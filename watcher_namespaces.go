package main

import (
	"regexp"
	"strings"

	"github.com/prometheus/client_golang/prometheus"

	log "github.com/sirupsen/logrus"
)

var regexToExtractArrayStats = map[string]string{
	STORAGE_ENGINE: "storage\\-engine\\.(?P<type>file|device)\\[(?P<idx>\\d+)\\]\\.(?P<metric>.+)",
	INDEX_TYPE:     "index\\-type\\.(?P<type>mount)\\[(?P<idx>\\d+)\\]\\.(?P<metric>.+)",
	SINDEX_TYPE:    "sindex\\-type\\.(?P<type>mount)\\[(?P<idx>\\d+)\\]\\.(?P<metric>.+)",
}

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

	log.Tracef("namespaces:%s", s)

	var infoKeys []string
	for _, k := range list {
		infoKeys = append(infoKeys, "namespace/"+k)
	}

	return infoKeys
}

func (nw *NamespaceWatcher) refresh(ott *Observer, infoKeys []string, rawMetrics map[string]string, ch chan<- prometheus.Metric) error {
	if nw.namespaceStats == nil {
		nw.namespaceStats = make(map[string]AerospikeStat)
	}

	// counter := 1
	for _, ns := range infoKeys {
		nsName := strings.ReplaceAll(ns, "namespace/", "")
		log.Tracef("namespace-stats:%s:%s", nsName, rawMetrics[ns])

		stats := parseStats(rawMetrics[ns], ";")
		for stat, value := range stats {

			pv, err := tryConvert(value)
			if err != nil {
				continue
			}

			// to find regular metric or index-type/storage-engine metric, check prefix and is it an array [i.e.: index-type.mount[0]]
			//
			var labels []string
			var labelValues []string
			constructedStatname := ""

			// check persistance-type
			deviceType, isArrayType := nw.checkStatPersistanceType(stat, stats)

			constructedStatname = stat
			labels = []string{METRIC_LABEL_CLUSTER_NAME, METRIC_LABEL_SERVICE, METRIC_LABEL_NS}
			labelValues = []string{rawMetrics[ikClusterName], rawMetrics[ikService], nsName}

			if isArrayType {
				constructedStatname, labels, labelValues = nw.handleArrayStats(nsName, stat, pv, stats, deviceType, rawMetrics, ch)
			}

			// check and include persistance-type if they are defined/found
			indexType := stats[INDEX_TYPE]
			sindexType := stats[SINDEX_TYPE]

			// if stat is index-type or sindex-type , append addl label
			if strings.HasPrefix(deviceType, INDEX_TYPE) && len(indexType) > 0 {
				labels = append(labels, "index")
				labelValues = append(labelValues, indexType)
			} else if strings.HasPrefix(deviceType, SINDEX_TYPE) && len(sindexType) > 0 {
				labels = append(labels, "sindex")
				labelValues = append(labelValues, sindexType)
			}

			// handleArrayStats(..) will return empty-string if unable to handle the array-stat
			if len(constructedStatname) > 0 {
				asMetric, exists := nw.namespaceStats[constructedStatname]

				if !exists {
					asMetric = newAerospikeStat(CTX_NAMESPACE, constructedStatname)
					nw.namespaceStats[constructedStatname] = asMetric
				}

				// push to prom-channel
				pushToPrometheus(asMetric, pv, labels, labelValues, ch)
			}
		}
	}
	return nil
}

// checks if the given stat is a storage-engine or index-type, depending on which type we decide how to process
//
// multiple values are returnd by server example: storage-engine.file[0].<remaining-stat-name>
func (nw *NamespaceWatcher) checkStatPersistanceType(statToProcess string,
	allNamespaceStats map[string]string) (string, bool) {

	// if starts-with "index-type", then
	//    return index-type, <is-array-or-normal-stat>

	if strings.HasPrefix(statToProcess, INDEX_TYPE) {
		return INDEX_TYPE, strings.Contains(statToProcess, "[")
	} else if strings.HasPrefix(statToProcess, SINDEX_TYPE) {
		return SINDEX_TYPE, strings.Contains(statToProcess, "[")
	} else if strings.HasPrefix(statToProcess, STORAGE_ENGINE) {
		return STORAGE_ENGINE, strings.Contains(statToProcess, "[")
	}

	return "", false
}

// Utility to handle array style stats like storage-engine or index-type etc.,
// example: storage-device..file[0].defrag_q
//
// Each part of the stat is split into 4 groups using a regex,
// each value of the 4 groups represents type like stats-type, index-number, sub-stat-name (like file[0].age)
// - example: group[0]=<full-stat> , group[1]= stat-type, group[2]= array-index, group[3]= sub-stat-name
func (nw *NamespaceWatcher) handleArrayStats(nsName string, statToProcess string, pv float64,
	allNamespaceStats map[string]string, deviceType string, rawMetrics map[string]string,
	ch chan<- prometheus.Metric) (string, []string, []string) {

	regexStr := regexToExtractArrayStats[deviceType]
	dynamicExtractor := regexp.MustCompile(regexStr)

	// to find regular metric or storage-engine metric, we split stat [using: seDynamicExtractor RegEx]
	//    after splitting, a index-type/storage-engine stat has 4 elements other stats have 3 elements

	match := dynamicExtractor.FindStringSubmatch(statToProcess)
	if len(match) != 4 {
		log.Warnf("namespace-stats: stat %s in unexpected format, length is not 4 as per regex", statToProcess)
		return "", nil, nil
	}

	statType := match[1]
	statIndex := match[2]
	statName := match[3]

	compositeStatName := deviceType + "_" + statType + "_" + statName
	deviceOrFileName := allNamespaceStats[deviceType+"."+statType+"["+statIndex+"]"]
	labels := []string{METRIC_LABEL_CLUSTER_NAME, METRIC_LABEL_SERVICE, METRIC_LABEL_NS, statType + "_index", statType}
	labelValues := []string{rawMetrics[ikClusterName], rawMetrics[ikService], nsName, statIndex, deviceOrFileName}

	return compositeStatName, labels, labelValues

}
