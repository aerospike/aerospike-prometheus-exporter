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

const (
	SECONDS_PER_TICK = 0.0001 // 1000 ticks per second

	// Read/write sectors are "standard UNIX 512-byte sectors" (https://www.kernel.org/doc/Documentation/block/stat.txt)
	DISK_SECTOR_SIZE_IN_UNIX = 512.0

	ONE_KILO_BYTE = 1024
)

const (
	ICS_SHM_PI_BASE_PREFIX = "pi"
	ICS_SHM_SINDEX_PREFIX  = "si"
	ICS_SHM_DATA_PREFIX    = "data"
	ICS_SHM_OTHER_PREFIX   = "other"
)

var (
	PROC_PATH         = procfs.DefaultMountPoint
	SYS_PATH          = "/sys"
	ROOTFS_PATH       = "/"
	NET_STAT_PATH     = "net/netstat"
	NET_DEV_STAT_PATH = "/proc/net/dev"
	ICS_SHM_PATH      = "sysvipc/shm" // actual path is /proc/sysvipc/shm
)

var (
	regexNetstatAcceptPattern = regexp.MustCompile(NET_STAT_ACCEPT_LIST)
)

var (
	shKeyNames = []string{"key", "shmid", "perms", "size", "cpid", "lpid", "nattch", "uid", "gid", "cuid", "cgid", "atime", "dtime", "ctime", "rss", "swap"}

	// index of int/uint keys in shmFields
	//   ignore key index 0, as it is the key itself
	shMemIntKeyIdx  = []int{1, 2, 4, 6, 11, 12, 13}
	shMemUIntKeyIdx = []int{3, 7, 8, 9, 10, 14, 15}
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

// IsAerospikeShmSegment reports whether a sysvipc shm key belongs to Aerospike,
// based on the encoded key prefix (PI/base=0xae, sindex=0xa2, data=0xad).
// This works across host, VM, Docker, and K8s sidecars without relying on cpid/lpid.
func IsAerospikeShmSegment(key int64) bool {
	switch byte(uint32(key) >> 24) {
	case 0xae, 0xa2, 0xad:
		return true
	default:
		return false
	}
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

func parseSysVSharedMemInfo(key int64, shmFields []string) map[string]string {
	// parse each field, if any field is invalid, return an error
	// total 16 fields - key,shmid,perms,size,cpid,lpid,nattch,uid,gid,cuid,cgid,atime,dtime,ctime,rss,swap
	// int-keys = {"key", "shmid", "perms", "cpid", "lpid", "atime", "dtime", "ctime"}
	// unsigned-int-keys {"size", "nattch", "uid", "gid", "cuid", "cgid", "rss", "swap"}

	arrSysInfoStats := make(map[string]string)

	// parse each field, if any field is invalid, return an error
	// NOTE: key is already parsed, so we are not re-parsing it here
	for _, keyIdx := range shMemIntKeyIdx {
		_, err := strconv.ParseInt(shmFields[keyIdx], 10, 64)
		if err != nil {
			log.Error("Error while parsing key: ", shKeyNames[keyIdx], " value: ", shmFields[keyIdx], " error: ", err)
			return nil
		}
		arrSysInfoStats[shKeyNames[keyIdx]] = shmFields[keyIdx] //strconv.FormatInt(value, 10)
	}

	for _, keyIdx := range shMemUIntKeyIdx {
		_, err := strconv.ParseUint(shmFields[keyIdx], 10, 64)
		if err != nil {
			log.Error("Error while parsing key: ", shKeyNames[keyIdx], " value: ", shmFields[keyIdx], " error: ", err)
			return nil
		}
		arrSysInfoStats[shKeyNames[keyIdx]] = shmFields[keyIdx]
	}

	hexKey, prefix, typeid, instanceID, namespaceID, suffix := decodeAerospikeShmKey(key)

	arrSysInfoStats["key"] = strconv.FormatInt(key, 10)
	arrSysInfoStats["hexKey"] = hexKey
	arrSysInfoStats["prefix"] = prefix
	arrSysInfoStats["typeid"] = typeid
	arrSysInfoStats["instanceid"] = instanceID
	arrSysInfoStats["namespaceid"] = namespaceID
	arrSysInfoStats["suffix"] = suffix

	return arrSysInfoStats
}

func decodeAerospikeShmKey(raw int64) (string, string, string, string, string, string) {
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
