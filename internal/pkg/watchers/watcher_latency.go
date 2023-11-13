package watchers

import (
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
		log.Tracef("latency-passtwokeys:%s", []string{"latencies:"})
		return []string{"latencies:"}
	}

	log.Tracef("latency-passtwokeys:%s", []string{"latency:"})
	return []string{"latency:"}
}

func (lw *LatencyWatcher) Refresh(infoKeys []string, rawMetrics map[string]string) ([]WatcherMetric, error) {

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

	var latencyStats map[string]commons.StatsMap
	log.Tracef("latencies-stats:%+v", rawMetrics["latencies:"])
	log.Tracef("latency-stats:%+v", rawMetrics["latency:"])

	if rawMetrics["latencies:"] != "" {
		latencyStats = parseLatencyInfo(rawMetrics["latencies:"], int(config.Cfg.Aerospike.LatencyBucketsCount))
	} else {
		latencyStats = parseLatencyInfoLegacy(rawMetrics["latency:"], int(config.Cfg.Aerospike.LatencyBucketsCount))
	}

	// log.Tracef("latency-stats:%+v", latencyStats)

	var metrics_to_send = []WatcherMetric{}

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

			for i, labelValue := range opLatencyStats.(commons.StatsMap)["bucketLabels"].([]string) {
				// aerospike_latencies_<operation>_<timeunit>_bucket metric - Less than or equal to histogram buckets

				labels := []string{commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE, commons.METRIC_LABEL_NS, commons.METRIC_LABEL_LE}
				labelValues := []string{ClusterName, Service, namespaceName, labelValue}
				pv := opLatencyStats.(commons.StatsMap)["bucketValues"].([]float64)[i]

				// pm := makeMetric("aerospike_latencies", operation+"_"+opLatencyStats.(commons.StatsMap)["timeUnit"].(string)+"_bucket", mtGauge, config.AeroProm.MetricLabels, METRIC_LABEL_CLUSTER_NAME, METRIC_LABEL_SERVICE, METRIC_LABEL_NS, METRIC_LABEL_LE)
				// ch <- prometheus.MustNewConstMetric(pm.desc, pm.valueType, pv, rawMetrics[ikClusterName], rawMetrics[ikService], namespaceName, labelValue)
				asMetric := commons.NewAerospikeStat(commons.CTX_LATENCIES, operation+"_"+opLatencyStats.(commons.StatsMap)["timeUnit"].(string)+"_bucket")
				metrics_to_send = append(metrics_to_send, WatcherMetric{asMetric, pv, labels, labelValues})

				// aerospike_latencies_<operation>_<timeunit>_count metric
				if i == 0 {
					labels := []string{commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE, commons.METRIC_LABEL_NS}
					labelValues := []string{ClusterName, Service, namespaceName}
					pv := opLatencyStats.(commons.StatsMap)["bucketValues"].([]float64)[i]

					// pm = makeMetric("aerospike_latencies", operation+"_"+opLatencyStats.(commons.StatsMap)["timeUnit"].(string)+"_count", mtGauge, config.AeroProm.MetricLabels, METRIC_LABEL_CLUSTER_NAME, METRIC_LABEL_SERVICE, METRIC_LABEL_NS)
					// ch <- prometheus.MustNewConstMetric(pm.desc, pm.valueType, pv, rawMetrics[ikClusterName], rawMetrics[ikService], namespaceName)
					asMetric := commons.NewAerospikeStat(commons.CTX_LATENCIES, operation+"_"+opLatencyStats.(commons.StatsMap)["timeUnit"].(string)+"_count")
					metrics_to_send = append(metrics_to_send, WatcherMetric{asMetric, pv, labels, labelValues})

				}
			}
		}
	}

	return metrics_to_send, nil
}
