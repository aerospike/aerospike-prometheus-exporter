package systeminfo

import (
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/dataprovider"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/statprocessors"
)

type NetStatInfoProcessor struct {
}

func (nsip NetStatInfoProcessor) Refresh() ([]statprocessors.AerospikeStat, error) {
	arrSysInfoStats := []statprocessors.AerospikeStat{}

	netStatInfoLabels := []string{}
	netStatInfoLabels = append(netStatInfoLabels, commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE)

	snmpStats := dataprovider.GetSystemProvider().GetNetStatInfo()

	//Net SNMP - includes TCP metrics like active_conn, established, retransmit etc.,
	clusterName := statprocessors.ClusterName
	service := statprocessors.Service
	labelValues := []string{clusterName, service}

	for statName, value := range snmpStats {
		sysMetric := statprocessors.NewAerospikeStat(commons.CTX_NET_STATS, statName, statName)
		sysMetric.Labels = netStatInfoLabels
		sysMetric.LabelValues = labelValues
		sysMetric.Value, _ = commons.TryConvert(value)

		arrSysInfoStats = append(arrSysInfoStats, sysMetric)
	}

	return arrSysInfoStats, nil
}
