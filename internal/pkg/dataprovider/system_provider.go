package dataprovider

import (
	"fmt"

	aero "github.com/aerospike/aerospike-client-go/v6"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/systeminfo"
)

// Inherits DataProvider interface
type SystemInfoProvider struct {
}

func (asm SystemInfoProvider) RequestInfo(infoKeys []string) (map[string]string, error) {
	fetchSystemStats()
	systeminfo.GetDiskStats()

	return nil, nil
}

func (asm SystemInfoProvider) FetchUsersDetails() (bool, []*aero.UserRoles, error) {
	return false, nil, nil
}

// System server info

func fetchSystemStats() {
	fmt.Println("\n\n* ===============================================================================================")
	fmt.Println(systeminfo.GetMemInfo())
}
