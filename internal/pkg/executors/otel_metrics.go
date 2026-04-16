package executors

import (
	"context"
	"fmt"
	"strings"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/statprocessors"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var METRIC_CONTEXT_SEPARATOR = map[string]string{
	"period":     ".",
	"underscore": "_",
}

func (oe *OtelExecutor) sendNodeUp(meter metric.Meter,
	labels []attribute.KeyValue, value int64) {

	var key = oe.constructMetricKey(oe.nodeUpMetricName, labels)
	nodeUpGauge := oe.getNodeUpGaugeMetric(key, meter, oe.nodeUpMetricName,
		"Aerospike node active status", labels)

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

func (oe *OtelExecutor) processAndPushStats(meter metric.Meter,
	commonLabels []attribute.KeyValue, refreshStats []statprocessors.AerospikeStat) {

	// create the required metered objects
	for _, stat := range refreshStats {

		// In OTEL all contexts have '.' separated names and actual stat can have _ etc.,
		//  Example: aerospike.server.namespace.master_objects
		//  metric-context-separator is used to separate the context from the metric name
		qualifiedName := fmt.Sprintf("%s%s%s%s%s",
			config.Cfg.Agent.Otel.MetricNamePrefix,
			METRIC_CONTEXT_SEPARATOR[config.Cfg.Agent.Otel.MetricContextSeparator],
			string(stat.Context),
			METRIC_CONTEXT_SEPARATOR[config.Cfg.Agent.Otel.MetricContextSeparator],
			NormalizeMetric(stat.Name),
		)

		desc := NormalizeMetric("description_" + stat.Name)

		labels := []attribute.KeyValue{}

		// label name to value mapped using index
		for idx, label := range stat.Labels {
			labels = append(labels, attribute.String(oe.getRenamedLabel(label), stat.LabelValues[idx]))
		}

		// append common labels
		labels = append(labels, commonLabels...)

		metricKey := oe.constructMetricKey(qualifiedName, labels)

		// Use or Create Otel metric
		switch stat.MType {
		case commons.MetricTypeCounter:

			// some providers still not supporting counters in all approaches in same way,
			// so we send them as gauges. example Datadog
			//    Otel Collector cannot send dalta of Counters
			if config.Cfg.Agent.Otel.AllMetricsAsGauge {
				gMetric := oe.getGaugeMetric(metricKey, meter, qualifiedName, desc, labels)

				if gMetric != nil {
					gMetric.value.Store(int64(stat.Value))
				}
			} else {
				cMetric := oe.getCounterMetric(metricKey, meter, qualifiedName, desc, labels)

				if cMetric != nil {
					cMetric.value.Store(int64(stat.Value))
				}
			}

		case commons.MetricTypeGauge:

			gMetric := oe.getGaugeMetric(metricKey, meter, qualifiedName, desc, labels)

			if gMetric != nil {
				gMetric.value.Store(int64(stat.Value))
			}

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
		log.Fatalf("Error while creating object for gauge stat %s: %v", metricName, err)
		return nil
	}

	gd := &GaugeMetrics{
		instrument: og,
		labels:     labels,
	}

	// Initialize the value to 0
	gd.value.Store(0)

	// Register callback ONCE
	_, err = meter.RegisterCallback(func(_ context.Context, o metric.Observer) error {

		// We send a value only if we are conected and correct value is available
		if oe.dataProvider.IsServerConnected() {
			o.ObserveInt64(gd.instrument, gd.value.Load(), metric.WithAttributes(labels...))
		}

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
		log.Fatalf("Error while creating object for counter stat %s: %v", metricName, err)
	}

	cd := &CounterMetrics{
		instrument: oc,
		labels:     labels,
	}
	// Initialize the value to 0
	cd.value.Store(0)

	// Register callback ONCE for this instrument
	_, err = meter.RegisterCallback(func(_ context.Context, o metric.Observer) error {

		// We send a value only if we are conected and correct value is available
		if oe.dataProvider.IsServerConnected() {
			o.ObserveInt64(cd.instrument, cd.value.Load(), metric.WithAttributes(cd.labels...))
		}

		return nil
	}, oc)

	if err != nil {
		log.Fatalf("Error while RegisterCallback for Counter stat %s: %v", metricName, err)
	}

	oe.counters[key] = cd
	return cd
}

// Send-up, this metric will have to be send irrespective of Aerospike Server is available or not.
func (oe *OtelExecutor) getNodeUpGaugeMetric(key string, meter metric.Meter, metricName string,
	desc string, labels []attribute.KeyValue) *GaugeMetrics {

	// Fast path
	if gd, ok := oe.gauges[key]; ok {
		return gd
	}

	// Create
	og, err := meter.Int64ObservableGauge(metricName, metric.WithDescription(desc))

	if err != nil {
		log.Fatalf("Error while creating object for gauge stat %s: %v", metricName, err)
		return nil
	}

	gd := &GaugeMetrics{
		instrument: og,
		labels:     labels,
	}

	// Initialize the value to 0
	gd.value.Store(0)

	// Register callback ONCE
	_, err = meter.RegisterCallback(func(_ context.Context, o metric.Observer) error {

		o.ObserveInt64(gd.instrument, gd.value.Load(), metric.WithAttributes(labels...))

		return nil
	}, og)

	if err != nil {
		log.Fatalf("Error while RegisterCallback for Gauge stat %s: %v", metricName, err)
	}

	oe.gauges[key] = gd
	return gd
}

func (oe *OtelExecutor) getRenamedLabel(label string) string {

	if value, ok := config.Cfg.Agent.Otel.RenamedLabels[label]; ok {
		return value
	}

	return label
}
