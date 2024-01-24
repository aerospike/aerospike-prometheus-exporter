package dataprovider

import (
	"fmt"

	aero "github.com/aerospike/aerospike-client-go/v6"
	"github.com/prometheus/procfs"
)

// Inherits DataProvider interface
type SystemInfoProvider struct {
}

func (asm SystemInfoProvider) RequestInfo(infoKeys []string) (map[string]string, error) {
	fetchProcStats()

	return nil, nil
}

func (asm SystemInfoProvider) FetchUsersDetails() (bool, []*aero.UserRoles, error) {
	return false, nil, nil
}

// System server info

func fetchProcStats() {
	fs, err := procfs.NewFS("/proc")
	handleError(err)
	stats, err := fs.Stat()
	handleError(err)
	fmt.Print(stats.CPU)
}

func handleError(e error) {
	fmt.Println("Error: ", e)
}
