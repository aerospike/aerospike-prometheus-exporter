package main

import (
	"regexp"
	"strings"

	"github.com/prometheus/client_golang/prometheus"

	log "github.com/sirupsen/logrus"
)

const (
	STORAGE_ENGINE = "storage-engine"
	INDEX_TYPE     = "index-type"
	SINDEX_TYPE    = "sindex-type"
)

var regexToExtractArrayStats = map[string]string{
	STORAGE_ENGINE: "storage\\-engine\\.(?P<type>file|device)\\[(?P<idx>\\d+)\\]\\.(?P<metric>.+)",
	INDEX_TYPE:     "index\\-type\\.(?P<type>mount)\\[(?P<idx>\\d+)\\]\\.(?P<metric>.+)",
	SINDEX_TYPE:    "sindex\\-type\\.(?P<type>mount)\\[(?P<idx>\\d+)\\]\\.(?P<metric>.+)",
}

type NamespaceWatcher struct {
	namespaceStats map[string]AerospikeStat
}

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

func (nw *NamespaceWatcher) refresh(ott *Observer, infoKeys []string, rawMetrics map[string]string, ch chan<- prometheus.Metric) error {
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

			// to find regular metric or index-type/storage-engine metric, check prefix and is it an array [i.e.: index-type.mount[0]]
			//
			deviceType, isArrayType := nw.checkStatPersistanceType(stat)
			if isArrayType {
				nw.handleArrayStats(nsName, stat, pv, stats, deviceType, rawMetrics, ch)
			} else {
				asMetric, exists := nw.namespaceStats[stat]

				if !exists {
					asMetric = newAerospikeStat(CTX_NAMESPACE, stat)
					nw.namespaceStats[stat] = asMetric
				}

				defer func() {
					// recover from panic if one occures. Set err to nil otherwise.
					if recover() != nil {
						log.Warnf("namespace-stats: recovered from panic while handling stat %s in %s", stat, nsName)
					}
				}()

				if asMetric.isAllowed {
					desc, valueType := asMetric.makePromMetric(METRIC_LABEL_CLUSTER_NAME, METRIC_LABEL_SERVICE, METRIC_LABEL_NS)
					ch <- prometheus.MustNewConstMetric(desc, valueType, pv, rawMetrics[ikClusterName], rawMetrics[ikService], nsName)
				}
			}
		}
	}
	return nil
}

// checks if the ficen stat is a storage-engine or index-type, depending on which type we decide how to process
//
//	multiple values will be returnd by server as storage-engine.file[0].<remaining-stat-name>
func (nw *NamespaceWatcher) checkStatPersistanceType(statToProcess string) (string, bool) {

	// if starts-with index-type
	//   if array-
	//   else
	if strings.HasPrefix(statToProcess, INDEX_TYPE) && strings.Contains(statToProcess, "[") {
		return INDEX_TYPE, true
	} else if strings.HasPrefix(statToProcess, SINDEX_TYPE) && strings.Contains(statToProcess, "[") {
		return SINDEX_TYPE, true
	} else if strings.HasPrefix(statToProcess, STORAGE_ENGINE) && strings.Contains(statToProcess, "[") {
		return STORAGE_ENGINE, true
	}

	return "", false
}

// Utility to handle array tyle stats like storage-engine or index-type etc.,
// example: storage-device..file[0].defrag_q
//
// Each part of the stat is split into 4 groups using a regex,
// each value of the 4 groups represents type like stats-type, index-number, sub-stat-name (like file[0].age)
// - example: group[0]=<full-stat> , group[1]= stat-type, group[2]= array-index, group[3]= sub-stat-name
func (nw *NamespaceWatcher) handleArrayStats(nsName string, statToProcess string, pv float64,
	allNamespaceStats map[string]string, deviceType string, rawMetrics map[string]string,
	ch chan<- prometheus.Metric) {

	regexStr := regexToExtractArrayStats[deviceType]
	dynamicExtractor := regexp.MustCompile(regexStr)

	// to find regular metric or storage-engine metric, we split stat [using: seDynamicExtractor RegEx]
	//    after splitting, a index-type/storage-engine stat has 4 elements other stats have 3 elements

	match := dynamicExtractor.FindStringSubmatch(statToProcess)
	if len(match) != 4 {
		//TODO: logWarn
		log.Warnf("namespace-stats: stat %s in unexpected format, length is not 4 as per regex", statToProcess)
		return
	}

	// get persistance-type
	indexType := allNamespaceStats[INDEX_TYPE]
	// sindexType := allNamespaceStats[SINDEX_TYPE]

	statType := match[1]
	statIndex := match[2]
	statName := match[3]

	compositeStatName := deviceType + "_" + statType + "_" + statName
	asMetric, exists := nw.namespaceStats[compositeStatName]

	if !exists {
		asMetric = newAerospikeStat(CTX_NAMESPACE, compositeStatName)
		nw.namespaceStats[compositeStatName] = asMetric
	}

	defer func() {
		// recover from panic if one occurs.
		if recover() != nil {
			log.Warnf("namespace-stats: recovered from panic while handling stat %s in %s", statToProcess, nsName)
		}
	}()

	if asMetric.isAllowed {
		deviceOrFileName := allNamespaceStats[deviceType+"."+statType+"["+statIndex+"]"]

		defer func() {
			// recover from panic if one occures. Set err to nil otherwise.
			if recover() != nil {
				log.Warnf("namespace-stats: recovered from panic while handling stat %s in %s", compositeStatName, nsName)
			}
		}()

		desc, valueType := asMetric.makePromMetric(METRIC_LABEL_CLUSTER_NAME, METRIC_LABEL_SERVICE, METRIC_LABEL_NS, statType+"_index", statType, "persistance")
		ch <- prometheus.MustNewConstMetric(desc, valueType, pv, rawMetrics[ikClusterName], rawMetrics[ikService], nsName, statIndex, deviceOrFileName, indexType)
	}

}
