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
	memstats := createMemInfoStats()

	stats = append(stats, memstats...)

	return stats
}

func createMemInfoStats() []SystemInfoStat {
	memStats := GetMemInfo()

	arrSysInfoStats := []SystemInfoStat{}

	clusterName := statprocessors.ClusterName
	service := statprocessors.Service

	labels := []string{commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE}
	labelValues := []string{clusterName, service}

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
	diskStats := GetDiskStats()
	fmt.Println("createDiskStats - diskStats: ", len(diskStats))
	return nil
}

func createFileSystemStats() []SystemInfoStat {
	fsStats := GetFileSystemInfo()
	fmt.Println("createFileSystemStats - fsStats: ", len(fsStats))
	return nil
}
