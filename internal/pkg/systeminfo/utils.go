package systeminfo

import (
	"path/filepath"

	"github.com/prometheus/procfs"
)

var (
	// The path of the proc filesystem.
	// procPath = kingpin.Flag("path.procfs", "procfs mountpoint.").Default(procfs.DefaultMountPoint).String()
	procPath = procfs.DefaultMountPoint
	// sysPath      = kingpin.Flag("path.sysfs", "sysfs mountpoint.").Default("/sys").String()
	// rootfsPath   = kingpin.Flag("path.rootfs", "rootfs mountpoint.").Default("/").String()
	// udevDataPath = kingpin.Flag("path.udev.data", "udev data path.").Default("/run/udev/data").String()
)

func procFilePath(name string) string {
	return filepath.Join(procPath, name)
}

// func sysFilePath(name string) string {
// 	return filepath.Join(*sysPath, name)
// }

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
