package systeminfo

import (
	"fmt"
)

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

	return stats
}

func createFileSystemStats() []SystemInfoStat {
	fsStats := GetFileSystemInfo()
	fmt.Println("createFileSystemStats - fsStats: ", len(fsStats))
	return nil
}
