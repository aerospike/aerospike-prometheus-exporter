package main

import (
	"strings"

	"github.com/prometheus/client_golang/prometheus"

	log "github.com/sirupsen/logrus"
)

type LatencyWatcher struct {
}

var LatencyBenchmarks = make(map[string]float64)

func (lw *LatencyWatcher) describe(ch chan<- *prometheus.Desc) {}

func (lw *LatencyWatcher) passOneKeys() []string {
	// return []string{"build"}
	return nil
}

func (lw *LatencyWatcher) passTwoKeys(rawMetrics map[string]string) (latencyCommands []string) {

	// return if this feature is disabled.
	if config.Aerospike.DisableLatenciesMetrics {
		// disabled
		return nil
	}

	latencyCommands = []string{"latencies:", "latency:"}

	ok, err := buildVersionGreaterThanOrEqual(rawMetrics, "5.1.0.0")
	if err != nil {
		log.Warn(err)
		return latencyCommands
	}

	if ok {
		return lw.getLatenciesCommands(rawMetrics)
	}

	return []string{"latency:"}
}

func (lw *LatencyWatcher) refresh(o *Observer, infoKeys []string, rawMetrics map[string]string, ch chan<- prometheus.Metric) error {

	allowedLatenciesList := make(map[string]struct{})
	blockedLatenciessList := make(map[string]struct{})

	if config.Aerospike.LatenciesMetricsAllowlistEnabled {
		for _, allowedLatencies := range config.Aerospike.LatenciesMetricsAllowlist {
			allowedLatenciesList[allowedLatencies] = struct{}{}
		}
	}

	if len(config.Aerospike.LatenciesMetricsBlocklist) > 0 {
		for _, blockedLatencies := range config.Aerospike.LatenciesMetricsBlocklist {
			blockedLatenciessList[blockedLatencies] = struct{}{}
		}
	}

	// loop all the latency infokeys
	for ik := range infoKeys {
		parseSingleLatenciesKey(infoKeys[ik], rawMetrics, allowedLatenciesList, blockedLatenciessList, ch)
	}

	return nil
}

func parseSingleLatenciesKey(singleLatencyKey string, rawMetrics map[string]string,
	allowedLatenciesList map[string]struct{},
	blockedLatenciessList map[string]struct{}, ch chan<- prometheus.Metric) error {

	var latencyStats map[string]StatsMap

	if rawMetrics["latencies:"] != "" {
		latencyStats = parseLatencyInfo(rawMetrics[singleLatencyKey], int(config.Aerospike.LatencyBucketsCount))
	} else {
		latencyStats = parseLatencyInfoLegacy(rawMetrics["latency:"], int(config.Aerospike.LatencyBucketsCount))
	}

	log.Tracef("latency-stats:%+v", latencyStats)

	for namespaceName, nsLatencyStats := range latencyStats {
		for operation, opLatencyStats := range nsLatencyStats {

			// operation comes from server as histogram-names
			if config.Aerospike.LatenciesMetricsAllowlistEnabled {
				if _, ok := allowedLatenciesList[operation]; !ok {
					continue
				}
			}

			if len(config.Aerospike.LatenciesMetricsBlocklist) > 0 {
				if _, ok := blockedLatenciessList[operation]; ok {
					continue
				}
			}

			for i, labelValue := range opLatencyStats.(StatsMap)["bucketLabels"].([]string) {
				// aerospike_latencies_<operation>_<timeunit>_bucket metric - Less than or equal to histogram buckets

				pm := makeMetric("aerospike_latencies", operation+"_"+opLatencyStats.(StatsMap)["timeUnit"].(string)+"_bucket", mtGauge, config.AeroProm.MetricLabels, METRIC_LABEL_CLUSTER_NAME, METRIC_LABEL_SERVICE, METRIC_LABEL_NS, METRIC_LABEL_LE)
				ch <- prometheus.MustNewConstMetric(pm.desc, pm.valueType, opLatencyStats.(StatsMap)["bucketValues"].([]float64)[i], rawMetrics[ikClusterName], rawMetrics[ikService], namespaceName, labelValue)

				// aerospike_latencies_<operation>_<timeunit>_count metric
				if i == 0 {
					pm = makeMetric("aerospike_latencies", operation+"_"+opLatencyStats.(StatsMap)["timeUnit"].(string)+"_count", mtGauge, config.AeroProm.MetricLabels, METRIC_LABEL_CLUSTER_NAME, METRIC_LABEL_SERVICE, METRIC_LABEL_NS)
					ch <- prometheus.MustNewConstMetric(pm.desc, pm.valueType, opLatencyStats.(StatsMap)["bucketValues"].([]float64)[i], rawMetrics[ikClusterName], rawMetrics[ikService], namespaceName)
				}
			}
		}
	}

	return nil
}

