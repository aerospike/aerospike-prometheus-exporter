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

	clusterName := sanitizeLabelValue(rawMetrics["cluster-name"])
	service := sanitizeLabelValue(rawMetrics["service"])

	for ns, stats := range rawStats {
		sNs := sanitizeLabelValue(ns)
		for opType, statsDetails := range stats {
			sOpType := sanitizeLabelValue(opType)
			total := 0.0

			for i, label := range statsDetails.(StatsMap)["buckets"].([]string) {
				sLabel := sanitizeLabelValue(label)
				total += statsDetails.(StatsMap)["valBuckets"].([]float64)[i]
				metricName := "gt_" + strings.Trim(label, "><=")
				labelValue := sanitizeLabelValue(strings.Trim(label, "><=ms "))
				pm := makeMetric("aerospike_latencies", metricName, mtGauge, "cluster_name", "service", "sNs", "op_type", "quartile", "quartile_sorted", "tags")
				ch <- prometheus.MustNewConstMetric(pm.desc, pm.valueType, math.Floor(statsDetails.(StatsMap)["valBuckets"].([]float64)[i]), clusterName, service, sNs, sOpType, sLabel, fmt.Sprintf("> %4s", labelValue), config.AeroProm.tags)
			}

			pm := makeMetric("aerospike_latencies", "lt_"+strings.Trim(statsDetails.(StatsMap)["buckets"].([]string)[0], "><="), mtGauge, "cluster_name", "service", "sNs", "op_type", "quartile", "quartile_sorted", "tags")
			val := math.Floor(statsDetails.(StatsMap)["tps"].(float64) - total)
			metricName := sanitizeLabelValue("<" + strings.Trim(statsDetails.(StatsMap)["buckets"].([]string)[0], "<>="))
			metricValue := strings.Trim(metricName, "><=ms ")
			ch <- prometheus.MustNewConstMetric(pm.desc, pm.valueType, val, clusterName, service, sNs, sOpType, metricName, fmt.Sprintf("< %4s", metricValue), config.AeroProm.tags)

			pm = makeMetric("aerospike_latencies", "total", mtGauge, "cluster_name", "service", "sNs", "op_type", "quartile", "quartile_sorted", "tags")
			val = math.Floor(statsDetails.(StatsMap)["tps"].(float64))
			ch <- prometheus.MustNewConstMetric(pm.desc, pm.valueType, val, clusterName, service, sNs, sOpType, "Total", "total", config.AeroProm.tags)
		}
	}

	// log.Println(rawStats)

	return nil
}
