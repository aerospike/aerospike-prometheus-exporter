package main

import (
	"strings"

	"github.com/prometheus/client_golang/prometheus"

	log "github.com/sirupsen/logrus"
)

type XdrWatcher struct{}

func (xw *XdrWatcher) describe(ch chan<- *prometheus.Desc) {}

func (xw *XdrWatcher) passOneKeys() []string {
	return []string{"get-config:context=xdr"}
}

func (xw *XdrWatcher) passTwoKeys(rawMetrics map[string]string) []string {
	res := rawMetrics["get-config:context=xdr"]
	list := parseStats(res, ";")
	dcsList := strings.Split(list["dcs"], ",")

	var infoKeys []string
	for _, dc := range dcsList {
		if dc != "" {
			infoKeys = append(infoKeys, "get-stats:context=xdr;dc="+dc)
		}
	}

	return infoKeys
}

// Filtered XDR metrics. Populated by getFilteredMetrics() based on the config.Aerospike.XdrMetricsAllowlist, config.Aerospike.XdrMetricsBlocklist and xdrRawMetrics.
var xdrMetrics = make(map[string]AerospikeStat)

func (xw *XdrWatcher) refresh(o *Observer, infoKeys []string, rawMetrics map[string]string, ch chan<- prometheus.Metric) error {

	if xdrMetrics == nil || isTestcaseMode() {
		// xdrMetrics = getFilteredMetrics(xdrRawMetrics, config.Aerospike.XdrMetricsAllowlist, config.Aerospike.XdrMetricsAllowlistEnabled, config.Aerospike.XdrMetricsBlocklist)
		xdrMetrics = make(map[string]AerospikeStat)

	}

	for _, dc := range infoKeys {
		dcName := strings.ReplaceAll(dc, "get-stats:context=xdr;dc=", "")
		log.Tracef("xdr-stats:%s:%s", dcName, rawMetrics[dc])

		// xdrObserver := make(MetricMap, len(xdrMetrics))
		// for m, t := range xdrMetrics {
		// 	xdrObserver[m] = makeMetric("aerospike_xdr", m, t, config.AeroProm.MetricLabels, "cluster_name", "service", "dc")
		// }

		stats := parseStats(rawMetrics[dc], ";")
		for stat, value := range stats {

			pv, err := tryConvert(value)
			if err != nil {
				continue
			}
			asMetric, exists := xdrMetrics[stat]
			if !exists {
				asMetric = newAerospikeStat(CTX_XDR, stat)
				xdrMetrics[stat] = asMetric
			}

			if asMetric.isAllowed {
				desc, valueType := asMetric.makePromeMetric(METRIC_LABEL_CLUSTER_NAME, METRIC_LABEL_SERVICE, METRIC_LABEL_DC_NAME)
				ch <- prometheus.MustNewConstMetric(desc, valueType, pv, rawMetrics[ikClusterName], rawMetrics[ikService], dcName)
			}
		}

		// for stat, pm := range xdrObserver {
		// 	v, exists := stats[stat]
		// 	if !exists {
		// 		// not found
		// 		continue
		// 	}

		// 	pv, err := tryConvert(v)
		// 	if err != nil {
		// 		continue
		// 	}

		// 	ch <- prometheus.MustNewConstMetric(pm.desc, pm.valueType, pv, rawMetrics[ikClusterName], rawMetrics[ikService], dcName)
		// }
	}

	return nil
}
