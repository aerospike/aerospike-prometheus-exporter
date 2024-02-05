package systeminfo

import (
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/dataprovider"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/statprocessors"
)

type VmstatInfoProcessor struct {
}

var (
	vmStatLabels []string
)

func (vip VmstatInfoProcessor) Refresh() ([]statprocessors.AerospikeStat, error) {

	arrSysInfoStats := []statprocessors.AerospikeStat{}
	vmStatLabels = []string{commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE}

	arrVmStats := dataprovider.GetSystemProvider().GetVmStats()

	for _, vmStats := range arrVmStats {
		for key, _ := range vmStats {
			arrSysInfoStats = append(arrSysInfoStats, vip.constructVmstat(key, vmStats))
		}
	}

	return arrSysInfoStats, nil
}

func (vip VmstatInfoProcessor) constructVmstat(statName string, stats map[string]string) statprocessors.AerospikeStat {
	clusterName := statprocessors.ClusterName
	service := statprocessors.Service

	labelValues := []string{clusterName, service}

	sysMetric := statprocessors.NewAerospikeStat(commons.CTX_VM_STATS, statName)
	sysMetric.Labels = vmStatLabels
	sysMetric.LabelValues = labelValues
	sysMetric.Value, _ = commons.TryConvert(stats[statName])

	return sysMetric
}
