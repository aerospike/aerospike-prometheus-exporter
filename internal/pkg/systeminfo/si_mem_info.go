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

	// All values are in KB, convert to bytes
	memStats["Active"] = GetFloatValue(meminfo.Active) * 1024
	memStats["Active_Anon"] = GetFloatValue(meminfo.ActiveAnon) * 1024
	memStats["Active_File"] = GetFloatValue(meminfo.ActiveFile) * 1024
	memStats["Anon_Pages"] = GetFloatValue(meminfo.AnonPages) * 1024
	memStats["Anon_Huge_Pages"] = GetFloatValue(meminfo.AnonHugePages) * 1024
	memStats["Bounce"] = GetFloatValue(meminfo.Bounce) * 1024
	memStats["Buffers"] = GetFloatValue(meminfo.Buffers) * 1024
	memStats["Cached"] = GetFloatValue(meminfo.Cached) * 1024
	memStats["CmaFree"] = GetFloatValue(meminfo.CmaFree) * 1024
	memStats["CmaTotal"] = GetFloatValue(meminfo.CmaTotal) * 1024
	memStats["Commit_Limit"] = GetFloatValue(meminfo.CommitLimit) * 1024
	memStats["Committed_AS"] = GetFloatValue(meminfo.CommittedAS) * 1024
	memStats["Direct_Map1G"] = GetFloatValue(meminfo.DirectMap1G) * 1024
	memStats["Direct_Map2M"] = GetFloatValue(meminfo.DirectMap2M) * 1024
	memStats["Direct_Map4k"] = GetFloatValue(meminfo.DirectMap4k) * 1024
	memStats["Dirty"] = GetFloatValue(meminfo.Dirty) * 1024
	memStats["Hardware_Corrupted"] = GetFloatValue(meminfo.HardwareCorrupted) * 1024
	memStats["Huge_Pages_Free"] = GetFloatValue(meminfo.HugePagesFree) * 1024
	memStats["Huge_Pages_Rsvd"] = GetFloatValue(meminfo.HugePagesRsvd) * 1024
	memStats["Huge_Pages_Surp"] = GetFloatValue(meminfo.HugePagesSurp) * 1024
	memStats["Huge_Pages_Total"] = GetFloatValue(meminfo.HugePagesTotal) * 1024
	memStats["Huge_page_size"] = GetFloatValue(meminfo.Hugepagesize) * 1024
	memStats["Inactive"] = GetFloatValue(meminfo.Inactive) * 1024
	memStats["Inactive_Anon"] = GetFloatValue(meminfo.InactiveAnon) * 1024
	memStats["Inactive_File"] = GetFloatValue(meminfo.InactiveFile) * 1024
	memStats["Kernel_Stack"] = GetFloatValue(meminfo.KernelStack) * 1024
	memStats["Mapped"] = GetFloatValue(meminfo.Mapped) * 1024
	memStats["Mem_Available"] = GetFloatValue(meminfo.MemAvailable) * 1024
	memStats["Mem_Free"] = GetFloatValue(meminfo.MemFree) * 1024
	memStats["Mem_Total"] = GetFloatValue(meminfo.MemTotal) * 1024
	memStats["Mlocked"] = GetFloatValue(meminfo.Mlocked) * 1024
	memStats["NFS_Unstable"] = GetFloatValue(meminfo.NFSUnstable) * 1024
	memStats["Page_Tables"] = GetFloatValue(meminfo.PageTables) * 1024
	memStats["SReclaimable"] = GetFloatValue(meminfo.SReclaimable) * 1024
	memStats["SUnreclaim"] = GetFloatValue(meminfo.SUnreclaim) * 1024
	memStats["Shmem"] = GetFloatValue(meminfo.Shmem) * 1024
	memStats["Shmem_Huge_Pages"] = GetFloatValue(meminfo.ShmemHugePages) * 1024
	memStats["Shmem_Pmd_Mapped"] = GetFloatValue(meminfo.ShmemPmdMapped) * 1024
	memStats["Slab"] = GetFloatValue(meminfo.Slab) * 1024
	memStats["Swap_Cached"] = GetFloatValue(meminfo.SwapCached) * 1024
	memStats["Swap_Free"] = GetFloatValue(meminfo.SwapFree) * 1024
	memStats["Swap_Total"] = GetFloatValue(meminfo.SwapTotal) * 1024
	memStats["Unevictable"] = GetFloatValue(meminfo.Unevictable) * 1024
	memStats["Vmalloc_Chunk"] = GetFloatValue(meminfo.VmallocChunk) * 1024
	memStats["Vmalloc_Total"] = GetFloatValue(meminfo.VmallocTotal) * 1024
	memStats["Vmalloc_Used"] = GetFloatValue(meminfo.VmallocUsed) * 1024
	memStats["Writeback"] = GetFloatValue(meminfo.Writeback) * 1024
	memStats["WritebackTmp"] = GetFloatValue(meminfo.WritebackTmp) * 1024

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
