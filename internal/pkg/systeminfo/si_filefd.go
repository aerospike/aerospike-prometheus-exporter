package systeminfo

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/statprocessors"
	log "github.com/sirupsen/logrus"
)

func GetFileFDInfo() []SystemInfoStat {
	arrSysInfoStats := []SystemInfoStat{}

	arrSysInfoStats = append(arrSysInfoStats, parseFilefdStats(GetProcFilePath("sys/fs/file-nr"))...)

	return arrSysInfoStats
}

func parseFilefdStats(fileName string) []SystemInfoStat {
	arrSysInfoStats := []SystemInfoStat{}

	file, err := os.Open(fileName)
	if err != nil {
		log.Error("Error while opening file,", fileName, " Error: ", err)
		return arrSysInfoStats
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		values := strings.Split(scanner.Text(), " ")

		fmt.Println("len-values : ", len(values), " values ", values[0], values[1], values[2])
		// for i := 1; i < len(statNames); i++ {
		// 	key := strings.ToLower(protocol + "_" + statNames[i])
		// 	val, _ := commons.TryConvert(values[i])
		// 	arrSysInfoStats = append(arrSysInfoStats, constructFileFDstat(key, val))
		// }
	}

	return arrSysInfoStats
}

func constructFileFDstat(netStatKey string, value float64) SystemInfoStat {
	clusterName := statprocessors.ClusterName
	service := statprocessors.Service

	labels := []string{}
	labels = append(labels, commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE)

	labelValues := []string{clusterName, service}

	sysMetric := NewSystemInfoStat(commons.CTX_FILEFD_STATS, netStatKey)
	sysMetric.Labels = labels
	sysMetric.LabelValues = labelValues
	sysMetric.Value = value

	return sysMetric
}
