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

	if err := scanner.Err(); err != nil {
		log.Error("Error while reading file,", fileName, " Error: ", err)
		return fileFDStats
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

func (sip SystemInfoProvider) GetIcsStats() map[string]string {
	fileIcsStats := make(map[string]string)

	fileName := getProcFilePath("sysvipc/shm")

	file, err := os.Open(fileName)

	if err != nil {
		log.Error("Error while opening file,", fileName, " Error: ", err)
		return fileIcsStats
	}

	defer file.Close() //nolint:errcheck

	scanner := bufio.NewScanner(file)

	lineNo := 0

	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())

		if line == "" {
			continue
		}

		// skip header line
		if lineNo == 1 && strings.HasPrefix(line, "key") {
			continue
		}

		fieldsIcs := strings.Fields(line)
		if len(fieldsIcs) < 16 {
			log.Errorf("Error while reading file, bad shm line %d: got %d fields", lineNo, len(fieldsIcs))
			return fileIcsStats
		}

		// key, _ := strconv.ParseInt(f[0], 10, 64)
		// shmid, _ := strconv.ParseInt(f[1], 10, 64)
		// size, _ := strconv.ParseUint(f[3], 10, 64)
		// cpid, _ := strconv.ParseInt(f[4], 10, 64)
		// lpid, _ := strconv.ParseInt(f[5], 10, 64)
		// nattch, _ := strconv.ParseUint(f[6], 10, 64)
		// uid, _ := strconv.ParseUint(f[7], 10, 64)
		// gid, _ := strconv.ParseUint(f[8], 10, 64)
		// cuid, _ := strconv.ParseUint(f[9], 10, 64)
		// cgid, _ := strconv.ParseUint(f[10], 10, 64)
		// atime, _ := strconv.ParseInt(f[11], 10, 64)
		// dtime, _ := strconv.ParseInt(f[12], 10, 64)
		// ctime, _ := strconv.ParseInt(f[13], 10, 64)
		// rss, _ := strconv.ParseUint(f[14], 10, 64)
		// swap, _ := strconv.ParseUint(f[15], 10, 64)

		fileIcsStats["key"] = fieldsIcs[0]
		fileIcsStats["shmid"] = fieldsIcs[1]
		fileIcsStats["size"] = fieldsIcs[3]
		fileIcsStats["cpid"] = fieldsIcs[4]
		fileIcsStats["lpid"] = fieldsIcs[5]
		fileIcsStats["nattch"] = fieldsIcs[6]
		fileIcsStats["uid"] = fieldsIcs[7]
		fileIcsStats["gid"] = fieldsIcs[8]
		fileIcsStats["cuid"] = fieldsIcs[9]

	}

	if err := scanner.Err(); err != nil {
		log.Error("Error while reading file,", fileName, " Error: ", err)
		return fileIcsStats
	}

	log.Debugf("FileFD Stats - Count of return stats %d", len(fileIcsStats))

	return fileIcsStats
}
