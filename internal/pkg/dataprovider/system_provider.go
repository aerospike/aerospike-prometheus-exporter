package dataprovider

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/prometheus/procfs"
	"github.com/prometheus/procfs/blockdevice"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
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

	log.Debug("GuestCPU Stats - Count of return stats ", len(arrGuestCpuStats))
	log.Debug("CPU Stats - Count of return stats ", len(arrCpuStats))
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
		if len(serial) == 0 {
			serial = "null"
		}

		diskStat["major_number"] = fmt.Sprint(stats.MajorNumber)
		diskStat["minor_number"] = fmt.Sprint(stats.MinorNumber)
		diskStat["serial"] = serial

		arrDiskStats = append(arrDiskStats, diskStat)
	}

	log.Debug("DiskStats - Count of return status ", len(arrDiskStats))
	return arrDiskStats
}

func (sip SystemInfoProvider) GetFileFD() []map[string]string {
	arrFileFdInfoStats := []map[string]string{}

	fileName := GetProcFilePath("sys/fs/file-nr")

	file, err := os.Open(fileName)
	if err != nil {
		log.Error("Error while opening file,", fileName, " Error: ", err)
		return arrFileFdInfoStats
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	index := 0
	for scanner.Scan() {
		//
		values := strings.Split(scanner.Text(), "\t")

		fileFDStats := make(map[string]string)
		fileFDStats["index"] = fmt.Sprint(index)
		fileFDStats["allocated"] = values[0]
		fileFDStats["maximum"] = values[2]

		index++
		arrFileFdInfoStats = append(arrFileFdInfoStats, fileFDStats)
	}

	return arrFileFdInfoStats
}

func (sip SystemInfoProvider) GetFileSystemStats() []map[string]string {

	arrFileSystemMountStats := []map[string]string{}

	mnts, err := procfs.GetMounts()
	if err != nil {
		return arrFileSystemMountStats
	}

	for _, mnt := range mnts {

		if ignoreFileSystem(mnt.Source) {
			log.Debug("\t ** FileSystem Stats -- Ignoring mount ", mnt.Source)
			continue
		}

		// local variables
		isreadonly := 0.0
		size, free, avail, files, filesFree, isError := 0.0, 0.0, 0.0, 0.0, 0.0, false
		mountStats := make(map[string]string)

		// if the disk is read only
		_, roKeyFound := mnt.OptionalFields["ro"]
		if roKeyFound {
			isreadonly = 1.0
		}

		size, free, avail, files, filesFree, isError = readDiskMountData(mnt.Source)
		if isError {
			log.Debug("Skipping, error during reading stats of mount-point ", mnt.MountPoint, " and mount-source ", mnt.Source)
			continue
		}

		mountStats["fstype"] = mnt.FSType
		mountStats["mount_point"] = mnt.MountPoint
		mountStats["source"] = mnt.Source
		mountStats["is_read_only"] = fmt.Sprint(isreadonly)

		mountStats["size_bytes"] = fmt.Sprint(size)
		mountStats["free_bytes"] = fmt.Sprint(free)
		mountStats["avail_byts"] = fmt.Sprint(avail)
		mountStats["files"] = fmt.Sprint(files)
		mountStats["files_free"] = fmt.Sprint(filesFree)

		arrFileSystemMountStats = append(arrFileSystemMountStats, mountStats)
	}

	return arrFileSystemMountStats
}

func readDiskMountData(mntpointsource string) (float64, float64, float64, float64, float64, bool) {
	buf := new(unix.Statfs_t)
	err := unix.Statfs(GetRootfsFilePath(mntpointsource), buf)
	// any kind of error
	if err != nil {
		log.Error("Error while fetching FileSystem stats for mount ", mntpointsource, ", hence, return all 0.0 --> error is ", err)
		return 0.0, 0.0, 0.0, 0.0, 0.0, true
	}

	size := float64(buf.Blocks) * float64(buf.Bsize)
	free := float64(buf.Bfree) * float64(buf.Bsize)
	avail := float64(buf.Bavail) * float64(buf.Bsize)
	files := float64(buf.Files)
	filesFree := float64(buf.Ffree)

	return size, free, avail, files, filesFree, false
}
