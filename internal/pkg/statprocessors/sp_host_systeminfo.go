package statprocessors

import (
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/dataprovider"
)

type HostSystemInfoProcessor struct {
	sharedState    *StatProcessorSharedState
	systemProvider *dataprovider.SystemInfoProvider
}

func NewHostSystemInfoProcessor(
	sharedState *StatProcessorSharedState) *HostSystemInfoProcessor {

	hostSystemInfoProcessor := &HostSystemInfoProcessor{sharedState: sharedState}
	hostSystemInfoProcessor.systemProvider = dataprovider.GetSystemProvider()

	return hostSystemInfoProcessor
}

func (hsi *HostSystemInfoProcessor) RefreshSystemInfo() ([]AerospikeStat, error) {
	arrSysInfoStats := []AerospikeStat{}

	if !config.Cfg.Agent.RefreshSystemStats {
		return arrSysInfoStats, nil
	}

	arrSysInfoStats = append(arrSysInfoStats, hsi.getFDInfo()...)
	arrSysInfoStats = append(arrSysInfoStats, hsi.getMemInfo()...)
	arrSysInfoStats = append(arrSysInfoStats, hsi.getNetStatInfo()...)
	arrSysInfoStats = append(arrSysInfoStats, hsi.getNetworkInfo()...)

	return arrSysInfoStats, nil
}

func (hsi *HostSystemInfoProcessor) getFDInfo() []AerospikeStat {
	arrSysInfoStats := []AerospikeStat{}

	labelValues := []string{hsi.sharedState.ClusterName, hsi.sharedState.Service}
	fileStatLabels := []string{commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE}

	fileFDStats := hsi.systemProvider.GetFileFD()

	statName := "allocated"
	statValue := fileFDStats[statName]
	value, err := commons.TryConvert(statValue)

	if err != nil {
		log.Errorf("Error while converting value of stat: %s and converted value is %s", statName, statValue)
		return arrSysInfoStats
	}

	allowed := isMetricAllowed(commons.CTX_SYSINFO_FILEFD_STATS, statName)
	sysMetric := NewAerospikeStat(commons.CTX_SYSINFO_FILEFD_STATS, statName, allowed)
	sysMetric.Labels = fileStatLabels
	sysMetric.LabelValues = labelValues
	sysMetric.Value = value

	arrSysInfoStats = append(arrSysInfoStats, sysMetric)

	return arrSysInfoStats
}

func (hsi *HostSystemInfoProcessor) getMemInfo() []AerospikeStat {
	arrSysInfoStats := []AerospikeStat{}

	memStats := hsi.systemProvider.GetMemInfoStats()
	memInfoLabels := []string{commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE}

	for statName, statValue := range memStats {
		value, err := commons.TryConvert(statValue)
		if err != nil {
			log.Error("Error while converting value of stat: ", statName, " and converted value is ", statValue)
			continue
		}

		labelValues := []string{hsi.sharedState.ClusterName, hsi.sharedState.Service}

		metricName := strings.ToLower(statName) + "_bytes"
		allowed := isMetricAllowed(commons.CTX_SYSINFO_MEMORY_STATS, statName)
		sysMetric := NewAerospikeStat(commons.CTX_SYSINFO_MEMORY_STATS, metricName, allowed)
		sysMetric.Labels = memInfoLabels
		sysMetric.LabelValues = labelValues
		sysMetric.Value = value

		arrSysInfoStats = append(arrSysInfoStats, sysMetric)
	}

	return arrSysInfoStats
}

func (hsi *HostSystemInfoProcessor) getNetStatInfo() []AerospikeStat {
	arrSysInfoStats := []AerospikeStat{}

	netStatInfoLabels := []string{}
	netStatInfoLabels = append(netStatInfoLabels, commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE)

	snmpStats := hsi.systemProvider.GetNetStatInfo()

	//Net SNMP - includes TCP metrics like active_conn, established, retransmit etc.,
	labelValues := []string{hsi.sharedState.ClusterName, hsi.sharedState.Service}

	for statName, statValue := range snmpStats {

		value, err := commons.TryConvert(statValue)

		if err != nil {
			log.Error("Error while converting value of stat: ", statName, " and converted value is ", statValue)
			continue
		}

		allowed := isMetricAllowed(commons.CTX_SYSINFO_NET_STATS, statName)
		sysMetric := NewAerospikeStat(commons.CTX_SYSINFO_NET_STATS, statName, allowed)
		sysMetric.Labels = netStatInfoLabels
		sysMetric.LabelValues = labelValues
		sysMetric.Value = value

		arrSysInfoStats = append(arrSysInfoStats, sysMetric)
	}

	return arrSysInfoStats
}

func (hsi *HostSystemInfoProcessor) getNetworkInfo() []AerospikeStat {

	arrSysInfoStats := []AerospikeStat{}

	networkLabels := []string{commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE, commons.METRIC_LABEL_DEVICE}

	arrReceiveStats, arrTransferStats := hsi.systemProvider.GetNetDevStats()

	// netdev receive
	for _, stats := range arrReceiveStats {
		deviceName := stats["device_name"]
		statName := "receive_bytes_total"
		value, err := commons.TryConvert(stats[statName])

		if err != nil {
			log.Debugf("Error while converting value of stat: %s and converted value is %s", statName, stats[statName])
			continue
		}

		labelValues := []string{hsi.sharedState.ClusterName, hsi.sharedState.Service, deviceName}

		allowed := isMetricAllowed(commons.CTX_SYSINFO_NETWORK_STATS, statName)
		sysMetric := NewAerospikeStat(commons.CTX_SYSINFO_NETWORK_STATS, statName, allowed)
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
			log.Debugf("Error while converting value of stat: %s and converted value is %s", statName, stats[statName])
			continue
		}

		labelValues := []string{hsi.sharedState.ClusterName, hsi.sharedState.Service, deviceName}
		allowed := isMetricAllowed(commons.CTX_SYSINFO_NETWORK_STATS, statName)
		sysMetric := NewAerospikeStat(commons.CTX_SYSINFO_NETWORK_STATS, statName, allowed)
		sysMetric.Labels = networkLabels
		sysMetric.LabelValues = labelValues
		sysMetric.Value = value

		arrSysInfoStats = append(arrSysInfoStats, sysMetric)
	}

	return arrSysInfoStats
}
