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
)

var (
	reParens = regexp.MustCompile(`\((.*)\)`)
)

func GetMemInfo() (map[string]float64, error) {
	stats := GetMemInfoUsingProcFS()

	return stats, nil
}

func GetMemInfoUsingProcFS() map[string]float64 {
	fs, err := procfs.NewFS("/proc")
	handleError(err)

	meminfo, err := fs.Meminfo()
	handleError(err)

	memStats := make(map[string]float64)

	memStats["Active"] = float64(*meminfo.Active)
	memStats["ActiveAnon"] = float64(*meminfo.ActiveAnon)
	memStats["ActiveFile"] = float64(*meminfo.ActiveFile)
	memStats["AnonHugePages"] = float64(*meminfo.AnonHugePages)
	memStats["Bounce"] = float64(*meminfo.Bounce)
	memStats["Buffers"] = float64(*meminfo.Buffers)
	memStats["Cached"] = float64(*meminfo.Cached)
	memStats["CmaFree"] = float64(*meminfo.CmaFree)
	memStats["CmaTotal"] = float64(*meminfo.CmaTotal)
	memStats["CommitLimit"] = float64(*meminfo.CommitLimit)
	memStats["CommittedAS"] = float64(*meminfo.CommittedAS)
	memStats["DirectMap1G"] = float64(*meminfo.DirectMap1G)
	memStats["DirectMap1G"] = float64(*meminfo.DirectMap1G)
	memStats["DirectMap4k"] = float64(*meminfo.DirectMap4k)
	memStats["Dirty"] = float64(*meminfo.Dirty)
	memStats["HardwareCorrupted"] = float64(*meminfo.HardwareCorrupted)
	memStats["HugePagesFree"] = float64(*meminfo.HugePagesFree)
	memStats["HugePagesRsvd"] = float64(*meminfo.HugePagesRsvd)
	memStats["HugePagesSurp"] = float64(*meminfo.HugePagesSurp)
	memStats["HugePagesTotal"] = float64(*meminfo.HugePagesTotal)
	memStats["Hugepagesize"] = float64(*meminfo.Hugepagesize)
	memStats["Inactive"] = float64(*meminfo.Inactive)
	memStats["Inactive"] = float64(*meminfo.Inactive)
	memStats["InactiveFile"] = float64(*meminfo.InactiveFile)
	memStats["KernelStack"] = float64(*meminfo.KernelStack)
	memStats["Mapped"] = float64(*meminfo.Mapped)
	memStats["MemAvailable"] = float64(*meminfo.MemAvailable)
	memStats["MemFree"] = float64(*meminfo.MemFree)
	memStats["MemTotal"] = float64(*meminfo.MemTotal)
	memStats["Mlocked"] = float64(*meminfo.Mlocked)
	memStats["NFSUnstable"] = float64(*meminfo.NFSUnstable)
	memStats["PageTables"] = float64(*meminfo.PageTables)
	memStats["SReclaimable"] = float64(*meminfo.SReclaimable)
	memStats["SUnreclaim"] = float64(*meminfo.SUnreclaim)
	memStats["Shmem"] = float64(*meminfo.Shmem)
	memStats["ShmemHugePages"] = float64(*meminfo.ShmemHugePages)
	memStats["ShmemPmdMapped"] = float64(*meminfo.ShmemPmdMapped)
	memStats["Slab"] = float64(*meminfo.Slab)
	memStats["SwapCached"] = float64(*meminfo.SwapCached)
	memStats["SwapFree"] = float64(*meminfo.SwapFree)
	memStats["SwapTotal"] = float64(*meminfo.SwapTotal)
	memStats["Unevictable"] = float64(*meminfo.Unevictable)
	memStats["VmallocChunk"] = float64(*meminfo.VmallocChunk)
	memStats["VmallocTotal"] = float64(*meminfo.VmallocTotal)
	memStats["VmallocUsed"] = float64(*meminfo.VmallocUsed)
	memStats["Writeback"] = float64(*meminfo.Writeback)
	memStats["WritebackTmp"] = float64(*meminfo.WritebackTmp)

	return memStats
}

func GetMemInfoParsingFile() (map[string]float64, error) {
	pathMeminfo := GetProcFilePath("meminfo")
	fmt.Println("\t\t *** pathMeminfo : ", pathMeminfo)
	file, err := os.Open(pathMeminfo)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// testing

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
