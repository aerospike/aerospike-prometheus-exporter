package systeminfo

import (
	"fmt"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/statprocessors"
	"github.com/prometheus/procfs"
	log "github.com/sirupsen/logrus"
)

func GetCpuInfo() []SystemInfoStat {
	arrSysInfoStats := parseCpuStats()
	return arrSysInfoStats
}

func parseCpuStats() []SystemInfoStat {
	arrSysInfoStats := []SystemInfoStat{}

	fs, err := procfs.NewFS(PROC_PATH)
	if err != nil {
		log.Debug("GetCpuStats Error while reading CPU Stats from ", PROC_PATH)
		return arrSysInfoStats
	}
	stats, err := fs.Stat()

	if err != nil {
		log.Debug("Eror while reading procfs.NewFS system,  error: ", err)
		return arrSysInfoStats
	}

	// fmt.Println("parsing CPU stats ", stats.CPU)
	for index, cpu := range stats.CPU {
		fmt.Println("parsing CPU stats ", index)
		arrSysInfoStats = append(arrSysInfoStats, constructCpuStats("node_cpu_guest_seconds_total", index, "user", cpu.Guest))
		arrSysInfoStats = append(arrSysInfoStats, constructCpuStats("node_cpu_guest_seconds_total", index, "nice", cpu.GuestNice))
	}

	fmt.Println(" si-cpu.go arrSysInfoStats... ", len(arrSysInfoStats))

	return arrSysInfoStats
}

func constructCpuStats(cpuStatName string, cpuNo int64, cpuMode string, value float64) SystemInfoStat {
	clusterName := statprocessors.ClusterName
	service := statprocessors.Service

	// add disk_info
	labels := []string{}
	labels = append(labels, commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE)
	labels = append(labels, commons.METRIC_LABEL_CPU, commons.METRIC_LABEL_CPU_MODE)

	labelValues := []string{clusterName, service, string(cpuNo), cpuMode}

	sysMetric := NewSystemInfoStat(commons.CTX_DISK_STATS, cpuStatName)
	sysMetric.Labels = labels
	sysMetric.LabelValues = labelValues
	sysMetric.Value = value

	return sysMetric
}

// func constructCpuStats(deviceName string, v_stats_info map[string]float64) []SystemInfoStat {
// 	arrSysInfoStats := []SystemInfoStat{}

// 	clusterName := statprocessors.ClusterName
// 	service := statprocessors.Service

// 	for sk, sv := range v_stats_info {
// 		labels := []string{commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE, commons.METRIC_LABEL_DEVICE}
// 		labelValues := []string{clusterName, service, deviceName}

// 		l_metricName := strings.ToLower(sk)
// 		sysMetric := NewSystemInfoStat(commons.CTX_CPU_STATS, l_metricName)

// 		sysMetric.Labels = labels
// 		sysMetric.LabelValues = labelValues
// 		sysMetric.Value = sv

// 		arrSysInfoStats = append(arrSysInfoStats, sysMetric)
// 	}

// 	return arrSysInfoStats
// }
