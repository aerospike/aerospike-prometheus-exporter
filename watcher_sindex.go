package main

import (
	"fmt"
	"strings"

	"github.com/prometheus/client_golang/prometheus"

	log "github.com/sirupsen/logrus"
)

type SindexWatcher struct {
	sindexMetrics map[string]AerospikeStat
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

// All (allowed/blocked) Sindex stats. Based on the config.Aerospike.SindexMetricsAllowlist, config.Aerospike.SindexMetricsBlocklist.
// var sindexMetrics = make(map[string]AerospikeStat)

func (siw *SindexWatcher) refresh(o *Observer, infoKeys []string, rawMetrics map[string]string, ch chan<- prometheus.Metric) error {
	if config.Aerospike.DisableSindexMetrics {
		// disabled
		return nil
	}

	if siw.sindexMetrics == nil {
		fmt.Println("Reinitializing sindexStats(...) ")

		siw.sindexMetrics = make(map[string]AerospikeStat)
	}

	for _, sindex := range infoKeys {
		sindexInfoKey := strings.ReplaceAll(sindex, "sindex/", "")
		sindexInfoKeySplit := strings.Split(sindexInfoKey, "/")
		nsName := sindexInfoKeySplit[0]
		sindexName := sindexInfoKeySplit[1]
		log.Tracef("sindex-stats:%s:%s:%s", nsName, sindexName, rawMetrics[sindex])

		stats := parseStats(rawMetrics[sindex], ";")
		for stat, value := range stats {
			pv, err := tryConvert(value)
			if err != nil {
				continue
			}
			asMetric, exists := siw.sindexMetrics[stat]

			if !exists {
				asMetric = newAerospikeStat(CTX_SINDEX, stat)
				siw.sindexMetrics[stat] = asMetric
			}

			if asMetric.isAllowed {
				desc, valueType := asMetric.makePromMetric(METRIC_LABEL_CLUSTER_NAME, METRIC_LABEL_SERVICE, METRIC_LABEL_NS, METRIC_LABEL_SINDEX)
				ch <- prometheus.MustNewConstMetric(desc, valueType, pv, rawMetrics[ikClusterName], rawMetrics[ikService], nsName, sindexName)
			}

		}

	}

	return nil
}
