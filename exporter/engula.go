package exporter

import (
	"strings"

	"github.com/gomodule/redigo/redis"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

func (e *Exporter) registEngulaMetrics() {
	// EAllocator
	e.metricMapCounters["ea_alloc_records_total"] = "ea_alloc_records_total"
	e.metricMapCounters["ea_alloc_record_size_bytes_total"] = "ea_alloc_record_size_bytes_total"
	e.metricMapCounters["ea_create_records_total"] = "ea_create_records_total"
	e.metricMapCounters["ea_create_record_size_bytes_total"] = "ea_create_record_size_bytes_total"
	e.metricMapCounters["ea_free_records_total"] = "ea_free_records_total"
	e.metricMapCounters["ea_free_record_size_bytes_total"] = "ea_free_record_size_bytes_total"
	e.metricMapCounters["ea_async_free_records_total"] = "ea_async_free_records_total"
	e.metricMapCounters["ea_find_total"] = "ea_find_total"
	e.metricMapCounters["ea_blind_update_total"] = "ea_blind_update_total"
	e.metricMapCounters["ea_rmw_update_total"] = "ea_rmw_update_total"

	e.metricMapCounters["ea_que_defrag_req_sends_total"] = "ea_que_defrag_req_sends_total"
	e.metricMapCounters["ea_que_defrag_completions_total"] = "ea_que_defrag_completions_total"
	e.metricMapCounters["ea_que_zip_req_sents_total"] = "ea_que_zip_req_sents_total"
	e.metricMapCounters["ea_que_zip_req_main_thread_total"] = "ea_que_zip_req_main_thread_total"
	e.metricMapCounters["ea_que_zip_completions_total"] = "ea_que_zip_completions_total"
	e.metricMapCounters["ea_que_rezip_req_sents_total"] = "ea_que_rezip_req_sents_total"
	e.metricMapCounters["ea_que_rezip_completions_total"] = "ea_que_rezip_completions_total"

	// EArena.Overall
	e.metricMapCounters["earena_block_size_bytes_total"] = "earena_block_size_bytes_total"
	e.metricMapGauges["earena_mutable_zone_bytes"] = "earena_mutable_zone_bytes"
	e.metricMapGauges["earena_indexed_blocks"] = "earena_indexed_blocks"
	e.metricMapGauges["earena_indexed_block_size_bytes"] = "earena_indexed_block_size_bytes"
	e.metricMapGauges["earena_direct_blocks"] = "earena_direct_blocks"
	e.metricMapGauges["earena_blocks"] = "earena_blocks"
	e.metricMapGauges["earena_block_size_bytes"] = "earena_block_size_bytes"
	e.metricMapGauges["earena_records"] = "earena_records"
	e.metricMapGauges["earena_record_size_bytes"] = "earena_record_size_bytes"

	e.metricMapCounters["earena_remark_records_total"] = "earena_remark_records_total"
	e.metricMapCounters["earena_remark_records_size_bytes_total"] = "earena_remark_records_size_bytes_total"

	// EArena.Defragmation
	e.metricMapGauges["defrag_fill_rate_bucket0_blocks"] = "defrag_fill_rate_bucket0_blocks"
	e.metricMapGauges["defrag_fill_rate_bucket1_blocks"] = "defrag_fill_rate_bucket1_blocks"
	e.metricMapGauges["defrag_fill_rate_bucket2_blocks"] = "defrag_fill_rate_bucket2_blocks"
	e.metricMapGauges["defrag_fill_rate_bucket3_blocks"] = "defrag_fill_rate_bucket3_blocks"
	e.metricMapGauges["defrag_fill_rate_bucket4_blocks"] = "defrag_fill_rate_bucket4_blocks"
	e.metricMapGauges["defrag_fill_rate_bucket5_blocks"] = "defrag_fill_rate_bucket5_blocks"
	e.metricMapGauges["defrag_fill_rate_bucket6_blocks"] = "defrag_fill_rate_bucket6_blocks"
	e.metricMapGauges["defrag_fill_rate_bucket7_blocks"] = "defrag_fill_rate_bucket7_blocks"
	e.metricMapGauges["defrag_fill_rate_bucket8_blocks"] = "defrag_fill_rate_bucket8_blocks"
	e.metricMapGauges["defrag_fill_rate_bucket9_blocks"] = "defrag_fill_rate_bucket9_blocks"

	e.metricMapCounters["defrag_task_create_requests_total"] = "defrag_task_create_requests_total"
	e.metricMapCounters["defrag_task_create_succ_total"] = "defrag_task_create_succ_total"
	e.metricMapCounters["defrag_task_create_succ_blocks_total"] = "defrag_task_create_succ_blocks_total"

	e.metricMapCounters["defrag_task_create_skip_threshold_total"] = "defrag_task_create_skip_threshold_total"
	e.metricMapCounters["defrag_task_create_skip_busy_total"] = "defrag_task_create_skip_busy_total"
	e.metricMapCounters["defrag_task_create_skip_insufficient_total"] = "defrag_task_create_skip_insufficient_total"
	e.metricMapCounters["defrag_task_create_stop_mutable_zone_total"] = "defrag_task_create_stop_mutable_zone_total"
	e.metricMapCounters["defrag_task_create_stop_busy_total"] = "defrag_task_create_stop_busy_total"
	e.metricMapCounters["defrag_task_create_stop_zip_dict_total"] = "defrag_task_create_stop_zip_dict_total"

	e.metricMapCounters["defrag_completion_old_blocks_total"] = "defrag_completion_old_blocks_total"
	e.metricMapCounters["defrag_completion_old_block_size_bytes_total"] = "defrag_completion_old_block_size_bytes_total"
	e.metricMapCounters["defrag_completion_new_blocks_total"] = "defrag_completion_new_blocks_total"
	e.metricMapCounters["defrag_completion_new_block_size_bytes_total"] = "defrag_completion_new_block_size_bytes_total"

	e.metricMapCounters["defrag_completion_old_records_total"] = "defrag_completion_old_records_total"
	e.metricMapCounters["defrag_completion_old_record_size_bytes_total"] = "defrag_completion_old_record_size_bytes_total"
	e.metricMapCounters["defrag_completion_new_records_total"] = "defrag_completion_new_records_total"
	e.metricMapCounters["defrag_completion_new_record_size_bytes_total"] = "defrag_completion_new_record_size_bytes_total"

	// EArena.Compress
	e.metricMapGauges["ec_dicts"] = "ec_dicts"
	e.metricMapGauges["ec_dict_cost_size_bytes"] = "ec_dict_cost_size_bytes"
	e.metricMapGauges["ec_dict_nsize_bucket0_dicts"] = "ec_dict_nsize_bucket0_dicts"
	e.metricMapGauges["ec_dict_nsize_bucket1_dicts"] = "ec_dict_nsize_bucket1_dicts"
	e.metricMapGauges["ec_dict_nsize_bucket2_dicts"] = "ec_dict_nsize_bucket2_dicts"
	e.metricMapGauges["ec_dict_nsize_bucket3_dicts"] = "ec_dict_nsize_bucket3_dicts"
	e.metricMapGauges["ec_dict_nsize_bucket4_dicts"] = "ec_dict_nsize_bucket4_dicts"
	e.metricMapGauges["ec_dict_nsize_bucket5_dicts"] = "ec_dict_nsize_bucket5_dicts"
	e.metricMapGauges["ec_dict_nsize_bucket6_dicts"] = "ec_dict_nsize_bucket6_dicts"
	e.metricMapGauges["ec_dict_nsize_bucket7_dicts"] = "ec_dict_nsize_bucket7_dicts"
	e.metricMapGauges["ec_dict_nsize_bucket8_dicts"] = "ec_dict_nsize_bucket8_dicts"
	e.metricMapGauges["ec_dict_nsize_bucket9_dicts"] = "ec_dict_nsize_bucket9_dicts"
	e.metricMapGauges["ec_dict_pending_delete_dicts"] = "ec_dict_pending_delete_dicts"
	e.metricMapGauges["ec_dict_async_deleted_dicts"] = "ec_dict_async_deleted_dicts"

	e.metricMapCounters["ec_zip_completion_old_blocks_total"] = "ec_zip_completion_old_blocks_total"
	e.metricMapCounters["ec_zip_completion_old_block_size_bytes_total"] = "ec_zip_completion_old_block_size_bytes_total"
	e.metricMapCounters["ec_zip_completion_old_records_total"] = "ec_zip_completion_old_records_total"
	e.metricMapCounters["ec_zip_completion_old_record_size_bytes_total"] = "ec_zip_completion_old_record_size_bytes_total"
	e.metricMapCounters["ec_zip_completion_new_blocks_total"] = "ec_zip_completion_new_blocks_total"
	e.metricMapCounters["ec_zip_completion_new_block_size_bytes_total"] = "ec_zip_completion_new_block_size_bytes_total"
	e.metricMapCounters["ec_zip_completion_new_records_total"] = "ec_zip_completion_new_records_total"
	e.metricMapCounters["ec_zip_completion_new_record_size_bytes_total"] = "ec_zip_completion_new_record_size_bytes_total"

	e.metricMapCounters["ec_rezip_completion_old_blocks_total"] = "ec_rezip_completion_old_blocks_total"
	e.metricMapCounters["ec_rezip_completion_old_block_size_bytes_total"] = "ec_rezip_completion_old_block_size_bytes_total"
	e.metricMapCounters["ec_rezip_completion_old_records_total"] = "ec_rezip_completion_old_records_total"
	e.metricMapCounters["ec_rezip_completion_old_record_size_bytes_total"] = "ec_rezip_completion_old_record_size_bytes_total"
	e.metricMapCounters["ec_rezip_completion_new_blocks_total"] = "ec_rezip_completion_new_blocks_total"
	e.metricMapCounters["ec_rezip_completion_new_block_size_bytes_total"] = "ec_rezip_completion_new_block_size_bytes_total"
	e.metricMapCounters["ec_rezip_completion_new_records_total"] = "ec_rezip_completion_new_records_total"
	e.metricMapCounters["ec_rezip_completion_new_record_size_bytes_total"] = "ec_rezip_completion_new_record_size_bytes_total"

	// EArena.Defragmation.Extra
	e.metricMapGauges["defrag_zmalloc_mem_used_bytes"] = "defrag_zmalloc_mem_used_bytes"
	e.metricMapGauges["defrag_overhead_bytes"] = "defrag_overhead_bytes"
	e.metricMapGauges["defrag_logic_mem_used_bytes"] = "defrag_logic_mem_used_bytes"
	e.metricMapGauges["defrag_mem_tofree_bytes"] = "defrag_mem_tofree_bytes"
	e.metricMapGauges["defrag_mutable_blocks_size_bytes"] = "defrag_mutable_blocks_size_bytes"
	e.metricMapGauges["defrag_immutable_blocks_size_bytes"] = "defrag_immutable_blocks_size_bytes"
	e.metricMapGauges["defrag_recyclable_space_bytes"] = "defrag_recyclable_space_bytes"

	e.metricMapCounters["defrag_triggered_by_cron_total"] = "defrag_triggered_by_cron_total"
	e.metricMapCounters["defrag_triggered_by_free_total"] = "defrag_triggered_by_free_total"
	e.metricMapCounters["defrag_triggered_by_blind_update_total"] = "defrag_triggered_by_blind_update_total"
	e.metricMapCounters["ea_que_defrag_send_reqs_total"] = "ea_que_defrag_send_reqs_total"
	e.metricMapCounters["ea_que_defrag_direct_reqs_total"] = "ea_que_defrag_direct_reqs_total"

	// EArena.Evict
	e.metricMapGauges["evict_timer_status"] = "evict_timer_status"
	e.metricMapCounters["evict_timer_start_total"] = "evict_timer_start_total"
	e.metricMapCounters["evict_defrag_triggered_by_evict_timer_total"] = "evict_defrag_triggered_by_evict_timer_total"
	e.metricMapCounters["evict_defrag_triggered_by_evict_func_total"] = "evict_defrag_triggered_by_evict_func_total"
}

func (e *Exporter) extractEngulaMetrics(ch chan<- prometheus.Metric, c redis.Conn) {
	mcInfo, err := redis.String(doRedisCmd(c, "ENGULA", "INFO"))
	if err != nil || mcInfo == "" {
		log.Infof("ENGULA INFO err: %s", err)
		return
	}
	// log.Infof("ENGULA INFO result: [%#v]", mcInfo)

	fieldClass := ""
	lines := strings.Split(mcInfo, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) > 0 && strings.HasPrefix(line, "# ") {
			fieldClass = line[2:]
			log.Debugf("set fieldClass: %s", fieldClass)
			continue
		}

		if (len(line) < 2) || (!strings.Contains(line, ":")) {
			continue
		}

		split := strings.SplitN(line, ":", 2)
		fieldKey := split[0]
		fieldValue := split[1]

		if !e.includeMetric(fieldKey) {
			continue
		}

		e.parseAndRegisterConstMetric(ch, fieldKey, fieldValue)
	}
}
