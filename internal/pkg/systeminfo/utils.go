package systeminfo

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
	vmstatAcceptList = "^(oom_kill|pgpg|pswp|pg.*fault).*"
)

var (
	vmstatAcceptPattern = regexp.MustCompile(vmstatAcceptList)
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

func acceptVmstat(name string) bool {
	return (vmstatAcceptPattern != nil && vmstatAcceptPattern.MatchString(name))
}

func GetMetricType(pContext commons.ContextType, pRawMetricName string) commons.MetricType {
	return commons.MetricTypeGauge
}

func isMetricAllowed(pContext commons.ContextType, pRawMetricName string) bool {
	return true
}
