package systeminfo

import (
	"fmt"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/statprocessors"
	"github.com/prometheus/procfs"
	log "github.com/sirupsen/logrus"
)

func GetNetworkStatsInfo() []SystemInfoStat {
	arrSysInfoStats := []SystemInfoStat{}

	arrSysInfoStats = append(arrSysInfoStats, parseNetworkStats()...)

	fmt.Println("\t GetNetworkStatsInfo **** arrSysInfoStats ", len(arrSysInfoStats))

	return arrSysInfoStats
}

func parseNetworkStats() []SystemInfoStat {
	arrSysInfoStats := []SystemInfoStat{}

	fs, err := procfs.NewFS(PROC_PATH)
	if err != nil {
		log.Debug("parseNetworkStats Error while reading Net_Dev Stats from ", PROC_PATH, " Error ", err)
		return arrSysInfoStats
	}

	stats, err := fs.NetDev()
	if err != nil {
		log.Debug("Eror while reading procfs.NewFS system, error: ", err)
		return arrSysInfoStats
	}

	for k, v := range stats {
		arrSysInfoStats = append(arrSysInfoStats, constructNetworkDevStat("group", k, 0))

		// network receive
		arrSysInfoStats = append(arrSysInfoStats, constructNetworkStat("receive_bytes_total", k, float64(v.RxBytes)))
		arrSysInfoStats = append(arrSysInfoStats, constructNetworkStat("receive_compressed_total", k, float64(v.RxCompressed)))
		arrSysInfoStats = append(arrSysInfoStats, constructNetworkStat("receive_dropped_total", k, float64(v.RxDropped)))
		arrSysInfoStats = append(arrSysInfoStats, constructNetworkStat("receive_errors_total", k, float64(v.RxErrors)))
		arrSysInfoStats = append(arrSysInfoStats, constructNetworkStat("receive_fifo_total", k, float64(v.RxFIFO)))
		arrSysInfoStats = append(arrSysInfoStats, constructNetworkStat("receive_frame_total", k, float64(v.RxFrame)))
		arrSysInfoStats = append(arrSysInfoStats, constructNetworkStat("receive_multicast_total", k, float64(v.RxMulticast)))
		arrSysInfoStats = append(arrSysInfoStats, constructNetworkStat("receive_packets_total", k, float64(v.RxPackets)))

		// network transfer
		arrSysInfoStats = append(arrSysInfoStats, constructNetworkStat("transfer_bytes_total", k, float64(v.TxBytes)))
		arrSysInfoStats = append(arrSysInfoStats, constructNetworkStat("transfer_carrier_total", k, float64(v.TxCarrier)))
		arrSysInfoStats = append(arrSysInfoStats, constructNetworkStat("transfer_collisions_total", k, float64(v.TxCollisions)))
		arrSysInfoStats = append(arrSysInfoStats, constructNetworkStat("transfer_compressed_total", k, float64(v.TxCompressed)))
		arrSysInfoStats = append(arrSysInfoStats, constructNetworkStat("transfer_errors_total", k, float64(v.TxErrors)))
		arrSysInfoStats = append(arrSysInfoStats, constructNetworkStat("transfer_fifo_total", k, float64(v.TxFIFO)))
		arrSysInfoStats = append(arrSysInfoStats, constructNetworkStat("transfer_packets_total", k, float64(v.TxPackets)))
	}

	return arrSysInfoStats
}

func constructNetworkDevStat(netStatKey string, deviceName string, value float64) SystemInfoStat {
	clusterName := statprocessors.ClusterName
	service := statprocessors.Service

	labels := []string{}
	labels = append(labels, commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE, commons.METRIC_LABEL_DEVICE)

	labelValues := []string{clusterName, service, deviceName}

	sysMetric := NewSystemInfoStat(commons.CTX_NET_DEV_STATS, netStatKey)
	sysMetric.Labels = labels
	sysMetric.LabelValues = labelValues
	sysMetric.Value = value

	return sysMetric
}

func constructNetworkStat(netStatKey string, deviceName string, value float64) SystemInfoStat {
	clusterName := statprocessors.ClusterName
	service := statprocessors.Service

	labels := []string{}
	labels = append(labels, commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE, commons.METRIC_LABEL_DEVICE)

	labelValues := []string{clusterName, service, deviceName}

	sysMetric := NewSystemInfoStat(commons.CTX_NETWORK_STATS, netStatKey)
	sysMetric.Labels = labels
	sysMetric.LabelValues = labelValues
	sysMetric.Value = value

	return sysMetric
}
