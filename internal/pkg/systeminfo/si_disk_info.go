package systeminfo

import (
	"strings"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/dataprovider"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/statprocessors"
)

type DiskInfoProcessor struct {
}

var (
	metricDiskInfoLabels = []string{}
	diskInfoLabels       = []string{}
)

func (dip DiskInfoProcessor) Refresh() ([]statprocessors.AerospikeStat, error) {

	// metric: diskinfo
	metricDiskInfoLabels = append(metricDiskInfoLabels, commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE, commons.METRIC_LABEL_DEVICE)
	metricDiskInfoLabels = append(metricDiskInfoLabels, commons.METRIC_LABEL_MAJOR, commons.METRIC_LABEL_MINOR, commons.METRIC_LABEL_SERIAL)

	// other disk metrics
	diskInfoLabels = []string{commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE, commons.METRIC_LABEL_DEVICE}

	arrSysInfoStats := dip.parseDiskStats()

	return arrSysInfoStats, nil
}

func (dip DiskInfoProcessor) parseDiskStats() []statprocessors.AerospikeStat {
	arrSysInfoStats := []statprocessors.AerospikeStat{}

	diskStats := dataprovider.GetSystemProvider().GetDiskStats()

	for _, stats := range diskStats {
		deviceName := stats["device_name"]

		arrSysInfoStats = append(arrSysInfoStats, dip.constructDiskinfoSystemStat(deviceName, "reads_completed_total", stats))
		arrSysInfoStats = append(arrSysInfoStats, dip.constructDiskinfoSystemStat(deviceName, "reads_merged_total", stats))
		arrSysInfoStats = append(arrSysInfoStats, dip.constructDiskinfoSystemStat(deviceName, "read_bytes_total", stats))
		arrSysInfoStats = append(arrSysInfoStats, dip.constructDiskinfoSystemStat(deviceName, "read_time_seconds_total", stats))
		arrSysInfoStats = append(arrSysInfoStats, dip.constructDiskinfoSystemStat(deviceName, "writes_completed_total", stats))
		arrSysInfoStats = append(arrSysInfoStats, dip.constructDiskinfoSystemStat(deviceName, "writes_merged_total", stats))
		arrSysInfoStats = append(arrSysInfoStats, dip.constructDiskinfoSystemStat(deviceName, "writes_bytes_total", stats))
		arrSysInfoStats = append(arrSysInfoStats, dip.constructDiskinfoSystemStat(deviceName, "write_time_seconds_total", stats))
		arrSysInfoStats = append(arrSysInfoStats, dip.constructDiskinfoSystemStat(deviceName, "io_now", stats))
		arrSysInfoStats = append(arrSysInfoStats, dip.constructDiskinfoSystemStat(deviceName, "io_time_seconds_total", stats))
		arrSysInfoStats = append(arrSysInfoStats, dip.constructDiskinfoSystemStat(deviceName, "io_time_weighted_seconds_total", stats))
		arrSysInfoStats = append(arrSysInfoStats, dip.constructDiskinfoSystemStat(deviceName, "discards_completed_total", stats))
		arrSysInfoStats = append(arrSysInfoStats, dip.constructDiskinfoSystemStat(deviceName, "discards_merged_total", stats))
		arrSysInfoStats = append(arrSysInfoStats, dip.constructDiskinfoSystemStat(deviceName, "discarded_sectors_total", stats))
		arrSysInfoStats = append(arrSysInfoStats, dip.constructDiskinfoSystemStat(deviceName, "discard_time_seconds_total", stats))
		arrSysInfoStats = append(arrSysInfoStats, dip.constructDiskinfoSystemStat(deviceName, "flush_requests_total", stats))
		arrSysInfoStats = append(arrSysInfoStats, dip.constructDiskinfoSystemStat(deviceName, "flush_requests_time_seconds_total", stats))

		arrSysInfoStats = append(arrSysInfoStats, dip.constructDiskInfo(deviceName, stats["major_number"], stats["minor_number"], stats["serial"]))

	}

	return arrSysInfoStats
}

func (dip DiskInfoProcessor) constructDiskInfo(deviceName string, major string, minor string, serial string) statprocessors.AerospikeStat {
	// 	[]string{"device", "major", "minor", "path", "wwn", "model", "serial", "revision"},
	// (stats.MajorNumber),(stats.MinorNumber), info[udevIDPath], info[udevIDWWN], info[udevIDModel], serial, info[udevIDRevision],
	clusterName := statprocessors.ClusterName
	service := statprocessors.Service

	labelValues := []string{clusterName, service, deviceName, major, minor, serial}

	sysMetric := statprocessors.NewAerospikeStat(commons.CTX_DISK_STATS, "info")
	sysMetric.Labels = metricDiskInfoLabels
	sysMetric.LabelValues = labelValues
	sysMetric.Value = 1

	return sysMetric

}

func (dip DiskInfoProcessor) constructDiskinfoSystemStat(deviceName string, statName string, diskStats map[string]string) statprocessors.AerospikeStat {

	clusterName := statprocessors.ClusterName
	service := statprocessors.Service

	labelValues := []string{clusterName, service, deviceName}

	l_metricName := strings.ToLower(statName)
	sysMetric := statprocessors.NewAerospikeStat(commons.CTX_DISK_STATS, l_metricName)

	sysMetric.Labels = diskInfoLabels
	sysMetric.LabelValues = labelValues
	value, _ := commons.TryConvert(diskStats[statName])
	sysMetric.Value = value

	return sysMetric
}
