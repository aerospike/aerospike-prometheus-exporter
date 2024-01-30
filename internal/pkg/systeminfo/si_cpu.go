package systeminfo

import (
	"fmt"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/dataprovider"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/statprocessors"
)

type CpuInfoProcessor struct {
}

func (cip CpuInfoProcessor) Refresh() ([]SystemInfoStat, error) {
	arrSysInfoStats := cip.parseCpuStats()
	return arrSysInfoStats, nil
}

func (cip CpuInfoProcessor) parseCpuStats() []SystemInfoStat {
	arrSysInfoStats := []SystemInfoStat{}

	guestCpuDetails, cpuDetails := dataprovider.GetSystemProvider().GetCPUDetails()

	for _, stat := range guestCpuDetails {
		arrSysInfoStats = append(arrSysInfoStats, cip.constructCpuStats("guest_seconds_total", fmt.Sprint(stat["index"]), "user", stat["user"]))
		arrSysInfoStats = append(arrSysInfoStats, cip.constructCpuStats("guest_seconds_total", fmt.Sprint(stat["index"]), "nice", stat["nice"]))
	}

	for _, stat := range cpuDetails {
		// fmt.Println("parsing CPU stats ", index)
		arrSysInfoStats = append(arrSysInfoStats, cip.constructCpuStats("seconds_total", fmt.Sprint(stat["index"]), "idle", stat["idle"]))
		arrSysInfoStats = append(arrSysInfoStats, cip.constructCpuStats("seconds_total", fmt.Sprint(stat["index"]), "irq", stat["irq"]))
		arrSysInfoStats = append(arrSysInfoStats, cip.constructCpuStats("seconds_total", fmt.Sprint(stat["index"]), "iowait", stat["iowait"]))
		arrSysInfoStats = append(arrSysInfoStats, cip.constructCpuStats("seconds_total", fmt.Sprint(stat["index"]), "nice", stat["nice"]))
		arrSysInfoStats = append(arrSysInfoStats, cip.constructCpuStats("seconds_total", fmt.Sprint(stat["index"]), "soft_irq", stat["soft_irq"]))
		arrSysInfoStats = append(arrSysInfoStats, cip.constructCpuStats("seconds_total", fmt.Sprint(stat["index"]), "steal", stat["steal"]))
		arrSysInfoStats = append(arrSysInfoStats, cip.constructCpuStats("seconds_total", fmt.Sprint(stat["index"]), "system", stat["system"]))
		arrSysInfoStats = append(arrSysInfoStats, cip.constructCpuStats("seconds_total", fmt.Sprint(stat["index"]), "user", stat["user"]))
	}

	return arrSysInfoStats
}

func (cip CpuInfoProcessor) constructCpuStats(cpuStatName string, cpuNo string, cpuMode string, value float64) SystemInfoStat {
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
