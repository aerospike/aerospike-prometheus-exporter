package main

import (
	"strings"

	"github.com/prometheus/client_golang/prometheus"

	log "github.com/sirupsen/logrus"
)

const (
	KEY_XDR_METADATA = "get-config:context=xdr"
	KEY_XDR_STAT     = "get-stats:context=xdr;dc="
	KEY_XDR_CONFIG   = "get-config:context=xdr;dc="
)

type XdrWatcher struct {
	xdrMetrics map[string]AerospikeStat
}

func (xw *XdrWatcher) describe(ch chan<- *prometheus.Desc) {}

func (xw *XdrWatcher) passOneKeys() []string {
	// this is used to fetch the dcs metadata, we send same get-config command to fetch the dc-names required in next steps
	return []string{KEY_XDR_METADATA}
}

func (xw *XdrWatcher) passTwoKeys(rawMetrics map[string]string) []string {
	res := rawMetrics[KEY_XDR_METADATA]
	list := parseStats(res, ";")
	dcsList := strings.Split(list["dcs"], ",")

	// XDR stats and configs are at Namespace level also
	listNamespaces := strings.Split(rawMetrics[KEY_NS_METADATA], ";")

	var infoKeys []string
	for _, dc := range dcsList {
		if dc != "" {
			infoKeys = append(infoKeys, KEY_XDR_STAT+dc)   // Existing: aerospike_xdr
			infoKeys = append(infoKeys, KEY_XDR_CONFIG+dc) // TODO: aerospike_xdr_dc

			// XDR configs and stats will be for each-namespace also
			// command structure will be like get-config:context=xdr;dc=backup_dc_as8;namespace=test
			// command structure will be like get-stat:context=xdr;dc=backup_dc_as8;namespace=test
			for _, ns := range listNamespaces {
				infoKeys = append(infoKeys, KEY_XDR_CONFIG+dc+";namespace="+ns) // TODO: aerospike_xdr_dc_namespace
				infoKeys = append(infoKeys, KEY_XDR_STAT+dc+";namespace="+ns)   // TODO: aerospike_xdr_dc_namespace
			}
		}
	}

	return infoKeys
}

// refresh prom metrics - parse the given rawMetrics (both config and stats ) and pushed into given channel
func (xw *XdrWatcher) refresh(o *Observer, infoKeys []string, rawMetrics map[string]string, ch chan<- prometheus.Metric) error {

	if xw.xdrMetrics == nil {
		xw.xdrMetrics = make(map[string]AerospikeStat)
	}

	clusterName := rawMetrics[ikClusterName]
	service := rawMetrics[ikService]
	for _, key := range infoKeys {
		xdrRawMetrics := rawMetrics[key]

		xw.handleRefreshe(o, key, xdrRawMetrics, clusterName, service, ch)

		// dcName, ns, newWay := xw.isProcessNewway(key)
		// if newWay {
		// 	xw.processValues(o, key, xdrRawMetrics, clusterName, service, dcName, ns, ch)
		// } else {
		// 	xw.refreshStatsOldway(o, key, xdrRawMetrics, clusterName, service, dcName, ns, ch)
		// }
	}

	return nil
}

func (xw *XdrWatcher) constructMetricName(infoKeyToProcess string, stat string) (string, string, string) {
	// get-stats:context=xdr;dc=xdr_backup_dc_asdev20 -- Process with old-metric-name format
	// get-config:context=xdr;dc=xdr_second_backup_dc_asdev20 -- Process with new-metric-name format
	// get-stats:context=xdr;dc=xdr_second_backup_dc_asdev20;namespace=ns_test_4 -- Process with new-metric-name format
	// get-config:context=xdr;dc=xdr_second_backup_dc_asdev20;namespace=vendors -- Process with new-metric-name format

	// splits the string into 3 parts, is-cfg/stat, dcname and namespace
	kvInfoKeyToProcess := parseStats(infoKeyToProcess, ";")
	_, cfgOk := kvInfoKeyToProcess["get-config:context"]
	_, statOk := kvInfoKeyToProcess["get-stats:context"]
	dcName := kvInfoKeyToProcess["dc"]
	nsName, nsOk := kvInfoKeyToProcess["namespace"]

	// either this is a config key or a stat having namespace (both are new use-cases) hence handle here

	if cfgOk && nsOk {
		return dcName, nsName, ("dc_namespace" + "_" + stat)
	} else if statOk && nsOk {
		return dcName, nsName, ("dc_namespace" + "_" + stat)
	} else if cfgOk {
		return dcName, nsName, ("dc" + "_" + stat)
	}

	return dcName, nsName, stat // default i.e. no suffix like "dc" / "dc_namespace"
}

