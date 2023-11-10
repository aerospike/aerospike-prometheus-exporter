package commons

type metricType byte

const (
	MetricTypeGauge   metricType = 'G'
	MetricTypeCounter metricType = 'C'
)
