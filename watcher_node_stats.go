package main

import (
	"github.com/prometheus/client_golang/prometheus"

	log "github.com/sirupsen/logrus"
)

// Node raw metrics
// gauge: true, counter: false
var statsRawMetrics = map[string]metricType{
	"cluster_size":                          mtGauge,
	"cluster_min_compatibility_id":          mtGauge,
	"cluster_max_compatibility_id":          mtGauge,
	"cluster_integrity":                     mtGauge,
	"cluster_is_member":                     mtGauge,
	"cluster_clock_skew_stop_writes_sec":    mtGauge,
	"cluster_clock_skew_ms":                 mtGauge,
	"cluster_generation":                    mtCounter,
	"uptime":                                mtCounter,
	"system_total_cpu_pct":                  mtGauge,
	"system_user_cpu_pct":                   mtGauge,
	"system_kernel_cpu_pct":                 mtGauge,
	"process_cpu_pct":                       mtGauge,
	"system_free_mem_pct":                   mtGauge,
	"heap_allocated_kbytes":                 mtGauge,
	"heap_active_kbytes":                    mtGauge,
	"heap_mapped_kbytes":                    mtGauge,
	"heap_efficiency_pct":                   mtGauge,
	"heap_site_count":                       mtGauge,
	"objects":                               mtGauge,
	"tombstones":                            mtGauge,
	"tsvc_queue":                            mtGauge,
	"info_queue":                            mtGauge,
	"rw_in_progress":                        mtGauge,
	"proxy_in_progress":                     mtGauge,
	"tree_gc_queue":                         mtGauge,
	"client_connections":                    mtGauge,
	"heartbeat_connections":                 mtGauge,
	"fabric_connections":                    mtGauge,
	"client_connections_opened":             mtCounter,
	"client_connections_closed":             mtCounter,
	"heartbeat_connections_opened":          mtCounter,
	"heartbeat_connections_closed":          mtCounter,
	"fabric_connections_opened":             mtCounter,
	"fabric_connections_closed":             mtCounter,
	"batch_index_proto_uncompressed_pct":    mtGauge,
	"batch_index_proto_compression_ratio":   mtGauge,
	"heartbeat_received_self":               mtCounter,
	"heartbeat_received_foreign":            mtCounter,
	"reaped_fds":                            mtCounter,
	"info_complete":                         mtCounter,
	"demarshal_error":                       mtCounter,
	"early_tsvc_client_error":               mtCounter,
	"early_tsvc_from_proxy_error":           mtCounter,
	"early_tsvc_batch_sub_error":            mtCounter,
	"early_tsvc_from_proxy_batch_sub_error": mtCounter,
	"early_tsvc_udf_sub_error":              mtCounter,
	"early_tsvc_ops_sub_error":              mtCounter,
	"batch_index_initiate":                  mtCounter,
	"batch_index_queue":                     mtGauge,
	"batch_index_complete":                  mtCounter,
	"batch_index_error":                     mtCounter,
	"batch_index_timeout":                   mtCounter,
	"batch_index_delay":                     mtCounter,
	"batch_index_unused_buffers":            mtGauge,
	"batch_index_huge_buffers":              mtGauge,
	"batch_index_created_buffers":           mtGauge,
	"batch_index_destroyed_buffers":         mtCounter,
	"scans_active":                          mtGauge,
	"queries_active":                        mtGauge, // deprecated
	"long_queries_active":                   mtGauge,
	"query_short_running":                   mtCounter,
	"query_long_running":                    mtCounter,
	"sindex_ucgarbage_found":                mtCounter,
	"sindex_gc_retries":                     mtCounter,
	"sindex_gc_list_creation_time":          mtGauge,
	"sindex_gc_list_deletion_time":          mtGauge,
	"sindex_gc_objects_validated":           mtCounter,
	"sindex_gc_garbage_found":               mtCounter,
	"sindex_gc_garbage_cleaned":             mtCounter,
	"time_since_rebalance":                  mtGauge,
	"migrate_allowed":                       mtCounter,
	"migrate_partitions_remaining":          mtGauge,
	"fabric_bulk_send_rate":                 mtGauge,
	"fabric_bulk_recv_rate":                 mtGauge,
	"fabric_ctrl_send_rate":                 mtGauge,
	"fabric_ctrl_recv_rate":                 mtGauge,
	"fabric_meta_send_rate":                 mtGauge,
	"fabric_meta_recv_rate":                 mtGauge,
	"fabric_rw_send_rate":                   mtGauge,
	"fabric_rw_recv_rate":                   mtGauge,
	"dlog_used_objects":                     mtGauge,
	"dlog_free_pct":                         mtGauge,
	"dlog_logged":                           mtGauge,
	"dlog_relogged":                         mtCounter,
	"dlog_processed_main":                   mtCounter,
	"dlog_processed_replica":                mtCounter,
	"dlog_processed_link_down":              mtCounter,
	"dlog_overwritten_error":                mtCounter,
	"xdr_ship_success":                      mtCounter,
	"xdr_ship_delete_success":               mtCounter,
	"xdr_ship_source_error":                 mtCounter,
	"xdr_ship_destination_error":            mtCounter,
	"xdr_ship_destination_permanent_error":  mtCounter,
	"xdr_ship_fullrecord":                   mtCounter,
	"xdr_ship_bytes":                        mtCounter,
	"xdr_ship_inflight_objects":             mtGauge,
	"xdr_ship_outstanding_objects":          mtGauge,
	"xdr_ship_latency_avg":                  mtGauge,
	"xdr_ship_compression_avg_pct":          mtGauge,
	"xdr_read_success":                      mtCounter,
	"xdr_read_error":                        mtCounter,
	"xdr_read_notfound":                     mtCounter,
	"xdr_read_latency_avg":                  mtGauge,
	"xdr_read_active_avg_pct":               mtGauge,
	"xdr_read_idle_avg_pct":                 mtGauge,
	"xdr_read_reqq_used":                    mtGauge,
	"xdr_read_respq_used":                   mtGauge,
	"xdr_read_reqq_used_pct":                mtGauge,
	"xdr_read_txnq_used":                    mtGauge,
	"xdr_read_txnq_used_pct":                mtGauge,
	"xdr_relogged_incoming":                 mtGauge,
	"xdr_relogged_outgoing":                 mtGauge,
	"xdr_queue_overflow_error":              mtCounter,
	"xdr_active_failed_node_sessions":       mtGauge,
	"xdr_active_link_down_sessions":         mtGauge,
	"xdr_hotkey_fetch":                      mtCounter,
	"xdr_hotkey_skip":                       mtCounter,
	"xdr_unknown_namespace_error":           mtCounter,
	"xdr_timelag":                           mtGauge,
	"xdr_throughput":                        mtGauge,
	"xdr_global_lastshiptime":               mtGauge,
	"threads_joinable":                      mtGauge,
	"threads_detached":                      mtGauge,
	"threads_pool_total":                    mtGauge,
	"threads_pool_active":                   mtGauge,
	"failed_best_practices":                 mtGauge,
}

