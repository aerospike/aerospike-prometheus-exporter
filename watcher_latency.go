package main

import (
	"github.com/prometheus/client_golang/prometheus"

	goversion "github.com/hashicorp/go-version"
	log "github.com/sirupsen/logrus"
)

type LatencyWatcher struct {
}

func (lw *LatencyWatcher) describe(ch chan<- *prometheus.Desc) {
	return
}

func (lw *LatencyWatcher) passOneKeys() []string {
	return []string{"build"}
}

func (lw *LatencyWatcher) passTwoKeys(rawMetrics map[string]string) (latencyCommands []string) {
	latencyCommands = []string{"latencies:", "latency:"}

	if len(rawMetrics["build"]) > 0 {
		ver := rawMetrics["build"]
		ref := "5.1.0.0"

		version, err := goversion.NewVersion(ver)
		if err != nil {
			log.Warnf("Error parsing build version %s: %v", ver, err)
			return latencyCommands
		}

		refVersion, err := goversion.NewVersion(ref)
		if err != nil {
			log.Warnf("Error parsing reference version %s: %v", ref, err)
			return latencyCommands
		}

		if version.GreaterThanOrEqual(refVersion) {
			return []string{"latencies:"}
		}

		return []string{"latency:"}
	}

	return latencyCommands
}

func (lw *LatencyWatcher) refresh(infoKeys []string, rawMetrics map[string]string, ch chan<- prometheus.Metric) error {
	var latencyStats map[string]StatsMap

	if rawMetrics["latencies:"] != "" {
		latencyStats = parseLatencyInfo(rawMetrics["latencies:"])
	} else {
		latencyStats = parseLatencyInfoLegacy(rawMetrics["latency:"])
	}

	log.Tracef("latency-stats:%+v", latencyStats)

	for namespaceName, nsLatencyStats := range latencyStats {
		for operation, opLatencyStats := range nsLatencyStats {
			for i, labelValue := range opLatencyStats.(StatsMap)["bucketLabels"].([]string) {
				// aerospike_latencies_<operation>_<timeunit>_bucket metric - Less than or equal to histogram buckets
				pm := makeMetric("aerospike_latencies", operation+"_"+opLatencyStats.(StatsMap)["timeUnit"].(string)+"_bucket", mtGauge, config.AeroProm.MetricLabels, "cluster_name", "service", "ns", "le")
				ch <- prometheus.MustNewConstMetric(pm.desc, pm.valueType, opLatencyStats.(StatsMap)["bucketValues"].([]float64)[i], rawMetrics[ikClusterName], rawMetrics[ikService], namespaceName, labelValue)

				// aerospike_latencies_<operation>_<timeunit>_count metric
				if i == 0 {
					pm = makeMetric("aerospike_latencies", operation+"_"+opLatencyStats.(StatsMap)["timeUnit"].(string)+"_count", mtGauge, config.AeroProm.MetricLabels, "cluster_name", "service", "ns")
					ch <- prometheus.MustNewConstMetric(pm.desc, pm.valueType, opLatencyStats.(StatsMap)["bucketValues"].([]float64)[i], rawMetrics[ikClusterName], rawMetrics[ikService], namespaceName)
				}
			}
		}
	}

	return nil
}
