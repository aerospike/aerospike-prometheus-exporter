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

	for index, cpu := range stats.CPU {
		fmt.Println("parsing CPU stats ", index)
		arrSysInfoStats = append(arrSysInfoStats, constructCpuStats("guest_seconds_total", index, "user", cpu.Guest))
		arrSysInfoStats = append(arrSysInfoStats, constructCpuStats("guest_seconds_total", index, "nice", cpu.GuestNice))
		arrSysInfoStats = append(arrSysInfoStats, constructCpuStats("seconds_total", index, "idle", cpu.Idle))
		arrSysInfoStats = append(arrSysInfoStats, constructCpuStats("seconds_total", index, "irq", cpu.IRQ))
		arrSysInfoStats = append(arrSysInfoStats, constructCpuStats("seconds_total", index, "iowait", cpu.Iowait))
		arrSysInfoStats = append(arrSysInfoStats, constructCpuStats("seconds_total", index, "nice", cpu.Nice))
		arrSysInfoStats = append(arrSysInfoStats, constructCpuStats("seconds_total", index, "soft_irq", cpu.SoftIRQ))
		arrSysInfoStats = append(arrSysInfoStats, constructCpuStats("seconds_total", index, "steal", cpu.Steal))
		arrSysInfoStats = append(arrSysInfoStats, constructCpuStats("seconds_total", index, "system", cpu.System))
		arrSysInfoStats = append(arrSysInfoStats, constructCpuStats("seconds_total", index, "user", cpu.User))
	}

	return arrSysInfoStats
}

func constructCpuStats(cpuStatName string, cpuNo int64, cpuMode string, value float64) SystemInfoStat {
	clusterName := statprocessors.ClusterName
	service := statprocessors.Service

	labels := []string{}
	labels = append(labels, commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE)
	labels = append(labels, commons.METRIC_LABEL_CPU, commons.METRIC_LABEL_CPU_MODE)

	labelValues := []string{clusterName, service, fmt.Sprint(cpuNo), cpuMode}

	sysMetric := NewSystemInfoStat(commons.CTX_CPU_STATS, cpuStatName)
	sysMetric.Labels = labels
	sysMetric.LabelValues = labelValues
	sysMetric.Value = value

	return sysMetric
}
