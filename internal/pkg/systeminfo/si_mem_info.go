package systeminfo

import (
	"strings"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/dataprovider"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/statprocessors"
)

type MemInfoProcessor struct {
}

func (mip MemInfoProcessor) Refresh() ([]statprocessors.AerospikeStat, error) {
	arrSysInfoStats := []statprocessors.AerospikeStat{}

	memStats := dataprovider.GetSystemProvider().GetMemInfoStats()

	memInfoLabels := []string{commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE}
	for stat, value := range memStats {
		clusterName := statprocessors.ClusterName
		service := statprocessors.Service

		labelValues := []string{clusterName, service}

		metricName := strings.ToLower(stat) + "_bytes"
		sysMetric := statprocessors.NewAerospikeStat(commons.CTX_MEMORY_STATS, metricName, metricName)
		sysMetric.Labels = memInfoLabels
		sysMetric.LabelValues = labelValues
		sysMetric.Value, _ = commons.TryConvert(value)

		arrSysInfoStats = append(arrSysInfoStats, sysMetric)
	}

	return arrSysInfoStats, nil
}