func (xw *XdrWatcher) handleRefreshe(o *Observer, infoKeyToProcess string, xdrRawMetrics string,
	clusterName string, service string, ch chan<- prometheus.Metric) {
	log.Tracef("xdr-%s:%s", infoKeyToProcess, xdrRawMetrics)

	stats := parseStats(xdrRawMetrics, ";")
	for stat, value := range stats {

		pv, err := tryConvert(value)
		if err != nil {
			continue
		}
		asMetric, exists := xw.xdrMetrics[stat]

		dcName, ns, dynamicStatname := xw.constructMetricName(infoKeyToProcess, stat)

		if !exists {
			asMetric = newAerospikeStat(CTX_XDR, dynamicStatname)
			xw.xdrMetrics[stat] = asMetric
		}

		if asMetric.isAllowed {

			labels := []string{METRIC_LABEL_CLUSTER_NAME, METRIC_LABEL_SERVICE, METRIC_LABEL_DC_NAME}
			labelsValues := []string{clusterName, service, dcName}

			// if namespace exists, add it to the label and label-values array
			if len(ns) > 0 {
				labels = []string{METRIC_LABEL_CLUSTER_NAME, METRIC_LABEL_SERVICE, METRIC_LABEL_DC_NAME, METRIC_LABEL_NS}
				labelsValues = []string{clusterName, service, dcName, ns}
			}

			pushToPrometheus(asMetric, pv, labels, labelsValues, ch)
		}

	}

}

// func (xw *XdrWatcher) isProcessNewway(infoKeyToProcess string) (string, string, bool) {
// 	// get-stats:context=xdr;dc=xdr_backup_dc_asdev20 -- Process with old-metric-name format
// 	// get-config:context=xdr;dc=xdr_second_backup_dc_asdev20 -- Process with new-metric-name format
// 	// get-stats:context=xdr;dc=xdr_second_backup_dc_asdev20;namespace=ns_test_4 -- Process with new-metric-name format
// 	// get-config:context=xdr;dc=xdr_second_backup_dc_asdev20;namespace=vendors -- Process with new-metric-name format

// 	// splits the string into 3 parts, is-cfg/stat, dcname and namespace
// 	kvInfoKeyToProcess := parseStats(infoKeyToProcess, ";")
// 	_, cfgOk := kvInfoKeyToProcess["get-config:context"]
// 	dcName := kvInfoKeyToProcess["dc"]
// 	ns, nsOk := kvInfoKeyToProcess["namespace"]

// 	// either this is a config key or a stat having namespace (both are new use-cases) hence handle here
// 	if cfgOk || nsOk {
// 		return dcName, ns, true
// 	}

// 	return dcName, ns, false
// }

// // internal utility process 3 categories xdr-dc-config, xdr-dc-namespace-config, xdr-dc-namespace-stats
// // this constructs metric-name in a hierarchy like xdr_dc_<stat-name>, xdr_dc_namespace_<stat-name>,
// // to avoid bct issues, only config/dc-namespace stats are parsed here
// func (xw *XdrWatcher) processValues(o *Observer, infoKeyToProcess string, xdrRawMetrics string,
// 	clusterName string, service string, dcName string, ns string,
// 	ch chan<- prometheus.Metric) {

