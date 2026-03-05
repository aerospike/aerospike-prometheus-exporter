package dataprovider

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/prometheus/procfs"
	log "github.com/sirupsen/logrus"
)

type SystemInfoProvider struct {
}

func (sip SystemInfoProvider) GetFileFD() map[string]string {
	fileFDStats := make(map[string]string)

	fileName := getProcFilePath("sys/fs/file-nr")

	file, err := os.Open(fileName)

	if err != nil {
		log.Error("Error while opening file,", fileName, " Error: ", err)
		return fileFDStats
	}

	defer file.Close() //nolint:errcheck

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		//
		values := strings.Split(scanner.Text(), "\t")

		fileFDStats["allocated"] = values[0]

	}

	log.Debugf("FileFD Stats - Count of return stats %d", len(fileFDStats))

	return fileFDStats
}

func (sip SystemInfoProvider) GetMemInfoStats() map[string]string {
	memStats := make(map[string]string)

	fs, err := procfs.NewFS(PROC_PATH)

	if err != nil {
		log.Debugf("Eror while reading procfs.NewFS system,  error: %s", err)
		return memStats
	}

	meminfo, err := fs.Meminfo()

	if err != nil {
		log.Debugf("Eror while reading MemInfo, error: %s", err)
		return memStats
	}

	// All values are in KB, convert to bytes
	memStats["Shmem"] = fmt.Sprint(getFloatValue(meminfo.Shmem) * ONE_KILO_BYTE)
	memStats["Swap_Cached"] = fmt.Sprint(getFloatValue(meminfo.SwapCached) * ONE_KILO_BYTE)

	log.Debugf("MemInfo Stats - Count of return stats %d", len(memStats))
	return memStats
}

func (sip SystemInfoProvider) GetNetStatInfo() map[string]string {

	arrSnmpStats := parseNetStats(getProcFilePath("net/snmp"))

	log.Debugf("NetStatsInfo SNMP - Count of return stats %d", len(arrSnmpStats))
	return arrSnmpStats
}

func (sip SystemInfoProvider) GetNetDevStats() ([]map[string]string, []map[string]string) {
	arrNetReceiveStats := []map[string]string{}
	arrNetTransferStats := []map[string]string{}

	fs, err := procfs.NewFS(PROC_PATH)

	if err != nil {
		log.Debugf("parseNetworkStats Error while reading Net_Dev Stats from %s, Error %s", PROC_PATH, err)
		return arrNetReceiveStats, arrNetTransferStats
	}

	stats, err := fs.NetDev()

	if err != nil {
		log.Debugf("Eror while reading procfs.NewFS system, error: %s", err)
		return arrNetReceiveStats, arrNetTransferStats
	}

	for deviceName, stats := range stats {

		receiveStats := make(map[string]string)
		transferStats := make(map[string]string)

		// network receive
		receiveStats["device_name"] = deviceName
		receiveStats["receive_bytes_total"] = fmt.Sprint(float64(stats.RxBytes))

		// network transfer
		transferStats["device_name"] = deviceName
		transferStats["transfer_bytes_total"] = fmt.Sprint(float64(stats.TxBytes))

		arrNetReceiveStats = append(arrNetReceiveStats, receiveStats)
		arrNetTransferStats = append(arrNetTransferStats, transferStats)
	}

	log.Debugf("NetDevStats - RECEIVE Count of return status %d", len(arrNetReceiveStats))
	log.Debugf("NetDevStats - TRANSFER Count of return status %d", len(arrNetTransferStats))

	return arrNetReceiveStats, arrNetTransferStats
}
