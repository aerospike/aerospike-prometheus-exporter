package systeminfo

import (
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/dataprovider"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/statprocessors"
	log "github.com/sirupsen/logrus"
)

type NetworkInfoProcessor struct {
}

func (nip NetworkInfoProcessor) Refresh() ([]statprocessors.AerospikeStat, error) {

	arrSysInfoStats := []statprocessors.AerospikeStat{}

	networkLabels := []string{commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE, commons.METRIC_LABEL_DEVICE}

	arrReceiveStats, arrTransferStats := dataprovider.GetSystemProvider().GetNetDevStats()

	// netdev receive
	clusterName := statprocessors.ClusterName
	service := statprocessors.Service
	for _, stats := range arrReceiveStats {
		deviceName := stats["device_name"]
		statName := "receive_bytes_total"
		value, err := commons.TryConvert(stats[statName])
		if err != nil {
			log.Debug("Error while converting value of stat: ", statName, " and converted value is ", stats[statName])
			continue
		}

		labelValues := []string{clusterName, service, deviceName}

		sysMetric := statprocessors.NewAerospikeStat(commons.CTX_NETWORK_STATS, statName, statName)
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

		sysMetric := statprocessors.NewAerospikeStat(commons.CTX_NETWORK_STATS, statName, statName)
		sysMetric.Labels = networkLabels
		sysMetric.LabelValues = labelValues
		sysMetric.Value = value

		arrSysInfoStats = append(arrSysInfoStats, sysMetric)
	}

	return arrSysInfoStats, nil
}
