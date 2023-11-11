package data

import (
	"fmt"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"
)

type DataProvider interface {
	RequestInfo(infokeys []string) (map[string]string, error)
}

func GetDataProvider() DataProvider {

	if config.Cfg.AeroProm.UseMockDatasource == 1 {
		fmt.Println(" Mock is enabled, going to use mock ")
		return &MockAerospikeServer{}
	}

	return &AerospikeServerProvider{}
}
