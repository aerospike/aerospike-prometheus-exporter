package handlers

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"

	commons "github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/watchers"
	log "github.com/sirupsen/logrus"
)

// Observer communicates with Aerospike and helps collecting metrices
type Observer struct {
	ticks prometheus.Counter
}

var (
	// aerospike_node_up metric descriptor
	nodeActiveDesc *prometheus.Desc

	mutex sync.Mutex
)

func NewObserver() (o *Observer) {
	// func NewObserver(server *aero.Host, user, pass string) (o *Observer, err error) {
	// initialize aerospike_node_up metric descriptor
	nodeActiveDesc = prometheus.NewDesc(
		"aerospike_node_up",
		"Aerospike node active status",
		[]string{"cluster_name", "service", "build"},
		commons.Cfg.AeroProm.MetricLabels,
	)

	o = &Observer{
		ticks: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: "aerospike",
				Subsystem: "node",
				Name:      "ticks",
				Help:      "Counter that detemines how many times the Aerospike node was scraped for metrics.",
			}),
	}

	return o
}

// Describe function of Prometheus' Collector interface
func (o *Observer) Describe(ch chan<- *prometheus.Desc) {}

// Collect function of Prometheus' Collector interface
func (o *Observer) Collect(ch chan<- prometheus.Metric) {
	// Protect against concurrent scrapes
	mutex.Lock()
	defer mutex.Unlock()

	o.ticks.Inc()
	ch <- o.ticks

	// refresh metrics from various watchers,
	watcher_metrics, err := watchers.Refresh()
	if err != nil {
		log.Errorln(err)
		ch <- prometheus.MustNewConstMetric(nodeActiveDesc, prometheus.GaugeValue, 0.0, watchers.ClusterName, watchers.Service, watchers.Build)
		return
	}

	// push the fetched metrics to prometheus
	for _, wm := range watcher_metrics {
		PushToPrometheus(wm.Metric, wm.Value, wm.Labels, wm.LabelValues, ch)
	}

	ch <- prometheus.MustNewConstMetric(nodeActiveDesc, prometheus.GaugeValue, 1.0, watchers.ClusterName, watchers.Service, watchers.Build)
}
