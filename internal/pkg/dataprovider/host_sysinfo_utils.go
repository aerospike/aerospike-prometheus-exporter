package dataprovider

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/prometheus/procfs"
	log "github.com/sirupsen/logrus"
)

const (
	// DISK_IGNORE_NAME_LIST = "^(z?ram|loop|fd|(h|s|v|xv)d[a-z]|nvme\\d+n\\d+p)\\d$"
	DISK_IGNORE_NAME_LIST = "^(z?ram|loop|fd|(h|s|xv)d[a-z]|nvme\\d+n\\d+p)\\d$"
	// FILE_STAT_IGNORE_LIST = "^(overlay|mqueue)$"
	FILE_STAT_IGNORE_LIST = "^(mqueue)$"

	// NET_STAT_ACCEPT_LIST = "^(.*_(inerrors|inerrs)|ip_forwarding|ip(6|ext)_(inoctets|outoctets)|icmp6?_(inmsgs|outmsgs)|tcpext_(listen.*|syncookies.*|tcpsynretrans|tcptimeouts|tcpofoqueue)|tcp_(activeopens|insegs|outsegs|outrsts|passiveopens|retranssegs|currestab)|udp6?_(indatagrams|outdatagrams|noports|rcvbuferrors|sndbuferrors))$"
	NET_STAT_ACCEPT_LIST = "tcp_(activeopens|retranssegs|currestab)"
)

var (
	PROC_PATH         = procfs.DefaultMountPoint
	SYS_PATH          = "/sys"
	ROOTFS_PATH       = "/"
	NET_STAT_PATH     = "net/netstat"
	NET_DEV_STAT_PATH = "/proc/net/dev"
	ICS_SHM_PATH      = "sysvipc/shm" // actual path is /proc/sysvipc/shm
)

const (
	SECONDS_PER_TICK = 0.0001 // 1000 ticks per second

	// Read/write sectors are "standard UNIX 512-byte sectors" (https://www.kernel.org/doc/Documentation/block/stat.txt)
	DISK_SECTOR_SIZE_IN_UNIX = 512.0

	ONE_KILO_BYTE = 1024
)

const (
	ICS_SHM_PI_BASE_PREFIX = "pi"
	ICS_SHM_SINDEX_PREFIX  = "sindex"
	ICS_SHM_DATA_PREFIX    = "data"
	ICS_SHM_OTHER_PREFIX   = "other"
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

	if err := scanner.Err(); err != nil {
		log.Error("Error while reading file,", fileName, " Error: ", err)
		return arrSysInfoStats
	}

	return arrSysInfoStats
}

