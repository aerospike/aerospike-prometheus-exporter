package main

import (
	"regexp"
	"strings"

	"github.com/prometheus/client_golang/prometheus"

	log "github.com/sirupsen/logrus"
)

// Namespace raw metrics
var namespaceRawMetrics = map[string]metricType{
	"allow-nonxdr-writes":                    mtGauge,
	"allow-xdr-writes":                       mtGauge,
	"cache_read_pct":                         mtGauge,
	"cold-start-evict-ttl":                   mtGauge,
	"conflict-resolution-policy":             mtGauge,
	"conflict-resolve-writes":                mtGauge,
	"data-in-index":                          mtGauge,
	"default-ttl":                            mtGauge,
	"disable-write-dup-res":                  mtGauge,
	"disallow-null-setname":                  mtGauge,
	"enable-benchmarks-batch-sub":            mtGauge,
	"enable-benchmarks-read":                 mtGauge,
	"enable-benchmarks-udf-sub":              mtGauge,
	"enable-benchmarks-udf":                  mtGauge,
	"enable-benchmarks-write":                mtGauge,
	"enable-benchmarks-ops-sub":              mtGauge,
	"enable-hist-proxy":                      mtGauge,
	"enable-xdr":                             mtGauge,
	"evict_ttl":                              mtGauge,
	"geo2dsphere-within.earth-radius-meters": mtGauge,
	"geo2dsphere-within.level-mod":           mtGauge,
	"geo2dsphere-within.max-cells":           mtGauge,
	"geo2dsphere-within.max-level":           mtGauge,
	"geo2dsphere-within.min-level":           mtGauge,
	"geo2dsphere-within.strict":              mtGauge,
	"max-ttl":                                mtGauge,
	"migrate-order":                          mtGauge,
	"migrate-retransmit-ms":                  mtGauge,
	"migrate-sleep":                          mtGauge,
	"ns-forward-xdr-writes":                  mtGauge,
	"nsup_cycle_duration":                    mtGauge,
	"nsup_cycle_sleep_pct":                   mtGauge,
	"obj-size-hist-max":                      mtGauge,
	"partition-tree-locks":                   mtGauge,
	"partition-tree-sprigs":                  mtGauge,
	"rack-id":                                mtGauge,
	"read-consistency-level-override":        mtGauge,
	"sets-enable-xdr":                        mtGauge,
	"sindex.num-partitions":                  mtGauge,
	"single-bin":                             mtGauge,
	"storage-engine":                         mtGauge,
	"tomb-raider-eligible-age":               mtGauge,
	"tomb-raider-period":                     mtGauge,
	"truncate_lut":                           mtGauge,
	"write-commit-level-override":            mtGauge,
	"allow-ttl-without-nsup":                 mtGauge,
	"background-scan-max-rps":                mtGauge,
	"disable-cold-start-eviction":            mtGauge,
	"index-stage-size":                       mtGauge,
	"nsup-hist-period":                       mtGauge,
	"nsup-period":                            mtGauge,
	"nsup-threads":                           mtGauge,
	"prefer-uniform-balance":                 mtGauge,
	"reject-non-xdr-writes":                  mtGauge,
	"reject-xdr-writes":                      mtGauge,
	"single-scan-threads":                    mtGauge,
	"strong-consistency":                     mtGauge,
	"strong-consistency-allow-expunge":       mtGauge,
	"transaction-pending-limit":              mtGauge,
	"truncate-threads":                       mtGauge,
	"xdr-tomb-raider-period":                 mtGauge,
	"xdr-tomb-raider-threads":                mtGauge,
	"xdr-bin-tombstone-ttl":                  mtGauge,
	"ignore-migrate-fill-delay":              mtGauge,

	"clock_skew_stop_writes":                 mtGauge,
	"dead_partitions":                        mtGauge,
	"unavailable_partitions":                 mtGauge,
	"available_bin_names":                    mtGauge,
	"defrag_q":                               mtGauge,
	"pmem_available_pct":                     mtGauge,
	"pmem_free_pct":                          mtGauge,
	"pmem_total_bytes":                       mtGauge,
	"pmem_used_bytes":                        mtGauge,
	"pmem_compression_ratio":                 mtGauge,
	"device_available_pct":                   mtGauge,
	"device_compression_ratio":               mtGauge,
	"device_free_pct":                        mtGauge,
	"device_total_bytes":                     mtGauge,
	"device_used_bytes":                      mtGauge,
	"effective_is_quiesced":                  mtGauge,
	"effective_replication_factor":           mtGauge,
	"evict-hist-buckets":                     mtGauge,
	"evict-tenths-pct":                       mtGauge,
	"high-water-disk-pct":                    mtGauge,
	"high-water-memory-pct":                  mtGauge,
	"hwm_breached":                           mtGauge,
	"index_pmem_used_bytes":                  mtGauge,
	"index_pmem_used_pct":                    mtGauge,
	"index_flash_used_bytes":                 mtGauge,
	"index_flash_used_pct":                   mtGauge,
	"index_flash_alloc_bytes":                mtGauge,
	"index_flash_alloc_pct":                  mtGauge,
	"index-type.mounts-high-water-pct":       mtGauge,
	"index-type.mounts-size-limit":           mtGauge,
	"master_objects":                         mtGauge,
	"master_tombstones":                      mtGauge,
	"memory_free_pct":                        mtGauge,
	"memory_used_bytes":                      mtGauge,
	"memory_used_data_bytes":                 mtGauge,
	"memory_used_index_bytes":                mtGauge,
	"memory_used_set_index_bytes":            mtGauge,
	"memory_used_sindex_bytes":               mtGauge,
	"memory-size":                            mtGauge,
	"migrate_record_receives":                mtGauge,
	"migrate_record_retransmits":             mtGauge,
	"migrate_records_skipped":                mtGauge,
	"migrate_records_transmitted":            mtGauge,
	"migrate_rx_instances":                   mtGauge,
	"migrate_rx_partitions_active":           mtGauge,
	"migrate_rx_partitions_initial":          mtGauge,
	"migrate_rx_partitions_remaining":        mtGauge,
	"migrate_signals_active":                 mtGauge,
	"migrate_signals_remaining":              mtGauge,
	"migrate_tx_instances":                   mtGauge,
	"migrate_tx_partitions_active":           mtGauge,
	"migrate_tx_partitions_imbalance":        mtGauge,
	"migrate_tx_partitions_initial":          mtGauge,
	"migrate_tx_partitions_remaining":        mtGauge,
	"n_nodes_quiesced":                       mtGauge,
	"non_expirable_objects":                  mtGauge,
	"non_replica_objects":                    mtGauge,
	"non_replica_tombstones":                 mtGauge,
	"ns_cluster_size":                        mtGauge,
	"objects":                                mtGauge,
	"pending_quiesce":                        mtGauge,
	"prole_objects":                          mtGauge,
	"prole_tombstones":                       mtGauge,
	"replication-factor":                     mtGauge,
	"shadow_write_q":                         mtGauge,
	"stop_writes":                            mtGauge,
	"stop-writes-pct":                        mtGauge,
	"tombstones":                             mtGauge,
	"write_q":                                mtGauge,
	"xdr_tombstones":                         mtGauge,
	"xdr_bin_cemeteries":                     mtGauge,
	"record_proto_uncompressed_pct":          mtGauge,
	"record_proto_compression_ratio":         mtGauge,
	"scan_proto_uncompressed_pct":            mtGauge,
	"scan_proto_compression_ratio":           mtGauge,
	"query_proto_uncompressed_pct":           mtGauge,
	"query_proto_compression_ratio":          mtGauge,
	"effective_prefer_uniform_balance":       mtGauge,
	"migrate_tx_partitions_lead_remaining":   mtGauge,
	"appeals_tx_active":                      mtGauge,
	"appeals_rx_active":                      mtGauge,
	"appeals_tx_remaining":                   mtGauge,
	"truncated_records":                      mtCounter,
	"batch_sub_proxy_complete":               mtCounter,
	"batch_sub_proxy_error":                  mtCounter,
	"batch_sub_proxy_timeout":                mtCounter,
	"batch_sub_read_error":                   mtCounter,
	"batch_sub_read_not_found":               mtCounter,
	"batch_sub_read_success":                 mtCounter,
	"batch_sub_read_timeout":                 mtCounter,
	"batch_sub_read_filtered_out":            mtCounter,
	"batch_sub_tsvc_error":                   mtCounter,
	"batch_sub_tsvc_timeout":                 mtCounter,
	"client_delete_error":                    mtCounter,
	"client_delete_not_found":                mtCounter,
	"client_delete_success":                  mtCounter,
	"client_delete_timeout":                  mtCounter,
	"client_delete_filtered_out":             mtCounter,
	"client_lang_delete_success":             mtCounter,
	"client_lang_error":                      mtCounter,
	"client_lang_read_success":               mtCounter,
	"client_lang_write_success":              mtCounter,
	"client_proxy_complete":                  mtCounter,
	"client_proxy_error":                     mtCounter,
	"client_proxy_timeout":                   mtCounter,
	"client_read_error":                      mtCounter,
	"client_read_not_found":                  mtCounter,
	"client_read_success":                    mtCounter,
	"client_read_timeout":                    mtCounter,
	"client_read_filtered_out":               mtCounter,
	"client_tsvc_error":                      mtCounter,
	"client_tsvc_timeout":                    mtCounter,
	"client_udf_complete":                    mtCounter,
	"client_udf_error":                       mtCounter,
	"client_udf_timeout":                     mtCounter,
	"client_udf_filtered_out":                mtCounter,
	"client_write_error":                     mtCounter,
	"client_write_success":                   mtCounter,
	"client_write_timeout":                   mtCounter,
	"client_write_filtered_out":              mtCounter,
	"defrag_reads":                           mtCounter,
	"defrag_writes":                          mtCounter,
	"evicted_objects":                        mtCounter,
	"expired_objects":                        mtCounter,
	"fail_generation":                        mtCounter,
	"fail_key_busy":                          mtCounter,
	"fail_record_too_big":                    mtCounter,
	"fail_xdr_forbidden":                     mtCounter,
	"fail_client_lost_conflict":              mtCounter,
	"fail_xdr_lost_conflict":                 mtCounter,
	"from_proxy_delete_error":                mtCounter,
	"from_proxy_delete_not_found":            mtCounter,
	"from_proxy_delete_success":              mtCounter,
	"from_proxy_delete_timeout":              mtCounter,
	"from_proxy_delete_filtered_out":         mtCounter,
	"from_proxy_read_error":                  mtCounter,
	"from_proxy_read_not_found":              mtCounter,
	"from_proxy_read_success":                mtCounter,
	"from_proxy_read_timeout":                mtCounter,
	"from_proxy_read_filtered_out":           mtCounter,
	"from_proxy_tsvc_error":                  mtCounter,
	"from_proxy_tsvc_timeout":                mtCounter,
	"from_proxy_write_error":                 mtCounter,
	"from_proxy_write_success":               mtCounter,
	"from_proxy_write_timeout":               mtCounter,
	"from_proxy_write_filtered_out":          mtCounter,
	"from_proxy_udf_complete":                mtCounter,
	"from_proxy_udf_error":                   mtCounter,
	"from_proxy_udf_timeout":                 mtCounter,
	"from_proxy_udf_filtered_out":            mtCounter,
	"from_proxy_lang_read_success":           mtCounter,
	"from_proxy_lang_write_success":          mtCounter,
	"from_proxy_lang_delete_success":         mtCounter,
	"from_proxy_lang_error":                  mtCounter,
	"from_proxy_batch_sub_tsvc_error":        mtCounter,
	"from_proxy_batch_sub_tsvc_timeout":      mtCounter,
	"from_proxy_batch_sub_read_success":      mtCounter,
	"from_proxy_batch_sub_read_error":        mtCounter,
	"from_proxy_batch_sub_read_timeout":      mtCounter,
	"from_proxy_batch_sub_read_not_found":    mtCounter,
	"from_proxy_batch_sub_read_filtered_out": mtCounter,
	"geo_region_query_cells":                 mtCounter,
	"geo_region_query_falsepos":              mtCounter,
	"geo_region_query_points":                mtCounter,
	"geo_region_query_reqs":                  mtCounter,
	"query_agg_abort":                        mtCounter,
	"query_agg_avg_rec_count":                mtCounter,
	"query_agg_error":                        mtCounter,
	"query_agg_success":                      mtCounter,
	"query_agg":                              mtCounter,
	"query_fail":                             mtCounter,
	"query_long_queue_full":                  mtCounter,
	"query_long_reqs":                        mtCounter,
	"query_lookup_abort":                     mtCounter,
	"query_lookup_avg_rec_count":             mtCounter,
	"query_lookup_error":                     mtCounter,
	"query_lookup_success":                   mtCounter,
	"query_lookups":                          mtCounter,
	"query_reqs":                             mtCounter,
	"query_short_queue_full":                 mtCounter,
	"query_short_reqs":                       mtCounter,
	"query_udf_bg_failure":                   mtCounter,
	"query_udf_bg_success":                   mtCounter,
	"query_ops_bg_success":                   mtCounter,
	"query_ops_bg_failure":                   mtCounter,
	"retransmit_batch_sub_dup_res":           mtCounter,
	"retransmit_client_delete_dup_res":       mtCounter,
	"retransmit_client_delete_repl_write":    mtCounter,
	"retransmit_client_read_dup_res":         mtCounter,
	"retransmit_client_udf_dup_res":          mtCounter,
	"retransmit_client_udf_repl_write":       mtCounter,
	"retransmit_client_write_dup_res":        mtCounter,
	"retransmit_client_write_repl_write":     mtCounter,
	"retransmit_nsup_repl_write":             mtCounter,
	"retransmit_udf_sub_dup_res":             mtCounter,
	"retransmit_udf_sub_repl_write":          mtCounter,
	"scan_aggr_abort":                        mtCounter,
	"scan_aggr_complete":                     mtCounter,
	"scan_aggr_error":                        mtCounter,
	"scan_basic_abort":                       mtCounter,
	"scan_basic_complete":                    mtCounter,
	"scan_basic_error":                       mtCounter,
	"scan_udf_bg_abort":                      mtCounter,
	"scan_udf_bg_complete":                   mtCounter,
	"scan_udf_bg_error":                      mtCounter,
	"scan_ops_bg_complete":                   mtCounter,
	"scan_ops_bg_error":                      mtCounter,
	"scan_ops_bg_abort":                      mtCounter,
	"udf_sub_lang_delete_success":            mtCounter,
	"udf_sub_lang_error":                     mtCounter,
	"udf_sub_lang_read_success":              mtCounter,
	"udf_sub_lang_write_success":             mtCounter,
	"udf_sub_tsvc_error":                     mtCounter,
	"udf_sub_tsvc_timeout":                   mtCounter,
	"udf_sub_udf_complete":                   mtCounter,
	"udf_sub_udf_error":                      mtCounter,
	"udf_sub_udf_timeout":                    mtCounter,
	"udf_sub_udf_filtered_out":               mtCounter,
	"xdr_write_error":                        mtCounter,
	"xdr_write_success":                      mtCounter,
	"xdr_write_timeout":                      mtCounter,
	"appeals_records_exonerated":             mtCounter,
	"xdr_client_write_success":               mtCounter,
	"xdr_client_write_error":                 mtCounter,
	"xdr_client_write_timeout":               mtCounter,
	"xdr_client_delete_success":              mtCounter,
	"xdr_client_delete_error":                mtCounter,
	"xdr_client_delete_timeout":              mtCounter,
	"xdr_client_delete_not_found":            mtCounter,
	"xdr_from_proxy_write_success":           mtCounter,
	"xdr_from_proxy_write_error":             mtCounter,
	"xdr_from_proxy_write_timeout":           mtCounter,
	"xdr_from_proxy_delete_success":          mtCounter,
	"xdr_from_proxy_delete_error":            mtCounter,
	"xdr_from_proxy_delete_timeout":          mtCounter,
	"xdr_from_proxy_delete_not_found":        mtCounter,
	"ops_sub_tsvc_error":                     mtCounter,
	"ops_sub_tsvc_timeout":                   mtCounter,
	"ops_sub_write_success":                  mtCounter,
	"ops_sub_write_error":                    mtCounter,
	"ops_sub_write_timeout":                  mtCounter,
	"ops_sub_write_filtered_out":             mtCounter,
	"dup_res_ask":                            mtCounter,
	"dup_res_respond_read":                   mtCounter,
	"dup_res_respond_no_read":                mtCounter,
	"retransmit_all_read_dup_res":            mtCounter,
	"retransmit_all_write_dup_res":           mtCounter,
	"retransmit_all_write_repl_write":        mtCounter,
	"retransmit_all_delete_dup_res":          mtCounter,
	"retransmit_all_delete_repl_write":       mtCounter,
	"retransmit_all_udf_dup_res":             mtCounter,
	"retransmit_all_udf_repl_write":          mtCounter,
	"retransmit_all_batch_sub_dup_res":       mtCounter,
	"retransmit_ops_sub_dup_res":             mtCounter,
	"retransmit_ops_sub_repl_write":          mtCounter,
	"re_repl_success":                        mtCounter,
	"re_repl_error":                          mtCounter,
	"re_repl_timeout":                        mtCounter,
	"deleted_last_bin":                       mtCounter,
	"current_time":                           mtCounter,
	"evict_void_time":                        mtCounter,
	"smd_evict_void_time":                    mtCounter,

	"storage-engine.filesize":                  mtGauge, //=1073741824
	"storage-engine.write-block-size":          mtGauge, //=1048576
	"storage-engine.data-in-memory":            mtGauge, //=false
	"storage-engine.cold-start-empty":          mtGauge, //=false
	"storage-engine.commit-to-device":          mtGauge, //=false
	"storage-engine.commit-min-size":           mtGauge, //=0
	"storage-engine.compression-level":         mtGauge, //=0
	"storage-engine.defrag-lwm-pct":            mtGauge, //=50
	"storage-engine.defrag-queue-min":          mtGauge, //=0
	"storage-engine.defrag-sleep":              mtGauge, //=1000
	"storage-engine.defrag-startup-minimum":    mtGauge, //=10
	"storage-engine.direct-files":              mtGauge, //=false
	"storage-engine.enable-benchmarks-storage": mtGauge, //=false
	"storage-engine.flush-max-ms":              mtGauge, //=1000
	"storage-engine.max-write-cache":           mtGauge, //=67108864
	"storage-engine.min-avail-pct":             mtGauge, //=5
	"storage-engine.post-write-queue":          mtGauge, //=256
	"storage-engine.read-page-cache":           mtGauge, //=false
	"storage-engine.serialize-tomb-raider":     mtGauge, //=false
	"storage-engine.tomb-raider-sleep":         mtGauge, //=1000
	"storage-engine.cache-replica-writes":      mtGauge, //=false
	"storage-engine.disable-odsync":            mtGauge, //=false

	"storage-engine.file.age":              mtGauge,
	"storage-engine.file.defrag_q":         mtGauge,
	"storage-engine.file.defrag_reads":     mtGauge,
	"storage-engine.file.defrag_writes":    mtGauge,
	"storage-engine.file.free_wblocks":     mtGauge,
	"storage-engine.file.shadow_write_q":   mtGauge,
	"storage-engine.file.used_bytes":       mtGauge,
	"storage-engine.file.write_q":          mtGauge,
	"storage-engine.file.writes":           mtGauge,
	"storage-engine.device.age":            mtGauge,
	"storage-engine.device.defrag_q":       mtGauge,
	"storage-engine.device.defrag_reads":   mtGauge,
	"storage-engine.device.defrag_writes":  mtGauge,
	"storage-engine.device.free_wblocks":   mtGauge,
	"storage-engine.device.shadow_write_q": mtGauge,
	"storage-engine.device.used_bytes":     mtGauge,
	"storage-engine.device.write_q":        mtGauge,
	"storage-engine.device.writes":         mtGauge,
}

