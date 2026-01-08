package statprocessors

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

type XdrStatsProcessor struct {
	xdrMetrics map[string]AerospikeStat
}

func (xw *XdrStatsProcessor) PassOneKeys() []string {
	// this is used to fetch the dcs metadata, we send same get-config command to fetch the dc-names required in next steps
	log.Tracef("xdr-passonekeys:%s", []string{KEY_XDR_METADATA})
	return []string{KEY_XDR_METADATA}
}

func (xw *XdrStatsProcessor) PassTwoKeys(passOneStats map[string]string) []string {
	log.Tracef("get-config:context=xdr:%s", passOneStats[KEY_XDR_METADATA])

	res := passOneStats[KEY_XDR_METADATA]
	list := commons.ParseStats(res, ";")
	dcsList := strings.Split(list["dcs"], ",")

	// XDR stats and configs are at Namespace level also
	listNamespaces := strings.Split(passOneStats[KEY_NS_METADATA], ";")

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

	log.Tracef("xdr-passtwokeys:%s", infoKeys)

	return infoKeys
}

// refresh prom metrics - parse the given rawMetrics (both config and stats ) and push to given channel
func (xw *XdrStatsProcessor) Refresh(infoKeys []string, rawMetrics map[string]string) ([]AerospikeStat, error) {

	if xw.xdrMetrics == nil {
		xw.xdrMetrics = make(map[string]AerospikeStat)
	}

	var allMetricsToSend = []AerospikeStat{}

	for _, key := range infoKeys {

		xdrRawMetrics := rawMetrics[key]
		// find and construct metric name
		dcName, ns, metricPrefix := xw.constructMetricNamePrefix(key)
		tmpXdrMetricsToSend := xw.handleRefresh(key, xdrRawMetrics, dcName, ns, metricPrefix)

		allMetricsToSend = append(allMetricsToSend, tmpXdrMetricsToSend...)
	}

	return allMetricsToSend, nil
}

// utility constructs the name of the metric according to the level of the stat
// according to the order stat dc/namespace & config dc/namespace
func (xw *XdrStatsProcessor) constructMetricNamePrefix(infoKeyToProcess string) (string, string, string) {
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

func (xw *XdrStatsProcessor) handleRefresh(infoKeyToProcess string, xdrRawMetrics string,
	dcName string, ns string, metricPrefix string) []AerospikeStat {
	log.Tracef("xdr-%s:%s", infoKeyToProcess, xdrRawMetrics)

	stats := commons.ParseStats(xdrRawMetrics, ";")
	var xdrMetricsToSend = []AerospikeStat{}
	for stat, value := range stats {

		pv, err := commons.TryConvert(value)
		if err != nil {
			continue
		}
		asMetric, exists := xw.xdrMetrics[stat]
		dynamicStatname := metricPrefix + stat

		if !exists {
			allowed := isMetricAllowed(commons.CTX_XDR, stat)
			asMetric = NewAerospikeStat(commons.CTX_XDR, dynamicStatname, allowed)
			xw.xdrMetrics[stat] = asMetric
		}

		labels := []string{commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE, commons.METRIC_LABEL_DC_NAME}
		labelValues := []string{ClusterName, Service, dcName}

		// if namespace exists, add it to the label and label-values array
		if len(ns) > 0 {
			labels = []string{commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE, commons.METRIC_LABEL_DC_NAME, commons.METRIC_LABEL_NS}
			labelValues = []string{ClusterName, Service, dcName, ns}
		}

		// pushToPrometheus(asMetric, pv, labels, labelsValues, ch)
		asMetric.updateValues(pv, labels, labelValues)
		xdrMetricsToSend = append(xdrMetricsToSend, asMetric)
	}

	return xdrMetricsToSend
}
