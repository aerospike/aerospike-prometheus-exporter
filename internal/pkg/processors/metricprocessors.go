package processors

type metricprocessors interface {
	Initialize() error
}

func GetMetricProcessors() map[string]metricprocessors {
	handles := map[string]metricprocessors{
		"prometheus": &AsmetricsHttpProcessor{},
	}

	return handles
}
