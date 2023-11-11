package watchers

import (
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"

	log "github.com/sirupsen/logrus"
)

const (
	KEY_SERVICE_CONFIG     = "get-config:context=service"
	KEY_SERVICE_STATISTICS = "statistics"
)

type NodeStatsWatcher struct {
	nodeMetrics map[string]commons.AerospikeStat
}

func (sw *NodeStatsWatcher) PassOneKeys() []string {
	return nil
}

func (sw *NodeStatsWatcher) PassTwoKeys(rawMetrics map[string]string) []string {
	// we need to fetch both configs and stat
	return []string{KEY_SERVICE_CONFIG, KEY_SERVICE_STATISTICS}
}

// All (allowed/blocked) node stats. Based on the config.Aerospike.NodeMetricsAllowlist, config.Aerospike.NodeMetricsBlocklist.
// var nodeMetrics = make(map[string]AerospikeStat)

func (sw *NodeStatsWatcher) Refresh(infoKeys []string, rawMetrics map[string]string) ([]WatcherMetric, error) {

	if sw.nodeMetrics == nil {
		sw.nodeMetrics = make(map[string]commons.AerospikeStat)
	}

	nodeConfigs := rawMetrics[KEY_SERVICE_CONFIG]
	nodeStats := rawMetrics[KEY_SERVICE_STATISTICS]
	log.Tracef("node-configs:%s", nodeConfigs)
	log.Tracef("node-stats:%s", nodeStats)

	clusterName := rawMetrics[commons.Infokey_ClusterName]
	service := rawMetrics[commons.Infokey_Service]

	// we are sending configs and stats in same refresh call, as both are being sent to prom, instead of doing prom-push in 2 functions
	// handle configs
	var metrics_to_send = []WatcherMetric{}

	l_cfg_metrics_to_send := sw.handleRefresh(nodeConfigs, clusterName, service)

	// handle stats
	l_stat_metrics_to_send := sw.handleRefresh(nodeStats, clusterName, service)

	// merge both array into single
	metrics_to_send = append(metrics_to_send, l_cfg_metrics_to_send...)
	metrics_to_send = append(metrics_to_send, l_stat_metrics_to_send...)

	return metrics_to_send, nil
}

func (sw *NodeStatsWatcher) handleRefresh(nodeRawMetrics string, clusterName string, service string) []WatcherMetric {

	stats := commons.ParseStats(nodeRawMetrics, ";")

	var metrics_to_send = []WatcherMetric{}

	for stat, value := range stats {
		pv, err := commons.TryConvert(value)
		if err != nil {
			continue
		}
		asMetric, exists := sw.nodeMetrics[stat]

		if !exists {
			asMetric = commons.NewAerospikeStat(commons.CTX_NODE_STATS, stat)
			sw.nodeMetrics[stat] = asMetric
		}

		labels := []string{commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE}
		labelValues := []string{clusterName, service}

		// pushToPrometheus(asMetric, pv, labels, labelsValues)
		metrics_to_send = append(metrics_to_send, WatcherMetric{asMetric, pv, labels, labelValues})

	}

	return metrics_to_send
}