package main

import (
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/prometheus/client_golang/prometheus"
)

// Set Raw metrics
var setRawMetrics = map[string]metricType{
	"objects":           mtGauge,
	"tombstones":        mtGauge,
	"memory_data_bytes": mtGauge,
	"truncate_lut":      mtGauge,
	"stop-writes-count": mtCounter,
	"disable-eviction":  mtGauge,
}

type SetWatcher struct{}

func (sw *SetWatcher) describe(ch chan<- *prometheus.Desc) {
	return
}

func (sw *SetWatcher) infoKeys() []string {
	return nil
}

func (sw *SetWatcher) detailKeys(rawMetrics map[string]string) []string {
	return []string{"sets"}
}

// Filtered set metrics. Populated by getFilteredMetrics() based on config.Aerospike.SetMetricsAllowlist, config.Aerospike.SetMetricsBlocklist and setRawMetrics.
var setMetrics map[string]metricType

func (sw *SetWatcher) refresh(infoKeys []string, rawMetrics map[string]string, accu map[string]interface{}, ch chan<- prometheus.Metric) error {
	setStats := strings.Split(rawMetrics["sets"], ";")
	log.Debug("Set Stats:", setStats)

	if setMetrics == nil {
		setMetrics = getFilteredMetrics(setRawMetrics, config.Aerospike.SetMetricsAllowlist, config.Aerospike.SetMetricsAllowlistEnabled, config.Aerospike.SetMetricsBlocklist, config.Aerospike.SetMetricsBlocklistEnabled)
	}

	for i := range setStats {
		setObserver := make(MetricMap, len(setMetrics))
		for m, t := range setMetrics {
			setObserver[m] = makeMetric("aerospike_sets", m, t, config.AeroProm.MetricLabels, "cluster_name", "service", "ns", "set")
		}

		stats := parseStats(setStats[i], ":")
		for stat, pm := range setObserver {
			v, exists := stats[stat]
			if !exists {
				// not found
				continue
			}

			pv, err := tryConvert(v)
			if err != nil {
				continue
			}

			ch <- prometheus.MustNewConstMetric(pm.desc, pm.valueType, pv, rawMetrics["cluster-name"], rawMetrics["service"], stats["ns"], stats["set"])
		}
	}

	return nil
}
