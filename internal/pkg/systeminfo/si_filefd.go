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
		values := strings.Split(scanner.Text(), "\t")

		fmt.Println("len-values : ", len(values), " values ", values[0])
		allocated, _ := commons.TryConvert(values[0])
		maximum, _ := commons.TryConvert(values[1])
		arrSysInfoStats = append(arrSysInfoStats, constructFileFDstat("allocated", allocated))
		arrSysInfoStats = append(arrSysInfoStats, constructFileFDstat("maximum", maximum))
	}

	return arrSysInfoStats
}

func constructFileFDstat(key string, value float64) SystemInfoStat {
	clusterName := statprocessors.ClusterName
	service := statprocessors.Service

	labels := []string{}
	labels = append(labels, commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE)

	labelValues := []string{clusterName, service}

	sysMetric := NewSystemInfoStat(commons.CTX_FILEFD_STATS, key)
	sysMetric.Labels = labels
	sysMetric.LabelValues = labelValues
	sysMetric.Value = value

	return sysMetric
}
