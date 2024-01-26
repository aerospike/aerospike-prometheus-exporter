package systeminfo

import (
	"fmt"
	"strings"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/statprocessors"
	"github.com/prometheus/procfs/blockdevice"
	log "github.com/sirupsen/logrus"
)

const (
	secondsPerTick = 1.0 / 1000.0

	// Read sectors and write sectors are the "standard UNIX 512-byte sectors, not any device- or filesystem-specific block size."
	// See also https://www.kernel.org/doc/Documentation/block/stat.txt
	unixSectorSize = 512.0

	// See udevadm(8).
	udevDevicePropertyPrefix = "E:"

	// Udev device properties.
	udevDMLVLayer               = "DM_LV_LAYER"
	udevDMLVName                = "DM_LV_NAME"
	udevDMName                  = "DM_NAME"
	udevDMUUID                  = "DM_UUID"
	udevDMVGName                = "DM_VG_NAME"
	udevIDATA                   = "ID_ATA"
	udevIDATARotationRateRPM    = "ID_ATA_ROTATION_RATE_RPM"
	udevIDATASATA               = "ID_ATA_SATA"
	udevIDATASATASignalRateGen1 = "ID_ATA_SATA_SIGNAL_RATE_GEN1"
	udevIDATASATASignalRateGen2 = "ID_ATA_SATA_SIGNAL_RATE_GEN2"
	udevIDATAWriteCache         = "ID_ATA_WRITE_CACHE"
	udevIDATAWriteCacheEnabled  = "ID_ATA_WRITE_CACHE_ENABLED"
	udevIDFSType                = "ID_FS_TYPE"
	udevIDFSUsage               = "ID_FS_USAGE"
	udevIDFSUUID                = "ID_FS_UUID"
	udevIDFSVersion             = "ID_FS_VERSION"
	udevIDModel                 = "ID_MODEL"
	udevIDPath                  = "ID_PATH"
	udevIDRevision              = "ID_REVISION"
	udevIDSerialShort           = "ID_SERIAL_SHORT"
	udevIDWWN                   = "ID_WWN"
	udevSCSIIdentSerial         = "SCSI_IDENT_SERIAL"
)

type DiskStats struct {
	stats_info map[string]float64
	udev_info  map[string]string
}

func GetDiskStats() []SystemInfoStat {

	arrSysInfoStats := parseDiskStats()
	fmt.Println("GetDiskStats - arrSysInfoStats: ", len(arrSysInfoStats))

	return arrSysInfoStats
}

