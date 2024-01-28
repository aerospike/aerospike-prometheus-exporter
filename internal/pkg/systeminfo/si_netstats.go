package systeminfo

import (
	"bufio"
	"os"
	"strings"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/statprocessors"
	log "github.com/sirupsen/logrus"
)

func GetNetStatInfo() []SystemInfoStat {
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
		statNames := strings.Split(scanner.Text(), " ")
		scanner.Scan()
		valueParts := strings.Split(scanner.Text(), " ")
		protocol := statNames[0][:len(statNames[0])-1]
		if len(statNames) != len(valueParts) {
			return arrSysInfoStats
		}
		for i := 1; i < len(statNames); i++ {
			key := strings.ToLower(protocol + "_" + statNames[i])
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
		snmp6Stat := strings.Fields(scanner.Text())
		if len(snmp6Stat) < 2 {
			continue
		}
		// statProtocolName will have IP6 as prefix
		statProtocolName := strings.ToLower(snmp6Stat[0])
		value := snmp6Stat[1]

		if acceptSnmp6(statProtocolName) {
			ele := strings.Split(statProtocolName, "6")
			// snmp6 metric format: ip6inreceives
			protocol := ele[0]
			stat := ele[1]

			key := protocol + "6_" + stat
			if acceptNetstat(key) {
				val, _ := commons.TryConvert(value)
				arrSysInfoStats = append(arrSysInfoStats, constructNetstat(key, val))
			}
		}

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
