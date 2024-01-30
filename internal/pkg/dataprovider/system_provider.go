package dataprovider

import (
	"bufio"
	"fmt"
	"os"
	"strings"

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

func (sip SystemInfoProvider) GetMemInfoStats() []map[string]string {
	arrMemInfoStats := []map[string]string{}

	fs, err := procfs.NewFS(PROC_PATH)

	if err != nil {
		log.Debug("Eror while reading procfs.NewFS system,  error: ", err)
		return arrMemInfoStats
	}

	meminfo, err := fs.Meminfo()
	if err != nil {
		log.Debug("Eror while reading MemInfo, error: ", err)
		return arrMemInfoStats
	}

	memStats := make(map[string]string)

	// All values are in KB, convert to bytes
	memStats["Active"] = fmt.Sprint(GetFloatValue(meminfo.Active) * ONE_KILO_BYTE)
	memStats["Active_Anon"] = fmt.Sprint(GetFloatValue(meminfo.ActiveAnon) * ONE_KILO_BYTE)
	memStats["Active_File"] = fmt.Sprint(GetFloatValue(meminfo.ActiveFile) * ONE_KILO_BYTE)
	memStats["Anon_Pages"] = fmt.Sprint(GetFloatValue(meminfo.AnonPages) * ONE_KILO_BYTE)
	memStats["Anon_Huge_Pages"] = fmt.Sprint(GetFloatValue(meminfo.AnonHugePages) * ONE_KILO_BYTE)
	memStats["Bounce"] = fmt.Sprint(GetFloatValue(meminfo.Bounce) * ONE_KILO_BYTE)
	memStats["Buffers"] = fmt.Sprint(GetFloatValue(meminfo.Buffers) * ONE_KILO_BYTE)
	memStats["Cached"] = fmt.Sprint(GetFloatValue(meminfo.Cached) * ONE_KILO_BYTE)
	memStats["CmaFree"] = fmt.Sprint(GetFloatValue(meminfo.CmaFree) * ONE_KILO_BYTE)
	memStats["CmaTotal"] = fmt.Sprint(GetFloatValue(meminfo.CmaTotal) * ONE_KILO_BYTE)
	memStats["Commit_Limit"] = fmt.Sprint(GetFloatValue(meminfo.CommitLimit) * ONE_KILO_BYTE)
	memStats["Committed_AS"] = fmt.Sprint(GetFloatValue(meminfo.CommittedAS) * ONE_KILO_BYTE)
	memStats["Direct_Map1G"] = fmt.Sprint(GetFloatValue(meminfo.DirectMap1G) * ONE_KILO_BYTE)
	memStats["Direct_Map2M"] = fmt.Sprint(GetFloatValue(meminfo.DirectMap2M) * ONE_KILO_BYTE)
	memStats["Direct_Map4k"] = fmt.Sprint(GetFloatValue(meminfo.DirectMap4k) * ONE_KILO_BYTE)
	memStats["Dirty"] = fmt.Sprint(GetFloatValue(meminfo.Dirty) * ONE_KILO_BYTE)
	memStats["Hardware_Corrupted"] = fmt.Sprint(GetFloatValue(meminfo.HardwareCorrupted) * ONE_KILO_BYTE)
	memStats["Huge_Pages_Free"] = fmt.Sprint(GetFloatValue(meminfo.HugePagesFree) * ONE_KILO_BYTE)
	memStats["Huge_Pages_Rsvd"] = fmt.Sprint(GetFloatValue(meminfo.HugePagesRsvd) * ONE_KILO_BYTE)
	memStats["Huge_Pages_Surp"] = fmt.Sprint(GetFloatValue(meminfo.HugePagesSurp) * ONE_KILO_BYTE)
	memStats["Huge_Pages_Total"] = fmt.Sprint(GetFloatValue(meminfo.HugePagesTotal) * ONE_KILO_BYTE)
	memStats["Huge_page_size"] = fmt.Sprint(GetFloatValue(meminfo.Hugepagesize) * ONE_KILO_BYTE)
	memStats["Inactive"] = fmt.Sprint(GetFloatValue(meminfo.Inactive) * ONE_KILO_BYTE)
	memStats["Inactive_Anon"] = fmt.Sprint(GetFloatValue(meminfo.InactiveAnon) * ONE_KILO_BYTE)
	memStats["Inactive_File"] = fmt.Sprint(GetFloatValue(meminfo.InactiveFile) * ONE_KILO_BYTE)
	memStats["Kernel_Stack"] = fmt.Sprint(GetFloatValue(meminfo.KernelStack) * ONE_KILO_BYTE)
	memStats["Mapped"] = fmt.Sprint(GetFloatValue(meminfo.Mapped) * ONE_KILO_BYTE)
	memStats["Mem_Available"] = fmt.Sprint(GetFloatValue(meminfo.MemAvailable) * ONE_KILO_BYTE)
	memStats["Mem_Free"] = fmt.Sprint(GetFloatValue(meminfo.MemFree) * ONE_KILO_BYTE)
	memStats["Mem_Total"] = fmt.Sprint(GetFloatValue(meminfo.MemTotal) * ONE_KILO_BYTE)
	memStats["Mlocked"] = fmt.Sprint(GetFloatValue(meminfo.Mlocked) * ONE_KILO_BYTE)
	memStats["NFS_Unstable"] = fmt.Sprint(GetFloatValue(meminfo.NFSUnstable) * ONE_KILO_BYTE)
	memStats["Page_Tables"] = fmt.Sprint(GetFloatValue(meminfo.PageTables) * ONE_KILO_BYTE)
	memStats["SReclaimable"] = fmt.Sprint(GetFloatValue(meminfo.SReclaimable) * ONE_KILO_BYTE)
	memStats["SUnreclaim"] = fmt.Sprint(GetFloatValue(meminfo.SUnreclaim) * ONE_KILO_BYTE)
	memStats["Shmem"] = fmt.Sprint(GetFloatValue(meminfo.Shmem) * ONE_KILO_BYTE)
	memStats["Shmem_Huge_Pages"] = fmt.Sprint(GetFloatValue(meminfo.ShmemHugePages) * ONE_KILO_BYTE)
	memStats["Shmem_Pmd_Mapped"] = fmt.Sprint(GetFloatValue(meminfo.ShmemPmdMapped) * ONE_KILO_BYTE)
	memStats["Slab"] = fmt.Sprint(GetFloatValue(meminfo.Slab) * ONE_KILO_BYTE)
	memStats["Swap_Cached"] = fmt.Sprint(GetFloatValue(meminfo.SwapCached) * ONE_KILO_BYTE)
	memStats["Swap_Free"] = fmt.Sprint(GetFloatValue(meminfo.SwapFree) * ONE_KILO_BYTE)
	memStats["Swap_Total"] = fmt.Sprint(GetFloatValue(meminfo.SwapTotal) * ONE_KILO_BYTE)
	memStats["Unevictable"] = fmt.Sprint(GetFloatValue(meminfo.Unevictable) * ONE_KILO_BYTE)
	memStats["Vmalloc_Chunk"] = fmt.Sprint(GetFloatValue(meminfo.VmallocChunk) * ONE_KILO_BYTE)
	memStats["Vmalloc_Total"] = fmt.Sprint(GetFloatValue(meminfo.VmallocTotal) * ONE_KILO_BYTE)
	memStats["Vmalloc_Used"] = fmt.Sprint(GetFloatValue(meminfo.VmallocUsed) * ONE_KILO_BYTE)
	memStats["Writeback"] = fmt.Sprint(GetFloatValue(meminfo.Writeback) * ONE_KILO_BYTE)
	memStats["WritebackTmp"] = fmt.Sprint(GetFloatValue(meminfo.WritebackTmp) * ONE_KILO_BYTE)

	return arrMemInfoStats
}
