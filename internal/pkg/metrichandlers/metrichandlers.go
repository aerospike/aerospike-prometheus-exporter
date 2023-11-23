package metrichandlers

type metrichandlers interface {
	Initialize() error
}

const (
	PROM = "prometheus"
)

func GetMetricHandlers() map[string]metrichandlers {
	handles := map[string]metrichandlers{
		PROM: &AsmetricsHttpProcessor{},
	}

	return handles
}
