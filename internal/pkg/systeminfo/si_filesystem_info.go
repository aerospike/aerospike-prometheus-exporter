package systeminfo

import (
	"strings"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/statprocessors"
	"github.com/prometheus/procfs"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

func GetFileSystemInfo() []SystemInfoStat {
	arrSysInfoStats := parseFileSystemInfo()
	return arrSysInfoStats
}

func parseFileSystemInfo() []SystemInfoStat {

	arrSysInfoStats := []SystemInfoStat{}

	mnts, err := procfs.GetMounts()
	if err != nil {
		return arrSysInfoStats
	}

	for _, mnt := range mnts {

		if ignoreFileSystem(mnt.Source) {
			log.Debug("\t ** FileSystem Stats -- Ignoring mount ", mnt.Source)
			continue
		}

		// local variables
		isreadonly := 0.0
		size, free, avail, files, filesFree, isError := 0.0, 0.0, 0.0, 0.0, 0.0, false
		l_mount_stats := make(map[string]float64)

		// if the disk is read only
		_, roKeyFound := mnt.OptionalFields["ro"]
		if roKeyFound {
			isreadonly = 1.0
		}

		size, free, avail, files, filesFree, isError = GetMountData(mnt.Source)
		if isError {
			log.Debug("Skipping, error during reading stats of mount-point ", mnt.MountPoint, " and mount-source ", mnt.Source)
			continue
		}

		l_mount_stats["size_bytes"] = size
		l_mount_stats["free_bytes"] = free
		l_mount_stats["avail_byts"] = avail
		l_mount_stats["files"] = files
		l_mount_stats["files_free"] = filesFree

		l_sysinfo_stats := constructFileSystemStats(mnt.FSType, mnt.MountPoint, mnt.Source, l_mount_stats)

		// add to return array
		arrSysInfoStats = append(arrSysInfoStats, l_sysinfo_stats...)

		// add disk-info
		statReadOnly := constructFileSystemReadOnly(mnt.FSType, mnt.MountPoint, mnt.Source, isreadonly)
		arrSysInfoStats = append(arrSysInfoStats, statReadOnly)

	}

	return arrSysInfoStats
}

func GetMountData(mntpointsource string) (float64, float64, float64, float64, float64, bool) {
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

func constructFileSystemReadOnly(fstype string, mountpoint string, deviceName string, isReadOnly float64) SystemInfoStat {
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
	sysMetric.Value = isReadOnly

	return sysMetric

}

func constructFileSystemStats(fstype string, mountpoint string, deviceName string, v_stats_info map[string]float64) []SystemInfoStat {
	// deviceName is same as source
	arrSysInfoStats := []SystemInfoStat{}

	clusterName := statprocessors.ClusterName
	service := statprocessors.Service

	for sk, sv := range v_stats_info {
		labels := []string{commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE, commons.METRIC_LABEL_FSTYPE, commons.METRIC_LABEL_DEVICE, commons.METRIC_LABEL_MOUNT_POINT}
		labelValues := []string{clusterName, service, fstype, deviceName, mountpoint}

		l_metricName := strings.ToLower(sk)
		sysMetric := NewSystemInfoStat(commons.CTX_FILESYSTEM_STATS, l_metricName)

		sysMetric.Labels = labels
		sysMetric.LabelValues = labelValues
		sysMetric.Value = sv

		arrSysInfoStats = append(arrSysInfoStats, sysMetric)
	}

	return arrSysInfoStats
}
