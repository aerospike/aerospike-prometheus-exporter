package watchers

import (
	"fmt"
	"strings"

	commons "github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"

	log "github.com/sirupsen/logrus"
)

type LatencyWatcher struct {
}

var LatencyBenchmarks = make(map[string]float64)

func (lw *LatencyWatcher) PassOneKeys() []string {
	// return []string{"build"}
	log.Tracef("latency-passonekeys:nil")

	return nil
}

func (lw *LatencyWatcher) PassTwoKeys(rawMetrics map[string]string) (latencyCommands []string) {

	// return if this feature is disabled.
	if config.Cfg.Aerospike.DisableLatenciesMetrics {
		// disabled
		return nil
	}

	latencyCommands = []string{"latencies:", "latency:"}

	ok, err := commons.BuildVersionGreaterThanOrEqual(rawMetrics, "5.1.0.0")
	if err != nil {
		log.Warn(err)
		return latencyCommands
	}

	if ok {
		return lw.getLatenciesCommands(rawMetrics)
	}

	// legacy / old version
	log.Tracef("latency-passtwokeys:%s", []string{"latency:"})
	return []string{"latency:"}
}

func (lw *LatencyWatcher) getLatenciesCommands(rawMetrics map[string]string) []string {
	var commands = []string{"latencies:"}

	// Hashmap content format := namespace-<histogram-key> = <0/1>
	for ns_latency_enabled_benchmark := range LatencyBenchmarks {
		l_value := LatencyBenchmarks[ns_latency_enabled_benchmark]
		// only if enabled, fetch the metrics
		if l_value == 1 {
			// format:= test-enable-benchmarks-read (or) test-enable-hist-proxy
			ns := strings.Split(ns_latency_enabled_benchmark, "-")[0]
			benchmarks_start_index := strings.LastIndex(ns_latency_enabled_benchmark, "-benchmarks-")
			l_command := ns_latency_enabled_benchmark[benchmarks_start_index:]
			l_command = "latencies:hist={" + ns + "}" + l_command
			// fmt.Println("ns_latency_enabled_benchmark: "+ns_latency_enabled_benchmark+"\t cmd: ", l_command)
			commands = append(commands, l_command)
		}
	}

	// Exceptions,
	// 1. re-repl ( is auto-enabled by default, but not returned in namespace configs )
	// 2. enable-hist-proxy -- this is not having same name pattern as enable-benchmarks-<operation>
	commands = append(commands, "latencies:hist={test}-re-repl")
	if LatencyBenchmarks["enable-hist-proxy"] == 1 { // only if enabled
		commands = append(commands, "latencies:hist={test}-proxy")
	}

	log.Tracef("latency-passtwokeys:%s", commands)
	return commands
}

func (lw *LatencyWatcher) Refresh(infoKeys []string, rawMetrics map[string]string) ([]AerospikeStat, error) {

	allowedLatenciesList := make(map[string]struct{})
	blockedLatenciessList := make(map[string]struct{})

	if config.Cfg.Aerospike.LatenciesMetricsAllowlistEnabled {
		for _, allowedLatencies := range config.Cfg.Aerospike.LatenciesMetricsAllowlist {
			allowedLatenciesList[allowedLatencies] = struct{}{}
		}
	}

	if len(config.Cfg.Aerospike.LatenciesMetricsBlocklist) > 0 {
		for _, blockedLatencies := range config.Cfg.Aerospike.LatenciesMetricsBlocklist {
			blockedLatenciessList[blockedLatencies] = struct{}{}
		}
	}

	// log.Tracef("latencies-stats:latencies:hist={test}-benchmarks-read -- %+v", rawMetrics["latencies:hist={test}-benchmarks-read"])
	log.Tracef("latency-stats:%+v", rawMetrics["latency:"])
	var metrics_to_send = []AerospikeStat{}

	// loop all the latency infokeys
	for ik := range infoKeys {
		l_metrics_to_send := parseSingleLatenciesKey(infoKeys[ik], rawMetrics, allowedLatenciesList, blockedLatenciessList)
		metrics_to_send = append(metrics_to_send, l_metrics_to_send...)
	}

	return metrics_to_send, nil
}

