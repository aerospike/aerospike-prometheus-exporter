package statprocessors

import (
	"strings"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"

	log "github.com/sirupsen/logrus"
)

const (
	KEY_SERVICE_CONFIG     = "get-config:context=service"
	KEY_SERVICE_STATISTICS = "statistics"
	KEY_SERVICE_LOGS       = "logs"
)

type NodeStatsProcessor struct {
	nodeMetrics map[string]AerospikeStat
}

func (sw *NodeStatsProcessor) PassOneKeys() []string {
	log.Tracef("node-passonekeys:logs")

	return []string{KEY_SERVICE_LOGS}
}

func (sw *NodeStatsProcessor) PassTwoKeys(rawMetrics map[string]string) []string {
	// we need to fetch both configs and stat

	// if Logs are configure/present, send individual sink log commands
	sinkCmds := sw.parseLogSinkDetails(rawMetrics)

	passTwoKeys := []string{KEY_SERVICE_CONFIG, KEY_SERVICE_STATISTICS}
	passTwoKeys = append(passTwoKeys, sinkCmds...)

	log.Tracef("node-passtwokeys:%s", passTwoKeys)

	return passTwoKeys
}

func (sw *NodeStatsProcessor) parseLogSinkDetails(rawMetrics map[string]string) []string {
	sinkLogCmds := []string{}

	sinkLogs := strings.Split(rawMetrics[KEY_SERVICE_LOGS], ";")

	// 0:stderr;1:/var/log/aerospike/aerospike.log
	for _, sinkInfo := range sinkLogs {
		sinkId := strings.Split(sinkInfo, ":")
		sinkLogCmds = append(sinkLogCmds, "log/"+sinkId[0])
	}

	return sinkLogCmds
}

// All (allowed/blocked) node stats. Based on the config.Aerospike.NodeMetricsAllowlist, config.Aerospike.NodeMetricsBlocklist.
// var nodeMetrics = make(map[string]AerospikeStat)

func (sw *NodeStatsProcessor) Refresh(infoKeys []string, rawMetrics map[string]string) ([]AerospikeStat, error) {

	if sw.nodeMetrics == nil {
		sw.nodeMetrics = make(map[string]AerospikeStat)
	}

	log.Tracef("node-configs:%s", rawMetrics[KEY_SERVICE_CONFIG])
	log.Tracef("node-stats:%s", rawMetrics[KEY_SERVICE_STATISTICS])

	// we are sending configs and stats in same refresh call, as both are being sent to prom, instead of doing prom-push in 2 functions
	// handle configs
	var allMetricsToSend = []AerospikeStat{}

	lCfgMetricsToSend := sw.handleRefresh(rawMetrics[KEY_SERVICE_CONFIG])

	// handle stats
	lStatMetricsToSend := sw.handleRefresh(rawMetrics[KEY_SERVICE_STATISTICS])

	// merge both array into single
	allMetricsToSend = append(allMetricsToSend, lCfgMetricsToSend...)
	allMetricsToSend = append(allMetricsToSend, lStatMetricsToSend...)

	// parse logs Sink
	allMetricsToSend = append(allMetricsToSend, sw.handleLogSinkStats(rawMetrics)...)

	return allMetricsToSend, nil
}

func (sw *NodeStatsProcessor) handleRefresh(nodeRawMetrics string) []AerospikeStat {

	stats := commons.ParseStats(nodeRawMetrics, ";")

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

			// pv==1 means histogram is enabled
			if pv == 1 {
				latencySubcommand := stat
				if strings.Contains(latencySubcommand, "enable-") {
					latencySubcommand = strings.ReplaceAll(latencySubcommand, "enable-", "")
				}
				// some histogram stats has 'hist-' in the config, but the latency command does not expect hist- when issue the command
				if strings.Contains(latencySubcommand, "hist-") {
					latencySubcommand = strings.ReplaceAll(latencySubcommand, "hist-", "")
				}

				ServiceLatencyBenchmarks[stat] = latencySubcommand
			} else {
				// pv==0 means histogram is disabled
				delete(ServiceLatencyBenchmarks, stat)
			}
		}
	}

	return refreshMetricsToSend
}

func (sw *NodeStatsProcessor) handleLogSinkStats(rawMetrics map[string]string) []AerospikeStat {

	var refreshMetricsToSend = []AerospikeStat{}

	debug_value := 0.0
	detail_value := 0.0
	for key, value := range rawMetrics {
		if strings.Contains(key, "log/") {

			if strings.Contains(value, ":DEBUG") {
				debug_value = 1.0
			}

			if strings.Contains(value, ":DETAIL") {
				detail_value = 1.0
			}
		}
	}
	refreshMetricsToSend = append(refreshMetricsToSend, sw.createLogSinkMetric("pseudo_log_debug", debug_value))
	refreshMetricsToSend = append(refreshMetricsToSend, sw.createLogSinkMetric("pseudo_log_detail", detail_value))

	return refreshMetricsToSend
}

func (sw *NodeStatsProcessor) createLogSinkMetric(statName string, statValue float64) AerospikeStat {
	asMetric, exists := sw.nodeMetrics[statName]

	allowed := isMetricAllowed(commons.CTX_NODE_STATS, statName)
	if !exists {
		asMetric = NewAerospikeStat(commons.CTX_NODE_STATS, statName, allowed)
		sw.nodeMetrics[statName] = asMetric
	}

	labels := []string{commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE}
	labelValues := []string{ClusterName, Service}

	asMetric.updateValues(statValue, labels, labelValues)

	return asMetric

}
