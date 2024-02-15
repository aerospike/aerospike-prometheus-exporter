package statprocessors

import (
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/dataprovider"
)

func RefreshSystemInfo() ([]AerospikeStat, error) {
	arrSysInfoStats := []AerospikeStat{}

	arrSysInfoStats = append(arrSysInfoStats, getCpuInfo()...)
	arrSysInfoStats = append(arrSysInfoStats, getFDInfo()...)
	arrSysInfoStats = append(arrSysInfoStats, getMemInfo()...)
	arrSysInfoStats = append(arrSysInfoStats, getNetStatInfo()...)
	arrSysInfoStats = append(arrSysInfoStats, getNetworkInfo()...)

	// TODO: Review and remove after finalizing the metrics to share
	arrSysInfoStats = append(arrSysInfoStats, getDiskInfo()...)
	arrSysInfoStats = append(arrSysInfoStats, getFileSystemInfo()...)
	arrSysInfoStats = append(arrSysInfoStats, getVmStatsInfo()...)

	return arrSysInfoStats, nil
}

func getCpuInfo() []AerospikeStat {
	arrSysInfoStats := []AerospikeStat{}
	cpuDetails := dataprovider.GetSystemProvider().GetCPUDetails()

	cpuStatLabels := []string{commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE}
	cpuStatLabels = append(cpuStatLabels, commons.METRIC_LABEL_CPU_MODE)

	clusterName := ClusterName
	service := Service

	// cpuStats["non_idle"] = fmt.Sprint(nonIdleCpuTotals)
	// cpuStats["idle"] = fmt.Sprint(idleCpuTotal)

	nonIdleTotal, err := commons.TryConvert(cpuDetails["non_idle"])
	if err != nil {
		log.Error("Error while converting value of float non_idle, and converted value is ", cpuDetails["non_idle"])
		return arrSysInfoStats
	}
	idleTotal, err := commons.TryConvert(cpuDetails["idle"])
	if err != nil {
		log.Error("Error while converting value of float idle, and converted value is ", cpuDetails["idle"])
		return arrSysInfoStats
	}

	if idleTotal <= 0 {
		return arrSysInfoStats
	}

	// calculate CPU utilization
	cpuUtilization := (nonIdleTotal / (nonIdleTotal + idleTotal)) * 100
	labelValues := []string{clusterName, service}

	allowed := isMetricAllowed(commons.CTX_SYSINFO_CPU_STATS, "cpu_utilzation")
	sysMetric := NewAerospikeStat(commons.CTX_SYSINFO_CPU_STATS, "cpu_utilzation", allowed)
	sysMetric.Labels = cpuStatLabels
	sysMetric.LabelValues = labelValues
	sysMetric.Value = cpuUtilization

	arrSysInfoStats = append(arrSysInfoStats, sysMetric)

	for statName, statValue := range cpuDetails {
		labelValues := []string{clusterName, service, statName}
		value, err := commons.TryConvert(statValue)
		if err != nil {
			log.Error("Error while converting value of stat: ", statName, " and converted value is ", statValue)
			continue
		}

		allowed := isMetricAllowed(commons.CTX_SYSINFO_CPU_STATS, "cpu_seconds_total")
		sysMetric := NewAerospikeStat(commons.CTX_SYSINFO_CPU_STATS, "cpu_seconds_total", allowed)
		sysMetric.Labels = cpuStatLabels
		sysMetric.LabelValues = labelValues
		sysMetric.Value = value

		arrSysInfoStats = append(arrSysInfoStats, sysMetric)

	}

	return arrSysInfoStats
}

func getFDInfo() []AerospikeStat {
	arrSysInfoStats := []AerospikeStat{}
	clusterName := ClusterName
	service := Service

	labelValues := []string{clusterName, service}
	fileStatLabels := []string{commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE}

	fileFDStats := dataprovider.GetSystemProvider().GetFileFD()

	statName := "allocated"
	statValue := fileFDStats[statName]
	value, err := commons.TryConvert(statValue)
	if err != nil {
		log.Error("Error while converting value of stat: ", statName, " and converted value is ", statValue)
		return arrSysInfoStats
	}

	allowed := isMetricAllowed(commons.CTX_SYSINFO_FILEFD_STATS, statName)
	sysMetric := NewAerospikeStat(commons.CTX_SYSINFO_FILEFD_STATS, statName, allowed)
	sysMetric.Labels = fileStatLabels
	sysMetric.LabelValues = labelValues
	sysMetric.Value = value

	arrSysInfoStats = append(arrSysInfoStats, sysMetric)

	return arrSysInfoStats
}

