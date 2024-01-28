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

func GetNetStatnfo() []SystemInfoStat {
	arrSysInfoStats := []SystemInfoStat{}

	arrSysInfoStats = append(arrSysInfoStats, parseNetStats(GetProcFilePath("net/netstat"))...)
	arrSysInfoStats = append(arrSysInfoStats, parseNetStats(GetProcFilePath("net/snmp"))...)
	arrSysInfoStats = append(arrSysInfoStats, parseSNMP6Stats(GetProcFilePath("net/snmp6"))...)

	return arrSysInfoStats
}

func parseNetStats(fileName string) []SystemInfoStat {
	arrSysInfoStats := []SystemInfoStat{}

	file, err := os.Open(fileName)
	if err != nil {
		log.Error("Error while opening file,", fileName, " Error: ", err)
		return arrSysInfoStats
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		nameParts := strings.Split(scanner.Text(), " ")
		scanner.Scan()
		valueParts := strings.Split(scanner.Text(), " ")
		protocol := nameParts[0][:len(nameParts[0])-1]
		if len(nameParts) != len(valueParts) {
			return arrSysInfoStats
		}
		for i := 1; i < len(nameParts); i++ {
			key := strings.ToLower(protocol + "_" + nameParts[i])
			// fmt.Println("key ", key, " acceptNetstat(key): ", acceptNetstat(key), " valueParts[i] ", valueParts[i])
			if acceptNetstat(key) {
				val, _ := commons.TryConvert(valueParts[i])
				arrSysInfoStats = append(arrSysInfoStats, constructNetstat(key, val))
			}
		}
	}

	return arrSysInfoStats
}

func constructNetstat(netStatKey string, value float64) SystemInfoStat {
	clusterName := statprocessors.ClusterName
	service := statprocessors.Service

	labels := []string{}
	labels = append(labels, commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE)

	labelValues := []string{clusterName, service}

	sysMetric := NewSystemInfoStat(commons.CTX_NET_STATS, netStatKey)
	sysMetric.Labels = labels
	sysMetric.LabelValues = labelValues
	sysMetric.Value = value

	return sysMetric
}

func parseSNMP6Stats(fileName string) []SystemInfoStat {
	arrSysInfoStats := []SystemInfoStat{}

	file, err := os.Open(fileName)
	if err != nil {
		log.Error("Error while opening file,", fileName, " Error: ", err)
		return arrSysInfoStats
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		stat := strings.Fields(scanner.Text())
		if len(stat) < 2 {
			continue
		}
		// key will have IP6 as prefix
		key := stat[0]
		value := stat[1]
		fmt.Println(" key ", key, " acceptNetstat(key) ", acceptNetstat(key), " value: ", value)

		// if sixIndex := strings.Index(stat[0], "6"); sixIndex != -1 {
		// 	protocol := stat[0][:sixIndex+1]
		// 	name := stat[0][sixIndex+1:]
		// 	if _, present := netStats[protocol]; !present {
		// 		netStats[protocol] = map[string]string{}
		// 	}
		// 	netStats[protocol][name] = stat[1]
		// }
	}

	return arrSysInfoStats
}
