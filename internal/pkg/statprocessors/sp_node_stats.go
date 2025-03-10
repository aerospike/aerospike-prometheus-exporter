package statprocessors

import (
	"fmt"
	"strings"

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
	log.Tracef("node-passonekeys:logs")

	return []string{"logs"}
}

func (sw *NodeStatsProcessor) PassTwoKeys(rawMetrics map[string]string) []string {
	// we need to fetch both configs and stat

	// if Logs are configure/present, send individual synk log commands
	pass_two_keys := []string{KEY_SERVICE_CONFIG, KEY_SERVICE_STATISTICS}
	sync_cmds := sw.parseLogSyncDetails(rawMetrics)
	pass_two_keys = append(pass_two_keys, sync_cmds...)

	log.Tracef("node-passtwokeys:%s", pass_two_keys)

	return pass_two_keys
}

func (sw *NodeStatsProcessor) parseLogSyncDetails(rawMetrics map[string]string) []string {
	synkLogCmds := []string{}

	sync_logs_info := rawMetrics["logs"]
	sync_logs := strings.Split(sync_logs_info, ";")

	// 0:stderr;1:/var/log/aerospike/aerospike.log
	for _, sync_info := range sync_logs {
		sync_id := strings.Split(sync_info, ":")
		synkLogCmds = append(synkLogCmds, "log/"+sync_id[0])
	}

	return synkLogCmds
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

	// parse logs sync
	allMetricsToSend = append(allMetricsToSend, sw.handleLogSyncStats(rawMetrics)...)

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

func (sw *NodeStatsProcessor) handleLogSyncStats(rawMetrics map[string]string) []AerospikeStat {

	var refreshMetricsToSend = []AerospikeStat{}

	for key, value := range rawMetrics {
		if strings.Contains(key, "log/") {
			stat := ""
			if strings.Contains(value, ":DEBUG") {
				stat = "log_debug"
				refreshMetricsToSend = append(refreshMetricsToSend, sw.createLogSyncMetric(stat))
			} else if strings.Contains(value, ":DETAIL") {
				stat = "log_detail"
				refreshMetricsToSend = append(refreshMetricsToSend, sw.createLogSyncMetric(stat))
			}
		}
	}

	fmt.Println(refreshMetricsToSend)

	return refreshMetricsToSend
}

func (sw *NodeStatsProcessor) createLogSyncMetric(statName string) AerospikeStat {
	asMetric, exists := sw.nodeMetrics[statName]

	if !exists {
		allowed := isMetricAllowed(commons.CTX_NODE_STATS, statName)
		asMetric = NewAerospikeStat(commons.CTX_NODE_STATS, statName, allowed)
		sw.nodeMetrics[statName] = asMetric
	}

	labels := []string{commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE}
	labelValues := []string{ClusterName, Service}

	asMetric.updateValues(1.0, labels, labelValues)

	return asMetric

}
