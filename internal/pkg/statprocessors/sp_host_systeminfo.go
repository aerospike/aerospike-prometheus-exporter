package statprocessors

import (
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/dataprovider"
)

type SysInfoProcessor struct {
}

func RefreshSystemInfo() ([]AerospikeStat, error) {
	arrSysInfoStats := []AerospikeStat{}

	arrSysInfoStats = append(arrSysInfoStats, getCpuInfo()...)
	arrSysInfoStats = append(arrSysInfoStats, getFDInfo()...)
	arrSysInfoStats = append(arrSysInfoStats, getMemInfo()...)
	arrSysInfoStats = append(arrSysInfoStats, getNetStatInfo()...)
	arrSysInfoStats = append(arrSysInfoStats, getNetworkInfo()...)

	return arrSysInfoStats, nil
}

func getCpuInfo() []AerospikeStat {
	arrSysInfoStats := []AerospikeStat{}
	cpuDetails := dataprovider.GetSystemProvider().GetCPUDetails()

	cpuStatLabels := []string{commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE}
	cpuStatLabels = append(cpuStatLabels, commons.METRIC_LABEL_CPU_MODE)

	clusterName := ClusterName
	service := Service

	for statName, statValue := range cpuDetails {
		labelValues := []string{clusterName, service, statName}
		value, err := commons.TryConvert(statValue)
		if err != nil {
			log.Error("Error while converting value of stat: ", statName, " and converted value is ", statValue)
			continue
		}

		sysMetric := NewAerospikeStat(commons.CTX_CPU_STATS, "cpu_seconds_total", statName)
		sysMetric.Labels = cpuStatLabels
		sysMetric.LabelValues = labelValues
		sysMetric.Value = value

		arrSysInfoStats = append(arrSysInfoStats, sysMetric)

	}

	return arrSysInfoStats
}

func getFDInfo() []AerospikeStat {
	arrSysInfoStats := []AerospikeStat{}
	clusterName := ClusterName
	service := Service

	labelValues := []string{clusterName, service}
	fileStatLabels := []string{commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE}

	fileFDStats := dataprovider.GetSystemProvider().GetFileFD()

	statName := "allocated"
	statValue := fileFDStats[statName]
	value, err := commons.TryConvert(statValue)
	if err != nil {
		log.Error("Error while converting value of stat: ", statName, " and converted value is ", statValue)
		return arrSysInfoStats
	}

	sysMetric := NewAerospikeStat(commons.CTX_FILEFD_STATS, statName, statName)
	sysMetric.Labels = fileStatLabels
	sysMetric.LabelValues = labelValues
	sysMetric.Value = value

	arrSysInfoStats = append(arrSysInfoStats, sysMetric)

	return arrSysInfoStats
}

func getMemInfo() []AerospikeStat {
	arrSysInfoStats := []AerospikeStat{}

	memStats := dataprovider.GetSystemProvider().GetMemInfoStats()

	memInfoLabels := []string{commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE}
	for statName, statValue := range memStats {
		clusterName := ClusterName
		service := Service
		value, err := commons.TryConvert(statValue)
		if err != nil {
			log.Error("Error while converting value of stat: ", statName, " and converted value is ", statValue)
			continue
		}

		labelValues := []string{clusterName, service}

		metricName := strings.ToLower(statName) + "_bytes"
		sysMetric := NewAerospikeStat(commons.CTX_MEMORY_STATS, metricName, metricName)
		sysMetric.Labels = memInfoLabels
		sysMetric.LabelValues = labelValues
		sysMetric.Value = value

		arrSysInfoStats = append(arrSysInfoStats, sysMetric)
	}

	return arrSysInfoStats
}

func getNetStatInfo() []AerospikeStat {
	arrSysInfoStats := []AerospikeStat{}

	netStatInfoLabels := []string{}
	netStatInfoLabels = append(netStatInfoLabels, commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE)

	snmpStats := dataprovider.GetSystemProvider().GetNetStatInfo()

	//Net SNMP - includes TCP metrics like active_conn, established, retransmit etc.,
	clusterName := ClusterName
	service := Service
	labelValues := []string{clusterName, service}

	for statName, statValue := range snmpStats {

		value, err := commons.TryConvert(statValue)
		if err != nil {
			log.Error("Error while converting value of stat: ", statName, " and converted value is ", statValue)
			continue
		}

		sysMetric := NewAerospikeStat(commons.CTX_NET_STATS, statName, statName)
		sysMetric.Labels = netStatInfoLabels
		sysMetric.LabelValues = labelValues
		sysMetric.Value = value

		arrSysInfoStats = append(arrSysInfoStats, sysMetric)
	}

	return arrSysInfoStats
}

func getNetworkInfo() []AerospikeStat {

	arrSysInfoStats := []AerospikeStat{}

	networkLabels := []string{commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE, commons.METRIC_LABEL_DEVICE}

	arrReceiveStats, arrTransferStats := dataprovider.GetSystemProvider().GetNetDevStats()

	// netdev receive
	clusterName := ClusterName
	service := Service
	for _, stats := range arrReceiveStats {
		deviceName := stats["device_name"]
		statName := "receive_bytes_total"
		value, err := commons.TryConvert(stats[statName])
		if err != nil {
			log.Debug("Error while converting value of stat: ", statName, " and converted value is ", stats[statName])
			continue
		}

		labelValues := []string{clusterName, service, deviceName}

		sysMetric := NewAerospikeStat(commons.CTX_NETWORK_STATS, statName, statName)
		sysMetric.Labels = networkLabels
		sysMetric.LabelValues = labelValues
		sysMetric.Value = value

		arrSysInfoStats = append(arrSysInfoStats, sysMetric)

	}

	// netdev transfer
	for _, stats := range arrTransferStats {
		deviceName := stats["device_name"]
		statName := "transfer_bytes_total"
		value, err := commons.TryConvert(stats[statName])
		if err != nil {
			log.Debug("Error while converting value of stat: ", statName, " and converted value is ", stats[statName])
			continue
		}

		labelValues := []string{clusterName, service, deviceName}

		sysMetric := NewAerospikeStat(commons.CTX_NETWORK_STATS, statName, statName)
		sysMetric.Labels = networkLabels
		sysMetric.LabelValues = labelValues
		sysMetric.Value = value

		arrSysInfoStats = append(arrSysInfoStats, sysMetric)
	}

	return arrSysInfoStats
}
