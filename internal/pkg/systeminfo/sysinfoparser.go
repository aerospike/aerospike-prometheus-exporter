package systeminfo

import "github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"

const (
	METRIC_LABEL_MEM = "memory"
)

func Refresh() []SystemInfoStat {
	var stats = []SystemInfoStat{}

	// Refresh System Info stats only when enabled
	if !config.Cfg.AeroProm.RefreshSystemStats {
		return stats
	}

	// Get Memory Stats
	stats = append(stats, GetMemInfo()...)
	stats = append(stats, GetDiskStats()...)
	stats = append(stats, GetFileSystemInfo()...)
	stats = append(stats, GetCpuInfo()...)
	stats = append(stats, GetNetStatInfo()...)
	stats = append(stats, GetNetworkStatsInfo()...)
	stats = append(stats, GetFileFDInfo()...)
	stats = append(stats, GetVmStatInfo()...)

	return stats
}
