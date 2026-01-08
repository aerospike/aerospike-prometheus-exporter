package executors

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/statprocessors"
	log "github.com/sirupsen/logrus"
)

// PrometheusImpl communicates with Aerospike and helps collecting metrices
type PrometheusImpl struct {
	ticks prometheus.Counter
}

var (
	// aerospike_node_up metric descriptor
	nodeActiveDesc *prometheus.Desc

	mutex sync.Mutex
)

func NewPrometheusImpl() (o *PrometheusImpl) {
	nodeActiveDesc = prometheus.NewDesc(
		"aerospike_node_up",
		"Aerospike node active status",
		[]string{"cluster_name", "service", "build"},
		config.Cfg.Agent.MetricLabels,
	)

	o = &PrometheusImpl{
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
func (o *PrometheusImpl) Describe(ch chan<- *prometheus.Desc) {}

// Collect function of Prometheus' Collector interface
func (o *PrometheusImpl) Collect(ch chan<- prometheus.Metric) {
	// Protect against concurrent scrapes
	mutex.Lock()
	defer mutex.Unlock()

	o.ticks.Inc()
	ch <- o.ticks

	// refresh metrics from various statprocessors,
	refreshedMetrics, err := statprocessors.Refresh()

	if err != nil {
		log.Errorln(err)
		ch <- prometheus.MustNewConstMetric(nodeActiveDesc, prometheus.GaugeValue, 0.0, statprocessors.ClusterName, statprocessors.Service, statprocessors.Build)
		return
	}

	// if kubernetes then send host-name/pod-name else send server-ip as-isnh
	if config.Cfg.Agent.IsKubernetes {
		statprocessors.Service = config.Cfg.Agent.KubernetesPodName
	}
	ch <- prometheus.MustNewConstMetric(nodeActiveDesc, prometheus.GaugeValue, 1.0, statprocessors.ClusterName, statprocessors.Service, statprocessors.Build)

	for _, wm := range refreshedMetrics {
		PushToPrometheus(wm, ch)
	}

	// System Metrics - Memory, Disk and Filesystem - push the fetched metrics to prometheus
	systemMetrics, err := statprocessors.RefreshSystemInfo()

	if err != nil {
		log.Errorln("Error while refreshing SystemInfo Stats, error: ", err)
	}

	// push the fetched metrics to prometheus
	for _, wm := range systemMetrics {
		PushToPrometheus(wm, ch)
	}

}
