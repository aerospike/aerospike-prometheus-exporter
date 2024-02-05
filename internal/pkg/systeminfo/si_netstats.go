package systeminfo

import (
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/dataprovider"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/statprocessors"
)

type NetStatInfoProcessor struct {
}

func (nsip NetStatInfoProcessor) Refresh() ([]statprocessors.AerospikeStat, error) {

	arrSysInfoStats := nsip.parseNetStats()
	return arrSysInfoStats, nil
}

func (nsip NetStatInfoProcessor) parseNetStats() []statprocessors.AerospikeStat {
	arrSysInfoStats := []statprocessors.AerospikeStat{}

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

	return arrSysInfoStats
}

func (nsip NetStatInfoProcessor) constructNetstat(netStatKey string, stats map[string]string) statprocessors.AerospikeStat {
	clusterName := statprocessors.ClusterName
	service := statprocessors.Service

	labels := []string{}
	labels = append(labels, commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE)

	labelValues := []string{clusterName, service}

	sysMetric := statprocessors.NewAerospikeStat(commons.CTX_NET_STATS, netStatKey)
	sysMetric.Labels = labels
	sysMetric.LabelValues = labelValues
	sysMetric.Value, _ = commons.TryConvert(stats[netStatKey])

	return sysMetric
}
