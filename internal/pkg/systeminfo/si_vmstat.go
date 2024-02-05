package systeminfo

import (
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/dataprovider"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/statprocessors"
)

type VmstatInfoProcessor struct {
}

func (vip VmstatInfoProcessor) Refresh() ([]statprocessors.AerospikeStat, error) {

	arrSysInfoStats := vip.parseVmStats()
	return arrSysInfoStats, nil
}

func (vip VmstatInfoProcessor) parseVmStats() []statprocessors.AerospikeStat {
	arrSysInfoStats := []statprocessors.AerospikeStat{}

	arrVmStats := dataprovider.GetSystemProvider().GetVmStats()

	for _, vmStats := range arrVmStats {
		for key, _ := range vmStats {
			arrSysInfoStats = append(arrSysInfoStats, vip.constructVmstat(key, vmStats))
		}
	}

	return arrSysInfoStats
}

func (vip VmstatInfoProcessor) constructVmstat(key string, stats map[string]string) statprocessors.AerospikeStat {
	clusterName := statprocessors.ClusterName
	service := statprocessors.Service

	labels := []string{}
	labels = append(labels, commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE)

	labelValues := []string{clusterName, service}

	sysMetric := statprocessors.NewAerospikeStat(commons.CTX_VM_STATS, key)
	sysMetric.Labels = labels
	sysMetric.LabelValues = labelValues
	sysMetric.Value, _ = commons.TryConvert(stats[key])

	return sysMetric
}

// func (vip VmstatInfoProcessor) constructVmstat(key string, stats map[string]string) SystemInfoStat {
// 	clusterName := statprocessors.ClusterName
// 	service := statprocessors.Service

// 	labels := []string{}
// 	labels = append(labels, commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE)

// 	labelValues := []string{clusterName, service}

// 	sysMetric := NewSystemInfoStat(commons.CTX_VM_STATS, key)
// 	sysMetric.Labels = labels
// 	sysMetric.LabelValues = labelValues
// 	sysMetric.Value, _ = commons.TryConvert(stats[key])

// 	return sysMetric
// }
