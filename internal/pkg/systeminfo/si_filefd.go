package systeminfo

import (
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/dataprovider"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/statprocessors"
)

type FileFDInfoProcessor struct {
}

func (ffdip FileFDInfoProcessor) Refresh() ([]SystemInfoStat, error) {
	arrSysInfoStats := ffdip.parseFilefdStats()
	return arrSysInfoStats, nil
}

func (ffdip FileFDInfoProcessor) parseFilefdStats() []SystemInfoStat {
	arrSysInfoStats := []SystemInfoStat{}

	fileFDStats := dataprovider.GetSystemProvider().GetFileFD()

	for _, stats := range fileFDStats {

		allocated, _ := commons.TryConvert(stats["allocated"])
		maximum, _ := commons.TryConvert(stats["maximum"])
		arrSysInfoStats = append(arrSysInfoStats, ffdip.constructFileFDstat("allocated", allocated))
		arrSysInfoStats = append(arrSysInfoStats, ffdip.constructFileFDstat("maximum", maximum))
	}

	return arrSysInfoStats
}

func (ffdip FileFDInfoProcessor) constructFileFDstat(key string, value float64) SystemInfoStat {
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
