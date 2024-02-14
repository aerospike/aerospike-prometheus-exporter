package systeminfo

import (
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/dataprovider"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/statprocessors"
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

	sysMetric := statprocessors.NewAerospikeStat(commons.CTX_FILEFD_STATS, "allocated", "allocated")
	sysMetric.Labels = fileStatLabels
	sysMetric.LabelValues = labelValues
	sysMetric.Value, _ = commons.TryConvert(fileFDStats["allocated"])

	arrSysInfoStats = append(arrSysInfoStats, sysMetric)

	return arrSysInfoStats, nil
}
