package systeminfo

const (
	METRIC_LABEL_MEM = "memory"
)

func Refresh() []SystemInfoStat {
	var stats = []SystemInfoStat{}

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
