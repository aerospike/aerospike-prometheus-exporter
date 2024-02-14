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

func (sip SystemInfoProvider) GetCPUDetails() map[string]string {

	// arrGuestCpuStats := []map[string]string{}
	cpuStats := map[string]string{}

	fs, err := procfs.NewFS(PROC_PATH)
	if err != nil {
		log.Debug("parseCpuStats Error while reading CPU Stats from ", PROC_PATH, " Error ", err)
		return cpuStats
	}

	stats, err := fs.Stat()
	if err != nil {
		log.Debug("Eror while reading procfs.NewFS system, error: ", err)
		return cpuStats
	}

	idleCpuTotal := stats.CPUTotal.Idle
	nonIdleCpuTotals := stats.CPUTotal.IRQ + stats.CPUTotal.Iowait + stats.CPUTotal.Nice + stats.CPUTotal.SoftIRQ
	nonIdleCpuTotals = nonIdleCpuTotals + stats.CPUTotal.Steal + stats.CPUTotal.System + stats.CPUTotal.User

	cpuStats["non_idle"] = fmt.Sprint(nonIdleCpuTotals)
	cpuStats["idle"] = fmt.Sprint(idleCpuTotal)

	log.Debug("non_idle ", nonIdleCpuTotals)
	log.Debug("idle ", idleCpuTotal)
	return cpuStats
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

func (sip SystemInfoProvider) GetFileFD() map[string]string {
	fileFDStats := make(map[string]string)

	fileName := getProcFilePath("sys/fs/file-nr")

	file, err := os.Open(fileName)
	if err != nil {
		log.Error("Error while opening file,", fileName, " Error: ", err)
		return fileFDStats
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		//
		values := strings.Split(scanner.Text(), "\t")

		fileFDStats["allocated"] = values[0]

	}

	log.Debug("FileFD Stats - Count of return stats ", len(fileFDStats))

	return fileFDStats
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

	log.Debug("FileSystem Stats - Count of return stats ", len(arrFileSystemMountStats))
	return arrFileSystemMountStats
}

func (sip SystemInfoProvider) GetMemInfoStats() map[string]string {
	memStats := make(map[string]string)

	fs, err := procfs.NewFS(PROC_PATH)

	if err != nil {
		log.Debug("Eror while reading procfs.NewFS system,  error: ", err)
		return memStats
	}

	meminfo, err := fs.Meminfo()
	if err != nil {
		log.Debug("Eror while reading MemInfo, error: ", err)
		return memStats
	}

	// All values are in KB, convert to bytes
	memStats["Shmem"] = fmt.Sprint(getFloatValue(meminfo.Shmem) * ONE_KILO_BYTE)
	memStats["Swap_Cached"] = fmt.Sprint(getFloatValue(meminfo.SwapCached) * ONE_KILO_BYTE)

	log.Debug("MemInfo Stats - Count of return stats ", memStats)
	return memStats
}

func (sip SystemInfoProvider) GetNetStatInfo() ([]map[string]string, []map[string]string, []map[string]string) {
	arrNetStats := []map[string]string{}
	arrSnmpStats := []map[string]string{}
	arrSnmp6Stats := []map[string]string{}

	arrNetStats = append(arrNetStats, parseNetStats(getProcFilePath("net/netstat")))
	// arrSnmpStats = append(arrSnmpStats, parseNetStats(getProcFilePath("net/snmp")))
	// arrSnmp6Stats = append(arrSnmp6Stats, parseSNMP6Stats(getProcFilePath("net/snmp6")))

	log.Debug("NetStatsInfo Net - Count of return stats ", len(arrNetStats))
	log.Debug("NetStatsInfo SNMP - Count of return stats ", len(arrSnmpStats))
	log.Debug("NetStatsInfo SNMP-6 - Count of return stats ", len(arrSnmp6Stats))
	return arrNetStats, arrSnmpStats, arrSnmp6Stats
}

func (sip SystemInfoProvider) GetNetDevStats() ([]map[string]string, []map[string]string, []map[string]string) {
	arrNetGroupStats := []map[string]string{}
	arrNetReceiveStats := []map[string]string{}
	arrNetTransferStats := []map[string]string{}

	fs, err := procfs.NewFS(PROC_PATH)
	if err != nil {
		log.Debug("parseNetworkStats Error while reading Net_Dev Stats from ", PROC_PATH, " Error ", err)
		return arrNetGroupStats, arrNetReceiveStats, arrNetTransferStats
	}

	stats, err := fs.NetDev()
	if err != nil {
		log.Debug("Eror while reading procfs.NewFS system, error: ", err)
		return arrNetGroupStats, arrNetReceiveStats, arrNetTransferStats
	}

	for deviceName, stats := range stats {

		groupStats := make(map[string]string)
		receiveStats := make(map[string]string)
		transferStats := make(map[string]string)

		// network group
		groupStats["device_name"] = deviceName
		groupStats[deviceName] = fmt.Sprint(0)

		// network receive
		receiveStats["device_name"] = deviceName
		receiveStats["receive_bytes_total"] = fmt.Sprint(float64(stats.RxBytes))
		receiveStats["receive_compressed_total"] = fmt.Sprint(float64(stats.RxCompressed))
		receiveStats["receive_dropped_total"] = fmt.Sprint(float64(stats.RxDropped))
		receiveStats["receive_errors_total"] = fmt.Sprint(float64(stats.RxErrors))
		receiveStats["receive_fifo_total"] = fmt.Sprint(float64(stats.RxFIFO))
		receiveStats["receive_frame_total"] = fmt.Sprint(float64(stats.RxFrame))
		receiveStats["receive_multicast_total"] = fmt.Sprint(float64(stats.RxMulticast))
		receiveStats["receive_packets_total"] = fmt.Sprint(float64(stats.RxPackets))

		// network transfer
		transferStats["device_name"] = deviceName
		transferStats["transfer_bytes_total"] = fmt.Sprint(float64(stats.TxBytes))
		transferStats["transfer_carrier_total"] = fmt.Sprint(float64(stats.TxCarrier))
		transferStats["transfer_collisions_total"] = fmt.Sprint(float64(stats.TxCollisions))
		transferStats["transfer_compressed_total"] = fmt.Sprint(float64(stats.TxCompressed))
		transferStats["transfer_errors_total"] = fmt.Sprint(float64(stats.TxErrors))
		transferStats["transfer_fifo_total"] = fmt.Sprint(float64(stats.TxFIFO))
		transferStats["transfer_packets_total"] = fmt.Sprint(float64(stats.TxPackets))

		arrNetGroupStats = append(arrNetGroupStats, groupStats)
		arrNetReceiveStats = append(arrNetReceiveStats, receiveStats)
		arrNetTransferStats = append(arrNetTransferStats, transferStats)
	}

	log.Debug("NetDevStats - GROUP Count of return status ", len(arrNetGroupStats))
	log.Debug("NetDevStats - RECEIVE Count of return status ", len(arrNetReceiveStats))
	log.Debug("NetDevStats - TRANSFER Count of return status ", len(arrNetTransferStats))
	return arrNetGroupStats, arrNetReceiveStats, arrNetTransferStats
}

func (sip SystemInfoProvider) GetVmStats() []map[string]string {
	arrSysInfoStats := []map[string]string{}

	file, err := os.Open(getProcFilePath("vmstat"))
	if err != nil {
		log.Error("Error while opening file, 'vmstat' Error: ", err)
		return arrSysInfoStats
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	vmStats := make(map[string]string)

	for scanner.Scan() {
		statElements := strings.Split(scanner.Text(), " ")

		if acceptVmstat(statElements[0]) {
			vmStats[statElements[0]] = statElements[1]
		}
	}

	arrSysInfoStats = append(arrSysInfoStats, vmStats)

	log.Debug("GetVmStats - VMSTATS Count of return status ", len(arrSysInfoStats))
	return arrSysInfoStats
}
