package main

import (
	"strings"

	"github.com/prometheus/client_golang/prometheus"

	log "github.com/sirupsen/logrus"
)

type SetWatcher struct {
	setMetrics map[string]AerospikeStat
}

func (sw *SetWatcher) describe(ch chan<- *prometheus.Desc) {}

func (sw *SetWatcher) passOneKeys() []string {
	return nil
}

func (sw *SetWatcher) passTwoKeys(rawMetrics map[string]string) []string {
	return []string{"sets"}
}

func (sw *SetWatcher) refresh(o *Observer, infoKeys []string, rawMetrics map[string]string, ch chan<- prometheus.Metric) error {
	setStats := strings.Split(rawMetrics["sets"], ";")
	log.Tracef("set-stats:%v", setStats)

	if sw.setMetrics == nil {
		sw.setMetrics = make(map[string]AerospikeStat)
	}

	for i := range setStats {
		clusterName := rawMetrics[ikClusterName]
		service := rawMetrics[ikService]

		stats := parseStats(setStats[i], ":")
		for stat, value := range stats {
			pv, err := tryConvert(value)
			if err != nil {
				continue
			}
			asMetric, exists := sw.setMetrics[stat]

			if !exists {
				asMetric = newAerospikeStat(CTX_SETS, stat)
				sw.setMetrics[stat] = asMetric
			}

			labels := []string{METRIC_LABEL_CLUSTER_NAME, METRIC_LABEL_SERVICE, METRIC_LABEL_NS, METRIC_LABEL_SET}
			labelsValues := []string{clusterName, service, stats["ns"], stats["set"]}
			pushToPrometheus(asMetric, pv, labels, labelsValues, ch)

		}

	}

	return nil
}
