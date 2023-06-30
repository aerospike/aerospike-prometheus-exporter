package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/prometheus/client_golang/prometheus"

	log "github.com/sirupsen/logrus"
)

type NamespaceWatcher struct{}

func (nw *NamespaceWatcher) describe(ch chan<- *prometheus.Desc) {}

func (nw *NamespaceWatcher) passOneKeys() []string {
	return []string{"namespaces"}
}

func (nw *NamespaceWatcher) passTwoKeys(rawMetrics map[string]string) []string {
	s := rawMetrics["namespaces"]
	list := strings.Split(s, ";")

	var infoKeys []string
	for _, k := range list {
		infoKeys = append(infoKeys, "namespace/"+k)
	}

	return infoKeys
}

// Filtered namespace metrics. Populated by getFilteredMetrics() based on the config.Aerospike.NamespaceMetricsAllowlist, config.Aerospike.NamespaceMetricsBlocklist and namespaceRawMetrics.
var namespaceMetrics = make(map[string]AerospikeStat)

// Regex for identifying storage-engine stats.
var seDynamicExtractor = regexp.MustCompile(`storage\-engine\.(?P<type>file|device)\[(?P<idx>\d+)\]\.(?P<metric>.+)`)

func (nw *NamespaceWatcher) refresh(ott *Observer, infoKeys []string, rawMetrics map[string]string, ch chan<- prometheus.Metric) error {
	if isTestcaseMode() {
		fmt.Println("Reinitializing namespaceMetrics(...) ")
		namespaceMetrics = make(map[string]AerospikeStat)
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

				compositeStatName := "storage-engine_" + statType + "_" + statName
				asMetric, exists := namespaceMetrics[compositeStatName]

				if !exists {
					asMetric = newAerospikeStat(CTX_NAMESPACE, compositeStatName)
					namespaceMetrics[compositeStatName] = asMetric
				}

				if asMetric.isAllowed {
					// fmt.Println("namespaces: checking for stat: ", compositeStatName, " is-ALLOWED? : ", nsMetric.isAllowed)
					deviceOrFileName := stats["storage-engine."+statType+"["+statIndex+"]"]

					desc, valueType := asMetric.makePromeMetric(METRIC_LABEL_CLUSTER_NAME, METRIC_LABEL_SERVICE, METRIC_LABEL_NS, statType+"_index", statType)
					ch <- prometheus.MustNewConstMetric(desc, valueType, pv, rawMetrics[ikClusterName], rawMetrics[ikService], nsName, statIndex, deviceOrFileName)
				}
			} else { // regular stat (i.e. non-storage-engine related)
				asMetric, exists := namespaceMetrics[stat]

				if !exists {
					asMetric = newAerospikeStat(CTX_NAMESPACE, stat)
					namespaceMetrics[stat] = asMetric
				}

				if asMetric.isAllowed {
					desc, valueType := asMetric.makePromeMetric(METRIC_LABEL_CLUSTER_NAME, METRIC_LABEL_SERVICE, METRIC_LABEL_NS)
					ch <- prometheus.MustNewConstMetric(desc, valueType, pv, rawMetrics[ikClusterName], rawMetrics[ikService], nsName)
				}

			}

		}
	}
	return nil
}
