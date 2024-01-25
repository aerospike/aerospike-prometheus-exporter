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
	pathMeminfo := GetProcFilePath("meminfo")
	fmt.Println("\t\t *** pathMeminfo : ", pathMeminfo)
	file, err := os.Open(pathMeminfo)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// testing
	testingProcsFS()

	return parseMemInfo(file)
}

func testingProcsFS() {
	fs, err := procfs.NewFS("/proc")
	handleError(err)
	stats, err := fs.Stat()
	handleError(err)
	fmt.Println("\t\t ====> stats : ", stats.CPU)

	// mem, err := procfs.NewFS(GetProcFilePath("meminfo"))
	meminfo, err := fs.Meminfo()
	handleError(err)
	fmt.Println("\t\t Memory free: unit64 ", meminfo.MemFree)
	fmt.Println("\t\t Memory free: float64 with pointer ", float64(*meminfo.MemFree))

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
