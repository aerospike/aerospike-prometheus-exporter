package dataprovider

import (
	"path/filepath"
	"regexp"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/prometheus/procfs"
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
	// diskstatsIgnoredDevices = "^(z?ram|loop|fd|(h|s|v|xv)d[a-z]|nvme\\d+n\\d+p)\\d+$"
	diskstatsIgnoredDevices = "^(z?ram|loop|fd|(h|s|v|xv)d[a-z]|nvme\\d+n\\d+p)\\d$"
	filestatIgnoreList      = "^(overlay|mqueue)$"

	netstatAcceptlist = "^(.*_(inerrors|inerrs)|ip_forwarding|ip(6|ext)_(inoctets|outoctets)|icmp6?_(inmsgs|outmsgs)|tcpext_(listen.*|syncookies.*|tcpsynretrans|tcptimeouts|tcpofoqueue)|tcp_(activeopens|insegs|outsegs|outrsts|passiveopens|retranssegs|currestab)|udp6?_(indatagrams|outdatagrams|noports|rcvbuferrors|sndbuferrors))$"
	snmp6Prefixlist   = "^(ip6.*|icmp6.*|udp6.*)"

	vmstatAcceptList = "^(oom_kill|pgpg|pswp|pg.*fault).*"
)

var (
	diskIgnorePattern    = regexp.MustCompile(diskstatsIgnoredDevices)
	fileIgnorePattern    = regexp.MustCompile(filestatIgnoreList)
	netstatAcceptPattern = regexp.MustCompile(netstatAcceptlist)
	snmp6PrefixPattern   = regexp.MustCompile(snmp6Prefixlist)
	vmstatAcceptPattern  = regexp.MustCompile(vmstatAcceptList)
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
	return (diskIgnorePattern != nil && diskIgnorePattern.MatchString(name))
}

// ignoreDisk returns whether the device should be ignoreDisk
func ignoreFileSystem(name string) bool {
	return (fileIgnorePattern != nil && fileIgnorePattern.MatchString(name))
}

func acceptNetstat(name string) bool {
	return (netstatAcceptPattern != nil && netstatAcceptPattern.MatchString(name))
}

func acceptSnmp6(name string) bool {
	return (snmp6PrefixPattern != nil && snmp6PrefixPattern.MatchString(name))
}

func acceptVmstat(name string) bool {
	return (vmstatAcceptPattern != nil && vmstatAcceptPattern.MatchString(name))
}

func GetMetricType(pContext commons.ContextType, pRawMetricName string) commons.MetricType {
	return commons.MetricTypeGauge
}

func isMetricAllowed(pContext commons.ContextType, pRawMetricName string) bool {
	return true
}
