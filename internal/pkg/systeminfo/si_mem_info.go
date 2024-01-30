package systeminfo

import (
	"fmt"
	"strings"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/dataprovider"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/statprocessors"
)

func GetMemInfo() []SystemInfoStat {
	arrSysInfoStats := []SystemInfoStat{}

	memStats := dataprovider.GetSystemProvider().GetMemInfoStats()

	for _, stats := range memStats {
		clusterName := statprocessors.ClusterName
		service := statprocessors.Service

		labels := []string{commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE}
		labelValues := []string{clusterName, service}

		for k, v := range stats {
			l_metricName := strings.ToLower(k) + "_bytes"
			sysMetric := NewSystemInfoStat(commons.CTX_MEMORY_STATS, l_metricName)
			sysMetric.Labels = labels
			sysMetric.LabelValues = labelValues
			sysMetric.Value, _ = commons.TryConvert(v)

			arrSysInfoStats = append(arrSysInfoStats, sysMetric)

		}
	}

	fmt.Println(" memory metrics ", arrSysInfoStats)

	return arrSysInfoStats
}
