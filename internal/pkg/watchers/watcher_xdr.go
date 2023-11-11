package watchers

import (
	"strings"

	commons "github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"

	log "github.com/sirupsen/logrus"
)

const (
	KEY_XDR_METADATA = "get-config:context=xdr"
	KEY_XDR_STAT     = "get-stats:context=xdr;dc="
	KEY_XDR_CONFIG   = "get-config:context=xdr;dc="
)

type XdrWatcher struct {
	xdrMetrics map[string]commons.AerospikeStat
}

func (xw *XdrWatcher) PassOneKeys() []string {
	// this is used to fetch the dcs metadata, we send same get-config command to fetch the dc-names required in next steps
	return []string{KEY_XDR_METADATA}
}

func (xw *XdrWatcher) PassTwoKeys(rawMetrics map[string]string) []string {
	log.Tracef("get-config:context=xdr %s", rawMetrics[KEY_XDR_METADATA])

	res := rawMetrics[KEY_XDR_METADATA]
	list := commons.ParseStats(res, ";")
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

	log.Tracef("xdr-passTwoKeys %s", infoKeys)

	return infoKeys
}

// refresh prom metrics - parse the given rawMetrics (both config and stats ) and push to given channel
func (xw *XdrWatcher) Refresh(infoKeys []string, rawMetrics map[string]string) ([]WatcherMetric, error) {

	if xw.xdrMetrics == nil {
		xw.xdrMetrics = make(map[string]commons.AerospikeStat)
	}

	clusterName := rawMetrics[commons.Infokey_ClusterName]
	service := rawMetrics[commons.Infokey_Service]

	var metrics_to_send = []WatcherMetric{}

	for _, key := range infoKeys {

		xdrRawMetrics := rawMetrics[key]
		// find and construct metric name
		dcName, ns, metricPrefix := xw.constructMetricNamePrefix(key)
		l_metrics_to_send := xw.handleRefresh(key, xdrRawMetrics, clusterName, service, dcName, ns, metricPrefix)
		metrics_to_send = append(metrics_to_send, l_metrics_to_send...)
	}

	return metrics_to_send, nil
}

// utility constructs the name of the metric according to the level of the stat
// according to the order stat dc/namespace & config dc/namespace
func (xw *XdrWatcher) constructMetricNamePrefix(infoKeyToProcess string) (string, string, string) {
	// get-stats:context=xdr;dc=xdr_backup_dc_asdev20 -- Process with old-metric-name format
	// get-config:context=xdr;dc=xdr_second_backup_dc_asdev20 -- Process with new-metric-name format
	// get-stats:context=xdr;dc=xdr_second_backup_dc_asdev20;namespace=ns_test_4 -- Process with new-metric-name format
	// get-config:context=xdr;dc=xdr_second_backup_dc_asdev20;namespace=vendors -- Process with new-metric-name format

	// splits the string into 3 parts, is-cfg/stat, dcname and namespace
	kvInfoKeyToProcess := commons.ParseStats(infoKeyToProcess, ";")
	_, cfgOk := kvInfoKeyToProcess["get-config:context"]
	_, statOk := kvInfoKeyToProcess["get-stats:context"]
	dcName := kvInfoKeyToProcess["dc"]
	nsName, nsOk := kvInfoKeyToProcess["namespace"]

	// either this is a config key or a stat having namespace (both are new use-cases) hence handle here

	if cfgOk && nsOk {
		return dcName, nsName, ("dc_namespace_")
	} else if statOk && nsOk {
		return dcName, nsName, ("dc_namespace_")
	} else if cfgOk {
		return dcName, nsName, ("dc_")
	}

	return dcName, nsName, "" // no-prefix/default i.e. no suffix like "dc" / "dc_namespace"
}

func (xw *XdrWatcher) handleRefresh(infoKeyToProcess string, xdrRawMetrics string,
	clusterName string, service string, dcName string, ns string, metricPrefix string) []WatcherMetric {
	log.Tracef("xdr-%s:%s", infoKeyToProcess, xdrRawMetrics)

	stats := commons.ParseStats(xdrRawMetrics, ";")
	var metrics_to_send = []WatcherMetric{}
	for stat, value := range stats {

		pv, err := commons.TryConvert(value)
		if err != nil {
			continue
		}
		asMetric, exists := xw.xdrMetrics[stat]
		dynamicStatname := metricPrefix + stat

		if !exists {
			asMetric = commons.NewAerospikeStat(commons.CTX_XDR, dynamicStatname)
			xw.xdrMetrics[stat] = asMetric
		}

		labels := []string{commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE, commons.METRIC_LABEL_DC_NAME}
		labelValues := []string{clusterName, service, dcName}

		// if namespace exists, add it to the label and label-values array
		if len(ns) > 0 {
			labels = []string{commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE, commons.METRIC_LABEL_DC_NAME, commons.METRIC_LABEL_NS}
			labelValues = []string{clusterName, service, dcName, ns}
		}

		// pushToPrometheus(asMetric, pv, labels, labelsValues, ch)
		metrics_to_send = append(metrics_to_send, WatcherMetric{asMetric, pv, labels, labelValues})
	}

	return metrics_to_send
}
