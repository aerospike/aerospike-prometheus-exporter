package systeminfo

import (
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/dataprovider"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/statprocessors"
	log "github.com/sirupsen/logrus"
)

type FileFDInfoProcessor struct {
}

func (ffdip FileFDInfoProcessor) Refresh() ([]statprocessors.AerospikeStat, error) {
	arrSysInfoStats := []statprocessors.AerospikeStat{}
	clusterName := statprocessors.ClusterName
	service := statprocessors.Service

	labelValues := []string{clusterName, service}
	fileStatLabels := []string{commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE}

	fileFDStats := dataprovider.GetSystemProvider().GetFileFD()

	statName := "allocated"
	statValue := fileFDStats[statName]
	value, err := commons.TryConvert(statValue)
	if err != nil {
		log.Error("Error while converting value of stat: ", statName, " and converted value is ", statValue)
		return arrSysInfoStats, nil
	}

	sysMetric := statprocessors.NewAerospikeStat(commons.CTX_FILEFD_STATS, statName, statName)
	sysMetric.Labels = fileStatLabels
	sysMetric.LabelValues = labelValues
	sysMetric.Value = value

	arrSysInfoStats = append(arrSysInfoStats, sysMetric)

	return arrSysInfoStats, nil
}
