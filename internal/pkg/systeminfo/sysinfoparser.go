package systeminfo

import (
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/statprocessors"
	log "github.com/sirupsen/logrus"
)

const (
	METRIC_LABEL_MEM = "memory"
)

type SysInfoProcessor interface {
	Refresh() ([]statprocessors.AerospikeStat, error)
}

var sysinfoprocessors = []SysInfoProcessor{
	&CpuInfoProcessor{},
	&DiskInfoProcessor{},
	&FileFDInfoProcessor{},
	&FileSystemInfoProcessor{},
	&MemInfoProcessor{},
	&NetStatInfoProcessor{},
	&NetworkInfoProcessor{},
	&VmstatInfoProcessor{},
}

func Refresh() ([]statprocessors.AerospikeStat, error) {
	var stats = []statprocessors.AerospikeStat{}

	// Refresh System Info stats only when enabled
	if !config.Cfg.Agent.RefreshSystemStats {
		return stats, nil
	}

	for _, processor := range sysinfoprocessors {
		siRefreshStats, err := processor.Refresh()
		if err != nil {
			log.Error("Error while Refreshing SystemInfo Stats, Error: ", err)
		} else {
			stats = append(stats, siRefreshStats...)
		}
	}

	return stats, nil
}
