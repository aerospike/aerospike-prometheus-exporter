package dataprovider

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
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
	// DISK_IGNORE_NAME_LIST = "^(z?ram|loop|fd|(h|s|v|xv)d[a-z]|nvme\\d+n\\d+p)\\d+$"
	DISK_IGNORE_NAME_LIST = "^(z?ram|loop|fd|(h|s|v|xv)d[a-z]|nvme\\d+n\\d+p)\\d$"
	FILE_STAT_IGNORE_LIST = "^(overlay|mqueue)$"

	netstatAcceptlist = "^(.*_(inerrors|inerrs)|ip_forwarding|ip(6|ext)_(inoctets|outoctets)|icmp6?_(inmsgs|outmsgs)|tcpext_(listen.*|syncookies.*|tcpsynretrans|tcptimeouts|tcpofoqueue)|tcp_(activeopens|insegs|outsegs|outrsts|passiveopens|retranssegs|currestab)|udp6?_(indatagrams|outdatagrams|noports|rcvbuferrors|sndbuferrors))$"
	snmp6Prefixlist   = "^(ip6.*|icmp6.*|udp6.*)"

	vmstatAcceptList = "^(oom_kill|pgpg|pswp|pg.*fault).*"
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
	regexSnmp6PrefixPattern   = regexp.MustCompile(snmp6Prefixlist)
	regexVmstatAcceptPattern  = regexp.MustCompile(vmstatAcceptList)
)

func GetProcFilePath(name string) string {
	return filepath.Join(PROC_PATH, name)
}

func GetSysFilePath(name string) string {
	return filepath.Join(SYS_PATH, name)
}

func GetUdevDataFilePath(name string) string {
	return filepath.Join(UDEV_DATA_PATH, name)
}

func GetRootfsFilePath(name string) string {
	return filepath.Join(ROOTFS_PATH, name)
}

// func rootfsStripPrefix(path string) string {
// 	if *rootfsPath == "/" {
// 		return path
// 	}
// 	stripped := strings.TrimPrefix(path, *rootfsPath)
// 	if stripped == "" {
// 		return "/"
// 	}
// 	return stripped
// }

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

func acceptSnmp6(name string) bool {
	return (regexSnmp6PrefixPattern != nil && regexSnmp6PrefixPattern.MatchString(name))
}

func acceptVmstat(name string) bool {
	return (regexVmstatAcceptPattern != nil && regexVmstatAcceptPattern.MatchString(name))
}

func GetMetricType(pContext commons.ContextType, pRawMetricName string) commons.MetricType {
	return commons.MetricTypeGauge
}

func isMetricAllowed(pContext commons.ContextType, pRawMetricName string) bool {
	return true
}

func getUdevDeviceProperties(major, minor uint32) (map[string]string, error) {
	filename := GetUdevDataFilePath(fmt.Sprintf("b%d:%d", major, minor))

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

func GetFloatValue(addr *uint64) float64 {
	if addr != nil {
		value := float64(*addr)
		return value
	}
	return 0.0
}
