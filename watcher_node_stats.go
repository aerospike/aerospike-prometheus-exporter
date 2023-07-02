package main

import (
	"github.com/prometheus/client_golang/prometheus"

	log "github.com/sirupsen/logrus"
)

type StatsWatcher struct{}

func (sw *StatsWatcher) describe(ch chan<- *prometheus.Desc) {}

func (sw *StatsWatcher) passOneKeys() []string {
	return nil
}

func (sw *StatsWatcher) passTwoKeys(rawMetrics map[string]string) []string {
	return []string{"statistics"}
}

// All (allowed/blocked) node stats. Based on the config.Aerospike.NodeMetricsAllowlist, config.Aerospike.NodeMetricsBlocklist.
var nodeMetrics = make(map[string]AerospikeStat)

func (sw *StatsWatcher) refresh(o *Observer, infoKeys []string, rawMetrics map[string]string, ch chan<- prometheus.Metric) error {

	log.Tracef("node-stats:%s", rawMetrics["statistics"])

	if nodeMetrics == nil || isTestcaseMode() {
		nodeMetrics = make(map[string]AerospikeStat)
	}

	stats := parseStats(rawMetrics["statistics"], ";")

	for stat, value := range stats {
		pv, err := tryConvert(value)
		if err != nil {
			continue
		}
		asMetric, exists := nodeMetrics[stat]

		if !exists {
			asMetric = newAerospikeStat(CTX_NODE_STATS, stat)
			nodeMetrics[stat] = asMetric
		}

		if asMetric.isAllowed {
			desc, valueType := asMetric.makePromeMetric(METRIC_LABEL_CLUSTER_NAME, METRIC_LABEL_SERVICE)
			ch <- prometheus.MustNewConstMetric(desc, valueType, pv, rawMetrics[ikClusterName], rawMetrics[ikService])
		}
	}

	return nil
}
