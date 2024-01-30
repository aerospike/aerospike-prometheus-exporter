package systeminfo

import (
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/dataprovider"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/statprocessors"
)

type NetworkInfoProcessor struct {
}

func (nip NetworkInfoProcessor) Refresh() ([]SystemInfoStat, error) {

	arrSysInfoStats := nip.parseNetworkStats()
	return arrSysInfoStats, nil
}

func (nip NetworkInfoProcessor) parseNetworkStats() []SystemInfoStat {
	arrSysInfoStats := []SystemInfoStat{}

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

	return arrSysInfoStats
}

func (nip NetworkInfoProcessor) constructNetworkDevStat(netStatKey string, deviceName string, stats map[string]string) SystemInfoStat {
	clusterName := statprocessors.ClusterName
	service := statprocessors.Service

	labels := []string{}
	labels = append(labels, commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE, commons.METRIC_LABEL_DEVICE)

	labelValues := []string{clusterName, service, deviceName}

	sysMetric := NewSystemInfoStat(commons.CTX_NET_DEV_STATS, netStatKey)
	sysMetric.Labels = labels
	sysMetric.LabelValues = labelValues
	sysMetric.Value, _ = commons.TryConvert(stats[deviceName])

	return sysMetric
}

func (nip NetworkInfoProcessor) constructNetworkStat(netStatKey string, deviceName string, stats map[string]string) SystemInfoStat {
	clusterName := statprocessors.ClusterName
	service := statprocessors.Service

	labels := []string{}
	labels = append(labels, commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE, commons.METRIC_LABEL_DEVICE)

	labelValues := []string{clusterName, service, deviceName}

	sysMetric := NewSystemInfoStat(commons.CTX_NETWORK_STATS, netStatKey)
	sysMetric.Labels = labels
	sysMetric.LabelValues = labelValues
	sysMetric.Value, _ = commons.TryConvert(stats[netStatKey])

	return sysMetric
}
