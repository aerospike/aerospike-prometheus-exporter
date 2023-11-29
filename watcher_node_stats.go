package main

import (
	"github.com/prometheus/client_golang/prometheus"

	log "github.com/sirupsen/logrus"
)

const (
	KEY_SERVICE_CONFIG     = "get-config:context=service"
	KEY_SERVICE_STATISTICS = "statistics"
)

type StatsWatcher struct {
	nodeMetrics map[string]AerospikeStat
}

func (sw *StatsWatcher) describe(ch chan<- *prometheus.Desc) {}

func (sw *StatsWatcher) passOneKeys() []string {
	return nil
}

func (sw *StatsWatcher) passTwoKeys(rawMetrics map[string]string) []string {
	// we need to fetch both configs and stat
	return []string{KEY_SERVICE_CONFIG, KEY_SERVICE_STATISTICS}
}

// All (allowed/blocked) node stats. Based on the config.Aerospike.NodeMetricsAllowlist, config.Aerospike.NodeMetricsBlocklist.
// var nodeMetrics = make(map[string]AerospikeStat)

func (sw *StatsWatcher) refresh(o *Observer, infoKeys []string, rawMetrics map[string]string, ch chan<- prometheus.Metric) error {

	if sw.nodeMetrics == nil {
		sw.nodeMetrics = make(map[string]AerospikeStat)
	}

	nodeConfigs := rawMetrics[KEY_SERVICE_CONFIG]
	nodeStats := rawMetrics[KEY_SERVICE_STATISTICS]
	log.Tracef("node-configs:%s", nodeConfigs)
	log.Tracef("node-stats:%s", nodeStats)

	clusterName := rawMetrics[ikClusterName]
	service := rawMetrics[ikService]

	// we are sending configs and stats in same refresh call, as both are being sent to prom, instead of doing prom-push in 2 functions
	// handle configs
	sw.handleRefresh(o, nodeConfigs, clusterName, service, ch)

	// handle stats
	sw.handleRefresh(o, nodeStats, clusterName, service, ch)

	return nil
}

func (sw *StatsWatcher) handleRefresh(o *Observer, nodeRawMetrics string, clusterName string, service string,
	ch chan<- prometheus.Metric) {

	stats := parseStats(nodeRawMetrics, ";")

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

		labels := []string{METRIC_LABEL_CLUSTER_NAME, METRIC_LABEL_SERVICE}
		labelsValues := []string{clusterName, service}

		pushToPrometheus(asMetric, pv, labels, labelsValues, ch)

		// check and if latency benchmarks stat, is it enabled (bool true==1 and false==0 after conversion)
		if isStatLatencyHistRelated(stat) && pv == 1 {
			LatencyBenchmarks["service-"+stat] = stat
		}

	}
}
