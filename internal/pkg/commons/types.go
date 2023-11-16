package commons

// used to define context of stat types (like namespace, set, xdr etc.,)
type ContextType string

type MetricType byte

const (
	MetricTypeGauge   MetricType = 'G'
	MetricTypeCounter MetricType = 'C'
)
