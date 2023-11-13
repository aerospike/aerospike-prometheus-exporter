package data

import (
	aero "github.com/aerospike/aerospike-client-go/v6"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"
)

type DataProvider interface {
	RequestInfo(infokeys []string) (map[string]string, error)
	FetchUsersDetails() (bool, []*aero.UserRoles, error)
}

// pre-create the instances
var dp_aerospike_server = &AerospikeServerProvider{}
var dp_mock_server = &MockAerospikeServer{}

func GetProvider() DataProvider {

	if config.Cfg.AeroProm.UseMockDatasource == 1 {
		// fmt.Println(" Mock is enabled, going to use mock ")
		// initialize, internally it will check if already initialized
		dp_mock_server.Initialize()

		return dp_mock_server
	}

	return dp_aerospike_server
}
