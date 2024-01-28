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

	// fs, err := procfs.NewFS(GetProcFilePath(NET_STAT_PATH))
	fs, err := procfs.NewFS("/proc/net")

	fmt.Println("\n\n ***** GetProcFilePath(net) ", GetProcFilePath("net"))
	if err != nil {
		log.Error("parseNetStats Error while reading NET Stats from ", NET_STAT_PATH, " Error ", err)
		return arrSysInfoStats
	}

	stats, err := fs.NetStat()
	if err != nil {
		log.Debug("Eror while reading procfs.NewFS system,  error: ", err)
		return arrSysInfoStats
	}

	fmt.Println(" \t\t **** Netsocket Stats  ", stats)

	return arrSysInfoStats
}
