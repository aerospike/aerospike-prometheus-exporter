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

var xdr_metric_labels []string

type XdrWatcher struct {
	xdrMetrics map[string]AerospikeStat
}

func (xw *XdrWatcher) describe(ch chan<- *prometheus.Desc) {}

func (xw *XdrWatcher) passOneKeys() []string {
	// this is used to fetch the dcs metadata
	return []string{KEY_XDR_METADATA}
}

func (xw *XdrWatcher) passTwoKeys(rawMetrics map[string]string) []string {
	res := rawMetrics[KEY_XDR_METADATA]
	list := parseStats(res, ";")
	dcsList := strings.Split(list["dcs"], ",")

	// XDR stats and configs are at Namespace level also
	s := rawMetrics[KEY_NS_METADATA]
	listNamespaces := strings.Split(s, ";")

	var infoKeys []string
	for _, dc := range dcsList {
		if dc != "" {
			infoKeys = append(infoKeys, KEY_XDR_STAT+dc)   // Existing: aerospike_xdr
			infoKeys = append(infoKeys, KEY_XDR_CONFIG+dc) // TODO: aerospike_xdr_dc

			// for all-namespaces
			//     infoKeys = append(infoKeys, "get-config:context=xdr;dc="+dc+";namespace="+ns) // TODO: aerospike_xdr_dc_namespace
			//     infoKeys = append(infoKeys, "get-stats:context=xdr;dc="+dc+";namespace="+ns) // TODO: aerospike_xdr_dc_namespace
			//  xdr_lag

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

// All (allowed/blocked) XDR stats. Based on the config.Aerospike.XdrMetricsAllowlist, config.Aerospike.XdrMetricsBlocklist.
func (xw *XdrWatcher) refresh(o *Observer, infoKeys []string, rawMetrics map[string]string, ch chan<- prometheus.Metric) error {

	if xw.xdrMetrics == nil {
		xw.xdrMetrics = make(map[string]AerospikeStat)
	}

	clusterName := rawMetrics[ikClusterName]
	service := rawMetrics[ikService]
	for _, key := range infoKeys {
		dcName, ns, newWay := xw.isProcessNewway(key)
		xdrRawMetrics := rawMetrics[key]
		if newWay {
			xw.processValues(o, key, xdrRawMetrics, clusterName, service, dcName, ns, ch)
		} else {
			xw.refreshStatsOldway(o, key, xdrRawMetrics, clusterName, service, dcName, ns, ch)
		}
	}

	return nil
}

func (xw *XdrWatcher) isProcessNewway(infoKeyToProcess string) (string, string, bool) {
	// get-stats:context=xdr;dc=xdr_backup_dc_asdev20 -- Process with old-metric-name format
	// get-config:context=xdr;dc=xdr_second_backup_dc_asdev20 -- Process with new-metric-name format
	// get-stats:context=xdr;dc=xdr_second_backup_dc_asdev20;namespace=ns_test_4 -- Process with new-metric-name format
	// get-config:context=xdr;dc=xdr_second_backup_dc_asdev20;namespace=vendors -- Process with new-metric-name format

	kvInfoKeyToProcess := parseStats(infoKeyToProcess, ";")
	_, cfgOk := kvInfoKeyToProcess["get-config:context"]
	dcName := kvInfoKeyToProcess["dc"]
	ns, nsOk := kvInfoKeyToProcess["namespace"]
	// either this is a config key or a stat having namespace (both are new use-cases) hence handle in new-way
	if cfgOk || nsOk {
		return dcName, ns, true
	}

	return dcName, ns, false
}

func (xw *XdrWatcher) processValues(o *Observer, infoKeyToProcess string, xdrRawMetrics string,
	clusterName string, service string, dcName string, ns string,
	ch chan<- prometheus.Metric) {

	list := parseStats(xdrRawMetrics, ";")

	// list of metric labels and corresponding label-vales
	labels := []string{METRIC_LABEL_CLUSTER_NAME, METRIC_LABEL_SERVICE, METRIC_LABEL_DC_NAME}
	labels_values := []string{clusterName, service, dcName}

	// If namespace exists in the infoKeyToProcess, then this is at dc+namespace level
	prefixToAppendToStatName := "dc"
	if len(ns) > 0 {
		prefixToAppendToStatName = "dc_namespace"
		labels = []string{METRIC_LABEL_CLUSTER_NAME, METRIC_LABEL_SERVICE, METRIC_LABEL_DC_NAME, METRIC_LABEL_NS}
		labels_values = []string{clusterName, service, dcName, ns}
	}
	for stat, value := range list {
		pv, err := tryConvert(value)
		if err != nil {
			continue
		}

		// construct composite stat-name, metric name will be in form: aerospike_xdr_dc_namespace_<stat-name>
		compositeStatName := prefixToAppendToStatName + "_" + stat

		asMetric, exists := xw.xdrMetrics[compositeStatName]
		if !exists {
			asMetric = newAerospikeStat(CTX_XDR, compositeStatName)
			xw.xdrMetrics[compositeStatName] = asMetric
		}

		// handle any panic from prometheus, this may occur when prom encounters a config/stat with special characters
		defer func() {
			if r := recover(); r != nil {
				log.Warnf("xdr-config: recovered from panic while handling config %s in %s", stat, dcName)
			}
		}()

		if asMetric.isAllowed {
			desc, valueType := asMetric.makePromMetric(labels...)
			ch <- prometheus.MustNewConstMetric(desc, valueType, pv, labels_values...)
		}

	}

}

func (xw *XdrWatcher) refreshStatsOldway(o *Observer, infoKeyToProcess string, xdrRawMetrics string,
	clusterName string, service string, dcName string, ns string,
	ch chan<- prometheus.Metric) {
	log.Tracef("xdr-stats:%s:%s", dcName, xdrRawMetrics)

	stats := parseStats(xdrRawMetrics, ";")
	for stat, value := range stats {

		pv, err := tryConvert(value)
		if err != nil {
			continue
		}
		asMetric, exists := xw.xdrMetrics[stat]
		if !exists {
			asMetric = newAerospikeStat(CTX_XDR, stat)
			xw.xdrMetrics[stat] = asMetric
		}

		// handle any panic from prometheus, this may occur when prom encounters a config/stat with special characters
		defer func() {
			if r := recover(); r != nil {
				log.Warnf("xdr-stats: recovered from panic while handling stat %s in %s", stat, dcName)
			}
		}()

		if asMetric.isAllowed {
			desc, valueType := asMetric.makePromMetric(METRIC_LABEL_CLUSTER_NAME, METRIC_LABEL_SERVICE, METRIC_LABEL_DC_NAME)
			ch <- prometheus.MustNewConstMetric(desc, valueType, pv, clusterName, service, dcName)
		}
	}

}
