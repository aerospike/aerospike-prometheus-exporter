package systeminfo

import (
	"strings"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/dataprovider"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/statprocessors"
	log "github.com/sirupsen/logrus"
)

type MemInfoProcessor struct {
}

func (mip MemInfoProcessor) Refresh() ([]statprocessors.AerospikeStat, error) {
	arrSysInfoStats := []statprocessors.AerospikeStat{}

	memStats := dataprovider.GetSystemProvider().GetMemInfoStats()

	memInfoLabels := []string{commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE}
	for statName, statValue := range memStats {
		clusterName := statprocessors.ClusterName
		service := statprocessors.Service
		value, err := commons.TryConvert(statValue)
		if err != nil {
			log.Error("Error while converting value of stat: ", statName, " and converted value is ", statValue)
			continue
		}

		labelValues := []string{clusterName, service}

		metricName := strings.ToLower(statName) + "_bytes"
		sysMetric := statprocessors.NewAerospikeStat(commons.CTX_MEMORY_STATS, metricName, metricName)
		sysMetric.Labels = memInfoLabels
		sysMetric.LabelValues = labelValues
		sysMetric.Value = value

		arrSysInfoStats = append(arrSysInfoStats, sysMetric)
	}

	return arrSysInfoStats, nil
}
