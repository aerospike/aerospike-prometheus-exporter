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

			// holds the labels, values and stat holds the object by normal-stat/storage-engine-stat
			var labels []string
			var labelValues []string
			var asMetric AerospikeStat

			// process storage engine stat
			constructedStatname := ""

			if len(match) == 4 {
				statType := match[1]
				statIndex := match[2]
				statName := match[3]

				constructedStatname = STORAGE_ENGINE + statType + "_" + statName
				deviceOrFileName := stats["storage-engine."+statType+"["+statIndex+"]"]

				labels = []string{METRIC_LABEL_CLUSTER_NAME, METRIC_LABEL_SERVICE, METRIC_LABEL_NS, statType + "_index", statType}
				labelValues = []string{rawMetrics[ikClusterName], rawMetrics[ikService], nsName, statIndex, deviceOrFileName}

			} else { // regular stat (i.e. non-storage-engine related)
				constructedStatname = stat
				labels = []string{METRIC_LABEL_CLUSTER_NAME, METRIC_LABEL_SERVICE, METRIC_LABEL_NS}
				labelValues = []string{rawMetrics[ikClusterName], rawMetrics[ikService], nsName}

			}

			asMetric, exists := nw.namespaceStats[constructedStatname]

			if !exists {
				asMetric = newAerospikeStat(CTX_NAMESPACE, constructedStatname)
				nw.namespaceStats[constructedStatname] = asMetric
			}

			if asMetric.isAllowed {
				pushToPrometheus(asMetric, pv, labels, labelValues, ch)
			}

		}
	}
	return nil
}
