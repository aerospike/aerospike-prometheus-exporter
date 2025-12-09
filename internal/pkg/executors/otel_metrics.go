package executors

import (
	"context"

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
			labels = append(labels, attribute.String(label, stat.LabelValues[idx]))
		}

		// append common labels
		labels = append(labels, commonLabels...)

		// oe.makeOtelGaugeMetric(meter, qualifiedName, desc, labels, stat.Value)

		// create Otel metric
		switch stat.MType {
		case commons.MetricTypeCounter:
			oe.makeOtelCounterMetric(meter, ctx, qualifiedName, desc, labels, stat.Value)
		case commons.MetricTypeGauge:
			oe.makeOtelGaugeMetric(meter, qualifiedName, desc, labels, stat.Value)

		default:
			log.Errorf("Unknown metric type: %d", stat.MType)
		}
	}
}

func (oe OtelExecutor) makeOtelCounterMetric(meter metric.Meter, ctx context.Context, metricName string, desc string, labels []attribute.KeyValue, value float64) {

	ometric, _ := meter.Float64Counter(
		metricName,
		metric.WithDescription(desc),
	)

	ometric.Add(ctx, value, metric.WithAttributes(labels...))
}

func (oe OtelExecutor) makeOtelGaugeMetric(meter metric.Meter, metricName string, desc string, labels []attribute.KeyValue, value float64) {

	ometric, _ := meter.Float64ObservableGauge(
		metricName,
		metric.WithDescription(desc),
	)
	_, err := meter.RegisterCallback(func(_ context.Context, o metric.Observer) error {
		a := value
		o.ObserveFloat64(ometric, a, metric.WithAttributes(labels...))
		return nil
	}, ometric)

	if err != nil {
		log.Fatalf("makeOtelGaugeMetric() Error while creating object for stat %s: %v", metricName, err)
	}
}
