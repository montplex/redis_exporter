package exporter

import (
	"strconv"
	"strings"

	"github.com/gomodule/redigo/redis"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

/*
Engula metrics, scraped from the `ENGULA INFO` command of a valkey-engula
server. The reply is formatted like INFO (`# Section` headers, `key:value`
lines) but is a separate command, so the exporter's own INFO path never sees it.

Only the user tier is scraped: `ENGULA INFO` with no arguments. The server also
has a developer tier behind `ENGULA INFO debug`, deliberately not collected -
it carries per-bucket distributions and task-level internals that change
between releases and would roughly triple the series count.

Names already follow the server-side contract in the valkey-engula repo
(docs/engula-metrics-design.md): engula_ prefix, _total only on counters,
explicit units. They are therefore exported unchanged apart from the unit
conversions below, and land in Prometheus as redis_engula_*. The engula_ prefix
is what keeps them out of the redis_defrag_* namespace already used by
jemalloc's active-defrag metrics.
*/

// Gauges: current values. Anything not ending in _total.
var engulaGauges = []string{
	// memory
	"engula_arena_blocks",
	"engula_arena_block_bytes",
	"engula_arena_records",
	"engula_arena_record_bytes",
	"engula_arena_mutable_zone_bytes",

	// pipeline
	"engula_defrag_queue_length",
	"engula_zip_queue_length",
	"engula_rezip_queue_length",
	"engula_fast_completion_queue_length",
	"engula_slow_completion_queue_length",
	"engula_tasks_inflight",

	// evict
	"engula_mem_usage_pressure_ratio",
	"engula_mem_defrag_pressure_ratio",
	"engula_forecast_inflow_bytes_per_second",
	"engula_forecast_reclaim_bytes_per_second",
	"engula_forecast_target_bytes_per_second",
	"engula_recyclable_bytes",
	"engula_recycling_bytes",
	"engula_evict_timer_active",

	// health - per-load snapshots, not counters: they can decrease
	"engula_rdb_last_load_checked",
	"engula_rdb_last_load_dropped",
	"engula_rdb_last_load_orphans_reclaimed",

	// coro
	"engula_coro_busy_level",
}

// Counters: monotonic. Must not be registered as gauges - a _total name
// exported as a gauge is a naming violation, and the inverse makes the metric
// name differ between the text and OpenMetrics exposition formats.
var engulaCounters = []string{
	// pipeline
	"engula_defrag_inline_urgent_total",
	"engula_defrag_inline_main_thread_total",
	"engula_zip_inline_urgent_total",
	"engula_zip_inline_main_thread_total",
	"engula_rezip_inline_main_thread_total",
	"engula_defrag_tasks_created_total",
	"engula_defrag_tasks_skipped_threshold_total",
	"engula_defrag_tasks_skipped_insufficient_total",
	"engula_defrag_retired_bytes_total",
	"engula_defrag_produced_bytes_total",

	// evict
	"engula_evict_forecast_keys_total",
	"engula_evict_timer_starts_total",
	"engula_evict_urgent_defrag_timer_total",
	"engula_evict_urgent_defrag_wall_total",

	// compress
	"engula_compress_encode_errors_total",

	// coro
	"engula_coro_schedule_total",
}

// The server keeps human-readable units; Prometheus wants base units. Rename
// and scale on the way out, as the exporter already does for latest_fork_usec.
var engulaUnitConversions = map[string]struct {
	name    string
	divisor float64
}{
	"engula_coro_run_useconds_total":  {"engula_coro_run_seconds_total", 1e6},
	"engula_forecast_time_to_wall_ms": {"engula_forecast_time_to_wall_seconds", 1e3},
}

// Metrics whose value is a string. Prometheus cannot hold one, so they become a
// constant-1 gauge carrying the strings as labels, like redis_instance_info.
// The server emits them as `name:k1=v1,k2=v2`; the label names come from the
// reply, so a server that adds a key does not need an exporter change.
var engulaInfoMetrics = map[string]bool{
	"engula_rdb_last_load_info": true,
}

func (e *Exporter) registEngulaMetrics() {
	for _, name := range engulaGauges {
		e.metricMapGauges[name] = name
	}
	for _, name := range engulaCounters {
		e.metricMapCounters[name] = name
	}
	for src, conv := range engulaUnitConversions {
		if strings.HasSuffix(src, "_total") {
			e.metricMapCounters[src] = conv.name
		} else {
			e.metricMapGauges[src] = conv.name
		}
	}
}

// Emits `name{k1="v1",k2="v2"} 1` from a `k1=v1,k2=v2` value. Label names and
// values are ordered as the server wrote them, which is stable per metric.
func (e *Exporter) registerEngulaInfoMetric(ch chan<- prometheus.Metric, metricName, fieldValue string) {
	var names, values []string
	for _, pair := range strings.Split(fieldValue, ",") {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) != 2 {
			log.Debugf("engula: %s has a malformed label pair %q", metricName, pair)
			return
		}
		names = append(names, strings.TrimSpace(kv[0]))
		values = append(values, strings.TrimSpace(kv[1]))
	}
	if len(names) == 0 {
		return
	}

	// findOrCreateMetricDescription() would take the label *values* as the
	// label *names*, so the description has to be built here.
	desc, found := e.metricDescriptions[metricName]
	if !found {
		desc = newMetricDescr(e.options.Namespace, metricName, metricName+" metric", names)
		e.metricDescriptions[metricName] = desc
	}

	m, err := prometheus.NewConstMetric(desc, prometheus.GaugeValue, 1, values...)
	if err != nil {
		log.Debugf("engula: NewConstMetric(%s) err: %s", metricName, err)
		return
	}
	ch <- m
}

func (e *Exporter) extractEngulaMetrics(ch chan<- prometheus.Metric, c redis.Conn) {
	info, err := redis.String(doRedisCmd(c, "ENGULA", "INFO"))
	if err != nil || info == "" {
		// Debug, not Info: the collector is on by default, so a plain Redis or
		// Valkey target would otherwise log an error on every scrape.
		log.Debugf("engula: ENGULA INFO err: %s", err)
		return
	}

	for _, line := range strings.Split(info, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") || len(line) < 2 || !strings.Contains(line, ":") {
			continue
		}

		split := strings.SplitN(line, ":", 2)
		fieldKey, fieldValue := split[0], split[1]

		if engulaInfoMetrics[fieldKey] {
			e.registerEngulaInfoMetric(ch, fieldKey, fieldValue)
			continue
		}

		if !e.includeMetric(fieldKey) {
			continue
		}

		if conv, ok := engulaUnitConversions[fieldKey]; ok {
			val, err := strconv.ParseFloat(fieldValue, 64)
			if err != nil {
				log.Debugf("engula: couldn't parse %s: %s", fieldKey, err)
				continue
			}
			// time_to_wall is -1 while the forecast controller is inactive.
			// Absent beats a sentinel: a gap reads as "no prediction" and
			// keeps threshold alerts from matching on a negative value.
			if val < 0 {
				continue
			}
			valType := prometheus.GaugeValue
			if strings.HasSuffix(conv.name, "_total") {
				valType = prometheus.CounterValue
			}
			e.registerConstMetric(ch, conv.name, val/conv.divisor, valType)
			continue
		}

		e.parseAndRegisterConstMetric(ch, fieldKey, fieldValue)
	}
}
