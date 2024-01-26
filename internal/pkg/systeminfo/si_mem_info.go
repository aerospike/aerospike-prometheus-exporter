package systeminfo

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/prometheus/procfs"
	log "github.com/sirupsen/logrus"
)

type MeminfoStats struct {
	mem_stats map[string]float64
}

var (
	reParens = regexp.MustCompile(`\((.*)\)`)
)

func GetMemInfo() MeminfoStats {
	stats := GetMemInfoUsingProcFS()

	return stats
}

func GetMemInfoUsingProcFS() MeminfoStats {
	fs, err := procfs.NewFS(PROC_PATH)
	handleError(err)

	meminfo, err := fs.Meminfo()
	if err != nil {
		log.Debug("Eror while reading MemInfo, error: ", err)
		return MeminfoStats{nil}
	}

	memStats := make(map[string]float64)

	memStats["Active"] = GetFloatValue(meminfo.Active)
	memStats["ActiveAnon"] = GetFloatValue(meminfo.ActiveAnon)
	memStats["ActiveFile"] = GetFloatValue(meminfo.ActiveFile)
	memStats["AnonHugePages"] = GetFloatValue(meminfo.AnonHugePages)
	memStats["Bounce"] = GetFloatValue(meminfo.Bounce)
	memStats["Buffers"] = GetFloatValue(meminfo.Buffers)
	memStats["Cached"] = GetFloatValue(meminfo.Cached)
	memStats["CmaFree"] = GetFloatValue(meminfo.CmaFree)
	memStats["CmaTotal"] = GetFloatValue(meminfo.CmaTotal)
	memStats["CommitLimit"] = GetFloatValue(meminfo.CommitLimit)
	memStats["CommittedAS"] = GetFloatValue(meminfo.CommittedAS)
	memStats["DirectMap1G"] = GetFloatValue(meminfo.DirectMap1G)
	memStats["DirectMap1G"] = GetFloatValue(meminfo.DirectMap1G)
	memStats["DirectMap4k"] = GetFloatValue(meminfo.DirectMap4k)
	memStats["Dirty"] = GetFloatValue(meminfo.Dirty)
	memStats["HardwareCorrupted"] = GetFloatValue(meminfo.HardwareCorrupted)
	memStats["HugePagesFree"] = GetFloatValue(meminfo.HugePagesFree)
	memStats["HugePagesRsvd"] = GetFloatValue(meminfo.HugePagesRsvd)
	memStats["HugePagesSurp"] = GetFloatValue(meminfo.HugePagesSurp)
	memStats["HugePagesTotal"] = GetFloatValue(meminfo.HugePagesTotal)
	memStats["Hugepagesize"] = GetFloatValue(meminfo.Hugepagesize)
	memStats["Inactive"] = GetFloatValue(meminfo.Inactive)
	memStats["Inactive"] = GetFloatValue(meminfo.Inactive)
	memStats["InactiveFile"] = GetFloatValue(meminfo.InactiveFile)
	memStats["KernelStack"] = GetFloatValue(meminfo.KernelStack)
	memStats["Mapped"] = GetFloatValue(meminfo.Mapped)
	memStats["MemAvailable"] = GetFloatValue(meminfo.MemAvailable)
	memStats["MemFree"] = GetFloatValue(meminfo.MemFree)
	memStats["MemTotal"] = GetFloatValue(meminfo.MemTotal)
	memStats["Mlocked"] = GetFloatValue(meminfo.Mlocked)
	memStats["NFSUnstable"] = GetFloatValue(meminfo.NFSUnstable)
	memStats["PageTables"] = GetFloatValue(meminfo.PageTables)
	memStats["SReclaimable"] = GetFloatValue(meminfo.SReclaimable)
	memStats["SUnreclaim"] = GetFloatValue(meminfo.SUnreclaim)
	memStats["Shmem"] = GetFloatValue(meminfo.Shmem)
	memStats["ShmemHugePages"] = GetFloatValue(meminfo.ShmemHugePages)
	memStats["ShmemPmdMapped"] = GetFloatValue(meminfo.ShmemPmdMapped)
	memStats["Slab"] = GetFloatValue(meminfo.Slab)
	memStats["SwapCached"] = GetFloatValue(meminfo.SwapCached)
	memStats["SwapFree"] = GetFloatValue(meminfo.SwapFree)
	memStats["SwapTotal"] = GetFloatValue(meminfo.SwapTotal)
	memStats["Unevictable"] = GetFloatValue(meminfo.Unevictable)
	memStats["VmallocChunk"] = GetFloatValue(meminfo.VmallocChunk)
	memStats["VmallocTotal"] = GetFloatValue(meminfo.VmallocTotal)
	memStats["VmallocUsed"] = GetFloatValue(meminfo.VmallocUsed)
	memStats["Writeback"] = GetFloatValue(meminfo.Writeback)
	memStats["WritebackTmp"] = GetFloatValue(meminfo.WritebackTmp)

	return MeminfoStats{memStats}
}

func GetMemInfoParsingFile() (map[string]float64, error) {
	pathMeminfo := GetProcFilePath("meminfo")
	fmt.Println("\t\t *** pathMeminfo : ", pathMeminfo)
	file, err := os.Open(pathMeminfo)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return parseMemInfo(file)
}

func parseMemInfo(r io.Reader) (map[string]float64, error) {
	var (
		memInfo = map[string]float64{}
		scanner = bufio.NewScanner(r)
	)

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		// Workaround for empty lines occasionally occur in CentOS 6.2 kernel 3.10.90.
		if len(parts) == 0 {
			continue
		}
		fv, err := strconv.ParseFloat(parts[1], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid value in meminfo: %w", err)
		}
		key := parts[0][:len(parts[0])-1] // remove trailing : from key
		// Active(anon) -> Active_anon
		key = reParens.ReplaceAllString(key, "_${1}")
		switch len(parts) {
		case 2: // no unit
		case 3: // has unit, we presume kB
			fv *= 1024
			key = key + "_bytes"
		default:
			return nil, fmt.Errorf("invalid line in meminfo: %s", line)
		}
		memInfo[key] = fv
	}

	return memInfo, scanner.Err()
}
