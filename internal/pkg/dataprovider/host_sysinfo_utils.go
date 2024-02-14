package dataprovider

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/prometheus/procfs"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

var (
	PROC_PATH         = procfs.DefaultMountPoint
	SYS_PATH          = "/sys"
	ROOTFS_PATH       = "/"
	UDEV_DATA_PATH    = "/run/udev/data"
	NET_STAT_PATH     = "net/netstat"
	NET_DEV_STAT_PATH = "/proc/net/dev"
)

const (
	// DISK_IGNORE_NAME_LIST = "^(z?ram|loop|fd|(h|s|v|xv)d[a-z]|nvme\\d+n\\d+p)\\d$"
	DISK_IGNORE_NAME_LIST = "^(z?ram|loop|fd|(h|s|xv)d[a-z]|nvme\\d+n\\d+p)\\d$"
	// FILE_STAT_IGNORE_LIST = "^(overlay|mqueue)$"
	FILE_STAT_IGNORE_LIST = "^(mqueue)$"

	// netstatAcceptlist = "^(.*_(inerrors|inerrs)|ip_forwarding|ip(6|ext)_(inoctets|outoctets)|icmp6?_(inmsgs|outmsgs)|tcpext_(listen.*|syncookies.*|tcpsynretrans|tcptimeouts|tcpofoqueue)|tcp_(activeopens|insegs|outsegs|outrsts|passiveopens|retranssegs|currestab)|udp6?_(indatagrams|outdatagrams|noports|rcvbuferrors|sndbuferrors))$"
	netstatAcceptlist = "tcp_(activeopens|retranssegs|currestab)"
	vmstatAcceptList  = "^(oom_kill|pgpg|pswp|pg.*fault).*"
)

const (
	SECONDS_PER_TICK = 0.0001 // 1000 ticks per second

	// Read/write sectors are "standard UNIX 512-byte sectors" (https://www.kernel.org/doc/Documentation/block/stat.txt)
	DISK_SECTOR_SIZE_IN_UNIX = 512.0

	ONE_KILO_BYTE = 1024

	// See udevadm(8).
	UDEV_PROP_PREFIX       = "E:"
	UDEV_ID_SERIAL_SHORT   = "ID_SERIAL_SHORT"
	UDEV_SCSI_IDENT_SERIAL = "SCSI_IDENT_SERIAL"
)

var (
	regexDiskIgnorePattern    = regexp.MustCompile(DISK_IGNORE_NAME_LIST)
	regexFileIgnorePattern    = regexp.MustCompile(FILE_STAT_IGNORE_LIST)
	regexNetstatAcceptPattern = regexp.MustCompile(netstatAcceptlist)
	regexVmstatAcceptPattern  = regexp.MustCompile(vmstatAcceptList)
)

func getProcFilePath(name string) string {
	return filepath.Join(PROC_PATH, name)
}

func getUdevDataFilePath(name string) string {
	return filepath.Join(UDEV_DATA_PATH, name)
}

func getRootfsFilePath(name string) string {
	return filepath.Join(ROOTFS_PATH, name)
}

// ignoreDisk returns whether the device should be ignoreDisk
func ignoreDisk(name string) bool {
	return (regexDiskIgnorePattern != nil && regexDiskIgnorePattern.MatchString(name))
}

// ignoreDisk returns whether the device should be ignoreDisk
func ignoreFileSystem(name string) bool {
	return (regexFileIgnorePattern != nil && regexFileIgnorePattern.MatchString(name))
}

func acceptNetstat(name string) bool {
	return (regexNetstatAcceptPattern != nil && regexNetstatAcceptPattern.MatchString(name))
}

func acceptVmstat(name string) bool {
	return (regexVmstatAcceptPattern != nil && regexVmstatAcceptPattern.MatchString(name))
}

func getUdevDeviceProperties(major, minor uint32) (map[string]string, error) {
	filename := getUdevDataFilePath(fmt.Sprintf("b%d:%d", major, minor))

	data, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer data.Close()

	info := make(map[string]string)

	scanner := bufio.NewScanner(data)
	for scanner.Scan() {
		line := scanner.Text()

		// We're only interested in device properties.
		if !strings.HasPrefix(line, UDEV_PROP_PREFIX) {
			continue
		}

		line = strings.TrimPrefix(line, UDEV_PROP_PREFIX)

		if name, value, found := strings.Cut(line, "="); found {
			info[name] = value
		}
	}

	return info, nil
}

func readDiskMountData(mntpointsource string) (float64, float64, float64, float64, float64, bool) {
	_, err := os.Stat(mntpointsource)
	if err != nil {
		// if mount point does not exist
		return 0.0, 0.0, 0.0, 0.0, 0.0, true
	}

	buf := new(unix.Statfs_t)
	err = unix.Statfs(getRootfsFilePath(mntpointsource), buf)
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

func getFloatValue(addr *uint64) float64 {
	if addr != nil {
		value := float64(*addr)
		return value
	}
	return 0.0
}

func parseNetStats(fileName string) map[string]string {
	arrSysInfoStats := make(map[string]string)

	file, err := os.Open(fileName)
	if err != nil {
		log.Error("Error while opening file,", fileName, " Error: ", err)
		return arrSysInfoStats
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		statNames := strings.Split(scanner.Text(), " ")
		scanner.Scan()
		values := strings.Split(scanner.Text(), " ")
		protocol := statNames[0][:len(statNames[0])-1]
		if len(statNames) != len(values) {
			return arrSysInfoStats
		}
		for i := 1; i < len(statNames); i++ {
			key := strings.ToLower(protocol + "_" + statNames[i])
			// fmt.Println("key ", key, " acceptNetstat(key): ", acceptNetstat(key), " valueParts[i] ", valueParts[i])
			if acceptNetstat(key) {
				arrSysInfoStats[key] = values[i]
			}
		}
	}

	return arrSysInfoStats
}
