package watchers

import (
	"strings"

	commons "github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"

	log "github.com/sirupsen/logrus"
)

type SetWatcher struct {
	setMetrics map[string]commons.AerospikeStat
}

func (sw *SetWatcher) PassOneKeys() []string {
	return nil
}

func (sw *SetWatcher) PassTwoKeys(rawMetrics map[string]string) []string {
	return []string{"sets"}
}

func (sw *SetWatcher) Refresh(infoKeys []string, rawMetrics map[string]string) ([]WatcherMetric, error) {
	setStats := strings.Split(rawMetrics["sets"], ";")
	log.Tracef("set-stats:%v", setStats)

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
