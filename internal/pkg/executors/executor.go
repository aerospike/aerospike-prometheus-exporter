package executors

import "github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"

type Executor interface {
	Initialize() error
}

func GetExecutors() map[string]Executor {
	executorsMap := map[string]Executor{
		commons.EXECUTOR_MODE_PROM: &PrometheusHttpExecutor{},
		commons.EXECUTOR_MODE_OTEL: &OtelExecutor{},
	}

	return executorsMap
}
