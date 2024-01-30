package systeminfo

import (
	"strings"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/dataprovider"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/statprocessors"
	log "github.com/sirupsen/logrus"
)

func GetDiskStats() []SystemInfoStat {

	arrSysInfoStats := parseDiskStats()
	// fmt.Println("GetDiskStats - arrSysInfoStats: ", len(arrSysInfoStats))

	return arrSysInfoStats
}

func parseDiskStats() []SystemInfoStat {
	arrSysInfoStats := []SystemInfoStat{}

	diskStats := dataprovider.GetSystemProvider().GetDiskStats()

	for _, stats := range diskStats {
		deviceName := stats["device_name"]

		if ignoreDisk(deviceName) {
			log.Debug("\t ** DiskStats -- Ignoring mount ", deviceName)
			continue
		}

		arrSysInfoStats = append(arrSysInfoStats, constructDiskinfoSystemStat(deviceName, "reads_completed_total", stats))
		arrSysInfoStats = append(arrSysInfoStats, constructDiskinfoSystemStat(deviceName, "reads_merged_total", stats))
		arrSysInfoStats = append(arrSysInfoStats, constructDiskinfoSystemStat(deviceName, "read_bytes_total", stats))
		arrSysInfoStats = append(arrSysInfoStats, constructDiskinfoSystemStat(deviceName, "read_time_seconds_total", stats))
		arrSysInfoStats = append(arrSysInfoStats, constructDiskinfoSystemStat(deviceName, "writes_completed_total", stats))
		arrSysInfoStats = append(arrSysInfoStats, constructDiskinfoSystemStat(deviceName, "writes_merged_total", stats))
		arrSysInfoStats = append(arrSysInfoStats, constructDiskinfoSystemStat(deviceName, "writes_bytes_total", stats))
		arrSysInfoStats = append(arrSysInfoStats, constructDiskinfoSystemStat(deviceName, "write_time_seconds_total", stats))
		arrSysInfoStats = append(arrSysInfoStats, constructDiskinfoSystemStat(deviceName, "io_now", stats))
		arrSysInfoStats = append(arrSysInfoStats, constructDiskinfoSystemStat(deviceName, "io_time_seconds_total", stats))
		arrSysInfoStats = append(arrSysInfoStats, constructDiskinfoSystemStat(deviceName, "io_time_weighted_seconds_total", stats))
		arrSysInfoStats = append(arrSysInfoStats, constructDiskinfoSystemStat(deviceName, "discards_completed_total", stats))
		arrSysInfoStats = append(arrSysInfoStats, constructDiskinfoSystemStat(deviceName, "discards_merged_total", stats))
		arrSysInfoStats = append(arrSysInfoStats, constructDiskinfoSystemStat(deviceName, "discarded_sectors_total", stats))
		arrSysInfoStats = append(arrSysInfoStats, constructDiskinfoSystemStat(deviceName, "discard_time_seconds_total", stats))
		arrSysInfoStats = append(arrSysInfoStats, constructDiskinfoSystemStat(deviceName, "flush_requests_total", stats))
		arrSysInfoStats = append(arrSysInfoStats, constructDiskinfoSystemStat(deviceName, "flush_requests_time_seconds_total", stats))

		arrSysInfoStats = append(arrSysInfoStats, constructDiskInfo(deviceName, stats["major_number"], stats["minor_number"], stats["serial"]))

	}

	return arrSysInfoStats
}

func constructDiskInfo(deviceName string, major string, minor string, serial string) SystemInfoStat {
	// 	[]string{"device", "major", "minor", "path", "wwn", "model", "serial", "revision"},
	// (stats.MajorNumber),(stats.MinorNumber), info[udevIDPath], info[udevIDWWN], info[udevIDModel], serial, info[udevIDRevision],
	clusterName := statprocessors.ClusterName
	service := statprocessors.Service

	// add disk_info
	labels := []string{}
	labels = append(labels, commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE, commons.METRIC_LABEL_DEVICE)
	labels = append(labels, commons.METRIC_LABEL_MAJOR, commons.METRIC_LABEL_MINOR, commons.METRIC_LABEL_SERIAL)

	labelValues := []string{clusterName, service, deviceName, major, minor, serial}

	sysMetric := NewSystemInfoStat(commons.CTX_DISK_STATS, "info")
	sysMetric.Labels = labels
	sysMetric.LabelValues = labelValues
	sysMetric.Value = 1

	return sysMetric

}

func constructDiskinfoSystemStat(deviceName string, statName string, diskStats map[string]string) SystemInfoStat {

	clusterName := statprocessors.ClusterName
	service := statprocessors.Service

	labels := []string{commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE, commons.METRIC_LABEL_DEVICE}
	labelValues := []string{clusterName, service, deviceName}

	l_metricName := strings.ToLower(statName)
	sysMetric := NewSystemInfoStat(commons.CTX_DISK_STATS, l_metricName)

	sysMetric.Labels = labels
	sysMetric.LabelValues = labelValues
	value, _ := commons.TryConvert(diskStats[statName])
	sysMetric.Value = value

	return sysMetric
}

// func constructDiskinfoStats(deviceName string, v_stats_info map[string]float64) []SystemInfoStat {
// 	arrSysInfoStats := []SystemInfoStat{}

