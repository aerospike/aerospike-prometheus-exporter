package dataprovider

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/prometheus/procfs"
	log "github.com/sirupsen/logrus"
)

var (
	PROC_PATH         = procfs.DefaultMountPoint
	SYS_PATH          = "/sys"
	ROOTFS_PATH       = "/"
	NET_STAT_PATH     = "net/netstat"
	NET_DEV_STAT_PATH = "/proc/net/dev"
)

const (
	// DISK_IGNORE_NAME_LIST = "^(z?ram|loop|fd|(h|s|v|xv)d[a-z]|nvme\\d+n\\d+p)\\d$"
	DISK_IGNORE_NAME_LIST = "^(z?ram|loop|fd|(h|s|xv)d[a-z]|nvme\\d+n\\d+p)\\d$"
	// FILE_STAT_IGNORE_LIST = "^(overlay|mqueue)$"
	FILE_STAT_IGNORE_LIST = "^(mqueue)$"

	// NET_STAT_ACCEPT_LIST = "^(.*_(inerrors|inerrs)|ip_forwarding|ip(6|ext)_(inoctets|outoctets)|icmp6?_(inmsgs|outmsgs)|tcpext_(listen.*|syncookies.*|tcpsynretrans|tcptimeouts|tcpofoqueue)|tcp_(activeopens|insegs|outsegs|outrsts|passiveopens|retranssegs|currestab)|udp6?_(indatagrams|outdatagrams|noports|rcvbuferrors|sndbuferrors))$"
	NET_STAT_ACCEPT_LIST = "tcp_(activeopens|retranssegs|currestab)"
)

const (
	SECONDS_PER_TICK = 0.0001 // 1000 ticks per second

	// Read/write sectors are "standard UNIX 512-byte sectors" (https://www.kernel.org/doc/Documentation/block/stat.txt)
	DISK_SECTOR_SIZE_IN_UNIX = 512.0

	ONE_KILO_BYTE = 1024
)

var (
	regexNetstatAcceptPattern = regexp.MustCompile(NET_STAT_ACCEPT_LIST)
)

func getProcFilePath(name string) string {
	return filepath.Join(PROC_PATH, name)
}

func acceptNetstat(name string) bool {
	return (regexNetstatAcceptPattern != nil && regexNetstatAcceptPattern.MatchString(name))
}

func getFloatValue(addr *uint64) float64 {
	if addr != nil {
		value := float64(*addr)
		return value
	}
	return 0.0
}

func parseNetStats(fileName string) map[string]string {
	arrSysInfoStats := make(map[string]string)

	file, err := os.Open(fileName)
	if err != nil {
		log.Error("Error while opening file,", fileName, " Error: ", err)
		return arrSysInfoStats
	}

	defer file.Close() //nolint:errcheck

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		statNames := strings.Split(scanner.Text(), " ")
		scanner.Scan()
		values := strings.Split(scanner.Text(), " ")
		protocol := statNames[0][:len(statNames[0])-1]
		if len(statNames) != len(values) {
			return arrSysInfoStats
		}
		for i := 1; i < len(statNames); i++ {
			key := strings.ToLower(protocol + "_" + statNames[i])
			if acceptNetstat(key) {
				arrSysInfoStats[key] = values[i]
			}
		}
	}

	return arrSysInfoStats
}
