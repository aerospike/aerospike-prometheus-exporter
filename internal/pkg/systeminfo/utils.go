package systeminfo

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/prometheus/procfs"
)

var (
	PROC_PATH      = procfs.DefaultMountPoint
	SYS_PATH       = "/sys"
	ROOTFS_PATH    = "/"
	UDEV_DATA_PATH = "/run/udev/data"
	NET_STAT_PATH  = "net/netstat"
)

const (
	// diskstatsIgnoredDevices = "^(z?ram|loop|fd|(h|s|v|xv)d[a-z]|nvme\\d+n\\d+p)\\d+$"
	diskstatsIgnoredDevices = "^(z?ram|loop|fd|(h|s|v|xv)d[a-z]|nvme\\d+n\\d+p)\\d$"
	filestatIgnoreList      = "^(overlay|mqueue)$"

	netstatAcceptlist = "^(.*_(InErrors|InErrs)|Ip_Forwarding|Ip(6|Ext)_(InOctets|OutOctets)|Icmp6?_(InMsgs|OutMsgs)|TcpExt_(Listen.*|Syncookies.*|TCPSynRetrans|TCPTimeouts|TCPOFOQueue)|Tcp_(ActiveOpens|InSegs|OutSegs|OutRsts|PassiveOpens|RetransSegs|CurrEstab)|Udp6?_(InDatagrams|OutDatagrams|NoPorts|RcvbufErrors|SndbufErrors))$"
)

var diskIgnorePattern = regexp.MustCompile(diskstatsIgnoredDevices)
var fileIgnorePattern = regexp.MustCompile(filestatIgnoreList)
var netstatAcceptPattern = regexp.MustCompile(netstatAcceptlist)

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
		if !strings.HasPrefix(line, udevDevicePropertyPrefix) {
			continue
		}

		line = strings.TrimPrefix(line, udevDevicePropertyPrefix)

		if name, value, found := strings.Cut(line, "="); found {
			info[name] = value
		}
	}

	return info, nil
}

func GetFloatValue(addr *uint64) float64 {
	if addr != nil {
		value := float64(*addr)
		return value
	}
	return 0.0
}

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

func GetMetricType(pContext commons.ContextType, pRawMetricName string) commons.MetricType {
	return commons.MetricTypeGauge
}

func isMetricAllowed(pContext commons.ContextType, pRawMetricName string) bool {
	return true
}
