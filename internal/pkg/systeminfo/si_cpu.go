package systeminfo

import (
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/dataprovider"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/statprocessors"
)

type CpuInfoProcessor struct {
}

var (
	cpuStatLabels []string
)

func (cip CpuInfoProcessor) Refresh() ([]statprocessors.AerospikeStat, error) {
	arrSysInfoStats := []statprocessors.AerospikeStat{}
	cpuDetails := dataprovider.GetSystemProvider().GetCPUDetails()

	cpuStatLabels = []string{commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE}
	cpuStatLabels = append(cpuStatLabels, commons.METRIC_LABEL_CPU_MODE)

	clusterName := statprocessors.ClusterName
	service := statprocessors.Service

	for statName, statValue := range cpuDetails {
		labelValues := []string{clusterName, service, statName}

		sysMetric := statprocessors.NewAerospikeStat(commons.CTX_CPU_STATS, "cpu_seconds_total", statName)
		sysMetric.Labels = cpuStatLabels
		sysMetric.LabelValues = labelValues
		sysMetric.Value, _ = commons.TryConvert(statValue)

		arrSysInfoStats = append(arrSysInfoStats, sysMetric)

	}

	return arrSysInfoStats, nil
}
