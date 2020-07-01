package main

import (
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

// XDR raw metrics
var xdrRawMetrics = map[string]metricType{
	"lag":                mtGauge,
	"in_queue":           mtGauge,
	"in_progress":        mtGauge,
	"recoveries_pending": mtGauge,
	"uncompressed_pct":   mtGauge,
	"compression_ratio":  mtGauge,
	"throughput":         mtGauge,
	"latency_ms":         mtGauge,
	"lap_us":             mtGauge,
	"success":            mtCounter,
	"abandoned":          mtCounter,
	"not_found":          mtCounter,
	"filtered_out":       mtCounter,
	"retry_conn_reset":   mtCounter,
	"retry_dest":         mtCounter,
	"recoveries":         mtCounter,
	"hot_keys":           mtCounter,
}

type XdrWatcher struct{}

func (xw *XdrWatcher) describe(ch chan<- *prometheus.Desc) {
	return
}

func (xw *XdrWatcher) infoKeys() []string {
	return []string{"get-config:context=xdr"}
}

func (xw *XdrWatcher) detailKeys(rawMetrics map[string]string) []string {
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
var xdrMetrics map[string]metricType

func (xw *XdrWatcher) refresh(infoKeys []string, rawMetrics map[string]string, accu map[string]interface{}, ch chan<- prometheus.Metric) error {

	if xdrMetrics == nil {
		xdrMetrics = getFilteredMetrics(xdrRawMetrics, config.Aerospike.XdrMetricsAllowlist, config.Aerospike.XdrMetricsAllowlistEnabled, config.Aerospike.XdrMetricsBlocklist, config.Aerospike.XdrMetricsBlocklistEnabled)
	}

	for _, dc := range infoKeys {
		dcName := strings.ReplaceAll(dc, "get-stats:context=xdr;dc=", "")
		xdrObserver := make(MetricMap, len(xdrMetrics))

		for m, t := range xdrMetrics {
			xdrObserver[m] = makeMetric("aerospike_xdr", m, t, config.AeroProm.MetricLabels, "cluster_name", "service", "dc")
		}

		stats := parseStats(rawMetrics[dc], ";")
		for stat, pm := range xdrObserver {
			v, exists := stats[stat]
			if !exists {
				// not found
				continue
			}

			pv, err := tryConvert(v)
			if err != nil {
				continue
			}

			ch <- prometheus.MustNewConstMetric(pm.desc, pm.valueType, pv, rawMetrics["cluster-name"], rawMetrics["service"], dcName)
		}
	}

	return nil
}
