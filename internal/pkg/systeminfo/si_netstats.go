package systeminfo

import (
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/dataprovider"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/statprocessors"
)

type NetStatInfoProcessor struct {
}

var (
	netStatInfoLabels []string
)

func (nsip NetStatInfoProcessor) Refresh() ([]statprocessors.AerospikeStat, error) {
	arrSysInfoStats := []statprocessors.AerospikeStat{}

	netStatInfoLabels = []string{}
	netStatInfoLabels = append(netStatInfoLabels, commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE)

	netStats, snmpStats, snmp6Stats := dataprovider.GetSystemProvider().GetNetStatInfo()

	// Net Dev
	for _, stats := range netStats {
		for key, _ := range stats {
			arrSysInfoStats = append(arrSysInfoStats, nsip.constructNetstat(key, stats))
		}
	}

	//Net SNMP
	for _, stats := range snmpStats {
		for key, _ := range stats {
			arrSysInfoStats = append(arrSysInfoStats, nsip.constructNetstat(key, stats))
		}
	}

	//Net SNMP6
	for _, stats := range snmp6Stats {
		for key, _ := range stats {
			arrSysInfoStats = append(arrSysInfoStats, nsip.constructNetstat(key, stats))
		}
	}

	return arrSysInfoStats, nil
}

func (nsip NetStatInfoProcessor) constructNetstat(statName string, stats map[string]string) statprocessors.AerospikeStat {
	clusterName := statprocessors.ClusterName
	service := statprocessors.Service

	labelValues := []string{clusterName, service}

	sysMetric := statprocessors.NewAerospikeStat(commons.CTX_NET_STATS, statName, statName)
	sysMetric.Labels = netStatInfoLabels
	sysMetric.LabelValues = labelValues
	sysMetric.Value, _ = commons.TryConvert(stats[statName])

	return sysMetric
}