func parseSingleLatenciesKey(singleLatencyKey string, rawMetrics map[string]string,
	allowedLatenciesList map[string]struct{}, blockedLatenciessList map[string]struct{}) []AerospikeStat {

	var latencyStats map[string]LatencyStatsMap

	if rawMetrics["latencies:"] != "" {
		// in latest aerospike server>5.1 version, latencies: will always come as infokey, so no need to check other conditions
		latencyStats = parseLatencyInfo(rawMetrics[singleLatencyKey], int(config.Cfg.Aerospike.LatencyBucketsCount))
	} else {
		latencyStats = parseLatencyInfoLegacy(rawMetrics["latency:"], int(config.Cfg.Aerospike.LatencyBucketsCount))
	}

	// log.Tracef("latency-stats:%+v", latencyStats)
	log.Tracef("latencies-stats:%+v:%+v", singleLatencyKey, rawMetrics[singleLatencyKey])

	var metrics_to_send = []AerospikeStat{}

	for namespaceName, nsLatencyStats := range latencyStats {
		fmt.Println("watcher-latency: namespaceName: ", namespaceName, "\t singleLatencyKey: ", singleLatencyKey,
			"\n\t rawMetrics[singleLatencyKey]: ", rawMetrics[singleLatencyKey], "\n\tnsLatencyStats: ", nsLatencyStats)
		for operation, opLatencyStats := range nsLatencyStats {

			// operation comes from server as histogram-names
			if config.Cfg.Aerospike.LatenciesMetricsAllowlistEnabled {
				if _, ok := allowedLatenciesList[operation]; !ok {
					continue
				}
			}

			if len(config.Cfg.Aerospike.LatenciesMetricsBlocklist) > 0 {
				if _, ok := blockedLatenciessList[operation]; ok {
					continue
				}
			}

			for i, labelValue := range opLatencyStats.(LatencyStatsMap)["bucketLabels"].([]string) {
				// aerospike_latencies_<operation>_<timeunit>_bucket metric - Less than or equal to histogram buckets

				labels := []string{commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE, commons.METRIC_LABEL_NS, commons.METRIC_LABEL_LE}
				labelValues := []string{ClusterName, Service, namespaceName, labelValue}
				pv := opLatencyStats.(LatencyStatsMap)["bucketValues"].([]float64)[i]

				// pm := makeMetric("aerospike_latencies", operation+"_"+opLatencyStats.(commons.StatsMap)["timeUnit"].(string)+"_bucket", mtGauge, config.AeroProm.MetricLabels, METRIC_LABEL_CLUSTER_NAME, METRIC_LABEL_SERVICE, METRIC_LABEL_NS, METRIC_LABEL_LE)
				// ch <- prometheus.MustNewConstMetric(pm.desc, pm.valueType, pv, rawMetrics[ikClusterName], rawMetrics[ikService], namespaceName, labelValue)
				asMetric := NewAerospikeStat(commons.CTX_LATENCIES, operation+"_"+opLatencyStats.(LatencyStatsMap)["timeUnit"].(string)+"_bucket")
				// asMetric.updateValues(pv, labels, labelValues)
				asMetric.updateValues(pv, labels, labelValues)
				metrics_to_send = append(metrics_to_send, asMetric)

				// aerospike_latencies_<operation>_<timeunit>_count metric
				if i == 0 {
					labels := []string{commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE, commons.METRIC_LABEL_NS}
					labelValues := []string{ClusterName, Service, namespaceName}
					pv := opLatencyStats.(LatencyStatsMap)["bucketValues"].([]float64)[i]

					// pm = makeMetric("aerospike_latencies", operation+"_"+opLatencyStats.(commons.StatsMap)["timeUnit"].(string)+"_count", mtGauge, config.AeroProm.MetricLabels, METRIC_LABEL_CLUSTER_NAME, METRIC_LABEL_SERVICE, METRIC_LABEL_NS)
					// ch <- prometheus.MustNewConstMetric(pm.desc, pm.valueType, pv, rawMetrics[ikClusterName], rawMetrics[ikService], namespaceName)
					asMetric := NewAerospikeStat(commons.CTX_LATENCIES, operation+"_"+opLatencyStats.(LatencyStatsMap)["timeUnit"].(string)+"_count")
					asMetric.updateValues(pv, labels, labelValues)
					// metrics_to_send = append(metrics_to_send, WatcherMetric{asMetric, pv, labels, labelValues})
					// asMetric.updateValues(pv, labels, labelValues)
					metrics_to_send = append(metrics_to_send, asMetric)

				}
			}
		}
	}

	return metrics_to_send
}