type NamespaceWatcher struct{}

func (nw *NamespaceWatcher) describe(ch chan<- *prometheus.Desc) {}

func (nw *NamespaceWatcher) passOneKeys() []string {
	return []string{"namespaces"}
}

func (nw *NamespaceWatcher) passTwoKeys(rawMetrics map[string]string) []string {
	s := rawMetrics["namespaces"]
	list := strings.Split(s, ";")

	var infoKeys []string
	for _, k := range list {
		infoKeys = append(infoKeys, "namespace/"+k)
	}

	return infoKeys
}

// Filtered namespace metrics. Populated by getFilteredMetrics() based on the config.Aerospike.NamespaceMetricsAllowlist, config.Aerospike.NamespaceMetricsBlocklist and namespaceRawMetrics.
var namespaceMetrics map[string]metricType

// Regex for identifying storage-engine stats.
var seDynamicExtractor = regexp.MustCompile(`storage\-engine\.(?P<type>file|device)\[(?P<idx>\d+)\]\.(?P<metric>.+)`)

func (nw *NamespaceWatcher) refresh(o *Observer, infoKeys []string, rawMetrics map[string]string, ch chan<- prometheus.Metric) error {
	if namespaceMetrics == nil {
		namespaceMetrics = getFilteredMetrics(namespaceRawMetrics, config.Aerospike.NamespaceMetricsAllowlist, config.Aerospike.NamespaceMetricsAllowlistEnabled, config.Aerospike.NamespaceMetricsBlocklist, config.Aerospike.NamespaceMetricsBlocklistEnabled)
	}

	for _, ns := range infoKeys {
		nsName := strings.ReplaceAll(ns, "namespace/", "")
		log.Tracef("namespace-stats:%s:%s", nsName, rawMetrics[ns])

		namespaceObserver := make(MetricMap, len(namespaceMetrics))
		for m, t := range namespaceMetrics {
			namespaceObserver[m] = makeMetric("aerospike_namespace", m, t, config.AeroProm.MetricLabels, "cluster_name", "service", "ns")
		}

		stats := parseStats(rawMetrics[ns], ";")
		for stat, pm := range namespaceObserver {
			v, exists := stats[stat]
			if !exists {
				// not found
				continue
			}

			pv, err := tryConvert(v)
			if err != nil {
				continue
			}

			ch <- prometheus.MustNewConstMetric(pm.desc, pm.valueType, pv, rawMetrics[ikClusterName], rawMetrics[ikService], nsName)
		}

		for stat, value := range stats {
			match := seDynamicExtractor.FindStringSubmatch(stat)
			if len(match) != 4 {
				continue
			}

			metricType := match[1]
			metricIndex := match[2]
			metricName := match[3]

			_, exists := namespaceMetrics["storage-engine."+metricType+"."+metricName]
			if !exists {
				continue
			}

			deviceOrFileName := stats["storage-engine."+metricType+"["+metricIndex+"]"]
			pm := makeMetric("aerospike_namespace", "storage-engine_"+metricType+"_"+metricName, mtGauge, config.AeroProm.MetricLabels, "cluster_name", "service", "ns", metricType+"_index", metricType)

			pv, err := tryConvert(value)
			if err != nil {
				continue
			}

			ch <- prometheus.MustNewConstMetric(pm.desc, pm.valueType, pv, rawMetrics[ikClusterName], rawMetrics[ikService], nsName, metricIndex, deviceOrFileName)
		}
	}

	return nil
}
