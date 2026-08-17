package exporter

import (
	"fmt"
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

The user tier - `ENGULA INFO` with no arguments - is what gets scraped by
default. The server also has a developer tier of per-bucket distributions and
task-level internals; --include-engula-debug-metrics switches the scrape to
`ENGULA INFO everything`, which returns both tiers in one reply. It replaces
the plain call rather than adding a second one, since `everything` is a
superset. Expect roughly three times the series.

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

// String-valued fields, which Prometheus cannot hold. They are collected while
// parsing and folded into one constant-1 gauge carrying them as labels, the
// same shape handleMetricsServer() builds redis_instance_info from. Order here
// is the label order of the emitted metric.
var engulaInfoLabels = []struct {
	field string
	label string
}{
	{"engula_rdb_last_load_mode", "mode"},
	{"engula_rdb_last_load_reason", "reason"},
}

const engulaInfoMetricName = "engula_rdb_last_load_info"

// Developer-tier names are unprefixed on the wire; see engulaDebugGauges.
const engulaDebugPrefix = "engula_"

/*
Developer tier. These names come straight from the server's internal structs
and carry no stability promise, so they are exported under an engula_ prefix
applied here: unprefixed, defrag_* would land in the redis_defrag_* namespace
that already holds jemalloc's active-defrag metrics.
*/

var engulaDebugGauges = []string{
	"earena_indexed_blocks",
	"earena_indexed_block_size_bytes",
	"earena_direct_blocks",
	"earena_index_bytes",

	"defrag_mutable_blocks",
	"defrag_immutable_recycling_blocks",

	"ec_dicts",
	"ec_dicts_cost_size_bytes",
	"ec_dicts_async_deleting",
	"ec_lines_skipped",
	"ec_lines_recycling",
}

var engulaDebugCounters = []string{
	"earena_block_size_bytes_total",
	"earena_remark_records_total",
	"earena_remark_records_size_bytes_total",

	"ea_alloc_records_total",
	"ea_alloc_record_size_bytes_total",
	"ea_create_records_total",
	"ea_create_record_size_bytes_total",
	"ea_free_records_total",
	"ea_async_free_records_total",
	"ea_find_total",
	"ea_blind_update_total",
	"ea_que_defrag_reqs_send_total",
	"ea_que_defrag_completions_recv_total",
	"ea_que_zip_reqs_send_total",
	"ea_que_zip_completions_recv_total",
	"ea_que_rezip_reqs_send_total",
	"ea_que_rezip_completions_recv_total",

	"defrag_task_create_requests_total",
	"defrag_task_create_succ_blocks_total",
	"defrag_completion_old_blocks_total",
	"defrag_completion_new_blocks_total",
	"defrag_completion_old_records_total",
	"defrag_completion_old_record_size_bytes_total",
	"defrag_completion_new_records_total",
	"defrag_completion_new_record_size_bytes_total",

	"ec_dicts_async_deleted_total",
	"ec_lines_created_total",
	"ec_lines_deleted_total",
	"ec_zip_completion_old_blocks_total",
	"ec_zip_completion_old_block_size_bytes_total",
	"ec_zip_completion_old_records_total",
	"ec_zip_completion_old_record_size_bytes_total",
	"ec_zip_completion_new_blocks_total",
	"ec_zip_completion_new_block_size_bytes_total",
	"ec_zip_completion_new_records_total",
	"ec_zip_completion_new_record_size_bytes_total",
	"ec_rezip_completion_old_blocks_total",
	"ec_rezip_completion_old_block_size_bytes_total",
	"ec_rezip_completion_old_records_total",
	"ec_rezip_completion_old_record_size_bytes_total",
	"ec_rezip_completion_new_blocks_total",
	"ec_rezip_completion_new_block_size_bytes_total",
	"ec_rezip_completion_new_records_total",
	"ec_rezip_completion_new_record_size_bytes_total",

	"coro_schedule_soft_limit_useconds_total",
	"coro_schedule_hard_limit_useconds_total",
	"coro_schedule_plan_useconds_total",
}

// Per-bucket gauges the server flattens into the metric name.
const (
	engulaDefragFillRateBuckets = 16
	engulaCompressNsizeBuckets  = 10
	engulaCoroBusyLevels        = 5
)

func isEngulaInfoField(field string) bool {
	for _, l := range engulaInfoLabels {
		if l.field == field {
			return true
		}
	}
	return false
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

	if !e.options.InclEngulaDebugMetrics {
		return
	}

	for _, name := range engulaDebugGauges {
		e.metricMapGauges[name] = engulaDebugPrefix + name
	}
	for _, name := range engulaDebugCounters {
		e.metricMapCounters[name] = engulaDebugPrefix + name
	}
	for i := 0; i < engulaDefragFillRateBuckets; i++ {
		name := fmt.Sprintf("defrag_fill_rate_bucket%d_blocks", i)
		e.metricMapGauges[name] = engulaDebugPrefix + name
	}
	for i := 0; i < engulaCompressNsizeBuckets; i++ {
		name := fmt.Sprintf("ec_lines_nsize_bucket%d", i)
		e.metricMapGauges[name] = engulaDebugPrefix + name
	}
	// The server flattens the level index onto _total, which would make
	// OpenMetrics append a second one. Rename to a well-formed counter.
	for i := 0; i < engulaCoroBusyLevels; i++ {
		e.metricMapCounters[fmt.Sprintf("coro_main_thread_busy_level_useconds_total%d", i)] =
			fmt.Sprintf("%scoro_main_thread_busy_level%d_useconds_total", engulaDebugPrefix, i)
	}
}

// The description is built here rather than through
// findOrCreateMetricDescription(), which would take the label *values* as the
// label *names*.
func (e *Exporter) registerEngulaInfoMetric(ch chan<- prometheus.Metric, values map[string]string) {
	labels := make([]string, 0, len(engulaInfoLabels))
	labelValues := make([]string, 0, len(engulaInfoLabels))
	for _, l := range engulaInfoLabels {
		labels = append(labels, l.label)
		labelValues = append(labelValues, values[l.field])
	}

	desc, found := e.metricDescriptions[engulaInfoMetricName]
	if !found {
		desc = newMetricDescr(e.options.Namespace, engulaInfoMetricName, "Engula's last RDB load", labels)
		e.metricDescriptions[engulaInfoMetricName] = desc
	}

	m, err := prometheus.NewConstMetric(desc, prometheus.GaugeValue, 1, labelValues...)
	if err != nil {
		log.Debugf("engula: NewConstMetric(%s) err: %s", engulaInfoMetricName, err)
		return
	}
	ch <- m
}

func (e *Exporter) extractEngulaMetrics(ch chan<- prometheus.Metric, c redis.Conn) {
	// "everything" is a superset of the default reply, so it replaces the
	// plain call instead of costing a second round trip.
	args := []interface{}{"INFO"}
	if e.options.InclEngulaDebugMetrics {
		args = append(args, "everything")
	}

	info, err := redis.String(doRedisCmd(c, "ENGULA", args...))
	if err != nil || info == "" {
		// Debug, not Info: the collector is on by default, so a plain Redis or
		// Valkey target would otherwise log an error on every scrape.
		log.Debugf("engula: ENGULA INFO err: %s", err)
		return
	}

	infoValues := map[string]string{}

	for _, line := range strings.Split(info, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") || len(line) < 2 || !strings.Contains(line, ":") {
			continue
		}

		split := strings.SplitN(line, ":", 2)
		fieldKey, fieldValue := split[0], split[1]

		if isEngulaInfoField(fieldKey) {
			infoValues[fieldKey] = strings.TrimSpace(fieldValue)
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

	if len(infoValues) > 0 {
		e.registerEngulaInfoMetric(ch, infoValues)
	}
}
