package data

import "github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"

type DataProvider interface {
	RequestInfo(infokeys []string) (map[string]string, error)
}

func GetDataProvider() DataProvider {

	if config.Cfg.AeroProm.UseMockDatasource == 0 {
		return nil
	}

	return &AerospikeServerProvider{}
}
