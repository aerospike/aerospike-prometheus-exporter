package main

import "github.com/prometheus/client_golang/prometheus"

type MetricMap map[string]promMetric

type metricType byte

const (
	mtGauge   metricType = 'G'
	mtCounter metricType = 'C'
)

type Watcher interface {
	infoKeys() []string
	detailKeys(rawMetrics map[string]string) []string
	refresh(infoKeys []string, rawMetrics map[string]string, ch chan<- prometheus.Metric) error
	describe(ch chan<- *prometheus.Desc)
}

type promMetric struct {
	origDesc  string
	desc      *prometheus.Desc
	valueType prometheus.ValueType
}

type StatsMap map[string]interface{}

// Value should be an int64 or a convertible string; otherwise defValue is returned
// this function never panics
func (s StatsMap) TryString(name string, defValue string, aliases ...string) string {
	field := s.Get(name, aliases...)
	if field != nil {
		if value, ok := field.(string); ok {
			return value
		}
	}
	return defValue
}

func (s StatsMap) Get(name string, aliases ...string) interface{} {
	if val, exists := s[name]; exists {
		return val
	}

	for _, alias := range aliases {
		if val, exists := s[alias]; exists {
			return val
		}
	}

	return nil
}

// Value should be an float64 or a convertible string; otherwise defValue is returned
// this function never panics
func (s StatsMap) TryFloat(name string, defValue float64, aliases ...string) float64 {
	field := s.Get(name, aliases...)
	if field != nil {
		if value, ok := field.(float64); ok {
			return value
		}
		if value, ok := field.(int64); ok {
			return float64(value)
		}
	}
	return defValue
}
