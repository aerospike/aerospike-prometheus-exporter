package systeminfo

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/prometheus/procfs"
)

var (
	// The path of the proc filesystem.
	// procPath = kingpin.Flag("path.procfs", "procfs mountpoint.").Default(procfs.DefaultMountPoint).String()
	procPath = procfs.DefaultMountPoint
	sysPath  = "/sys"
	// rootfsPath   = kingpin.Flag("path.rootfs", "rootfs mountpoint.").Default("/").String()
	udevDataPath = "/run/udev/data"
)

func GetProcFilePath(name string) string {
	return filepath.Join(procPath, name)
}

func GetSysFilePath(name string) string {
	return filepath.Join(sysPath, name)
}

func GetUdevDataFilePath(name string) string {
	return filepath.Join(udevDataPath, name)
}

func handleError(e error) {
	fmt.Println("Error :- ", e)
}

// func rootfsFilePath(name string) string {
// 	return filepath.Join(*rootfsPath, name)
// }

// func udevDataFilePath(name string) string {
// 	return filepath.Join(*udevDataPath, name)
// }

// func rootfsStripPrefix(path string) string {
// 	if *rootfsPath == "/" {
// 		return path
// 	}
// 	stripped := strings.TrimPrefix(path, *rootfsPath)
// 	if stripped == "" {
// 		return "/"
// 	}
// 	return stripped
// }

func getUdevDeviceProperties(major, minor uint32) (map[string]string, error) {
	filename := GetUdevDataFilePath(fmt.Sprintf("b%d:%d", major, minor))

	data, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer data.Close()

	info := make(map[string]string)

	scanner := bufio.NewScanner(data)
	for scanner.Scan() {
		line := scanner.Text()

		// We're only interested in device properties.
		if !strings.HasPrefix(line, udevDevicePropertyPrefix) {
			continue
		}

		line = strings.TrimPrefix(line, udevDevicePropertyPrefix)

		if name, value, found := strings.Cut(line, "="); found {
			info[name] = value
		}
	}

	return info, nil
}

func GetFloatValue(addr *uint64) float64 {
	if addr != nil {
		value := float64(*addr)
		return value
	}
	return -1.0
}
