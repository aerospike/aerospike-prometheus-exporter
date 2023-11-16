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
	var commands = []string{"latencies:", "latencies:hist={test}-benchmarks-read"}
	log.Tracef("latency-passtwokeys:%s", commands)

	// list of namespaces
	s := rawMetrics[KEY_NS_METADATA]
	ns_list := strings.Split(s, ";")
	fmt.Println(" \n*** namespaces: ", ns_list)

	for k := range rawMetrics {
		fmt.Println(" rawmetrics keys are : ", k)
	}
	// for ns_idx := range ns_list {
	// 	ns := ns_list[ns_idx]
	// 	fmt.Println("\n******* list of namespace: ", ns, " \n\t ns_metrics: ", rawMetrics[KEY_NS_NAMESPACE+"/"+ns])
	// }

	log.Tracef("namespaces:%s", s)

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

	var latencyStats map[string]LatencyStatsMap
	log.Tracef("latencies-stats:%+v", rawMetrics["latencies:"])
	log.Tracef("latencies-stats:latencies:hist={test}-benchmarks-read -- %+v", rawMetrics["latencies:hist={test}-benchmarks-read"])

	log.Tracef("latency-stats:%+v", rawMetrics["latency:"])

	if rawMetrics["latencies:"] != "" {
		latencyStats = parseLatencyInfo(rawMetrics["latencies:"], int(config.Cfg.Aerospike.LatencyBucketsCount))
	} else {
		latencyStats = parseLatencyInfoLegacy(rawMetrics["latency:"], int(config.Cfg.Aerospike.LatencyBucketsCount))
	}

	// log.Tracef("latency-stats:%+v", latencyStats)

	var metrics_to_send = []AerospikeStat{}

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

	return metrics_to_send, nil
}
