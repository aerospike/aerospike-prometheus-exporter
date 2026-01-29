package executors

import (
	"context"
	"os"
	"strings"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/statprocessors"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const AEROSPIKE_NODE_UP = "aerospike.server.node.up"

var OTEL_LABELS_MAPPING = map[string]string{
	commons.METRIC_LABEL_CLUSTER_NAME: "aerospike_cluster",
	commons.METRIC_LABEL_SERVICE:      "aerospike_service",
}

func (oe *OtelExecutor) sendNodeUp(meter metric.Meter,
	labels []attribute.KeyValue, value int64) {

	metricKey := oe.constructMetricKey(AEROSPIKE_NODE_UP, labels)

	nodeUpGauge := oe.getGaugeMetric(metricKey, meter, AEROSPIKE_NODE_UP, "Aerospike node active status", labels)
	nodeUpGauge.value.Store(value)
}

func (oe *OtelExecutor) getCommonLabels() []attribute.KeyValue {
	mlabels := config.Cfg.Agent.MetricLabels
	attrkv := []attribute.KeyValue{}

	if len(mlabels) > 0 {
		for k, v := range mlabels {
			attrkv = append(attrkv, attribute.String(k, v))
		}
	}

	return attrkv
}

func (oe *OtelExecutor) allMetricsAreGauges() bool {
	envConfig := os.Getenv("AEROSPIKE_ALL_METRICS_ARE_GAUGES")
	return envConfig != "" && strings.ToLower(envConfig) == "true"
}

func (oe *OtelExecutor) processAndPushStats(meter metric.Meter, ctx context.Context,
	commonLabels []attribute.KeyValue, refreshStats []statprocessors.AerospikeStat) {

	// create the required metered objectes
	for _, stat := range refreshStats {

		qualifiedName := statprocessors.PREFIX_AEROSPIKE_OTEL + "." + string(stat.Context)
		qualifiedName = qualifiedName + "." + NormalizeMetric(stat.Name)

		desc := NormalizeMetric("description_" + stat.Name)

		labels := []attribute.KeyValue{}
		// label name to value mapped using index
		for idx, label := range stat.Labels {
			//TODO: handle if label value is null or not present
			if renamedLabel, ok := OTEL_LABELS_MAPPING[label]; ok {
				labels = append(labels, attribute.String(renamedLabel, stat.LabelValues[idx]))
			} else {
				labels = append(labels, attribute.String(label, stat.LabelValues[idx]))
			}
			// labels = append(labels, attribute.String(label, stat.LabelValues[idx]))
		}

		// append common labels
		labels = append(labels, commonLabels...)

		metricKey := oe.constructMetricKey(qualifiedName, labels)

		// Use or Create Otel metric
		allMetricsAreGauges := oe.allMetricsAreGauges()
		switch stat.MType {
		case commons.MetricTypeCounter:
			if allMetricsAreGauges {
				gMetric := oe.getGaugeMetric(metricKey, meter, qualifiedName, desc, labels)
				gMetric.value.Store(int64(stat.Value))
			} else {
				cMetric := oe.getCounterMetric(metricKey, meter, qualifiedName, desc, labels)

				// If server restarts while exporter running, delta will be negative, so we don't send it
				// TODO: discuss with sunil on this
				cMetric.value.Store(int64(stat.Value))
			}

		case commons.MetricTypeGauge:

			gMetric := oe.getGaugeMetric(metricKey, meter, qualifiedName, desc, labels)
			gMetric.value.Store(int64(stat.Value))
		default:
			log.Errorf("Unknown metric type: %d", stat.MType)
		}
	}
}

func (oe *OtelExecutor) constructMetricKey(metricName string, labels []attribute.KeyValue) string {
	var b strings.Builder
	b.WriteString(metricName)

	for _, l := range labels {
		b.WriteString("|")
		b.WriteString(string(l.Key))
		b.WriteString("=")
		b.WriteString(l.Value.Emit())
	}

	return b.String()
}

func (oe *OtelExecutor) getGaugeMetric(key string, meter metric.Meter, metricName string,
	desc string, labels []attribute.KeyValue) *GaugeMetrics {

	// Fast path
	if gd, ok := oe.gauges[key]; ok {
		return gd
	}

	// Create
	og, err := meter.Int64ObservableGauge(metricName, metric.WithDescription(desc))

	if err != nil {
		log.Fatalf("getOrCreateGauge() Error while creating object for stat %s: %v", metricName, err)
	}

	gd := &GaugeMetrics{
		instrument: og,
		labels:     labels,
	}

	// Initialize the value to 0
	gd.value.Store(0)

	// Register callback ONCE
	_, err = meter.RegisterCallback(func(_ context.Context, o metric.Observer) error {
		v := gd.value.Load()
		o.ObserveInt64(gd.instrument, v, metric.WithAttributes(labels...))

		return nil
	}, og)

	if err != nil {
		log.Fatalf("Error while RegisterCallback for Gauge stat %s: %v", metricName, err)
	}

	oe.gauges[key] = gd
	return gd
}

func (oe *OtelExecutor) getCounterMetric(key string, meter metric.Meter, metricName string,
	desc string, labels []attribute.KeyValue) *CounterMetrics {

	// Fast path
	if cd, ok := oe.counters[key]; ok {
		return cd
	}

	// Create ObservableCounter
	oc, err := meter.Int64ObservableCounter(
		metricName,
		metric.WithDescription(desc),
	)

	if err != nil {
		log.Fatalf("getCounterMetric() Error while creating object for stat %s: %v", metricName, err)
	}

	cd := &CounterMetrics{
		instrument: oc,
		labels:     labels,
	}
	// Initialize the value to 0
	cd.value.Store(0)

	// Register callback ONCE for this instrument
	_, err = meter.RegisterCallback(func(_ context.Context, o metric.Observer) error {
		v := cd.value.Load()
		o.ObserveInt64(cd.instrument, v, metric.WithAttributes(cd.labels...))
		return nil
	}, oc)

	if err != nil {
		log.Fatalf("Error while RegisterCallback for Counter stat %s: %v", metricName, err)
	}

	oe.counters[key] = cd
	return cd
}
