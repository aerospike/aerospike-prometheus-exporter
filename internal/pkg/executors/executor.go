package executors

type Executor interface {
	Initialize() error
}

const (
	PROMETHEUS = "prometheus"
)

func GetExecutors() map[string]Executor {
	executorsMap := map[string]Executor{
		PROMETHEUS: &PrometheusHttpExecutor{},
	}

	return executorsMap
}