// 	log.Tracef("xdr-%s:%s", infoKeyToProcess, xdrRawMetrics)

// 	list := parseStats(xdrRawMetrics, ";")

// 	// list of metric labels and corresponding label-vales
// 	labels := []string{METRIC_LABEL_CLUSTER_NAME, METRIC_LABEL_SERVICE, METRIC_LABEL_DC_NAME}
// 	labelsValues := []string{clusterName, service, dcName}

// 	// If namespace exists in the infoKeyToProcess, then this is at dc+namespace level
// 	prefixToAppendToStatName := "dc"
// 	if len(ns) > 0 {
// 		prefixToAppendToStatName = "dc_namespace"
// 		labels = []string{METRIC_LABEL_CLUSTER_NAME, METRIC_LABEL_SERVICE, METRIC_LABEL_DC_NAME, METRIC_LABEL_NS}
// 		labelsValues = []string{clusterName, service, dcName, ns}
// 	}
// 	for stat, value := range list {
// 		pv, err := tryConvert(value)
// 		if err != nil {
// 			continue
// 		}

// 		// construct composite stat-name, metric name will be in form: aerospike_xdr_dc_namespace_<stat-name>
// 		compositeStatName := prefixToAppendToStatName + "_" + stat

// 		asMetric, exists := xw.xdrMetrics[compositeStatName]
// 		if !exists {
// 			asMetric = newAerospikeStat(CTX_XDR, compositeStatName)
// 			xw.xdrMetrics[compositeStatName] = asMetric
// 		}

// 		// // handle any panic from prometheus, this may occur when prom encounters a config/stat with special characters
// 		// defer func() {
// 		// 	if r := recover(); r != nil {
// 		// 		log.Warnf("xdr-config: recovered from panic while handling config %s in %s", stat, dcName)
// 		// 	}
// 		// }()

// 		if asMetric.isAllowed {
// 			// desc, valueType := asMetric.makePromMetric(labels...)
// 			// ch <- prometheus.MustNewConstMetric(desc, valueType, pv, labels_values...)
// 			pushToPrometheus(asMetric, pv, labels, labelsValues, ch)
// 		}

// 	}

// }

// func (xw *XdrWatcher) refreshStatsOldway(o *Observer, infoKeyToProcess string, xdrRawMetrics string,
// 	clusterName string, service string, dcName string, ns string,
// 	ch chan<- prometheus.Metric) {
// 	log.Tracef("xdr-%s:%s", infoKeyToProcess, xdrRawMetrics)

// 	stats := parseStats(xdrRawMetrics, ";")
// 	for stat, value := range stats {

// 		pv, err := tryConvert(value)
// 		if err != nil {
// 			continue
// 		}
// 		asMetric, exists := xw.xdrMetrics[stat]
// 		if !exists {
// 			asMetric = newAerospikeStat(CTX_XDR, stat)
// 			xw.xdrMetrics[stat] = asMetric
// 		}

// 		// // handle any panic from prometheus, this may occur when prom encounters a config/stat with special characters
// 		// defer func() {
// 		// 	if r := recover(); r != nil {
// 		// 		log.Warnf("xdr-stats: recovered from panic while handling stat %s in %s", stat, dcName)
// 		// 	}
// 		// }()

// 		if asMetric.isAllowed {
// 			// desc, valueType := asMetric.makePromMetric(METRIC_LABEL_CLUSTER_NAME, METRIC_LABEL_SERVICE, METRIC_LABEL_DC_NAME)
// 			// ch <- prometheus.MustNewConstMetric(desc, valueType, pv, clusterName, service, dcName)

// 			labels := []string{METRIC_LABEL_CLUSTER_NAME, METRIC_LABEL_SERVICE, METRIC_LABEL_DC_NAME}
// 			labelsValues := []string{clusterName, service, dcName}

// 			pushToPrometheus(asMetric, pv, labels, labelsValues, ch)

// 		}

// 	}

// }
