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
	ICS_SHM_SINDEX_PREFIX  = "si"
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

// IsAerospikeShmSegment reports whether a sysvipc shm key belongs to Aerospike,
// based on the encoded key prefix (PI/base=0xae, sindex=0xa2, data=0xad).
// This works across host, VM, Docker, and K8s sidecars without relying on cpid/lpid.
func IsAerospikeShmSegment(key int32) bool {
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

func parseSysVSharedMemInfo(shmFields []string) (map[string]string, error) {

	// parse each field, if any field is invalid, return an error
	// total 16 fields - key,shmid,perms,size,cpid,lpid,nattch,uid,gid,cuid,cgid,atime,dtime,ctime,rss,swap
	key, err := strconv.ParseInt(shmFields[0], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("key %q: %w", shmFields[0], err)
	}

	shmid, err := strconv.ParseInt(shmFields[1], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("shmid %q: %w", shmFields[1], err)
	}

	size, err := strconv.ParseUint(shmFields[3], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("size %q: %w", shmFields[3], err)
	}

	cpid, err := strconv.ParseInt(shmFields[4], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("cpid %q: %w", shmFields[4], err)
	}

	lpid, err := strconv.ParseInt(shmFields[5], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("lpid %q: %w", shmFields[5], err)
	}

	nattch, err := strconv.ParseUint(shmFields[6], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("nattch %q: %w", shmFields[6], err)
	}

	uid, err := strconv.ParseUint(shmFields[7], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("uid %q: %w", shmFields[7], err)
	}

	gid, err := strconv.ParseUint(shmFields[8], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("gid %q: %w", shmFields[8], err)
	}

	cuid, err := strconv.ParseUint(shmFields[9], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("cuid %q: %w", shmFields[9], err)
	}

	cgid, err := strconv.ParseUint(shmFields[10], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("cgid %q: %w", shmFields[10], err)
	}

	atime, err := strconv.ParseInt(shmFields[11], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("atime %q: %w", shmFields[11], err)
	}

	dtime, err := strconv.ParseInt(shmFields[12], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("dtime %q: %w", shmFields[12], err)
	}

	ctime, err := strconv.ParseInt(shmFields[13], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("ctime %q: %w", shmFields[13], err)
	}

	rss, err := strconv.ParseUint(shmFields[14], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("rss %q: %w", shmFields[14], err)
	}

	swap, err := strconv.ParseUint(shmFields[15], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("swap %q: %w", shmFields[15], err)
	}

	hexKey, prefix, typeid, instanceID, namespaceID, suffix := decodeAerospikeShmKey(int32(key))

	return map[string]string{
		"key":         strconv.FormatInt(key, 10),
		"shmid":       strconv.FormatInt(shmid, 10),
		"size":        strconv.FormatUint(size, 10),
		"cpid":        strconv.FormatInt(cpid, 10),
		"lpid":        strconv.FormatInt(lpid, 10),
		"nattch":      strconv.FormatUint(nattch, 10),
		"uid":         strconv.FormatUint(uid, 10),
		"gid":         strconv.FormatUint(gid, 10),
		"cuid":        strconv.FormatUint(cuid, 10),
		"cgid":        strconv.FormatUint(cgid, 10),
		"atime":       strconv.FormatInt(atime, 10),
		"dtime":       strconv.FormatInt(dtime, 10),
		"ctime":       strconv.FormatInt(ctime, 10),
		"rss":         strconv.FormatUint(rss, 10),
		"swap":        strconv.FormatUint(swap, 10),
		"hexKey":      hexKey,
		"prefix":      prefix,
		"typeid":      typeid,
		"instanceid":  instanceID,
		"namespaceid": namespaceID,
		"suffix":      suffix,
	}, nil
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
