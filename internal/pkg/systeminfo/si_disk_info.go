package systeminfo

import (
	"fmt"

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

// func GetDiskStats() ([]*iostat.DriveStats, error) {
func GetDiskStats() map[string]DiskStats {
	// acceptPattern = regexp.MustCompile(acceptPattern)

	device_stats := parseDiskStats()

	// for k, v := range device_stats {
	// 	fmt.Println("\t *** Device name is ", k, " and \n\t *** v.stats_info ", v.stats_info)
	// }

	return device_stats
}

func parseDiskStats() map[string]DiskStats {
	deviceStats := make(map[string]DiskStats)

	var diskLabelNames = []string{"device"}
	fmt.Println(" diskLabelNames --> ", diskLabelNames)

	fs, err := blockdevice.NewFS(PROC_PATH, SYS_PATH)
	if err != nil {
		return deviceStats
	}
	diskStats, err := fs.ProcDiskstats()
	if err != nil {
		return deviceStats
	}

	for _, stats := range diskStats {
		dev := stats.DeviceName

		if ignoreDisk(dev) {
			fmt.Println("\t ** DiskStats -- Ignoring mount ", dev)
			continue
		}

		l_stats_info := make(map[string]float64)
		l_udev_info := make(map[string]string)

		ds := DiskStats{l_stats_info, l_udev_info}
		deviceStats[dev] = ds

		// l_stats_info["Read_IOs"] = float64(stats.ReadIOs)
		// l_stats_info["Read_Merges"] = float64(stats.ReadMerges)
		// l_stats_info["Read_Sectors"] = float64(stats.ReadSectors) * unixSectorSize
		// l_stats_info["Read_Ticks"] = float64(stats.ReadTicks) * secondsPerTick
		// l_stats_info["Write_IOs"] = float64(stats.WriteIOs)
		// l_stats_info["Write_Merges"] = float64(stats.WriteMerges)
		// l_stats_info["Write_Sectors"] = float64(stats.WriteSectors) * unixSectorSize
		// l_stats_info["Write_Ticks"] = float64(stats.WriteTicks) * secondsPerTick
		// l_stats_info["IOs_InProgress"] = float64(stats.IOsInProgress)
		// l_stats_info["IOs_Total_Ticks"] = float64(stats.IOsTotalTicks) * secondsPerTick
		// l_stats_info["Weighted_IO_Ticks"] = float64(stats.WeightedIOTicks) * secondsPerTick
		// l_stats_info["Discard_IOs"] = float64(stats.DiscardIOs)
		// l_stats_info["Discard_Merges"] = float64(stats.DiscardMerges)
		// l_stats_info["Discard_Sectors"] = float64(stats.DiscardSectors)
		// l_stats_info["Discard_Ticks"] = float64(stats.DiscardTicks) * secondsPerTick
		// l_stats_info["Flush_Requests_Completed"] = float64(stats.FlushRequestsCompleted)
		// l_stats_info["Time_Spent_Flushing"] = float64(stats.TimeSpentFlushing) * secondsPerTick
		// l_stats_info["Major_Number"] = float64(stats.MajorNumber)
		// l_stats_info["Minor_Number"] = float64(stats.MinorNumber)

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
		// l_stats_info["Major_Number"] = float64(stats.MajorNumber)
		// l_stats_info["Minor_Number"] = float64(stats.MinorNumber)

		info, err := getUdevDeviceProperties(stats.MajorNumber, stats.MinorNumber)

		if err != nil {
			fmt.Println("\t\t *** msg", "Failed to parse udev info", "err", err)
			log.Debug("msg", "Failed to parse udev info", "err", err)
		}

		for k, v := range info {
			fmt.Println("info k: ", k, "\t v: ", v)
			l_udev_info[k] = v
		}
		// This is usually the serial printed on the disk label.
		serial := info[udevSCSIIdentSerial]

		// If it's undefined, fallback to ID_SERIAL_SHORT instead.
		if serial == "" {
			serial = info[udevIDSerialShort]
		}
		l_udev_info["serial"] = serial
	}

	return deviceStats
}
