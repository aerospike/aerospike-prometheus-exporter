package dataprovider

import (
	"fmt"

	"github.com/prometheus/procfs"
	"github.com/prometheus/procfs/blockdevice"
	log "github.com/sirupsen/logrus"
)

func (sip SystemInfoProvider) GetCPUDetails() ([]map[string]float64, []map[string]float64) {

	arrGuestCpuStats := []map[string]float64{}
	arrCpuStats := []map[string]float64{}

	fs, err := procfs.NewFS(PROC_PATH)
	if err != nil {
		log.Debug("parseCpuStats Error while reading CPU Stats from ", PROC_PATH, " Error ", err)
		return arrGuestCpuStats, arrCpuStats
	}

	stats, err := fs.Stat()
	if err != nil {
		log.Debug("Eror while reading procfs.NewFS system, error: ", err)
		return arrGuestCpuStats, arrCpuStats
	}

	for index, cpu := range stats.CPU {
		// fmt.Println("parsing CPU stats ", index)
		guestCpuValues := make(map[string]float64)
		guestCpuValues["index"] = float64(index)
		guestCpuValues["user"] = cpu.Guest
		guestCpuValues["nice"] = cpu.GuestNice

		cpuValues := make(map[string]float64)
		cpuValues["index"] = float64(index)
		cpuValues["user"] = cpu.Guest
		cpuValues["idle"] = cpu.Idle
		cpuValues["irq"] = cpu.IRQ
		cpuValues["iowait"] = cpu.Iowait
		cpuValues["nice"] = cpu.Nice
		cpuValues["soft_irq"] = cpu.SoftIRQ
		cpuValues["steal"] = cpu.Steal
		cpuValues["system"] = cpu.System
		cpuValues["user"] = cpu.User

		arrGuestCpuStats = append(arrGuestCpuStats, guestCpuValues)
		arrCpuStats = append(arrCpuStats, cpuValues)

	}

	return arrGuestCpuStats, arrCpuStats
}

func (sip SystemInfoProvider) GetDiskStats() []map[string]string {

	arrDiskStats := []map[string]string{}

	fs, err := blockdevice.NewFS(PROC_PATH, SYS_PATH)
	if err != nil {
		return arrDiskStats
	}

	procDiskStats, err := fs.ProcDiskstats()
	if err != nil {
		return arrDiskStats
	}

	for index, stats := range procDiskStats {

		deviceName := stats.DeviceName

		if ignoreDisk(deviceName) {
			log.Debug("\t ** DiskStats -- Ignoring mount ", deviceName)
			continue
		}

		diskStat := make(map[string]string)

		diskStat["index"] = fmt.Sprint(index)
		diskStat["device_name"] = deviceName
		diskStat["reads_completed_total"] = fmt.Sprint(stats.ReadIOs)
		diskStat["reads_merged_total"] = fmt.Sprint(stats.ReadMerges)
		diskStat["read_bytes_total"] = fmt.Sprint(float64(stats.ReadSectors) * DISK_SECTOR_SIZE_IN_UNIX)
		diskStat["read_time_seconds_total"] = fmt.Sprint(float64(stats.ReadTicks) * SECONDS_PER_TICK)
		diskStat["writes_completed_total"] = fmt.Sprint(stats.WriteIOs)
		diskStat["writes_merged_total"] = fmt.Sprint(stats.WriteMerges)
		diskStat["writes_bytes_total"] = fmt.Sprint(float64(stats.WriteSectors) * DISK_SECTOR_SIZE_IN_UNIX)
		diskStat["write_time_seconds_total"] = fmt.Sprint(float64(stats.WriteTicks) * SECONDS_PER_TICK)
		diskStat["io_now"] = fmt.Sprint(stats.IOsInProgress)
		diskStat["io_time_seconds_total"] = fmt.Sprint(float64(stats.IOsTotalTicks) * SECONDS_PER_TICK)
		diskStat["io_time_weighted_seconds_total"] = fmt.Sprint(float64(stats.WeightedIOTicks) * SECONDS_PER_TICK)
		diskStat["discards_completed_total"] = fmt.Sprint(stats.DiscardIOs)
		diskStat["discards_merged_total"] = fmt.Sprint(stats.DiscardMerges)
		diskStat["discarded_sectors_total"] = fmt.Sprint(stats.DiscardSectors)
		diskStat["discard_time_seconds_total"] = fmt.Sprint(float64(stats.DiscardTicks) * SECONDS_PER_TICK)
		diskStat["flush_requests_total"] = fmt.Sprint(stats.FlushRequestsCompleted)
		diskStat["flush_requests_time_seconds_total"] = fmt.Sprint(float64(stats.TimeSpentFlushing) * SECONDS_PER_TICK)
		diskStat["flush_requests_time_seconds_total"] = fmt.Sprint(float64(stats.TimeSpentFlushing) * SECONDS_PER_TICK)
		diskStat["major_number"] = fmt.Sprint(stats.MajorNumber)
		diskStat["minor_number"] = fmt.Sprint(stats.MinorNumber)

		udevDeviceProps, err := getUdevDeviceProperties(stats.MajorNumber, stats.MinorNumber)

		if err != nil {
			log.Debug("msg", "Failed to parse udev info", "err", err)
		}

		// a serial number printed on the disk label.
		serial, ok := udevDeviceProps[UDEV_SCSI_IDENT_SERIAL]

		// If it's undefined, fallback to ID_SERIAL_SHORT instead.
		if !ok || len(serial) == 0 {
			serial = udevDeviceProps[UDEV_ID_SERIAL_SHORT]
		}
		diskStat["major_number"] = fmt.Sprint(stats.MajorNumber)
		diskStat["minor_number"] = fmt.Sprint(stats.MinorNumber)
		diskStat["serial"] = serial

		arrDiskStats = append(arrDiskStats, diskStat)
	}

	fmt.Println("\n\n\t*****DiskStats - Count of return status ", len(arrDiskStats))
	log.Debug("DiskStats - Count of return status ", len(arrDiskStats))
	return arrDiskStats
}