// 	clusterName := statprocessors.ClusterName
// 	service := statprocessors.Service

// 	for sk, sv := range v_stats_info {
// 		labels := []string{commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE, commons.METRIC_LABEL_DEVICE}
// 		labelValues := []string{clusterName, service, deviceName}

// 		l_metricName := strings.ToLower(sk)
// 		sysMetric := NewSystemInfoStat(commons.CTX_DISK_STATS, l_metricName)

// 		sysMetric.Labels = labels
// 		sysMetric.LabelValues = labelValues
// 		sysMetric.Value = sv

// 		arrSysInfoStats = append(arrSysInfoStats, sysMetric)
// 	}

// 	return arrSysInfoStats
// }

// func parseDiskStats() []SystemInfoStat {
// 	arrSysInfoStats := []SystemInfoStat{}
// 	// deviceStats := make(map[string]DiskStats)
// 	// var diskLabelNames = []string{"device"}
// 	// fmt.Println(" diskLabelNames --> ", diskLabelNames)

// 	fs, err := blockdevice.NewFS(PROC_PATH, SYS_PATH)
// 	if err != nil {
// 		return arrSysInfoStats
// 	}

// 	procDiskStats, err := fs.ProcDiskstats()
// 	if err != nil {
// 		return arrSysInfoStats
// 	}

// 	for _, stats := range procDiskStats {
// 		deviceName := stats.DeviceName

// 		if ignoreDisk(deviceName) {
// 			log.Debug("\t ** DiskStats -- Ignoring mount ", deviceName)
// 			continue
// 		}

// 		l_stats_info := make(map[string]float64)
// 		l_udev_info := make(map[string]string)

// 		// ds := DiskStats{l_stats_info, l_udev_info}
// 		// deviceStats[deviceName] = ds

// 		l_stats_info["reads_completed_total"] = float64(stats.ReadIOs)
// 		l_stats_info["reads_merged_total"] = float64(stats.ReadMerges)
// 		l_stats_info["read_bytes_total"] = float64(stats.ReadSectors) * unixSectorSize
// 		l_stats_info["read_time_seconds_total"] = float64(stats.ReadTicks) * secondsPerTick
// 		l_stats_info["writes_completed_total"] = float64(stats.WriteIOs)
// 		l_stats_info["writes_merged_total"] = float64(stats.WriteMerges)
// 		l_stats_info["writes_bytes_total"] = float64(stats.WriteSectors) * unixSectorSize
// 		l_stats_info["write_time_seconds_total"] = float64(stats.WriteTicks) * secondsPerTick
// 		l_stats_info["io_now"] = float64(stats.IOsInProgress)
// 		l_stats_info["io_time_seconds_total"] = float64(stats.IOsTotalTicks) * secondsPerTick
// 		l_stats_info["io_time_weighted_seconds_total"] = float64(stats.WeightedIOTicks) * secondsPerTick
// 		l_stats_info["discards_completed_total"] = float64(stats.DiscardIOs)
// 		l_stats_info["discards_merged_total"] = float64(stats.DiscardMerges)
// 		l_stats_info["discarded_sectors_total"] = float64(stats.DiscardSectors)
// 		l_stats_info["discard_time_seconds_total"] = float64(stats.DiscardTicks) * secondsPerTick
// 		l_stats_info["flush_requests_total"] = float64(stats.FlushRequestsCompleted)
// 		l_stats_info["flush_requests_time_seconds_total"] = float64(stats.TimeSpentFlushing) * secondsPerTick
// 		l_stats_info["flush_requests_time_seconds_total"] = float64(stats.TimeSpentFlushing) * secondsPerTick
// 		// l_stats_info["Major_Number"] = float64(stats.MajorNumber)
// 		// l_stats_info["Minor_Number"] = float64(stats.MinorNumber)

// 		udevDeviceProps, err := getUdevDeviceProperties(stats.MajorNumber, stats.MinorNumber)

// 		if err != nil {
// 			// fmt.Println("\t\t *** msg", "Failed to parse udev info", "err", err)
// 			log.Debug("msg", "Failed to parse udev info", "err", err)
// 		}

// 		for k, v := range udevDeviceProps {
// 			// fmt.Println("info k: ", k, "\t v: ", v)
// 			l_udev_info[k] = v
// 		}
// 		// This is usually the serial printed on the disk label.
// 		serial := udevDeviceProps[udevSCSIIdentSerial]

// 		// If it's undefined, fallback to ID_SERIAL_SHORT instead.
// 		if serial == "" {
// 			serial = udevDeviceProps[udevIDSerialShort]
// 		}
// 		l_udev_info["serial"] = serial

// 		// create Stat objects now
// 		l_sysinfo_stats := constructDiskinfoStats(deviceName, l_stats_info)

// 		// add to return array
// 		arrSysInfoStats = append(arrSysInfoStats, l_sysinfo_stats...)

// 		// add disk-info
// 		statDiskInfo := constructDiskInfo(deviceName, fmt.Sprint(stats.MajorNumber), fmt.Sprint(stats.MinorNumber), serial)
// 		arrSysInfoStats = append(arrSysInfoStats, statDiskInfo)

// 	}

// 	return arrSysInfoStats
// }
