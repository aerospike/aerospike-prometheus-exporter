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
var dp_aerospike_server = &AerospikeServer{}
var dp_mock_server = &MockAerospikeServer{}
var sys_info_provider = &SystemInfoProvider{}

func GetProvider() DataProvider {

	if config.Cfg.Agent.UseMockDatasource {
		// initialize, internally it will check if already initialized
		dp_mock_server.Initialize()

		// a := &FakeDataProvider{}

		return dp_mock_server
	}

	return dp_aerospike_server
}

func GetSystemProvider() *SystemInfoProvider {
	return sys_info_provider
}
