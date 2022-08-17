package main

import (
	"strings"

	"github.com/prometheus/client_golang/prometheus"

	log "github.com/sirupsen/logrus"
)

// Sindex raw metrics
var sindexRawMetrics = map[string]metricType{
	"keys":                      mtGauge, // removed in server6.0
	"entries":                   mtGauge,
	"ibtr_memory_used":          mtGauge, // removed in server6.0
	"nbtr_memory_used":          mtGauge, // removed in server6.0
	"load_pct":                  mtGauge,
	"loadtime":                  mtGauge,   // removed in server6.0
	"write_success":             mtCounter, // removed in server6.0
	"write_error":               mtCounter, // removed in server6.0
	"delete_success":            mtCounter, // removed in server6.0
	"delete_error":              mtCounter, // removed in server6.0
	"stat_gc_recs":              mtCounter,
	"query_basic_complete":      mtCounter, // removed in server6.0
	"query_basic_error":         mtCounter, // removed in server6.0
	"query_basic_abort":         mtCounter, // removed in server6.0
	"query_basic_avg_rec_count": mtGauge,   // removed in server6.0
	"histogram":                 mtGauge,   // removed in server6.0
	"memory_used":               mtGauge,
	"load_time":                 mtGauge,
	"entries_per_rec":           mtGauge,
	"entries_per_bval":          mtGauge,
}

type SindexWatcher struct {
}

func (siw *SindexWatcher) describe(ch chan<- *prometheus.Desc) {}

func (siw *SindexWatcher) passOneKeys() []string {
	if config.Aerospike.DisableSindexMetrics {
		// disabled
		return nil
	}

	return []string{"sindex"}
}

func (siw *SindexWatcher) passTwoKeys(rawMetrics map[string]string) (sindexCommands []string) {
	if config.Aerospike.DisableSindexMetrics {
		// disabled
		return nil
	}

	sindexesMeta := strings.Split(rawMetrics["sindex"], ";")
	sindexCommands = siw.getSindexCommands(sindexesMeta)

	return sindexCommands
}

// getSindexCommands returns list of commands to fetch sindex statistics
func (siw *SindexWatcher) getSindexCommands(sindexesMeta []string) (sindexCommands []string) {
	for _, sindex := range sindexesMeta {
		stats := parseStats(sindex, ":")
		sindexCommands = append(sindexCommands, "sindex/"+stats["ns"]+"/"+stats["indexname"])
	}

	return sindexCommands
}

var sindexMetrics map[string]metricType

func (siw *SindexWatcher) refresh(o *Observer, infoKeys []string, rawMetrics map[string]string, ch chan<- prometheus.Metric) error {
	if config.Aerospike.DisableSindexMetrics {
		// disabled
		return nil
	}

	if sindexMetrics == nil {
		sindexMetrics = getFilteredMetrics(sindexRawMetrics, config.Aerospike.SindexMetricsAllowlist, config.Aerospike.SindexMetricsAllowlistEnabled, config.Aerospike.SindexMetricsBlocklist, config.Aerospike.SindexMetricsBlocklistEnabled)
	}

	for _, sindex := range infoKeys {
		sindexInfoKey := strings.ReplaceAll(sindex, "sindex/", "")
		sindexInfoKeySplit := strings.Split(sindexInfoKey, "/")
		nsName := sindexInfoKeySplit[0]
		sindexName := sindexInfoKeySplit[1]
		log.Tracef("sindex-stats:%s:%s:%s", nsName, sindexName, rawMetrics[sindex])

		sindexObserver := make(MetricMap, len(sindexMetrics))
		for m, t := range sindexMetrics {
			sindexObserver[m] = makeMetric("aerospike_sindex", m, t, config.AeroProm.MetricLabels, "cluster_name", "service", "ns", "sindex")
		}

		stats := parseStats(rawMetrics[sindex], ";")
		for stat, pm := range sindexObserver {
			v, exists := stats[stat]
			if !exists {
				// not found
				continue
			}

			pv, err := tryConvert(v)
			if err != nil {
				continue
			}

			ch <- prometheus.MustNewConstMetric(pm.desc, pm.valueType, pv, rawMetrics[ikClusterName], rawMetrics[ikService], nsName, sindexName)
		}
	}

	return nil
}
