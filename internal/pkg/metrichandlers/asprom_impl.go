package metrichandlers

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/statprocessors"
	log "github.com/sirupsen/logrus"
)

// AsPromImpl communicates with Aerospike and helps collecting metrices
type AsPromImpl struct {
	ticks prometheus.Counter
}

var (
	// aerospike_node_up metric descriptor
	nodeActiveDesc *prometheus.Desc

	mutex sync.Mutex
)

func NewAsPromImpl() (o *AsPromImpl) {
	// func NewObserver(server *aero.Host, user, pass string) (o *Observer, err error) {
	// initialize aerospike_node_up metric descriptor
	nodeActiveDesc = prometheus.NewDesc(
		"aerospike_node_up",
		"Aerospike node active status",
		[]string{"cluster_name", "service", "build"},
		config.Cfg.AeroProm.MetricLabels,
	)

	o = &AsPromImpl{
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
func (o *AsPromImpl) Describe(ch chan<- *prometheus.Desc) {}

// Collect function of Prometheus' Collector interface
func (o *AsPromImpl) Collect(ch chan<- prometheus.Metric) {
	// Protect against concurrent scrapes
	mutex.Lock()
	defer mutex.Unlock()

	o.ticks.Inc()
	ch <- o.ticks

	// refresh metrics from various statprocessors,
	refreshed_metrics, err := statprocessors.Refresh()
	if err != nil {
		log.Errorln(err)
		ch <- prometheus.MustNewConstMetric(nodeActiveDesc, prometheus.GaugeValue, 0.0, statprocessors.ClusterName, statprocessors.Service, statprocessors.Build)
		return
	}

	// push the fetched metrics to prometheus
	for _, wm := range refreshed_metrics {
		PushToPrometheus(wm, ch)
	}

	ch <- prometheus.MustNewConstMetric(nodeActiveDesc, prometheus.GaugeValue, 1.0, statprocessors.ClusterName, statprocessors.Service, statprocessors.Build)
}
