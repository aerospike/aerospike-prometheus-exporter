package dataprovider

import (
	aero "github.com/aerospike/aerospike-client-go/v6"
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

func GetProvider() DataProvider {

	if config.Cfg.AeroProm.UseMockDatasource == 1 {
		// initialize, internally it will check if already initialized
		dp_mock_server.Initialize()

		// a := &FakeDataProvider{}

		return dp_mock_server
	}

	return dp_aerospike_server
}
