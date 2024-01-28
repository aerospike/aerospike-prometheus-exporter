package systeminfo

import "fmt"

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

	cpuStats := GetCpuInfo()
	fmt.Println("\n\t **** # of CPU Stats ... ", len(cpuStats))
	stats = append(stats, cpuStats...)

	return stats
}
