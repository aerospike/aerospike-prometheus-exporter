package executors

type Executor interface {
	Initialize() error
}

const (
	PROMETHEUS = "prometheus"
	OTELGRPC   = "otel"
)

func GetExecutors() map[string]Executor {
	executorsMap := map[string]Executor{
		PROMETHEUS: &PrometheusHttpExecutor{},
		OTELGRPC:   &OtelExecutor{},
	}

	return executorsMap
}
