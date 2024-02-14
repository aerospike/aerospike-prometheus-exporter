package systeminfo

import (
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/dataprovider"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/statprocessors"
)

type NetworkInfoProcessor struct {
}

var (
	netDevStatLabels []string
	networkLabels    []string
)

func (nip NetworkInfoProcessor) Refresh() ([]statprocessors.AerospikeStat, error) {

	arrSysInfoStats := []statprocessors.AerospikeStat{}

	netDevStatLabels = []string{commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE, commons.METRIC_LABEL_DEVICE}
	networkLabels = []string{commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE, commons.METRIC_LABEL_DEVICE}

	arrGroupStats, arrReceiveStats, arrTransferStats := dataprovider.GetSystemProvider().GetNetDevStats()

	// netdev group
	for _, stats := range arrGroupStats {
		arrSysInfoStats = append(arrSysInfoStats, nip.constructNetworkDevStat("group", stats["device_name"], stats))
	}

	// netdev receive
	for _, stats := range arrReceiveStats {
		deviceName := stats["device_name"]
		// fmt.Println("Netdev Receive device: ", deviceName)
		arrSysInfoStats = append(arrSysInfoStats, nip.constructNetworkStat("receive_bytes_total", deviceName, stats))
		arrSysInfoStats = append(arrSysInfoStats, nip.constructNetworkStat("receive_compressed_total", deviceName, stats))
		arrSysInfoStats = append(arrSysInfoStats, nip.constructNetworkStat("receive_dropped_total", deviceName, stats))
		arrSysInfoStats = append(arrSysInfoStats, nip.constructNetworkStat("receive_errors_total", deviceName, stats))
		arrSysInfoStats = append(arrSysInfoStats, nip.constructNetworkStat("receive_fifo_total", deviceName, stats))
		arrSysInfoStats = append(arrSysInfoStats, nip.constructNetworkStat("receive_frame_total", deviceName, stats))
		arrSysInfoStats = append(arrSysInfoStats, nip.constructNetworkStat("receive_multicast_total", deviceName, stats))
		arrSysInfoStats = append(arrSysInfoStats, nip.constructNetworkStat("receive_packets_total", deviceName, stats))

	}

	// netdev transfer
	for _, stats := range arrTransferStats {
		deviceName := stats["device_name"]
		// fmt.Println("Netdev Transfer device: ", deviceName)
		arrSysInfoStats = append(arrSysInfoStats, nip.constructNetworkStat("transfer_bytes_total", deviceName, stats))
		arrSysInfoStats = append(arrSysInfoStats, nip.constructNetworkStat("transfer_carrier_total", deviceName, stats))
		arrSysInfoStats = append(arrSysInfoStats, nip.constructNetworkStat("transfer_collisions_total", deviceName, stats))
		arrSysInfoStats = append(arrSysInfoStats, nip.constructNetworkStat("transfer_compressed_total", deviceName, stats))
		arrSysInfoStats = append(arrSysInfoStats, nip.constructNetworkStat("transfer_errors_total", deviceName, stats))
		arrSysInfoStats = append(arrSysInfoStats, nip.constructNetworkStat("transfer_fifo_total", deviceName, stats))
		arrSysInfoStats = append(arrSysInfoStats, nip.constructNetworkStat("transfer_packets_total", deviceName, stats))
	}

	return arrSysInfoStats, nil
}

func (nip NetworkInfoProcessor) constructNetworkDevStat(statName string, deviceName string, stats map[string]string) statprocessors.AerospikeStat {
	clusterName := statprocessors.ClusterName
	service := statprocessors.Service

	labelValues := []string{clusterName, service, deviceName}

	sysMetric := statprocessors.NewAerospikeStat(commons.CTX_NET_DEV_STATS, statName, statName)
	sysMetric.Labels = netDevStatLabels
	sysMetric.LabelValues = labelValues
	sysMetric.Value, _ = commons.TryConvert(stats[deviceName])

	return sysMetric
}

func (nip NetworkInfoProcessor) constructNetworkStat(statName string, deviceName string, stats map[string]string) statprocessors.AerospikeStat {
	clusterName := statprocessors.ClusterName
	service := statprocessors.Service

	labelValues := []string{clusterName, service, deviceName}

	sysMetric := statprocessors.NewAerospikeStat(commons.CTX_NETWORK_STATS, statName, statName)
	sysMetric.Labels = networkLabels
	sysMetric.LabelValues = labelValues
	sysMetric.Value, _ = commons.TryConvert(stats[statName])

	return sysMetric
}
