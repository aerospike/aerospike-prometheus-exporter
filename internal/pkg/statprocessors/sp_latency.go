package statprocessors

import (
	"strings"

	commons "github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	config "github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"

	log "github.com/sirupsen/logrus"
)

type LatencyStatsProcessor struct {
}

func (lw *LatencyStatsProcessor) PassOneKeys() []string {
	// return []string{"build"}
	log.Tracef("latency-passonekeys:nil")

	return nil
}

func (lw *LatencyStatsProcessor) PassTwoKeys(passOneStats map[string]string) (latencyCommands []string) {

	// return if this feature is disabled.
	if config.Cfg.Aerospike.DisableLatenciesMetrics {
		// disabled
		return nil
	}

	// latencyCommands = []string{"latencies:", "latency:"}

	ge, err := isBuildVersionGreaterThanOrEqual(passOneStats["build"], "5.1.0.0")

	if err != nil {
		return nil
	}

	if ge {
		return lw.getLatenciesCommands(passOneStats)
	}

	// legacy / old version
	log.Tracef("latency-passtwokeys:%s", []string{"latency:"})
	return []string{"latency:"}
}

func (lw *LatencyStatsProcessor) getLatenciesCommands(passOneStats map[string]string) []string {
	var commands = []string{"latencies:"}

	// below latency-command are added to the auto-enabled list, i.e. latencies: command
	// re-repl is auto-enabled, but not coming as part of latencies: list, hence we are adding it explicitly
	// command will be like
	//      latencies:hist={NAMESPACE}-proxy / latencies:hist={NAMESPACE}-benchmarks-read
	//      latencies:hist=info

	for nsName := range NamespaceLatencyBenchmarks {
		for _, latencyCommand := range NamespaceLatencyBenchmarks[nsName] {
			histCommand := "latencies:hist=" + latencyCommand
			commands = append(commands, histCommand)
		}
	}

	for _, latencyCommand := range ServiceLatencyBenchmarks {
		histCommand := "latencies:hist=" + latencyCommand
		commands = append(commands, histCommand)
	}

	log.Tracef("latency-getLatenciesCommands:%s", commands)

	return commands
}

// checks if a stat can be considered for latency stat retrieval
func isStatLatencyHistRelated(stat string) bool {
	// is not enable-benchmarks-storage and (enable-benchmarks-* or enable-hist-*)
	return (!strings.Contains(stat, "enable-benchmarks-storage")) && (strings.Contains(stat, "enable-benchmarks-") ||
		strings.Contains(stat, "enable-hist-")) // hist-proxy & hist-info - both at service level
}

func (lw *LatencyStatsProcessor) Refresh(infoKeys []string, rawMetrics map[string]string) ([]AerospikeStat, error) {

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
	var allMetricsToSend = []AerospikeStat{}

	// loop all the latency infokeys
	for ik := range infoKeys {
		latencyMetricsToSend := parseSingleLatenciesKey(infoKeys[ik], rawMetrics, allowedLatenciesList, blockedLatenciessList)
		allMetricsToSend = append(allMetricsToSend, latencyMetricsToSend...)
	}

	return allMetricsToSend, nil
}

func parseSingleLatenciesKey(singleLatencyKey string, rawMetrics map[string]string,
	allowedLatenciesList map[string]struct{}, blockedLatenciessList map[string]struct{}) []AerospikeStat {

	var latencyStats map[string]LatencyStatsMap

	if rawMetrics["latencies:"] != "" {
		// in latest aerospike server>5.1 version, latencies: will always come as infokey, so no need to check other latency commands
		latencyStats = parseLatencyInfo(rawMetrics[singleLatencyKey], int(config.Cfg.Aerospike.LatencyBucketsCount))
	} else {
		latencyStats = parseLatencyInfoLegacy(rawMetrics["latency:"], int(config.Cfg.Aerospike.LatencyBucketsCount))
	}

	// log.Tracef("latency-stats:%+v", latencyStats)
	log.Tracef("latencies-stats:%+v:%+v", singleLatencyKey, rawMetrics[singleLatencyKey])
	var latencyMetricsToSend = []AerospikeStat{}

	for namespaceName, nsLatencyStats := range latencyStats {
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

				statName := operation + "_" + opLatencyStats.(LatencyStatsMap)["timeUnit"].(string) + "_bucket"

				allowed := isMetricAllowed(commons.CTX_LATENCIES, statName)
				asMetric := NewAerospikeStat(commons.CTX_LATENCIES, statName, allowed)
				asMetric.updateValues(pv, labels, labelValues)
				latencyMetricsToSend = append(latencyMetricsToSend, asMetric)

				// aerospike_latencies_<operation>_<timeunit>_count metric
				if i == 0 {
					labels := []string{commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE, commons.METRIC_LABEL_NS}
					labelValues := []string{ClusterName, Service, namespaceName}
					pv := opLatencyStats.(LatencyStatsMap)["bucketValues"].([]float64)[i]

					statName := operation + "_" + opLatencyStats.(LatencyStatsMap)["timeUnit"].(string) + "_count"

					allowed := isMetricAllowed(commons.CTX_LATENCIES, statName)
					asMetric := NewAerospikeStat(commons.CTX_LATENCIES, statName, allowed)
					asMetric.updateValues(pv, labels, labelValues)
					latencyMetricsToSend = append(latencyMetricsToSend, asMetric)

				}
			}
		}
	}

	return latencyMetricsToSend
}