type StatsWatcher struct{}

func (sw *StatsWatcher) describe(ch chan<- *prometheus.Desc) {}

func (sw *StatsWatcher) passOneKeys() []string {
	return nil
}

func (sw *StatsWatcher) passTwoKeys(rawMetrics map[string]string) []string {
	return []string{"statistics"}
}

// Filtered node statistics. Populated by getFilteredMetrics() based on config.Aerospike.NodeMetricsAllowlist, config.Aerospike.NodeMetricsBlocklist and statsRawMetrics.
var nodeMetrics map[string]metricType

func (sw *StatsWatcher) refresh(o *Observer, infoKeys []string, rawMetrics map[string]string, ch chan<- prometheus.Metric) error {
	log.Tracef("node-stats:%s", rawMetrics["statistics"])

	if nodeMetrics == nil {
		nodeMetrics = getFilteredMetrics(statsRawMetrics, config.Aerospike.NodeMetricsAllowlist, config.Aerospike.NodeMetricsAllowlistEnabled, config.Aerospike.NodeMetricsBlocklist)
	}

	statsObserver := make(MetricMap, len(nodeMetrics))
	for m, t := range nodeMetrics {
		statsObserver[m] = makeMetric("aerospike_node_stats", m, t, config.AeroProm.MetricLabels, "cluster_name", "service")
	}

	stats := parseStats(rawMetrics["statistics"], ";")

	for stat, pm := range statsObserver {
		v, exists := stats[stat]
		if !exists {
			// not found
			continue
		}

		pv, err := tryConvert(v)
		if err != nil {
			continue
		}

		ch <- prometheus.MustNewConstMetric(pm.desc, pm.valueType, pv, rawMetrics[ikClusterName], rawMetrics[ikService])
	}

	return nil
}
