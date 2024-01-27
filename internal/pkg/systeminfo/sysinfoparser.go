package systeminfo

const (
	METRIC_LABEL_MEM = "memory"
)

func Refresh() []SystemInfoStat {
	var stats = []SystemInfoStat{}

	// Get Memory Stats
	memStats := GetMemInfo()
	stats = append(stats, memStats...)

	diskStats := GetDiskStats()
	stats = append(stats, diskStats...)

	fileSystemStats := GetFileSystemInfo()
	stats = append(stats, fileSystemStats...)

	return stats
}
