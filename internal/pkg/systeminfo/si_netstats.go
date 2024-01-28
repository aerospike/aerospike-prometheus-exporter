package systeminfo

import (
	"fmt"

	"github.com/prometheus/procfs"
	log "github.com/sirupsen/logrus"
)

func GetNetStatnfo() []SystemInfoStat {
	arrSysInfoStats := parseNetStats()
	return arrSysInfoStats
}

func parseNetStats() []SystemInfoStat {
	arrSysInfoStats := []SystemInfoStat{}

	fs, err := procfs.NewFS(PROC_PATH)
	if err != nil {
		log.Debug("GetCpuStats Error while reading CPU Stats from ", PROC_PATH)
		return arrSysInfoStats
	}

	stats, err := fs.NetStat()
	if err != nil {
		log.Debug("Eror while reading procfs.NewFS system,  error: ", err)
		return arrSysInfoStats
	}

	for k, v := range stats {
		fmt.Println("key -- ", k, " -- value ", v)
	}

	return arrSysInfoStats
}
