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

	if err != nil {
		log.Debug("Eror while reading procfs.NewFS system,  error: ", err)
		return MeminfoStats{nil}
	}

	meminfo, err := fs.Meminfo()
	if err != nil {
		log.Debug("Eror while reading MemInfo, error: ", err)
		return MeminfoStats{nil}
	}

	memStats := make(map[string]float64)

	memStats["Active"] = GetFloatValue(meminfo.Active)
	memStats["Active_Anon"] = GetFloatValue(meminfo.ActiveAnon)
	memStats["Active_File"] = GetFloatValue(meminfo.ActiveFile)
	memStats["Anon_Pages"] = GetFloatValue(meminfo.AnonPages)
	memStats["Anon_Huge_Pages"] = GetFloatValue(meminfo.AnonHugePages)
	memStats["Bounce"] = GetFloatValue(meminfo.Bounce)
	memStats["Buffers"] = GetFloatValue(meminfo.Buffers)
	memStats["Cached"] = GetFloatValue(meminfo.Cached)
	memStats["CmaFree"] = GetFloatValue(meminfo.CmaFree)
	memStats["CmaTotal"] = GetFloatValue(meminfo.CmaTotal)
	memStats["Commit_Limit"] = GetFloatValue(meminfo.CommitLimit)
	memStats["Committed_AS"] = GetFloatValue(meminfo.CommittedAS)
	memStats["Direct_Map1G"] = GetFloatValue(meminfo.DirectMap1G)
	memStats["Direct_Map2M"] = GetFloatValue(meminfo.DirectMap2M)
	memStats["Direct_Map4k"] = GetFloatValue(meminfo.DirectMap4k)
	memStats["Dirty"] = GetFloatValue(meminfo.Dirty)
	memStats["Hardware_Corrupted"] = GetFloatValue(meminfo.HardwareCorrupted)
	memStats["Huge_Pages_Free"] = GetFloatValue(meminfo.HugePagesFree)
	memStats["Huge_Pages_Rsvd"] = GetFloatValue(meminfo.HugePagesRsvd)
	memStats["Huge_Pages_Surp"] = GetFloatValue(meminfo.HugePagesSurp)
	memStats["Huge_Pages_Total"] = GetFloatValue(meminfo.HugePagesTotal)
	memStats["Huge_page_size"] = GetFloatValue(meminfo.Hugepagesize)
	memStats["Inactive"] = GetFloatValue(meminfo.Inactive)
	memStats["Inactive_Anon"] = GetFloatValue(meminfo.InactiveAnon)
	memStats["Inactive_File"] = GetFloatValue(meminfo.InactiveFile)
	memStats["Kernel_Stack"] = GetFloatValue(meminfo.KernelStack)
	memStats["Mapped"] = GetFloatValue(meminfo.Mapped)
	memStats["Mem_Available"] = GetFloatValue(meminfo.MemAvailable)
	memStats["Mem_Free"] = GetFloatValue(meminfo.MemFree)
	memStats["Mem_Total"] = GetFloatValue(meminfo.MemTotal)
	memStats["Mlocked"] = GetFloatValue(meminfo.Mlocked)
	memStats["NFS_Unstable"] = GetFloatValue(meminfo.NFSUnstable)
	memStats["Page_Tables"] = GetFloatValue(meminfo.PageTables)
	memStats["SReclaimable"] = GetFloatValue(meminfo.SReclaimable)
	memStats["SUnreclaim"] = GetFloatValue(meminfo.SUnreclaim)
	memStats["Shmem"] = GetFloatValue(meminfo.Shmem)
	memStats["Shmem_Huge_Pages"] = GetFloatValue(meminfo.ShmemHugePages)
	memStats["Shmem_Pmd_Mapped"] = GetFloatValue(meminfo.ShmemPmdMapped)
	memStats["Slab"] = GetFloatValue(meminfo.Slab)
	memStats["Swap_Cached"] = GetFloatValue(meminfo.SwapCached)
	memStats["Swap_Free"] = GetFloatValue(meminfo.SwapFree)
	memStats["Swap_Total"] = GetFloatValue(meminfo.SwapTotal)
	memStats["Unevictable"] = GetFloatValue(meminfo.Unevictable)
	memStats["Vmalloc_Chunk"] = GetFloatValue(meminfo.VmallocChunk)
	memStats["Vmalloc_Total"] = GetFloatValue(meminfo.VmallocTotal)
	memStats["Vmalloc_Used"] = GetFloatValue(meminfo.VmallocUsed)
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
