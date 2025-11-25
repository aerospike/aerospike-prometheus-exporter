package statprocessors

import (
	"encoding/base64"
	"strings"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

const (
	USER_AGENTS = "user-agents"
)

type UserAgentsStatsProcessor struct {
	userAgentsMetrics map[string]AerospikeStat
}

func (ua *UserAgentsStatsProcessor) PassOneKeys() []string {
	// this is used to fetch the dcs metadata, we send same get-config command to fetch the dc-names required in next steps
	return nil
}

func (ua *UserAgentsStatsProcessor) PassTwoKeys(rawMetrics map[string]string) []string {
	log.Tracef("user-agent-passonekeys:%s", []string{USER_AGENTS})
	ge, err := isBuildVersionGreaterThanOrEqual(rawMetrics["build"], "8.0.0.0")

	if err != nil {
		log.Warn(err)
		return nil
	}

	if ge {
		return []string{USER_AGENTS}
	}

	log.Debug("user-agent-passonekeys: ignoring user-agents command for build version < 8.1.0.0")
	return nil
}

// refresh prom metrics - parse the given rawMetrics (both config and stats ) and push to given channel
func (ua *UserAgentsStatsProcessor) Refresh(infoKeys []string, rawMetrics map[string]string) ([]AerospikeStat, error) {

	if ua.userAgentsMetrics == nil {
		ua.userAgentsMetrics = make(map[string]AerospikeStat)
	}

	var allMetricsToSend = []AerospikeStat{}

	for _, key := range infoKeys {

		userAgentsMetrics := rawMetrics[key]
		uaMetricsToSend, err := ua.handleRefresh(userAgentsMetrics)

		if err != nil {
			return nil, err
		}

		allMetricsToSend = append(allMetricsToSend, uaMetricsToSend...)
	}

	return allMetricsToSend, nil
}

func (ua *UserAgentsStatsProcessor) handleRefresh(uaRawMetrics string) ([]AerospikeStat, error) {

	stats := strings.Split(uaRawMetrics, ";")
	var uaMetricsToSend = []AerospikeStat{}

	for _, stat := range stats {
		if len(stat) == 0 {
			continue
		}
		// stat = user-agent=MSxhc2FkbS00LjAuMix1bmtub3du:count=1
		clientLibraryVersion, appId, err := ua.getUserAgentInfo(stat)

		if err != nil {
			continue
		}

		// Count value
		uaClientVersionCount := strings.ReplaceAll(strings.Split(stat, ":")[1], "count=", "")

		pv, err := commons.TryConvert(uaClientVersionCount)
		if err != nil {
			logrus.Error("Error converting user agent client version count: ", uaClientVersionCount, " error: ", err)
			continue
		}

		asMetric, exists := ua.userAgentsMetrics[stat]
		dynamicStatname := "details"

		if !exists {
			allowed := isMetricAllowed(commons.CTX_USER_AGENTS, stat)
			asMetric = NewAerospikeStat(commons.CTX_USER_AGENTS, dynamicStatname, allowed)
			ua.userAgentsMetrics[stat] = asMetric
		}

		labels := []string{commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE, commons.METRIC_LABEL_UA_CLIENT_LIBRARY_VERSION, commons.METRIC_LABEL_UA_CLIENT_APP_ID}
		labelValues := []string{ClusterName, Service, clientLibraryVersion, appId}

		asMetric.updateValues(pv, labels, labelValues)
		uaMetricsToSend = append(uaMetricsToSend, asMetric)
	}

	return uaMetricsToSend, nil
}

func (ua *UserAgentsStatsProcessor) getUserAgentInfo(uaKeyWithAllInfo string) (string, string, error) {

	clientLibraryVersion, appId := "unknown", "unknown"

	// user-agent=MSxhc2FkbS00LjAuMix1bmtub3du:count=1, get the first part
	uaKey := strings.ReplaceAll(strings.Split(uaKeyWithAllInfo, ":")[0], "user-agent=", "")
	uaInfo, err := base64.StdEncoding.DecodeString(uaKey)

	if err != nil {
		logrus.Error("Error decoding user agent client version: encoded value: ", uaKey, " error: ", err)
		return clientLibraryVersion, appId, err
	}

	uaInfoValues := strings.Split(string(uaInfo), ",")

	// older clients, apps with no user-agent logic then we get "unknown" values
	// userAgentVersion = uaInfoValues[0]
	if len(uaInfoValues) > 1 {
		clientLibraryVersion = uaInfoValues[1]
	}
	if len(uaInfoValues) > 2 {
		appId = uaInfoValues[2]
	}

	return clientLibraryVersion, appId, nil
}
