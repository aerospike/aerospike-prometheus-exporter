package systeminfo

import (
	"strings"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/dataprovider"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/statprocessors"
)

func GetFileSystemInfo() []SystemInfoStat {
	arrSysInfoStats := parseFileSystemInfo()
	return arrSysInfoStats
}

func parseFileSystemInfo() []SystemInfoStat {

	arrSysInfoStats := []SystemInfoStat{}

	arrFileSystemMountStats := dataprovider.GetSystemProvider().GetFileSystemStats()

	for _, stats := range arrFileSystemMountStats {

		isreadonly := stats["is_read_only"]
		source := stats["source"]
		mountPoint := stats["mount_point"]
		fsType := stats["mount_point"]

		arrSysInfoStats = append(arrSysInfoStats, constructFileSystemSysInfoStats(fsType, mountPoint, source, "size_bytes", stats))
		arrSysInfoStats = append(arrSysInfoStats, constructFileSystemSysInfoStats(fsType, mountPoint, source, "free_bytes", stats))
		arrSysInfoStats = append(arrSysInfoStats, constructFileSystemSysInfoStats(fsType, mountPoint, source, "avail_byts", stats))
		arrSysInfoStats = append(arrSysInfoStats, constructFileSystemSysInfoStats(fsType, mountPoint, source, "files", stats))
		arrSysInfoStats = append(arrSysInfoStats, constructFileSystemSysInfoStats(fsType, mountPoint, source, "files_free", stats))

		// add disk-info
		statReadOnly := constructFileSystemReadOnly(fsType, mountPoint, source, isreadonly)
		arrSysInfoStats = append(arrSysInfoStats, statReadOnly)

	}

	return arrSysInfoStats
}

func constructFileSystemReadOnly(fstype string, mountpoint string, deviceName string, isReadOnly string) SystemInfoStat {
	clusterName := statprocessors.ClusterName
	service := statprocessors.Service

	// add disk_info
	labels := []string{}
	labels = append(labels, commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE)
	labels = append(labels, commons.METRIC_LABEL_FSTYPE, commons.METRIC_LABEL_DEVICE, commons.METRIC_LABEL_MOUNT_POINT)
	labelValues := []string{clusterName, service, fstype, deviceName, mountpoint}

	sysMetric := NewSystemInfoStat(commons.CTX_FILESYSTEM_STATS, "readonly")
	sysMetric.Labels = labels
	sysMetric.LabelValues = labelValues
	sysMetric.Value, _ = commons.TryConvert(isReadOnly)

	return sysMetric

}

func constructFileSystemSysInfoStats(fstype string, mountpoint string, deviceName string, statName string, stats map[string]string) SystemInfoStat {

	clusterName := statprocessors.ClusterName
	service := statprocessors.Service

	labels := []string{commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE, commons.METRIC_LABEL_FSTYPE, commons.METRIC_LABEL_DEVICE, commons.METRIC_LABEL_MOUNT_POINT}
	labelValues := []string{clusterName, service, fstype, deviceName, mountpoint}

	l_metricName := strings.ToLower(statName)
	sysMetric := NewSystemInfoStat(commons.CTX_FILESYSTEM_STATS, l_metricName)

	sysMetric.Labels = labels
	sysMetric.LabelValues = labelValues

	value, _ := commons.TryConvert(stats[statName])
	sysMetric.Value = value

	return sysMetric
}
