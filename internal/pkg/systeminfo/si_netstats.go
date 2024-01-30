package systeminfo

import (
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/dataprovider"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/statprocessors"
)

func GetNetStatInfo() []SystemInfoStat {

	arrSysInfoStats := parseNetStats()
	return arrSysInfoStats
}

func parseNetStats() []SystemInfoStat {
	arrSysInfoStats := []SystemInfoStat{}

	netStats := dataprovider.GetSystemProvider().GetNetStatInfo()

	for _, stats := range netStats {
		for key, _ := range stats {
			arrSysInfoStats = append(arrSysInfoStats, constructNetstat(key, stats))
		}
	}

	return arrSysInfoStats
}

func constructNetstat(netStatKey string, stats map[string]string) SystemInfoStat {
	clusterName := statprocessors.ClusterName
	service := statprocessors.Service

	labels := []string{}
	labels = append(labels, commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE)

	labelValues := []string{clusterName, service}

	sysMetric := NewSystemInfoStat(commons.CTX_NET_STATS, netStatKey)
	sysMetric.Labels = labels
	sysMetric.LabelValues = labelValues
	sysMetric.Value, _ = commons.TryConvert(stats[netStatKey])

	return sysMetric
}
