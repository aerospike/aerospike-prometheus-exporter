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
	fmt.Println("stats.Total().Name: ... 1", PROC_PATH)

	if err != nil {
		log.Debug("parseNetworkStats Error while reading Net_Dev Stats from ", PROC_PATH, " Error ", err)
		return arrSysInfoStats
	}

	stats, err := fs.NetDev()
	fmt.Println("stats.Total().Name: ... 2", stats)
	if err != nil {
		log.Debug("Eror while reading procfs.NewFS system, error: ", err)
		return arrSysInfoStats
	}

	for k, v := range stats {
		// fmt.Println("stats.Total().Name: ... key: ", k, " value: ", v.Name)
		arrSysInfoStats = append(arrSysInfoStats, constructNetworkDevStat("group", k, 0))
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
