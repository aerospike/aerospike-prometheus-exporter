package statprocessors

import (
	"strings"

	commons "github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"

	log "github.com/sirupsen/logrus"
)

type SetsStatsProcessor struct {
	setMetrics map[string]AerospikeStat
}

const (
	KEY_SETS = "sets"
)

func (sw *SetsStatsProcessor) PassOneKeys() []string {
	log.Tracef("sets-passonekeys:nil")

	return nil
}

func (sw *SetsStatsProcessor) PassTwoKeys(passOneStats map[string]string) []string {
	log.Tracef("sets-passtwokeys:%s", []string{KEY_SETS})

	return []string{KEY_SETS}
}

func (sw *SetsStatsProcessor) Refresh(infoKeys []string, rawMetrics map[string]string) ([]AerospikeStat, error) {

	setStats := strings.Split(rawMetrics[KEY_SETS], ";")

	log.Tracef("set-stats:%v", rawMetrics[KEY_SETS])

	if sw.setMetrics == nil {
		sw.setMetrics = make(map[string]AerospikeStat)
	}

	var allMetricsToSend = []AerospikeStat{}

	for i := range setStats {
		stats := commons.ParseStats(setStats[i], ":")
		for stat, value := range stats {
			pv, err := commons.TryConvert(value)
			if err != nil {
				continue
			}
			asMetric, exists := sw.setMetrics[stat]

			if !exists {
				allowed := isMetricAllowed(commons.CTX_SETS, stat)
				asMetric = NewAerospikeStat(commons.CTX_SETS, stat, allowed)
				sw.setMetrics[stat] = asMetric
			}

			labels := []string{commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE, commons.METRIC_LABEL_NS, commons.METRIC_LABEL_SET}
			labelValues := []string{ClusterName, Service, stats["ns"], stats["set"]}

			// pushToPrometheus(asMetric, pv, labels, labelsValues, ch)
			asMetric.updateValues(pv, labels, labelValues)
			allMetricsToSend = append(allMetricsToSend, asMetric)
		}

	}

	return allMetricsToSend, nil
}
