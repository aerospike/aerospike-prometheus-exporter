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
	IsServerConnected() bool
}

// pre-create the instances
var dpPromAerospikeServer = NewAerospikeProvider()
var dpOtelAerospikeServer = NewAerospikeProvider()

var dpSysInfoProvider = &SystemInfoProvider{}

func GetProvider(executorMode string) DataProvider {

	switch executorMode {
	case commons.EXECUTOR_MODE_PROM:
		return dpPromAerospikeServer
	case commons.EXECUTOR_MODE_OTEL:
		return dpOtelAerospikeServer
	default:
		return dpPromAerospikeServer
	}
}

func GetSystemProvider() *SystemInfoProvider {
	return dpSysInfoProvider
}
