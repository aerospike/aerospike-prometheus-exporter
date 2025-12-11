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

func (oe OtelExecutor) sendNodeUp(meter metric.Meter, commonLabels []attribute.KeyValue, value float64) {

	nodeActiveDesc, _ := meter.Float64ObservableGauge(
		"aerospike_node_up",
		metric.WithDescription("Aerospike node active status"),
	)

	if config.Cfg.Agent.IsKubernetes {
		statprocessors.Service = config.Cfg.Agent.KubernetesPodName
	}

	labels := []attribute.KeyValue{
		attribute.String("cluster_name", statprocessors.ClusterName),
		attribute.String("service", statprocessors.Service),
		attribute.String("build", statprocessors.Build),
	}

	// append common labels
	labels = append(labels, commonLabels...)

	_, err := meter.RegisterCallback(func(_ context.Context, o metric.Observer) error {
		o.ObserveFloat64(nodeActiveDesc, value, metric.WithAttributes(labels...))
		return nil
	}, nodeActiveDesc)

	if err != nil {
		log.Fatalf("sendNodeUp() Error while creating object for stat 'aerospike_node_up': %v", err)
	}
}

func (oe OtelExecutor) getCommonLabels() []attribute.KeyValue {
	mlabels := config.Cfg.Agent.MetricLabels
	attrkv := []attribute.KeyValue{}
	if len(mlabels) > 0 {
		for k, v := range mlabels {
			attrkv = append(attrkv, attribute.String(k, v))
		}
	}

	return attrkv
}

func (oe OtelExecutor) processAndPushStats(meter metric.Meter, ctx context.Context, commonLabels []attribute.KeyValue, refreshStats []statprocessors.AerospikeStat) {

	// create the required metered objectes
	for _, stat := range refreshStats {

		qualifiedName := stat.QualifyMetricContext() + "_" + NormalizeMetric(stat.Name)
		desc := NormalizeMetric("description_" + stat.Name)

		labels := []attribute.KeyValue{}
		// label name to value mapped using index
		for idx, label := range stat.Labels {
			//TODO: handle if label value is null or not present
			labels = append(labels, attribute.String(label, stat.LabelValues[idx]))
		}

		// append common labels
		labels = append(labels, commonLabels...)

		key := oe.constructMetricKey(qualifiedName, labels)

		// create Otel metric
		switch stat.MType {
		case commons.MetricTypeCounter:
			counter := oe.getCounterMetric(key, meter, qualifiedName, desc, labels)

			// Only zero (first run) or positive deltas are sent
			if (stat.Value - counter.value) >= 0 {
				if strings.Contains(qualifiedName, "client_write_success") {
					fmt.Println("Adding counter delta: qualifiedName - ", qualifiedName, " : delta - ", (stat.Value - counter.value))
				}
				counter.instrument.Add(ctx, (stat.Value - counter.value), metric.WithAttributes(counter.labels...))
			}

			counter.value = stat.Value

		case commons.MetricTypeGauge:

			gauge := oe.getGaugeMetric(key, meter, qualifiedName, desc, labels)
			gauge.value.Store(stat.Value)
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

func (oe *OtelExecutor) getGaugeMetric(key string, meter metric.Meter, metricName string, desc string, labels []attribute.KeyValue) *GaugeMetrics {

	// Fast path
	if gd, ok := oe.gauges[key]; ok {
		return gd
	}

	// Create
	og, err := meter.Float64ObservableGauge(metricName, metric.WithDescription(desc))

	if err != nil {
		log.Fatalf("getOrCreateGauge() Error while creating object for stat %s: %v", metricName, err)
	}

	gd := &GaugeMetrics{
		instrument: og,
		labels:     labels,
	}

	// Initialize the value to 0
	gd.value.Store(float64(0))

	// Register callback ONCE
	_, err = meter.RegisterCallback(func(_ context.Context, o metric.Observer) error {
		vAny := gd.value.Load()
		v := vAny.(float64)
		o.ObserveFloat64(gd.instrument, v, metric.WithAttributes(labels...))
		return nil
	}, og)

	if err != nil {
		log.Fatalf("Error while RegisterCallback for Gauge stat %s: %v", metricName, err)
	}

	oe.gauges[key] = gd
	return gd
}

func (oe *OtelExecutor) getCounterMetric(key string, meter metric.Meter, metricName string, desc string, labels []attribute.KeyValue) *CounterMetrics {

	if cd, ok := oe.counters[key]; ok {
		return cd
	}

	instr, err := meter.Float64Counter(
		metricName,
		metric.WithDescription(desc),
	)
	if err != nil {
		panic(err)
	}

	cd := &CounterMetrics{
		instrument: instr,
		value:      0,
	}

	oe.counters[key] = cd
	return cd
}
