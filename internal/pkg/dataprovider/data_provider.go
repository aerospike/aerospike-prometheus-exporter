package dataprovider

import (
	aero "github.com/aerospike/aerospike-client-go/v8"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
)

// //go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate
//
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 . DataProvider
type DataProvider interface {
	RequestInfo(infokeys []string) (map[string]string, error)
	FetchUsersDetails() (bool, []*aero.UserRoles, error)
}

// pre-create the instances
var dpPromAerospikeServer = &AerospikeServer{}
var dpOtelAerospikeServer = &AerospikeServer{}

var dpMockServer = &MockAerospikeServer{}
var dpSysInfoProvider = &SystemInfoProvider{}

func GetProvider(executorMode string) DataProvider {

	switch executorMode {
	case commons.EXECUTOR_MODE_PROM:
		return dpPromAerospikeServer
	case commons.EXECUTOR_MODE_OTEL:
		return dpOtelAerospikeServer
	case "mock":
		return dpMockServer
	default:
		return dpPromAerospikeServer
	}
}

func GetSystemProvider() *SystemInfoProvider {
	return dpSysInfoProvider
}
