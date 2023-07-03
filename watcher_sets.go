package main

import (
	"strings"

	"github.com/prometheus/client_golang/prometheus"

	log "github.com/sirupsen/logrus"
)

type SetWatcher struct{}

func (sw *SetWatcher) describe(ch chan<- *prometheus.Desc) {}

func (sw *SetWatcher) passOneKeys() []string {
	return nil
}

func (sw *SetWatcher) passTwoKeys(rawMetrics map[string]string) []string {
	return []string{"sets"}
}

// All (allowed/blocked) Sets stats. Based on the config.Aerospike.SetsMetricsAllowlist, config.Aerospike.SetsMetricsBlocklist.
var setMetrics = make(map[string]AerospikeStat)

func (sw *SetWatcher) refresh(o *Observer, infoKeys []string, rawMetrics map[string]string, ch chan<- prometheus.Metric) error {
	setStats := strings.Split(rawMetrics["sets"], ";")
	log.Tracef("set-stats:%v", setStats)

	if isTestcaseMode() {
		setMetrics = make(map[string]AerospikeStat)
	}

	for i := range setStats {

		stats := parseStats(setStats[i], ":")
		for stat, value := range stats {
			pv, err := tryConvert(value)
			if err != nil {
				continue
			}
			asMetric, exists := setMetrics[stat]

			if !exists {
				asMetric = newAerospikeStat(CTX_SETS, stat)
				setMetrics[stat] = asMetric
			}

			if asMetric.isAllowed {
				desc, valueType := asMetric.makePromMetric(METRIC_LABEL_CLUSTER_NAME, METRIC_LABEL_SERVICE, METRIC_LABEL_NS, METRIC_LABEL_SET)
				ch <- prometheus.MustNewConstMetric(desc, valueType, pv, rawMetrics[ikClusterName], rawMetrics[ikService], stats["ns"], stats["set"])
			}

		}

	}

	return nil
}
