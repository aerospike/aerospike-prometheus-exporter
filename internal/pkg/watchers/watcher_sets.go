package watchers

import (
	"strings"

	commons "github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"

	log "github.com/sirupsen/logrus"
)

type SetWatcher struct {
	setMetrics map[string]commons.AerospikeStat
}

const (
	KEY_SETS = "sets"
)

func (sw *SetWatcher) PassOneKeys() []string {
	log.Tracef("sets-passonekeys:nil")

	return nil
}

func (sw *SetWatcher) PassTwoKeys(rawMetrics map[string]string) []string {
	log.Tracef("sets-passtwokeys:%s", []string{KEY_SETS})

	return []string{KEY_SETS}
}

func (sw *SetWatcher) Refresh(infoKeys []string, rawMetrics map[string]string) ([]WatcherMetric, error) {

	setStats := strings.Split(rawMetrics[KEY_SETS], ";")

	log.Tracef("set-stats:%v", rawMetrics[KEY_SETS])

	if sw.setMetrics == nil {
		sw.setMetrics = make(map[string]commons.AerospikeStat)
	}

	var metrics_to_send = []WatcherMetric{}

	for i := range setStats {
		clusterName := rawMetrics[commons.Infokey_ClusterName]
		service := rawMetrics[commons.Infokey_Service]

		stats := commons.ParseStats(setStats[i], ":")
		for stat, value := range stats {
			pv, err := commons.TryConvert(value)
			if err != nil {
				continue
			}
			asMetric, exists := sw.setMetrics[stat]

			if !exists {
				asMetric = commons.NewAerospikeStat(commons.CTX_SETS, stat)
				sw.setMetrics[stat] = asMetric
			}

			labels := []string{commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE, commons.METRIC_LABEL_NS, commons.METRIC_LABEL_SET}
			labelValues := []string{clusterName, service, stats["ns"], stats["set"]}

			// pushToPrometheus(asMetric, pv, labels, labelsValues, ch)
			metrics_to_send = append(metrics_to_send, WatcherMetric{asMetric, pv, labels, labelValues})
		}

	}

	return metrics_to_send, nil
}
