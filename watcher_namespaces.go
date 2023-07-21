package main

import (
	"regexp"
	"strings"

	"github.com/prometheus/client_golang/prometheus"

	log "github.com/sirupsen/logrus"
)

const (
	KEY_NS_METADATA = "namespaces"
)

type NamespaceWatcher struct {
	namespaceStats map[string]AerospikeStat
}

func (nw *NamespaceWatcher) describe(ch chan<- *prometheus.Desc) {}

func (nw *NamespaceWatcher) passOneKeys() []string {
	// we are sending key "namespaces", server returns all the configs and stats in single call, unlike node-stats, xdr
	return []string{KEY_NS_METADATA}
}

func (nw *NamespaceWatcher) passTwoKeys(rawMetrics map[string]string) []string {
	s := rawMetrics[KEY_NS_METADATA]
	list := strings.Split(s, ";")

	var infoKeys []string
	for _, k := range list {
		infoKeys = append(infoKeys, "namespace/"+k)
	}

	return infoKeys
}

// All (allowed/blocked) namespace stats. Based on the config.Aerospike.NamespaceMetricsAllowlist, config.Aerospike.NamespaceMetricsBlocklist.
// TODO: move this to NamespaceWatcher -- as thie belongs there
// var namespaceStats = make(map[string]AerospikeStat)

// Regex for identifying storage-engine stats.

func (nw *NamespaceWatcher) refresh(ott *Observer, infoKeys []string, rawMetrics map[string]string, ch chan<- prometheus.Metric) error {
	seDynamicExtractor := regexp.MustCompile(`storage\-engine\.(?P<type>file|device)\[(?P<idx>\d+)\]\.(?P<metric>.+)`)
	if nw.namespaceStats == nil {
		nw.namespaceStats = make(map[string]AerospikeStat)
	}

	for _, ns := range infoKeys {
		nsName := strings.ReplaceAll(ns, "namespace/", "")
		log.Tracef("namespace-stats:%s:%s", nsName, rawMetrics[ns])

		stats := parseStats(rawMetrics[ns], ";")
		for stat, value := range stats {

			pv, err := tryConvert(value)
			if err != nil {
				continue
			}

			// to find regular metric or storage-engine metric, we split stat [using: seDynamicExtractor RegEx]
			//    after splitting, a storage-engine stat has 4 elements other stats have 3 elements
			match := seDynamicExtractor.FindStringSubmatch(stat)

			// process storage engine stat
			if len(match) == 4 {
				statType := match[1]
				statIndex := match[2]
				statName := match[3]

				compositeStatName := STORAGE_ENGINE + statType + "_" + statName
				asMetric, exists := nw.namespaceStats[compositeStatName]

				if !exists {
					asMetric = newAerospikeStat(CTX_NAMESPACE, compositeStatName)
					nw.namespaceStats[compositeStatName] = asMetric
				}

				if asMetric.isAllowed {
					deviceOrFileName := stats["storage-engine."+statType+"["+statIndex+"]"]

					// handle any panic from prometheus, this may occur when prom encounters a config/stat with special characters
					defer func() {
						if r := recover(); r != nil {
							log.Tracef("namespace-stats: recovered from panic while handling stat %s in %s", stat, nsName)
						}
					}()

					desc, valueType := asMetric.makePromMetric(METRIC_LABEL_CLUSTER_NAME, METRIC_LABEL_SERVICE, METRIC_LABEL_NS, statType+"_index", statType)
					ch <- prometheus.MustNewConstMetric(desc, valueType, pv, rawMetrics[ikClusterName], rawMetrics[ikService], nsName, statIndex, deviceOrFileName)
					// desc, valueType := asMetric.makePromMetric(METRIC_LABEL_CLUSTER_NAME, METRIC_LABEL_SERVICE, METRIC_LABEL_NS, statType+"_index", statType)
					// nw.pushMetricToPrometheus(stat, pv, rawMetrics[ikClusterName], rawMetrics[ikService], nsName, desc, valueType, ch)
				}
			} else { // regular stat (i.e. non-storage-engine related)
				asMetric, exists := nw.namespaceStats[stat]

				if !exists {
					asMetric = newAerospikeStat(CTX_NAMESPACE, stat)
					nw.namespaceStats[stat] = asMetric
				}

				if asMetric.isAllowed {

					// // handle any panic from prometheus, this may occur when prom encounters a config/stat with special characters
					// defer func() {
					// 	if r := recover(); r != nil {
					// 		log.Tracef("namespace-stats: recovered from panic while handling stat %s in %s", stat, nsName)
					// 	}
					// }()

					// desc, valueType := asMetric.makePromMetric(METRIC_LABEL_CLUSTER_NAME, METRIC_LABEL_SERVICE, METRIC_LABEL_NS)
					// ch <- prometheus.MustNewConstMetric(desc, valueType, pv, rawMetrics[ikClusterName], rawMetrics[ikService], nsName)

					desc, valueType := asMetric.makePromMetric(METRIC_LABEL_CLUSTER_NAME, METRIC_LABEL_SERVICE, METRIC_LABEL_NS)
					nw.pushMetricToPrometheus(stat, pv, rawMetrics[ikClusterName], rawMetrics[ikService], nsName, desc, valueType, ch)
				}

			}

		}
	}
	return nil
}

func (nw *NamespaceWatcher) pushMetricToPrometheus(stat string, pv float64, clusterName string, service string, nsName string,
	desc *prometheus.Desc, valueType prometheus.ValueType, ch chan<- prometheus.Metric) {

	// handle any panic from prometheus, this may occur when prom encounters a config/stat with special characters
	defer func() {
		if r := recover(); r != nil {
			log.Tracef("namespace-stats: recovered from panic while handling stat %s in %s", stat, nsName)
		}
	}()

	ch <- prometheus.MustNewConstMetric(desc, valueType, pv, clusterName, service, nsName)

}