func parseDiskStats() []SystemInfoStat {
	arrSysInfoStats := []SystemInfoStat{}
	deviceStats := make(map[string]DiskStats)
	// var diskLabelNames = []string{"device"}
	// fmt.Println(" diskLabelNames --> ", diskLabelNames)

	fs, err := blockdevice.NewFS(PROC_PATH, SYS_PATH)
	if err != nil {
		return arrSysInfoStats
	}
	diskStats, err := fs.ProcDiskstats()
	if err != nil {
		return arrSysInfoStats
	}

	for _, stats := range diskStats {
		deviceName := stats.DeviceName

		if ignoreDisk(deviceName) {
			fmt.Println("\t ** DiskStats -- Ignoring mount ", deviceName)
			continue
		}

		l_stats_info := make(map[string]float64)
		l_udev_info := make(map[string]string)

		ds := DiskStats{l_stats_info, l_udev_info}
		deviceStats[deviceName] = ds

		l_stats_info["reads_completed_total"] = float64(stats.ReadIOs)
		l_stats_info["reads_merged_total"] = float64(stats.ReadMerges)
		l_stats_info["read_bytes_total"] = float64(stats.ReadSectors) * unixSectorSize
		l_stats_info["read_time_seconds_total"] = float64(stats.ReadTicks) * secondsPerTick
		l_stats_info["writes_completed_total"] = float64(stats.WriteIOs)
		l_stats_info["writes_merged_total"] = float64(stats.WriteMerges)
		l_stats_info["writes_bytes_total"] = float64(stats.WriteSectors) * unixSectorSize
		l_stats_info["write_time_seconds_total"] = float64(stats.WriteTicks) * secondsPerTick
		l_stats_info["io_now"] = float64(stats.IOsInProgress)
		l_stats_info["io_time_seconds_total"] = float64(stats.IOsTotalTicks) * secondsPerTick
		l_stats_info["io_time_weighted_seconds_total"] = float64(stats.WeightedIOTicks) * secondsPerTick
		l_stats_info["discards_completed_total"] = float64(stats.DiscardIOs)
		l_stats_info["discards_merged_total"] = float64(stats.DiscardMerges)
		l_stats_info["discarded_sectors_total"] = float64(stats.DiscardSectors)
		l_stats_info["discard_time_seconds_total"] = float64(stats.DiscardTicks) * secondsPerTick
		l_stats_info["flush_requests_total"] = float64(stats.FlushRequestsCompleted)
		l_stats_info["flush_requests_time_seconds_total"] = float64(stats.TimeSpentFlushing) * secondsPerTick
		l_stats_info["flush_requests_time_seconds_total"] = float64(stats.TimeSpentFlushing) * secondsPerTick
		// l_stats_info["Major_Number"] = float64(stats.MajorNumber)
		// l_stats_info["Minor_Number"] = float64(stats.MinorNumber)

		udevDeviceProps, err := getUdevDeviceProperties(stats.MajorNumber, stats.MinorNumber)

		if err != nil {
			fmt.Println("\t\t *** msg", "Failed to parse udev info", "err", err)
			log.Debug("msg", "Failed to parse udev info", "err", err)
		}

		for k, v := range udevDeviceProps {
			fmt.Println("info k: ", k, "\t v: ", v)
			l_udev_info[k] = v
		}
		// This is usually the serial printed on the disk label.
		serial := udevDeviceProps[udevSCSIIdentSerial]

		// If it's undefined, fallback to ID_SERIAL_SHORT instead.
		if serial == "" {
			serial = udevDeviceProps[udevIDSerialShort]
		}
		l_udev_info["serial"] = serial

		// create Stat objects now
		l_sysinfo_stats := constructDiskinfoStats(deviceName, l_stats_info)

		// add to return array
		arrSysInfoStats = append(arrSysInfoStats, l_sysinfo_stats...)

		// add disk-info
		statDiskInfo := constructDiskInfo(deviceName, fmt.Sprint(stats.MajorNumber), fmt.Sprint(stats.MinorNumber), serial)
		arrSysInfoStats = append(arrSysInfoStats, statDiskInfo)

		// 	desc: prometheus.NewDesc(prometheus.BuildFQName(namespace, diskSubsystem, "info"),
		// 	"Info of /sys/block/<block_device>.",
		// 	[]string{"device", "major", "minor", "path", "wwn", "model", "serial", "revision"},
		// 	nil,
		// ), valueType: prometheus.GaugeValue,

		// fmt.Sprint(stats.MajorNumber),
		// fmt.Sprint(stats.MinorNumber),
		// info[udevIDPath],
		// info[udevIDWWN],
		// info[udevIDModel],
		// serial,
		// info[udevIDRevision],

	}

	return arrSysInfoStats
}

func constructDiskInfo(deviceName string, major string, minor string, serial string) SystemInfoStat {
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

func constructDiskinfoStats(deviceName string, v_stats_info map[string]float64) []SystemInfoStat {
	arrSysInfoStats := []SystemInfoStat{}

	clusterName := statprocessors.ClusterName
	service := statprocessors.Service

	for sk, sv := range v_stats_info {
		labels := []string{commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE, commons.METRIC_LABEL_DEVICE}
		labelValues := []string{clusterName, service, deviceName}

		l_metricName := strings.ToLower(sk)
		sysMetric := NewSystemInfoStat(commons.CTX_DISK_STATS, l_metricName)

		sysMetric.Labels = labels
		sysMetric.LabelValues = labelValues
		sysMetric.Value = sv

		arrSysInfoStats = append(arrSysInfoStats, sysMetric)
	}

	return arrSysInfoStats
}
