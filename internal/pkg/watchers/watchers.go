package watchers

import (
	commons "github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
)

type WatcherMetric struct {
	Metric      commons.AerospikeStat
	Value       float64
	Labels      []string
	LabelValues []string
}

type Watcher interface {
	PassOneKeys() []string
	PassTwoKeys(rawMetrics map[string]string) []string
	// refresh( o *Observer, infoKeys []string, rawMetrics map[string]string, ch chan<- prometheus.Metric) error
	Refresh(infoKeys []string, rawMetrics map[string]string) ([]WatcherMetric, error)
}
