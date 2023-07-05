package main

import (
	"fmt"
	"strings"

	"github.com/prometheus/client_golang/prometheus"

	log "github.com/sirupsen/logrus"
)

type XdrWatcher struct {
	xdrMetrics map[string]AerospikeStat
}

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

// All (allowed/blocked) XDR stats. Based on the config.Aerospike.XdrMetricsAllowlist, config.Aerospike.XdrMetricsBlocklist.
// var xdrMetrics = make(map[string]AerospikeStat)

func (xw *XdrWatcher) refresh(o *Observer, infoKeys []string, rawMetrics map[string]string, ch chan<- prometheus.Metric) error {

	if xw.xdrMetrics == nil {
		fmt.Println("Reinitializing xdrStats(...) ")
		xw.xdrMetrics = make(map[string]AerospikeStat)
	}

	for _, dc := range infoKeys {
		dcName := strings.ReplaceAll(dc, "get-stats:context=xdr;dc=", "")
		log.Tracef("xdr-stats:%s:%s", dcName, rawMetrics[dc])

		stats := parseStats(rawMetrics[dc], ";")
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

			if asMetric.isAllowed {
				desc, valueType := asMetric.makePromMetric(METRIC_LABEL_CLUSTER_NAME, METRIC_LABEL_SERVICE, METRIC_LABEL_DC_NAME)
				ch <- prometheus.MustNewConstMetric(desc, valueType, pv, rawMetrics[ikClusterName], rawMetrics[ikService], dcName)
			}
		}

	}

	return nil
}
