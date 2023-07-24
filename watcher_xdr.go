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
			infoKeys = append(infoKeys, KEY_XDR_STAT+dc) // Existing: aerospike_xdr
			infoKeys = append(infoKeys, KEY_XDR_CONFIG+dc)
			// XDR configs and stats will be for each-namespace also
			// command structure will be like get-config:context=xdr;dc=backup_dc_as8;namespace=test
			// command structure will be like get-stat:context=xdr;dc=backup_dc_as8;namespace=test
			for _, ns := range listNamespaces {
				infoKeys = append(infoKeys, KEY_XDR_CONFIG+dc+";namespace="+ns)
				infoKeys = append(infoKeys, KEY_XDR_STAT+dc+";namespace="+ns)
			}
		}
	}

	return infoKeys
}

// refresh prom metrics - parse the given rawMetrics (both config and stats ) and push to given channel
func (xw *XdrWatcher) refresh(o *Observer, infoKeys []string, rawMetrics map[string]string, ch chan<- prometheus.Metric) error {

	if xw.xdrMetrics == nil {
		xw.xdrMetrics = make(map[string]AerospikeStat)
	}

	clusterName := rawMetrics[ikClusterName]
	service := rawMetrics[ikService]
	for _, key := range infoKeys {

		xdrRawMetrics := rawMetrics[key]
		// find and construct metric name
		dcName, ns, metricPrefix := xw.constructMetricNamePrefix(key)
		xw.handleRefresh(o, key, xdrRawMetrics, clusterName, service, dcName, ns, metricPrefix, ch)
	}

	return nil
}

// utility constructs the name of the metric according to the level of the stat
// according to the order stat dc/namespace & config dc/namespace
func (xw *XdrWatcher) constructMetricNamePrefix(infoKeyToProcess string) (string, string, string) {
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
		return dcName, nsName, ("dc_namespace")
	} else if statOk && nsOk {
		return dcName, nsName, ("dc_namespace")
	} else if cfgOk {
		return dcName, nsName, ("dc")
	}

	return dcName, nsName, "" // no-prefix/default i.e. no suffix like "dc" / "dc_namespace"
}

func (xw *XdrWatcher) handleRefresh(o *Observer, infoKeyToProcess string, xdrRawMetrics string,
	clusterName string, service string, dcName string, ns string, metricPrefix string,
	ch chan<- prometheus.Metric) {
	log.Tracef("xdr-%s:%s", infoKeyToProcess, xdrRawMetrics)

	stats := parseStats(xdrRawMetrics, ";")
	for stat, value := range stats {

		pv, err := tryConvert(value)
		if err != nil {
			continue
		}
		asMetric, exists := xw.xdrMetrics[stat]
		dynamicStatname := metricPrefix + "_" + stat

		if !exists {
			asMetric = newAerospikeStat(CTX_XDR, dynamicStatname)
			xw.xdrMetrics[stat] = asMetric
		}

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
