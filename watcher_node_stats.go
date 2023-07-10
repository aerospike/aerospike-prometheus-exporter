package main

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"

	log "github.com/sirupsen/logrus"
)

type StatsWatcher struct {
	nodeMetrics map[string]AerospikeStat
}

func (sw *StatsWatcher) describe(ch chan<- *prometheus.Desc) {}

func (sw *StatsWatcher) passOneKeys() []string {
	return nil
}

func (sw *StatsWatcher) passTwoKeys(rawMetrics map[string]string) []string {
	return []string{"statistics"}
}

// All (allowed/blocked) node stats. Based on the config.Aerospike.NodeMetricsAllowlist, config.Aerospike.NodeMetricsBlocklist.
// var nodeMetrics = make(map[string]AerospikeStat)

func (sw *StatsWatcher) refresh(o *Observer, infoKeys []string, rawMetrics map[string]string, ch chan<- prometheus.Metric) error {

	log.Tracef("node-stats:%s", rawMetrics["statistics"])

	if sw.nodeMetrics == nil {
		fmt.Println("Reinitializing nodeStats(...) ")
		sw.nodeMetrics = make(map[string]AerospikeStat)
	}

	stats := parseStats(rawMetrics["statistics"], ";")

	for stat, value := range stats {
		pv, err := tryConvert(value)
		if err != nil {
			continue
		}
		asMetric, exists := sw.nodeMetrics[stat]

		if !exists {
			asMetric = newAerospikeStat(CTX_NODE_STATS, stat)
			sw.nodeMetrics[stat] = asMetric
		}

		if asMetric.isAllowed {
			desc, valueType := asMetric.makePromMetric(METRIC_LABEL_CLUSTER_NAME, METRIC_LABEL_SERVICE)
			ch <- prometheus.MustNewConstMetric(desc, valueType, pv, rawMetrics[ikClusterName], rawMetrics[ikService])
		}
	}

	return nil
}
