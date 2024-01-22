package executors

import (
	"strings"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/statprocessors"
)

var (
	descReplacerFunc = strings.NewReplacer("_", " ", "-", " ", ".", " ")
	// TODO: re-check why we need below replacer, is it because of the replace char sequences
	metricReplacerFunc = strings.NewReplacer(".", "_", "-", "_", " ", "_")
)

// Utility functions

func NormalizeDesc(s string) string {
	return descReplacerFunc.Replace(s)
}

func NormalizeMetric(s string) string {
	return metricReplacerFunc.Replace(s)
}

func getMetricMapKey(metricName string, stat statprocessors.AerospikeStat) string {
	var key = metricName

	for _, v := range stat.LabelValues {
		key = key + "_" + v
	}

	return key
}
