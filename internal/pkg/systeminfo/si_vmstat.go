package systeminfo

import (
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/dataprovider"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/statprocessors"
)

func GetVmStatInfo() []SystemInfoStat {

	arrSysInfoStats := parseVmStats()
	return arrSysInfoStats
}

func parseVmStats() []SystemInfoStat {
	arrSysInfoStats := []SystemInfoStat{}

	arrVmStats := dataprovider.GetSystemProvider().GetVmStats()

	for _, vmStats := range arrVmStats {
		for key, _ := range vmStats {
			arrSysInfoStats = append(arrSysInfoStats, constructVmstat(key, vmStats))
		}
	}

	return arrSysInfoStats
}

func constructVmstat(key string, stats map[string]string) SystemInfoStat {
	clusterName := statprocessors.ClusterName
	service := statprocessors.Service

	labels := []string{}
	labels = append(labels, commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE)

	labelValues := []string{clusterName, service}

	sysMetric := NewSystemInfoStat(commons.CTX_VM_STATS, key)
	sysMetric.Labels = labels
	sysMetric.LabelValues = labelValues
	sysMetric.Value, _ = commons.TryConvert(stats[key])

	return sysMetric
}