// Utility methods
// checks if a stat can be considered for latency stat retrieval
func canConsiderLatencyCommand(stat string) bool {
	return (strings.Contains(stat, "enable-benchmarks-") ||
		strings.Contains(stat, "enable-hist-")) // hist-proxy & hist-info - both at service level
}

func (lw *LatencyWatcher) getLatenciesCommands(rawMetrics map[string]string) []string {
	var commands = []string{"latencies:"}

	// below latency-command are added to the auto-enabled list, i.e. latencies: command
	// re-repl is auto-enabled, but not coming as part of latencies: list, hence we are adding it explicitly
	//
	// Hashmap content format := namespace-<histogram-key> = <0/1>
	for ns_latency_enabled_benchmark := range LatencyBenchmarks {
		l_value := LatencyBenchmarks[ns_latency_enabled_benchmark]
		// only if enabled, fetch the metrics
		if l_value == 1 {
			// if enable-hist-proxy
			//    command = latencies:hist={test}-proxy
			// else if enable-benchmarks-fabric
			//    command = latencies:hist=benchmarks-fabric
			// else if re-repl
			//    command = latencies:hist={test}-re-repl

			if strings.Contains(ns_latency_enabled_benchmark, "re-repl") {
				// Exception case
				ns := strings.Split(ns_latency_enabled_benchmark, "-")[0]
				l_command := "latencies:hist={" + ns + "}-re-repl"
				commands = append(commands, l_command)
			} else if strings.Contains(ns_latency_enabled_benchmark, "enable-hist-proxy") {
				// Exception case
				ns := strings.Split(ns_latency_enabled_benchmark, "-")[0]
				l_command := "latencies:hist={" + ns + "}-proxy"
				commands = append(commands, l_command)
			} else if strings.Contains(ns_latency_enabled_benchmark, "enable-benchmarks-fabric") {
				// Exception case
				l_command := "latencies:hist=benchmarks-fabric"
				commands = append(commands, l_command)
			} else if strings.Contains(ns_latency_enabled_benchmark, "enable-hist-info") {
				// Exception case
				l_command := "latencies:hist=info"
				commands = append(commands, l_command)
			} else if strings.Contains(ns_latency_enabled_benchmark, "-benchmarks-") {
				// remaining enabled benchmark latencies like
				//         enable-benchmarks-fabric, enable-benchmarks-ops-sub, enable-benchmarks-read
				//         enable-benchmarks-write, enable-benchmarks-udf, enable-benchmarks-udf-sub, enable-benchmarks-batch-sub

				// format:= test-enable-benchmarks-read (or) test-enable-hist-proxy
				ns := strings.Split(ns_latency_enabled_benchmark, "-")[0]
				benchmarks_start_index := strings.LastIndex(ns_latency_enabled_benchmark, "-benchmarks-")
				l_command := ns_latency_enabled_benchmark[benchmarks_start_index:]
				l_command = "latencies:hist={" + ns + "}" + l_command
				commands = append(commands, l_command)
			}
		}
	}

	log.Tracef("latency-passtwokeys:%s", commands)

	return commands
}
