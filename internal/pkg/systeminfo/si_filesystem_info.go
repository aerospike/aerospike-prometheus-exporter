package systeminfo

import (
	"strings"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/dataprovider"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/statprocessors"
)

type FileSystemInfoProcessor struct {
}

var (
	fsReadOnlyLabels []string
	fsInfoLabels     []string
)

func (fsip FileSystemInfoProcessor) Refresh() ([]statprocessors.AerospikeStat, error) {
	arrFileSystemMountStats := dataprovider.GetSystemProvider().GetFileSystemStats()

	// global labels
	fsReadOnlyLabels = []string{}
	fsReadOnlyLabels = append(fsReadOnlyLabels, commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE)
	fsReadOnlyLabels = append(fsReadOnlyLabels, commons.METRIC_LABEL_FSTYPE, commons.METRIC_LABEL_DEVICE, commons.METRIC_LABEL_MOUNT_POINT)

	fsInfoLabels = []string{commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE, commons.METRIC_LABEL_FSTYPE, commons.METRIC_LABEL_DEVICE, commons.METRIC_LABEL_MOUNT_POINT}

	arrSysInfoStats := []statprocessors.AerospikeStat{}
	for _, stats := range arrFileSystemMountStats {

		isreadonly := stats["is_read_only"]
		source := stats["source"]
		mountPoint := stats["mount_point"]
		fsType := stats["mount_point"]

		arrSysInfoStats = append(arrSysInfoStats, fsip.constructFileSystemSysInfoStats(fsType, mountPoint, source, "size_bytes", stats))
		arrSysInfoStats = append(arrSysInfoStats, fsip.constructFileSystemSysInfoStats(fsType, mountPoint, source, "free_bytes", stats))
		arrSysInfoStats = append(arrSysInfoStats, fsip.constructFileSystemSysInfoStats(fsType, mountPoint, source, "avail_byts", stats))
		arrSysInfoStats = append(arrSysInfoStats, fsip.constructFileSystemSysInfoStats(fsType, mountPoint, source, "files", stats))
		arrSysInfoStats = append(arrSysInfoStats, fsip.constructFileSystemSysInfoStats(fsType, mountPoint, source, "files_free", stats))

		// add disk-info
		statReadOnly := fsip.constructFileSystemReadOnly(fsType, mountPoint, source, isreadonly)
		arrSysInfoStats = append(arrSysInfoStats, statReadOnly)
	}

	return arrSysInfoStats, nil
}

func (fsip FileSystemInfoProcessor) constructFileSystemReadOnly(fstype string, mountpoint string, deviceName string, isReadOnly string) statprocessors.AerospikeStat {
	clusterName := statprocessors.ClusterName
	service := statprocessors.Service

	// add disk_info
	labelValues := []string{clusterName, service, fstype, deviceName, mountpoint}

	sysMetric := statprocessors.NewAerospikeStat(commons.CTX_FILESYSTEM_STATS, "readonly")
	sysMetric.Labels = fsReadOnlyLabels
	sysMetric.LabelValues = labelValues
	sysMetric.Value, _ = commons.TryConvert(isReadOnly)

	return sysMetric

}

func (fsip FileSystemInfoProcessor) constructFileSystemSysInfoStats(fstype string, mountpoint string, deviceName string, statName string, stats map[string]string) statprocessors.AerospikeStat {

	clusterName := statprocessors.ClusterName
	service := statprocessors.Service

	labelValues := []string{clusterName, service, fstype, deviceName, mountpoint}

	metricName := strings.ToLower(statName)
	sysMetric := statprocessors.NewAerospikeStat(commons.CTX_FILESYSTEM_STATS, metricName)

	sysMetric.Labels = fsInfoLabels
	sysMetric.LabelValues = labelValues

	value, _ := commons.TryConvert(stats[statName])
	sysMetric.Value = value

	return sysMetric
}