func getMemInfo() []AerospikeStat {
	arrSysInfoStats := []AerospikeStat{}

	memStats := dataprovider.GetSystemProvider().GetMemInfoStats()

	memInfoLabels := []string{commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE}
	for statName, statValue := range memStats {
		clusterName := ClusterName
		service := Service
		value, err := commons.TryConvert(statValue)
		if err != nil {
			log.Error("Error while converting value of stat: ", statName, " and converted value is ", statValue)
			continue
		}

		labelValues := []string{clusterName, service}

		metricName := strings.ToLower(statName) + "_bytes"
		allowed := isMetricAllowed(commons.CTX_SYSINFO_MEMORY_STATS, statName)
		sysMetric := NewAerospikeStat(commons.CTX_SYSINFO_MEMORY_STATS, metricName, allowed)
		sysMetric.Labels = memInfoLabels
		sysMetric.LabelValues = labelValues
		sysMetric.Value = value

		arrSysInfoStats = append(arrSysInfoStats, sysMetric)
	}

	return arrSysInfoStats
}

func getNetStatInfo() []AerospikeStat {
	arrSysInfoStats := []AerospikeStat{}

	netStatInfoLabels := []string{}
	netStatInfoLabels = append(netStatInfoLabels, commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE)

	snmpStats := dataprovider.GetSystemProvider().GetNetStatInfo()

	//Net SNMP - includes TCP metrics like active_conn, established, retransmit etc.,
	clusterName := ClusterName
	service := Service
	labelValues := []string{clusterName, service}

	for statName, statValue := range snmpStats {

		value, err := commons.TryConvert(statValue)
		if err != nil {
			log.Error("Error while converting value of stat: ", statName, " and converted value is ", statValue)
			continue
		}

		allowed := isMetricAllowed(commons.CTX_SYSINFO_NET_STATS, statName)
		sysMetric := NewAerospikeStat(commons.CTX_SYSINFO_NET_STATS, statName, allowed)
		sysMetric.Labels = netStatInfoLabels
		sysMetric.LabelValues = labelValues
		sysMetric.Value = value

		arrSysInfoStats = append(arrSysInfoStats, sysMetric)
	}

	return arrSysInfoStats
}

func getNetworkInfo() []AerospikeStat {

	arrSysInfoStats := []AerospikeStat{}

	networkLabels := []string{commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE, commons.METRIC_LABEL_DEVICE}

	arrReceiveStats, arrTransferStats := dataprovider.GetSystemProvider().GetNetDevStats()

	// netdev receive
	clusterName := ClusterName
	service := Service
	for _, stats := range arrReceiveStats {
		deviceName := stats["device_name"]
		statName := "receive_bytes_total"
		value, err := commons.TryConvert(stats[statName])
		if err != nil {
			log.Debug("Error while converting value of stat: ", statName, " and converted value is ", stats[statName])
			continue
		}

		labelValues := []string{clusterName, service, deviceName}

		allowed := isMetricAllowed(commons.CTX_SYSINFO_NETWORK_STATS, statName)
		sysMetric := NewAerospikeStat(commons.CTX_SYSINFO_NETWORK_STATS, statName, allowed)
		sysMetric.Labels = networkLabels
		sysMetric.LabelValues = labelValues
		sysMetric.Value = value

		arrSysInfoStats = append(arrSysInfoStats, sysMetric)

	}

	// netdev transfer
	for _, stats := range arrTransferStats {
		deviceName := stats["device_name"]
		statName := "transfer_bytes_total"
		value, err := commons.TryConvert(stats[statName])
		if err != nil {
			log.Debug("Error while converting value of stat: ", statName, " and converted value is ", stats[statName])
			continue
		}

		labelValues := []string{clusterName, service, deviceName}
		allowed := isMetricAllowed(commons.CTX_SYSINFO_NETWORK_STATS, statName)
		sysMetric := NewAerospikeStat(commons.CTX_SYSINFO_NETWORK_STATS, statName, allowed)
		sysMetric.Labels = networkLabels
		sysMetric.LabelValues = labelValues
		sysMetric.Value = value

		arrSysInfoStats = append(arrSysInfoStats, sysMetric)
	}

	return arrSysInfoStats
}

// disk, filesystem and vmstat - TODO: clean-up

