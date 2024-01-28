package commons

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
	METRIC_LABEL_CLUSTER_NAME   = "cluster_name"
	METRIC_LABEL_SERVICE        = "service"
	METRIC_LABEL_NS             = "ns"
	METRIC_LABEL_SET            = "set"
	METRIC_LABEL_LE             = "le"
	METRIC_LABEL_DC_NAME        = "dc"
	METRIC_LABEL_INDEX          = "index"
	METRIC_LABEL_SINDEX         = "sindex"
	METRIC_LABEL_STORAGE_ENGINE = "storage_engine"
	METRIC_LABEL_USER           = "user"
)

// constants used to identify type of metrics
const (
	STORAGE_ENGINE = "storage-engine"
	INDEX_TYPE     = "index-type"
	SINDEX_TYPE    = "sindex-type"
)

const (
	CTX_MEMORY_STATS     ContextType = "memory_stats"
	CTX_DISK_STATS       ContextType = "disk_stats"
	CTX_FILESYSTEM_STATS ContextType = "filesystem_stats"
	CTX_CPU_STATS        ContextType = "cpu"
	CTX_NET_STATS        ContextType = "netstat"
	CTX_NET_DEV_STATS    ContextType = "net_dev"
	CTX_NETWORK_STATS    ContextType = "network"
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