func parseIcsKey(icsShmFields []string) map[string]string {

	fileIcsStats := make(map[string]string)

	// key:-1392500736
	// cat / proc / sysvipc / shm
	// 	key      shmid perms                  size  cpid  lpid nattch   uid   gid  cuid  cgid      atime      dtime      ctime                   rss                  swap
	// -1375727360          0   666            1073741824   152   288      1     0     0     0     0 1781083729 1781083717 1781083714                  4096                     0
	// -1392504832          1   666             536870912   152   288      1     0     0     0     0 1781083729 1781083717 1781083714               8388608                     0
	// -1392504831          2   666             536870912   152   288      1     0     0     0     0 1781083729 1781083717 1781083714               8388608                     0
	// -1392504830          3   666             536870912   152   288      1     0     0     0     0 1781083729 1781083717 1781083714               8388608                     0
	// -1392504829          4   666             536870912   152   288      1     0     0     0     0 1781083729 1781083717 1781083714               8388608                     0
	// -1392504828          5   666             536870912   152   288      1     0     0     0     0 1781083729 1781083717 1781083714               8388608                     0
	// -1392504827          6   666             536870912   152   288      1     0     0     0     0 1781083729 1781083717 1781083714               8388608                     0
	// -1392504826          7   666             536870912   152   288      1     0     0     0     0 1781083729 1781083717 1781083714               8388608                     0
	// -1392504825          8   666             536870912   152   288      1     0     0     0     0 1781083729 1781083717 1781083714               8388608                     0

	// 0xae -> PI/base
	// 0xa2 -> SI
	// 0xad -> data

	// for sindex its 0xa2 + 1 letter for instanceid + 2 letters for namespaceid + 3 digits for stages
	// for data in shmem
	// 0xad + 1 letter for instance id + 2 letters for namespaceID + 3 (last bit for stripe ids 0,1,2,3)

	key, _ := strconv.ParseInt(icsShmFields[0], 10, 64)
	shmid, _ := strconv.ParseInt(icsShmFields[1], 10, 64)
	size, _ := strconv.ParseUint(icsShmFields[3], 10, 64)
	cpid, _ := strconv.ParseInt(icsShmFields[4], 10, 64)
	lpid, _ := strconv.ParseInt(icsShmFields[5], 10, 64)
	nattch, _ := strconv.ParseUint(icsShmFields[6], 10, 64)
	uid, _ := strconv.ParseUint(icsShmFields[7], 10, 64)
	gid, _ := strconv.ParseUint(icsShmFields[8], 10, 64)
	cuid, _ := strconv.ParseUint(icsShmFields[9], 10, 64)
	cgid, _ := strconv.ParseUint(icsShmFields[10], 10, 64)
	atime, _ := strconv.ParseInt(icsShmFields[11], 10, 64)
	dtime, _ := strconv.ParseInt(icsShmFields[12], 10, 64)
	ctime, _ := strconv.ParseInt(icsShmFields[13], 10, 64)
	rss, _ := strconv.ParseUint(icsShmFields[14], 10, 64)
	swap, _ := strconv.ParseUint(icsShmFields[15], 10, 64)

	fileIcsStats["key"] = strconv.FormatInt(int64(key), 10)
	fileIcsStats["shmid"] = strconv.FormatInt(int64(shmid), 10)
	fileIcsStats["size"] = strconv.FormatUint(size, 10)
	fileIcsStats["cpid"] = strconv.FormatInt(int64(cpid), 10)
	fileIcsStats["lpid"] = strconv.FormatInt(int64(lpid), 10)
	fileIcsStats["nattch"] = strconv.FormatUint(nattch, 10)
	fileIcsStats["uid"] = strconv.FormatUint(uid, 10)
	fileIcsStats["gid"] = strconv.FormatUint(gid, 10)
	fileIcsStats["cuid"] = strconv.FormatUint(cuid, 10)
	fileIcsStats["cgid"] = strconv.FormatUint(cgid, 10)
	fileIcsStats["atime"] = strconv.FormatInt(atime, 10)
	fileIcsStats["dtime"] = strconv.FormatInt(dtime, 10)
	fileIcsStats["ctime"] = strconv.FormatInt(ctime, 10)
	fileIcsStats["rss"] = strconv.FormatUint(rss, 10)
	fileIcsStats["swap"] = strconv.FormatUint(swap, 10)

	hexKey, prefix, typeid, instanceID, namespaceID, suffix := decodeAerospikeShmKey(int32(key))

	fileIcsStats["hexKey"] = hexKey
	fileIcsStats["prefix"] = prefix
	fileIcsStats["typeid"] = typeid
	fileIcsStats["instanceid"] = instanceID
	fileIcsStats["namespaceid"] = namespaceID
	fileIcsStats["suffix"] = suffix

	return fileIcsStats
}

func decodeAerospikeShmKey(raw int32) (string, string, string, string, string, string) {
	u := uint32(raw)

	prefix := byte(u >> 24)

	var typeid string

	switch prefix {
	case 0xae:
		typeid = ICS_SHM_PI_BASE_PREFIX
	case 0xa2:
		typeid = ICS_SHM_SINDEX_PREFIX
	case 0xad:
		typeid = ICS_SHM_DATA_PREFIX
	default:
		typeid = ICS_SHM_OTHER_PREFIX
	}

	hexKey := fmt.Sprintf("0x%08x", u)
	instanceID := strconv.FormatUint(uint64(((u >> 20) & 0xF)), 10)
	namespaceID := strconv.FormatUint(uint64(((u >> 12) & 0xFF)), 10)
	suffix := strconv.FormatUint(uint64((u & 0xFFF)), 10)

	// rawKey = strconv.FormatInt(int64(raw), 10),
	return hexKey, fmt.Sprintf("0x%02x", prefix),
		typeid, instanceID, namespaceID, suffix

}
