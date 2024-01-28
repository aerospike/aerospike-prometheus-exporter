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

	arrSysInfoStats = append(arrSysInfoStats, parseNetworkStats(GetProcFilePath("net/netstat"))...)

	return arrSysInfoStats
}

func parseNetworkStats(fileName string) []SystemInfoStat {
	arrSysInfoStats := []SystemInfoStat{}

	fs, err := procfs.NewFS(NET_DEV_STAT_PATH)
	if err != nil {
		log.Debug("parseCpuStats Error while reading Network/Net-Dev Stats from ", NET_DEV_STAT_PATH, " Error ", err)
		return arrSysInfoStats
	}

	stats, err := fs.NetDev()
	if err != nil {
		log.Debug("Eror while reading procfs.NewFS system, error: ", err)
		return arrSysInfoStats
	}

	fmt.Println("stats.Total().Name: ", stats.Total().Name)

	return arrSysInfoStats
}

func constructNetworkStat(netStatKey string, value float64) SystemInfoStat {
	clusterName := statprocessors.ClusterName
	service := statprocessors.Service

	labels := []string{}
	labels = append(labels, commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE)

	labelValues := []string{clusterName, service}

	sysMetric := NewSystemInfoStat(commons.CTX_NET_DEV_STATS, netStatKey)
	sysMetric.Labels = labels
	sysMetric.LabelValues = labelValues
	sysMetric.Value = value

	return sysMetric
}
