package main

import (
	"github.com/prometheus/client_golang/prometheus"
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
	return []string{"latency:"}
}

func (lw *LatencyWatcher) refresh(infoKeys []string, rawMetrics map[string]string, ch chan<- prometheus.Metric) error {
	latencyStats := parseLatencyInfo(rawMetrics["latency:"])

	for namespaceName, nsLatencyStats := range latencyStats {
		for operation, opLatencyStats := range nsLatencyStats {
			for i, labelValue := range opLatencyStats.(StatsMap)["bucketLabels"].([]string) {
				// aerospike_latencies_<operation>_bucket metric - Less than or equal to histogram buckets
				pm := makeMetric("aerospike_latencies", operation+"_ms_bucket", mtGauge, config.AeroProm.MetricLabels, "cluster_name", "service", "ns", "le")
				ch <- prometheus.MustNewConstMetric(pm.desc, pm.valueType, opLatencyStats.(StatsMap)["bucketValues"].([]float64)[i], rawMetrics["cluster-name"], rawMetrics["service"], namespaceName, labelValue)

				// aerospike_latencies_<operation>_count metric
				if i == 0 {
					pm = makeMetric("aerospike_latencies", operation+"_ms_count", mtGauge, config.AeroProm.MetricLabels, "cluster_name", "service", "ns")
					ch <- prometheus.MustNewConstMetric(pm.desc, pm.valueType, opLatencyStats.(StatsMap)["bucketValues"].([]float64)[i], rawMetrics["cluster-name"], rawMetrics["service"], namespaceName)
				}
			}
		}
	}

	return nil
}
