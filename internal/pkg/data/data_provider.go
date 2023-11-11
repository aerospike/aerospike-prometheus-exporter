package data

import (
	"fmt"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"
)

type DataProvider interface {
	RequestInfo(infokeys []string) (map[string]string, error)
}

// pre-create the instances
var dp_aerospike_server = &AerospikeServerProvider{}
var dp_mock_server = &MockAerospikeServer{}

func GetDataProvider() DataProvider {

	if config.Cfg.AeroProm.UseMockDatasource == 1 {
		fmt.Println(" Mock is enabled, going to use mock ")
		return dp_mock_server
	}

	return dp_aerospike_server
}
