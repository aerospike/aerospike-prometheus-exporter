package processors

type metricprocessors interface {
	Initialize() error
}

const (
	PROM = "prometheus"
)

func GetMetricProcessors() map[string]metricprocessors {
	handles := map[string]metricprocessors{
		PROM: &AsmetricsHttpProcessor{},
	}

	return handles
}
