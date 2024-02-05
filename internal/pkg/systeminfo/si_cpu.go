package systeminfo

import (
	"fmt"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/dataprovider"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/statprocessors"
)

type CpuInfoProcessor struct {
}

func (cip CpuInfoProcessor) Refresh() ([]statprocessors.AerospikeStat, error) {
	arrSysInfoStats := cip.parseCpuStats()
	return arrSysInfoStats, nil
}

func (cip CpuInfoProcessor) parseCpuStats() []statprocessors.AerospikeStat {
	arrSysInfoStats := []statprocessors.AerospikeStat{}

	guestCpuDetails, cpuDetails := dataprovider.GetSystemProvider().GetCPUDetails()

	for _, stats := range guestCpuDetails {
		arrSysInfoStats = append(arrSysInfoStats, cip.constructCpuStats("guest_seconds_total", fmt.Sprint(stats["index"]), "user", stats))
		arrSysInfoStats = append(arrSysInfoStats, cip.constructCpuStats("guest_seconds_total", fmt.Sprint(stats["index"]), "nice", stats))
	}

	for _, stats := range cpuDetails {
		// fmt.Println("parsing CPU stats ", index)
		arrSysInfoStats = append(arrSysInfoStats, cip.constructCpuStats("seconds_total", fmt.Sprint(stats["index"]), "idle", stats))
		arrSysInfoStats = append(arrSysInfoStats, cip.constructCpuStats("seconds_total", fmt.Sprint(stats["index"]), "irq", stats))
		arrSysInfoStats = append(arrSysInfoStats, cip.constructCpuStats("seconds_total", fmt.Sprint(stats["index"]), "iowait", stats))
		arrSysInfoStats = append(arrSysInfoStats, cip.constructCpuStats("seconds_total", fmt.Sprint(stats["index"]), "nice", stats))
		arrSysInfoStats = append(arrSysInfoStats, cip.constructCpuStats("seconds_total", fmt.Sprint(stats["index"]), "soft_irq", stats))
		arrSysInfoStats = append(arrSysInfoStats, cip.constructCpuStats("seconds_total", fmt.Sprint(stats["index"]), "steal", stats))
		arrSysInfoStats = append(arrSysInfoStats, cip.constructCpuStats("seconds_total", fmt.Sprint(stats["index"]), "system", stats))
		arrSysInfoStats = append(arrSysInfoStats, cip.constructCpuStats("seconds_total", fmt.Sprint(stats["index"]), "user", stats))
	}

	return arrSysInfoStats
}

func (cip CpuInfoProcessor) constructCpuStats(cpuStatName string, cpuNo string, cpuMode string, stats map[string]string) statprocessors.AerospikeStat {
	clusterName := statprocessors.ClusterName
	service := statprocessors.Service

	labels := []string{}
	labels = append(labels, commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE)
	labels = append(labels, commons.METRIC_LABEL_CPU, commons.METRIC_LABEL_CPU_MODE)

	labelValues := []string{clusterName, service, fmt.Sprint(cpuNo), cpuMode}

	sysMetric := statprocessors.NewAerospikeStat(commons.CTX_CPU_STATS, cpuStatName)
	sysMetric.Labels = labels
	sysMetric.LabelValues = labelValues
	sysMetric.Value, _ = commons.TryConvert(stats[cpuMode])

	return sysMetric
}
