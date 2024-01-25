package systeminfo

import (
	"fmt"

	"github.com/lufia/iostat"
)

// func GetDiskStats() ([]*iostat.DriveStats, error) {
func GetDiskStats() {
	driveStats, err := parseDiskStats()
	fmt.Println(driveStats[0].Name)
	fmt.Println("error ", err)
}

func parseDiskStats() ([]*iostat.DriveStats, error) {
	driveStats, err := iostat.ReadDriveStats()

	return driveStats, err
}
