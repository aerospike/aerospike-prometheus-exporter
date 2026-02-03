package statprocessors

import (
	"encoding/base64"
	"strconv"
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
	nodeMetrics  map[string]AerospikeStat
	logSinkCount int
}

func (sw *NodeStatsProcessor) PassOneKeys() []string {
	log.Tracef("node-passonekeys:logs")

	return []string{KEY_SERVICE_LOGS}
}

func (sw *NodeStatsProcessor) PassTwoKeys(passOneStats map[string]string) []string {
	// we need to fetch both configs and stat

	// if Logs are configure/present, send individual sink log commands
	sinkCmds := sw.parseLogSinkDetails(passOneStats)

	passTwoKeys := []string{KEY_SERVICE_CONFIG, KEY_SERVICE_STATISTICS}
	passTwoKeys = append(passTwoKeys, sinkCmds...)

	// add user-agents command if build version is >= 8.1.0.0
	ge, err := isBuildVersionGreaterThanOrEqual(passOneStats["build"], "8.1.0.0")

	if err != nil {
		return passTwoKeys
	}

	if ge {
		passTwoKeys = append(passTwoKeys, "user-agents")
	}

	log.Tracef("node-passtwokeys:%s", passTwoKeys)

	return passTwoKeys
}

func (sw *NodeStatsProcessor) parseLogSinkDetails(rawMetrics map[string]string) []string {
	logSinkCmds := []string{}

	logSinks := strings.Split(rawMetrics[KEY_SERVICE_LOGS], ";")

	// reset the logSinkCount to 0 always, if server restarts by changing debug level, no need to fetch
	sw.logSinkCount = 0

	// 0:stderr;1:/var/log/aerospike/aerospike.log
	for _, logSink := range logSinks {
		logSinkId := strings.Split(logSink, ":")
		logSinkCmds = append(logSinkCmds, "log/"+logSinkId[0])
		sw.logSinkCount++
	}

	return logSinkCmds
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

	// Config
	allMetricsToSend = append(allMetricsToSend, sw.handleRefresh(rawMetrics[KEY_SERVICE_CONFIG])...)

	// handle stats
	allMetricsToSend = append(allMetricsToSend, sw.handleRefresh(rawMetrics[KEY_SERVICE_STATISTICS])...)

	// parse logs Sink
	allMetricsToSend = append(allMetricsToSend, sw.handleLogSinkStats(rawMetrics)...)

	// handle user-agents
	if _, exists := rawMetrics["user-agents"]; exists {
		allMetricsToSend = append(allMetricsToSend, sw.handleUserAgentsStats(rawMetrics)...)
	}

	return allMetricsToSend, nil
}

func (sw *NodeStatsProcessor) handleRefresh(rawMetrics string) []AerospikeStat {

	stats := commons.ParseStats(rawMetrics, ";")

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

	debugValue := 0.0
	detailValue := 0.0

	// log-sink-ids will be from 0..(n-1)
	for idx := 0; idx < sw.logSinkCount; idx++ {
		logSinkKey := "log/" + strconv.Itoa(idx)
		value := rawMetrics[logSinkKey]

		if strings.Contains(value, ":DEBUG") {
			debugValue = 1.0
		}

		if strings.Contains(value, ":DETAIL") {
			detailValue = 1.0
		}
	}

	refreshMetricsToSend = append(refreshMetricsToSend, sw.createLogSinkMetric("pseudo_log_debug", debugValue))
	refreshMetricsToSend = append(refreshMetricsToSend, sw.createLogSinkMetric("pseudo_log_detail", detailValue))

	return refreshMetricsToSend
}

func (sw *NodeStatsProcessor) createLogSinkMetric(statName string, statValue float64) AerospikeStat {
	asMetric, exists := sw.nodeMetrics[statName]

	if !exists {
		allowed := isMetricAllowed(commons.CTX_NODE_STATS, statName)
		asMetric = NewAerospikeStat(commons.CTX_NODE_STATS, statName, allowed)
		sw.nodeMetrics[statName] = asMetric
	}

	labels := []string{commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE}
	labelValues := []string{ClusterName, Service}

	asMetric.updateValues(statValue, labels, labelValues)

	return asMetric

}

// handleUserAgentsStats handles the user-agents stats and returns the metrics to send
func (sw *NodeStatsProcessor) handleUserAgentsStats(rawMetrics map[string]string) []AerospikeStat {

	var refreshMetricsToSend = []AerospikeStat{}

	userAgentsMetrics := rawMetrics["user-agents"]
	stats := strings.Split(userAgentsMetrics, ";")

	for _, stat := range stats {

		if len(stat) == 0 {
			continue
		}
		// stat = user-agent=MSxhc2FkbS00LjAuMix1bmtub3du:count=1
		clientLibraryVersion, appId, uaClientVersionCount, err := sw.getUserAgentInfo(stat)

		if err != nil {
			continue
		}

		// Count value
		pv, err := commons.TryConvert(uaClientVersionCount)

		if err != nil {
			log.Error("Error converting user agent client version count: ", uaClientVersionCount, " error: ", err)
			continue
		}

		asMetric, exists := sw.nodeMetrics[stat]
		dynamicStatname := "user_agent_details"

		if !exists {
			allowed := isMetricAllowed(commons.CTX_NODE_STATS, stat)
			asMetric = NewAerospikeStat(commons.CTX_NODE_STATS, dynamicStatname, allowed)
			sw.nodeMetrics[stat] = asMetric
		}

		labels := []string{commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE, commons.METRIC_LABEL_UA_CLIENT_LIBRARY_VERSION, commons.METRIC_LABEL_UA_CLIENT_APP_ID}
		labelValues := []string{ClusterName, Service, clientLibraryVersion, appId}

		asMetric.updateValues(pv, labels, labelValues)
		refreshMetricsToSend = append(refreshMetricsToSend, asMetric)
	}

	return refreshMetricsToSend
}

func (sw *NodeStatsProcessor) getUserAgentInfo(uaKeyWithAllInfo string) (string, string, string, error) {

	clientLibraryVersion, appId := "unknown", "unknown"
	uaClientVersionCount := "0"

	// user-agent=MSxhc2FkbS00LjAuMix1bmtub3du:count=1, first part is user-agent, second part is count
	uaKeyWithAllInfoParts := strings.Split(uaKeyWithAllInfo, ":")

	// SplitN because encoded values can have multiple = signs
	uaKey := strings.SplitN(uaKeyWithAllInfoParts[0], "=", 2)[1]

	uaInfo, err := base64.StdEncoding.DecodeString(uaKey)

	if err != nil {
		log.Error("Error decoding user agent client version: encoded value: ", uaKey, " error: ", err)
		return clientLibraryVersion, appId, uaClientVersionCount, err
	}

	uaInfoValues := strings.Split(string(uaInfo), ",")

	// older clients, apps with no user-agent logic then we get "unknown" values
	// example: 1,go-1.0.0,ape-1.0.0 - for now we are not using userAgentVersion
	if len(uaInfoValues) > 1 {
		clientLibraryVersion = uaInfoValues[1]
	}

	if len(uaInfoValues) > 2 {
		appId = uaInfoValues[2]
	}

	// count value is the second part of the user-agent key
	uaClientVersionCount = strings.SplitN(uaKeyWithAllInfoParts[1], "=", 2)[1]

	return clientLibraryVersion, appId, uaClientVersionCount, nil
}
