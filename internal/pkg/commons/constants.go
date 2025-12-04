package commons

import "github.com/prometheus/procfs"

const (
	CTX_USERS      ContextType = "users"
	CTX_NAMESPACE  ContextType = "namespace"
	CTX_NODE_STATS ContextType = "node_stats"
	CTX_SETS       ContextType = "sets"
	CTX_SINDEX     ContextType = "sindex"
	CTX_XDR        ContextType = "xdr"
	CTX_LATENCIES  ContextType = "latencies"
)

// below constant represent the labels we send along with metrics to Prometheus or something
const (
	METRIC_LABEL_CLUSTER_NAME              = "cluster_name"
	METRIC_LABEL_SERVICE                   = "service"
	METRIC_LABEL_NS                        = "ns"
	METRIC_LABEL_SET                       = "set"
	METRIC_LABEL_LE                        = "le"
	METRIC_LABEL_DC_NAME                   = "dc"
	METRIC_LABEL_INDEX                     = "index"
	METRIC_LABEL_SINDEX                    = "sindex"
	METRIC_LABEL_STORAGE_ENGINE            = "storage_engine"
	METRIC_LABEL_USER                      = "user"
	METRIC_LABEL_UA_CLIENT_LIBRARY_VERSION = "client_library_version"
	METRIC_LABEL_UA_CLIENT_APP_ID          = "client_app_id"
)

// constants used to identify type of metrics
const (
	STORAGE_ENGINE = "storage-engine"
	INDEX_TYPE     = "index-type"
	SINDEX_TYPE    = "sindex-type"
)

const (
	CTX_SYSINFO_MEMORY_STATS     ContextType = "sysinfo_memory_stats"
	CTX_SYSINFO_DISK_STATS       ContextType = "sysinfo_disk_stats"
	CTX_SYSINFO_FILESYSTEM_STATS ContextType = "sysinfo_filesystem_stats"
	CTX_SYSINFO_CPU_STATS        ContextType = "sysinfo_cpu"
	CTX_SYSINFO_NET_STATS        ContextType = "sysinfo_netstat"
	CTX_SYSINFO_NET_DEV_STATS    ContextType = "sysinfo_net_dev"
	CTX_SYSINFO_NETWORK_STATS    ContextType = "sysinfo_network"
	CTX_SYSINFO_FILEFD_STATS     ContextType = "sysinfo_filefd"
	CTX_SYSINFO_VM_STATS         ContextType = "sysinfo_vmstat"
)
const (
	METRIC_LABEL_DEVICE      = "device"
	METRIC_LABEL_MAJOR       = "major_version"
	METRIC_LABEL_MINOR       = "minor_version"
	METRIC_LABEL_SERIAL      = "serial"
	METRIC_LABEL_MOUNT_POINT = "mountpoint"
	METRIC_LABEL_FSTYPE      = "fstype"
	METRIC_LABEL_CPU         = "cpu"
	METRIC_LABEL_CPU_MODE    = "mode"
)

var (
	PROC_PATH         = procfs.DefaultMountPoint
	SYS_PATH          = "/sys"
	ROOTFS_PATH       = "/"
	UDEV_DATA_PATH    = "/run/udev/data"
	NET_STAT_PATH     = "net/netstat"
	NET_DEV_STAT_PATH = "/proc/net/dev"
)
