package systeminfo

import (
	log "github.com/sirupsen/logrus"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/dataprovider"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/statprocessors"
)

type CpuInfoProcessor struct {
}

func (cip CpuInfoProcessor) Refresh() ([]statprocessors.AerospikeStat, error) {
	arrSysInfoStats := []statprocessors.AerospikeStat{}
	cpuDetails := dataprovider.GetSystemProvider().GetCPUDetails()

	cpuStatLabels := []string{commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE}
	cpuStatLabels = append(cpuStatLabels, commons.METRIC_LABEL_CPU_MODE)

	clusterName := statprocessors.ClusterName
	service := statprocessors.Service

	for statName, statValue := range cpuDetails {
		labelValues := []string{clusterName, service, statName}
		value, err := commons.TryConvert(statValue)
		if err != nil {
			log.Error("Error while converting value of stat: ", statName, " and converted value is ", statValue)
			continue
		}

		sysMetric := statprocessors.NewAerospikeStat(commons.CTX_CPU_STATS, "cpu_seconds_total", statName)
		sysMetric.Labels = cpuStatLabels
		sysMetric.LabelValues = labelValues
		sysMetric.Value = value

		arrSysInfoStats = append(arrSysInfoStats, sysMetric)

	}

	return arrSysInfoStats, nil
}
