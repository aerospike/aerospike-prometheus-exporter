package handlers

type MetricHandlers interface {
	Initialize() error
}

func GetMetricHandlers() map[string]MetricHandlers {
	handles := map[string]MetricHandlers{
		"prometheus": &PrometheusMetrics{},
	}

	return handles
}
