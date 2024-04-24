package statprocessors

import (
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"

	log "github.com/sirupsen/logrus"
)

const (
	KEY_SERVICE_CONFIG     = "get-config:context=service"
	KEY_SERVICE_STATISTICS = "statistics"
)

type NodeStatsProcessor struct {
	nodeMetrics map[string]AerospikeStat
}

func (sw *NodeStatsProcessor) PassOneKeys() []string {
	log.Tracef("node-passonekeys:nil")

	return nil
}

func (sw *NodeStatsProcessor) PassTwoKeys(rawMetrics map[string]string) []string {
	// we need to fetch both configs and stat
	log.Tracef("node-passtwokeys:%s", []string{KEY_SERVICE_CONFIG, KEY_SERVICE_STATISTICS})

	return []string{KEY_SERVICE_CONFIG, KEY_SERVICE_STATISTICS}
}

// All (allowed/blocked) node stats. Based on the config.Aerospike.NodeMetricsAllowlist, config.Aerospike.NodeMetricsBlocklist.
// var nodeMetrics = make(map[string]AerospikeStat)

func (sw *NodeStatsProcessor) Refresh(infoKeys []string, rawMetrics map[string]string) ([]AerospikeStat, error) {

	if sw.nodeMetrics == nil {
		sw.nodeMetrics = make(map[string]AerospikeStat)
	}

	nodeConfigs := rawMetrics[KEY_SERVICE_CONFIG]
	nodeStats := rawMetrics[KEY_SERVICE_STATISTICS]
	log.Tracef("node-configs:%s", nodeConfigs)
	log.Tracef("node-stats:%s", nodeStats)

	// we are sending configs and stats in same refresh call, as both are being sent to prom, instead of doing prom-push in 2 functions
	// handle configs
	var allMetricsToSend = []AerospikeStat{}

	lCfgMetricsToSend := sw.handleRefresh(nodeConfigs)

	// handle stats
	lStatMetricsToSend := sw.handleRefresh(nodeStats)

	// merge both array into single
	allMetricsToSend = append(allMetricsToSend, lCfgMetricsToSend...)
	allMetricsToSend = append(allMetricsToSend, lStatMetricsToSend...)

	return allMetricsToSend, nil
}

func (sw *NodeStatsProcessor) handleRefresh(nodeRawMetrics string) []AerospikeStat {

	stats := commons.ParseStats(nodeRawMetrics, ";")
	if len(NodeId) == 0 {
		NodeId = stats["node-id"]
	}

	var refreshMetricsToSend = []AerospikeStat{}

	for stat, value := range stats {
		pv, err := commons.TryConvert(value)
		if err != nil {
			continue
		}
		asMetric, exists := sw.nodeMetrics[stat]

		if !exists {
			allowed := isMetricAllowed(commons.CTX_NODE_STATS, stat)
			asMetric = NewAerospikeStat(commons.CTX_NODE_STATS, stat, allowed)
			sw.nodeMetrics[stat] = asMetric
		}

		labels := []string{commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE}
		labelValues := []string{ClusterName, Service}

		// pushToPrometheus(asMetric, pv, labels, labelsValues)
		asMetric.updateValues(pv, labels, labelValues)
		refreshMetricsToSend = append(refreshMetricsToSend, asMetric)

		// check and if latency benchmarks stat, is it enabled (bool true==1 and false==0 after conversion)
		if isStatLatencyHistRelated(stat) {

			// remove old value as microbenchmark may get enabled / disable on-the-fly at server so we cannot rely on value
			delete(LatencyBenchmarks, "service-"+stat)

			if pv == 1 {
				LatencyBenchmarks["service-"+stat] = stat
			}
		}
	}

	return refreshMetricsToSend
}
