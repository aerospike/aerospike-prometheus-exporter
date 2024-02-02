package systeminfo

import (
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"
	log "github.com/sirupsen/logrus"
)

const (
	METRIC_LABEL_MEM = "memory"
)

type SysInfoProcessor interface {
	Refresh() ([]SystemInfoStat, error)
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

func Refresh() ([]SystemInfoStat, error) {
	var stats = []SystemInfoStat{}

	// Refresh System Info stats only when enabled
	if !config.Cfg.Agent.RefreshSystemStats {
		return stats, nil
	}

	for _, processor := range sysinfoprocessors {
		siRefreshStats, err := processor.Refresh()
		if err != nil {
			log.Error("Error while Refreshing SystemInfoStats, Error: ", err)
		} else {
			stats = append(stats, siRefreshStats...)
		}
	}

	return stats, nil
}
