package main

import (
	"fmt"
	"math"
	"strings"

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

func (lw *LatencyWatcher) refresh(infoKeys []string, rawMetrics map[string]string, accu map[string]interface{}, ch chan<- prometheus.Metric) error {
	rawStats := parseLatencyInfo(rawMetrics["latency:"])

	for ns, stats := range rawStats {
		for opType, statsDetails := range stats {
			total := 0.0

			for i, label := range statsDetails.(StatsMap)["buckets"].([]string) {
				total += statsDetails.(StatsMap)["valBuckets"].([]float64)[i]
				metricName := "gt_" + strings.Trim(label, "><=")
				labelValue := strings.Trim(label, "><=ms ")
				pm := makeMetric("aerospike_latencies", metricName, mtGauge, config.AeroProm.MetricLabels, "cluster_name", "service", "ns", "op_type", "quartile", "quartile_sorted")
				ch <- prometheus.MustNewConstMetric(pm.desc, pm.valueType, math.Floor(statsDetails.(StatsMap)["valBuckets"].([]float64)[i]), rawMetrics["cluster-name"], rawMetrics["service"], ns, opType, label, fmt.Sprintf("> %4s", labelValue))
			}

			pm := makeMetric("aerospike_latencies", "lt_"+strings.Trim(statsDetails.(StatsMap)["buckets"].([]string)[0], "><="), mtGauge, config.AeroProm.MetricLabels, "cluster_name", "service", "ns", "op_type", "quartile", "quartile_sorted")
			val := math.Floor(statsDetails.(StatsMap)["tps"].(float64) - total)
			metricName := "<" + strings.Trim(statsDetails.(StatsMap)["buckets"].([]string)[0], "<>=")
			metricValue := strings.Trim(metricName, "><=ms ")
			ch <- prometheus.MustNewConstMetric(pm.desc, pm.valueType, val, rawMetrics["cluster-name"], rawMetrics["service"], ns, opType, metricName, fmt.Sprintf("< %4s", metricValue))

			pm = makeMetric("aerospike_latencies", "total", mtGauge, config.AeroProm.MetricLabels, "cluster_name", "service", "ns", "op_type", "quartile", "quartile_sorted")
			val = math.Floor(statsDetails.(StatsMap)["tps"].(float64))
			ch <- prometheus.MustNewConstMetric(pm.desc, pm.valueType, val, rawMetrics["cluster-name"], rawMetrics["service"], ns, opType, "Total", "total")
		}
	}

	// log.Println(rawStats)

	return nil
}