func getDiskInfo() []AerospikeStat {
	arrSysInfoStats := []AerospikeStat{}
	diskStats := dataprovider.GetSystemProvider().GetDiskStats()

	// metric: diskinfo
	metricDiskInfoLabels := []string{commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE, commons.METRIC_LABEL_DEVICE}
	metricDiskInfoLabels = append(metricDiskInfoLabels, commons.METRIC_LABEL_MAJOR, commons.METRIC_LABEL_MINOR, commons.METRIC_LABEL_SERIAL)

	// other disk metrics
	diskInfoLabels := []string{commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE, commons.METRIC_LABEL_DEVICE}

	for _, stats := range diskStats {
		deviceName := stats["device_name"]

		arrSysInfoStats = append(arrSysInfoStats, constructDiskinfoSystemStat(deviceName, "reads_completed_total", stats, diskInfoLabels))
		arrSysInfoStats = append(arrSysInfoStats, constructDiskinfoSystemStat(deviceName, "reads_merged_total", stats, diskInfoLabels))
		arrSysInfoStats = append(arrSysInfoStats, constructDiskinfoSystemStat(deviceName, "read_bytes_total", stats, diskInfoLabels))
		arrSysInfoStats = append(arrSysInfoStats, constructDiskinfoSystemStat(deviceName, "read_time_seconds_total", stats, diskInfoLabels))
		arrSysInfoStats = append(arrSysInfoStats, constructDiskinfoSystemStat(deviceName, "writes_completed_total", stats, diskInfoLabels))
		arrSysInfoStats = append(arrSysInfoStats, constructDiskinfoSystemStat(deviceName, "writes_merged_total", stats, diskInfoLabels))
		arrSysInfoStats = append(arrSysInfoStats, constructDiskinfoSystemStat(deviceName, "writes_bytes_total", stats, diskInfoLabels))
		arrSysInfoStats = append(arrSysInfoStats, constructDiskinfoSystemStat(deviceName, "write_time_seconds_total", stats, diskInfoLabels))
		arrSysInfoStats = append(arrSysInfoStats, constructDiskinfoSystemStat(deviceName, "io_now", stats, diskInfoLabels))
		arrSysInfoStats = append(arrSysInfoStats, constructDiskinfoSystemStat(deviceName, "io_time_seconds_total", stats, diskInfoLabels))
		arrSysInfoStats = append(arrSysInfoStats, constructDiskinfoSystemStat(deviceName, "io_time_weighted_seconds_total", stats, diskInfoLabels))
		arrSysInfoStats = append(arrSysInfoStats, constructDiskinfoSystemStat(deviceName, "discards_completed_total", stats, diskInfoLabels))
		arrSysInfoStats = append(arrSysInfoStats, constructDiskinfoSystemStat(deviceName, "discards_merged_total", stats, diskInfoLabels))
		arrSysInfoStats = append(arrSysInfoStats, constructDiskinfoSystemStat(deviceName, "discarded_sectors_total", stats, diskInfoLabels))
		arrSysInfoStats = append(arrSysInfoStats, constructDiskinfoSystemStat(deviceName, "discard_time_seconds_total", stats, diskInfoLabels))
		arrSysInfoStats = append(arrSysInfoStats, constructDiskinfoSystemStat(deviceName, "flush_requests_total", stats, diskInfoLabels))
		arrSysInfoStats = append(arrSysInfoStats, constructDiskinfoSystemStat(deviceName, "flush_requests_time_seconds_total", stats, diskInfoLabels))

		arrSysInfoStats = append(arrSysInfoStats, constructDiskInfo(deviceName, stats["major_number"], stats["minor_number"], stats["serial"], metricDiskInfoLabels))

	}

	return arrSysInfoStats
}

func constructDiskInfo(deviceName string, major string, minor string, serial string, metricDiskInfoLabels []string) AerospikeStat {
	// 	[]string{"device", "major", "minor", "path", "wwn", "model", "serial", "revision"},
	// (stats.MajorNumber),(stats.MinorNumber), info[udevIDPath], info[udevIDWWN], info[udevIDModel], serial, info[udevIDRevision],
	clusterName := ClusterName
	service := Service

	labelValues := []string{clusterName, service, deviceName, major, minor, serial}

	allowed := isMetricAllowed(commons.CTX_SYSINFO_DISK_STATS, "info")

	sysMetric := NewAerospikeStat(commons.CTX_SYSINFO_DISK_STATS, "info", allowed)
	sysMetric.Labels = metricDiskInfoLabels
	sysMetric.LabelValues = labelValues
	sysMetric.Value = 1

	return sysMetric
}

func constructDiskinfoSystemStat(deviceName string, statName string, diskStats map[string]string, diskInfoLabels []string) AerospikeStat {

	clusterName := ClusterName
	service := Service

	labelValues := []string{clusterName, service, deviceName}

	metricName := strings.ToLower(statName)
	allowed := isMetricAllowed(commons.CTX_SYSINFO_DISK_STATS, statName)
	sysMetric := NewAerospikeStat(commons.CTX_SYSINFO_DISK_STATS, metricName, allowed)

	sysMetric.Labels = diskInfoLabels
	sysMetric.LabelValues = labelValues
	value, _ := commons.TryConvert(diskStats[statName])
	sysMetric.Value = value

	return sysMetric
}

