package systeminfo

import (
	"strconv"
	"strings"

	"github.com/prometheus/procfs"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

const ()

type FileSystemStat struct {
	mount_stats map[string]float64
	mount_info  map[string]string
	is_error    int
}

// var filesystem_stats map[string]FileSystemStat

func GetFileSystemInfo() map[string]FileSystemStat {

	filesystem_stats := make(map[string]FileSystemStat)

	mnts, err := procfs.GetMounts()
	handleError(err)

	for _, mnt := range mnts {

		if ignoreFileSystem(mnt.Source) {
			log.Debug("\t ** FileSystem Stats -- Ignoring mount ", mnt.Source)
			continue
		}

		// local variables
		isreadonly := false
		size, free, avail, files, filesFree, is_error := 0.0, 0.0, 0.0, 0.0, 0.0, 0
		l_mount_stats := make(map[string]float64)
		l_mount_info := make(map[string]string)

		l_mount_info["FSType"] = mnt.FSType
		l_mount_info["MajorMinorVer"] = mnt.MajorMinorVer
		l_mount_info["MountID"] = strconv.Itoa(mnt.MountID)
		l_mount_info["MountPoint"] = mnt.MountPoint
		l_mount_info["Root"] = mnt.Root
		l_mount_info["Source"] = mnt.Source
		l_mount_info["ParentID"] = strconv.Itoa(mnt.ParentID)

		for k, v := range mnt.OptionalFields {
			if strings.Contains(k, "ro") {
				isreadonly = true
			}
			l_mount_info["optionfields_"+k] = v
		}
		l_mount_info["isreadonly"] = strconv.FormatBool(isreadonly)

		for k, v := range mnt.Options {
			l_mount_info["options_"+k] = v
		}
		for k, v := range mnt.SuperOptions {
			l_mount_info["superoptions_"+k] = v
		}

		size, free, avail, files, filesFree, is_error = GetMountData(mnt.Source, isreadonly)

		l_mount_stats["size"] = size
		l_mount_stats["free"] = free
		l_mount_stats["avail"] = avail
		l_mount_stats["files"] = files
		l_mount_stats["filesFree"] = filesFree

		// add this mount to the root-hashmap
		filesystem_stats[strconv.Itoa(mnt.MountID)] = FileSystemStat{l_mount_stats, l_mount_info, is_error}
	}

	return filesystem_stats
}

func GetMountData(mntpointsource string, isreadonly bool) (float64, float64, float64, float64, float64, int) {
	buf := new(unix.Statfs_t)
	err := unix.Statfs(GetRootfsFilePath(mntpointsource), buf)
	// any kind of error
	if err != nil {
		log.Error("Error while fetching FileSystem stats for mount ", mntpointsource, ", hence, return all 0.0 --> error is ", err)
		return 0.0, 0.0, 0.0, 0.0, 0.0, 1
	}

	size := float64(buf.Blocks) * float64(buf.Bsize)
	free := float64(buf.Bfree) * float64(buf.Bsize)
	avail := float64(buf.Bavail) * float64(buf.Bsize)
	files := float64(buf.Files)
	filesFree := float64(buf.Ffree)

	return size, free, avail, files, filesFree, 0

}

func GetFileSystemInfoUsingUnix() {
	buf := new(unix.Statfs_t)
	err := unix.Statfs(GetRootfsFilePath("/dev/vda1"), buf)
	handleError(err)

	// buf.
}
