package systeminfo

import (
	"fmt"
	"strings"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/statprocessors"
)

const (
	METRIC_LABEL_MEM = "memory"
)

func Refresh() []SystemInfoStat {
	var stats = []SystemInfoStat{}

	// Get Memory Stats
	memStats := createMemInfoStats()
	stats = append(stats, memStats...)

	diskStats := createDiskStats()
	stats = append(stats, diskStats...)

	return stats
}

func createMemInfoStats() []SystemInfoStat {
	arrSysInfoStats := []SystemInfoStat{}

	clusterName := statprocessors.ClusterName
	service := statprocessors.Service

	labels := []string{commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE}
	labelValues := []string{clusterName, service}

	memStats := GetMemInfo()
	for k, v := range memStats.mem_stats {
		l_metricName := strings.ToLower(k) + "_bytes"
		sysMetric := NewSystemInfoStat(commons.CTX_MEMORY_STATS, l_metricName)
		sysMetric.Labels = labels
		sysMetric.LabelValues = labelValues
		sysMetric.Value = v

		arrSysInfoStats = append(arrSysInfoStats, sysMetric)
	}

	return arrSysInfoStats
}

func createDiskStats() []SystemInfoStat {

	arrSysInfoStats := []SystemInfoStat{}

	clusterName := statprocessors.ClusterName
	service := statprocessors.Service

	diskStats := GetDiskStats()
	for k, v := range diskStats {
		fmt.Println(" processing disk-device stat k: ", k)

		for sk, sv := range v.stats_info {
			labels := []string{commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE, commons.METRIC_LABEL_DEVICE}
			labelValues := []string{clusterName, service, k}

			l_metricName := strings.ToLower(sk)
			sysMetric := NewSystemInfoStat(commons.CTX_DISK_STATS, l_metricName)

			sysMetric.Labels = labels
			sysMetric.LabelValues = labelValues
			sysMetric.Value = sv

			arrSysInfoStats = append(arrSysInfoStats, sysMetric)
		}
	}

	fmt.Println("createDiskStats - diskStats: ", len(diskStats), " arrSysInfoStats: ", len(arrSysInfoStats))
	return arrSysInfoStats
}

func createFileSystemStats() []SystemInfoStat {
	fsStats := GetFileSystemInfo()
	fmt.Println("createFileSystemStats - fsStats: ", len(fsStats))
	return nil
}