func getFileSystemInfo() []AerospikeStat {
	arrFileSystemMountStats := dataprovider.GetSystemProvider().GetFileSystemStats()

	// global labels
	fsReadOnlyLabels := []string{}
	fsReadOnlyLabels = append(fsReadOnlyLabels, commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE)
	fsReadOnlyLabels = append(fsReadOnlyLabels, commons.METRIC_LABEL_FSTYPE, commons.METRIC_LABEL_DEVICE, commons.METRIC_LABEL_MOUNT_POINT)

	fsInfoLabels := []string{commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE, commons.METRIC_LABEL_FSTYPE, commons.METRIC_LABEL_DEVICE, commons.METRIC_LABEL_MOUNT_POINT}

	arrSysInfoStats := []AerospikeStat{}
	for _, stats := range arrFileSystemMountStats {

		isreadonly := stats["is_read_only"]
		source := stats["source"]
		mountPoint := stats["mount_point"]
		fsType := stats["mount_point"]

		arrSysInfoStats = append(arrSysInfoStats, constructFileSystemSysInfoStats(fsType, mountPoint, source, "size_bytes", stats, fsInfoLabels))
		arrSysInfoStats = append(arrSysInfoStats, constructFileSystemSysInfoStats(fsType, mountPoint, source, "free_bytes", stats, fsInfoLabels))
		arrSysInfoStats = append(arrSysInfoStats, constructFileSystemSysInfoStats(fsType, mountPoint, source, "avail_byts", stats, fsInfoLabels))
		arrSysInfoStats = append(arrSysInfoStats, constructFileSystemSysInfoStats(fsType, mountPoint, source, "files", stats, fsInfoLabels))
		arrSysInfoStats = append(arrSysInfoStats, constructFileSystemSysInfoStats(fsType, mountPoint, source, "files_free", stats, fsInfoLabels))

		// add disk-info
		statReadOnly := constructFileSystemReadOnly(fsType, mountPoint, source, isreadonly, fsReadOnlyLabels)
		arrSysInfoStats = append(arrSysInfoStats, statReadOnly)
	}

	return arrSysInfoStats
}

func constructFileSystemReadOnly(fstype string, mountpoint string, deviceName string, isReadOnly string, fsReadOnlyLabels []string) AerospikeStat {
	clusterName := ClusterName
	service := Service

	// add disk_info
	labelValues := []string{clusterName, service, fstype, deviceName, mountpoint}

	allowed := isMetricAllowed(commons.CTX_SYSINFO_FILESYSTEM_STATS, "readonly")
	sysMetric := NewAerospikeStat(commons.CTX_SYSINFO_FILESYSTEM_STATS, "readonly", allowed)
	sysMetric.Labels = fsReadOnlyLabels
	sysMetric.LabelValues = labelValues
	sysMetric.Value, _ = commons.TryConvert(isReadOnly)

	return sysMetric

}

func constructFileSystemSysInfoStats(fstype string, mountpoint string, deviceName string, statName string, stats map[string]string, fsInfoLabels []string) AerospikeStat {

	clusterName := ClusterName
	service := Service

	labelValues := []string{clusterName, service, fstype, deviceName, mountpoint}

	metricName := strings.ToLower(statName)
	allowed := isMetricAllowed(commons.CTX_SYSINFO_FILESYSTEM_STATS, statName)
	sysMetric := NewAerospikeStat(commons.CTX_SYSINFO_FILESYSTEM_STATS, metricName, allowed)

	sysMetric.Labels = fsInfoLabels
	sysMetric.LabelValues = labelValues

	value, _ := commons.TryConvert(stats[statName])
	sysMetric.Value = value

	return sysMetric
}

func getVmStatsInfo() []AerospikeStat {

	arrSysInfoStats := []AerospikeStat{}
	vmStatLabels := []string{commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE}

	arrVmStats := dataprovider.GetSystemProvider().GetVmStats()

	for _, vmStats := range arrVmStats {
		for key, _ := range vmStats {
			arrSysInfoStats = append(arrSysInfoStats, constructVmstat(key, vmStatLabels, vmStats))
		}
	}

	return arrSysInfoStats
}

func constructVmstat(statName string, vmStatLabels []string, stats map[string]string) AerospikeStat {
	clusterName := ClusterName
	service := Service

	labelValues := []string{clusterName, service}

	allowed := isMetricAllowed(commons.CTX_SYSINFO_VM_STATS, statName)
	sysMetric := NewAerospikeStat(commons.CTX_SYSINFO_VM_STATS, statName, allowed)
	sysMetric.Labels = vmStatLabels
	sysMetric.LabelValues = labelValues
	sysMetric.Value, _ = commons.TryConvert(stats[statName])

	return sysMetric
}
