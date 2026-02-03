package dataprovider

import (
	aero "github.com/aerospike/aerospike-client-go/v8"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"
)

// //go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate
//
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 . DataProvider
type DataProvider interface {
	RequestInfo(infokeys []string) (map[string]string, error)
	FetchUsersDetails() (bool, []*aero.UserRoles, error)
}

// pre-create the instances
var dpAerospikeServer = &AerospikeServer{}
var dpMockServer = &MockAerospikeServer{}
var dpSysInfoProvider = &SystemInfoProvider{}

func GetProvider() DataProvider {

	if config.Cfg.Agent.UseMockDatasource {
		// initialize, internally it will check if already initialized
		dpMockServer.Initialize()

		// a := &FakeDataProvider{}

		return dpMockServer
	}

	return dpAerospikeServer
}

func GetSystemProvider() *SystemInfoProvider {
	return dpSysInfoProvider
}
