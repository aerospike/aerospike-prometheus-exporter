package main

import (
	"github.com/prometheus/client_golang/prometheus"

	log "github.com/sirupsen/logrus"
)

type LatencyWatcher struct {
}

func (lw *LatencyWatcher) describe(ch chan<- *prometheus.Desc) {
	return
}

func (lw *LatencyWatcher) infoKeys() []string {
	return nil
}

func (lw *LatencyWatcher) detailKeys(rawMetrics map[string]string) []string {
	return []string{"latency:", "latencies:"}
}

func (lw *LatencyWatcher) refresh(infoKeys []string, rawMetrics map[string]string, ch chan<- prometheus.Metric) error {
	latencyStats := make(map[string]StatsMap)

	if rawMetrics["latencies:"] != "" {
		latencyStats = parseLatencyInfoNew(rawMetrics["latencies:"])
	} else {
		latencyStats = parseLatencyInfoLegacy(rawMetrics["latency:"])
	}

	log.Tracef("latency-stats:%+v", latencyStats)

	for namespaceName, nsLatencyStats := range latencyStats {
		for operation, opLatencyStats := range nsLatencyStats {
			for i, labelValue := range opLatencyStats.(StatsMap)["bucketLabels"].([]string) {
				// aerospike_latencies_<operation>_bucket metric - Less than or equal to histogram buckets
				pm := makeMetric("aerospike_latencies", operation+"_"+opLatencyStats.(StatsMap)["timeUnit"].(string)+"_bucket", mtGauge, config.AeroProm.MetricLabels, "cluster_name", "service", "ns", "le")
				ch <- prometheus.MustNewConstMetric(pm.desc, pm.valueType, opLatencyStats.(StatsMap)["bucketValues"].([]float64)[i], rawMetrics["cluster-name"], rawMetrics["service"], namespaceName, labelValue)

				// aerospike_latencies_<operation>_count metric
				if i == 0 {
					pm = makeMetric("aerospike_latencies", operation+"_"+opLatencyStats.(StatsMap)["timeUnit"].(string)+"_count", mtGauge, config.AeroProm.MetricLabels, "cluster_name", "service", "ns")
					ch <- prometheus.MustNewConstMetric(pm.desc, pm.valueType, opLatencyStats.(StatsMap)["bucketValues"].([]float64)[i], rawMetrics["cluster-name"], rawMetrics["service"], namespaceName)
				}
			}
		}
	}

	return nil
}
