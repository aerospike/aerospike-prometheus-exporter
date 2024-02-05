package systeminfo

import (
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/dataprovider"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/statprocessors"
)

type FileFDInfoProcessor struct {
}

var (
	fileStatLabels []string
)

func (ffdip FileFDInfoProcessor) Refresh() ([]statprocessors.AerospikeStat, error) {
	arrSysInfoStats := []statprocessors.AerospikeStat{}

	fileStatLabels = []string{commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE}

	fileFDStats := dataprovider.GetSystemProvider().GetFileFD()
	for _, stats := range fileFDStats {

		allocated, _ := commons.TryConvert(stats["allocated"])
		maximum, _ := commons.TryConvert(stats["maximum"])
		arrSysInfoStats = append(arrSysInfoStats, ffdip.constructFileFDstat("allocated", allocated))
		arrSysInfoStats = append(arrSysInfoStats, ffdip.constructFileFDstat("maximum", maximum))
	}

	return arrSysInfoStats, nil
}

func (ffdip FileFDInfoProcessor) constructFileFDstat(statName string, value float64) statprocessors.AerospikeStat {
	clusterName := statprocessors.ClusterName
	service := statprocessors.Service

	labelValues := []string{clusterName, service}

	sysMetric := statprocessors.NewAerospikeStat(commons.CTX_FILEFD_STATS, statName)
	sysMetric.Labels = fileStatLabels
	sysMetric.LabelValues = labelValues
	sysMetric.Value = value

	return sysMetric
}
