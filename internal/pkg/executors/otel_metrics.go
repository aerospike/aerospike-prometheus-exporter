package executors

import (
	"context"
	"fmt"
	"strings"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/statprocessors"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/systeminfo"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	// "go.opentelemetry.io/otel/label"
)

func sendNodeUp(meter metric.Meter, ctx context.Context, commonLabels []attribute.KeyValue, value float64) {

	nodeActiveDesc, _ := meter.Float64ObservableGauge(
		"aerospike_node_up",
		metric.WithDescription("Aerospike node active status"),
	)

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

	handleErr(err, "sendNodeUp() Error while creating object for stat 'aerospike_node_up' ")
}

func getCommonLabels() []attribute.KeyValue {
	mlabels := config.Cfg.AeroProm.MetricLabels
	attrkv := []attribute.KeyValue{}
	if len(mlabels) > 0 {
		for k, v := range mlabels {
			attrkv = append(attrkv, attribute.String(k, v))
		}
	}

	return attrkv
}

func processAerospikeStats(meter metric.Meter, ctx context.Context, commonLabels []attribute.KeyValue, refreshStats []statprocessors.AerospikeStat) {

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

		// create Otel metric
		if stat.MType == commons.MetricTypeCounter {
			value := stat.Value
			// value = calcAerospikeStatValueToUse(qualifiedName, stat)

			makeOtelCounterMetric(meter, ctx, qualifiedName, desc, labels, value)

		} else if stat.MType == commons.MetricTypeGauge {
			makeOtelGaugeMetric(meter, ctx, qualifiedName, desc, labels, stat.Value)
		}

		// Add stat to previous-process-map
		previousRefreshStats[getMetricMapKey(qualifiedName, stat)] = stat
	}

}

func calcAerospikeStatValueToUse(metricName string, stat statprocessors.AerospikeStat) float64 {
	value := stat.Value

	// if previous value exists, then set value as DIFFerence ( current_value , previous_value)
	prevStatState, ok := previousRefreshStats[getMetricMapKey(metricName, stat)]
	// only if this is a stat and not a config,
	if ok && !stat.IsConfig {
		value = stat.Value - prevStatState.Value
	}

	return value
}

func processSystemInfoStats(meter metric.Meter, ctx context.Context, commonLabels []attribute.KeyValue, refreshStats []systeminfo.SystemInfoStat) {

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

		// create Otel metric
		if stat.MType == commons.MetricTypeCounter {
			value := stat.Value
			// value = calcSysInfoStatValueToUse(qualifiedName, stat)

			makeOtelCounterMetric(meter, ctx, qualifiedName, desc, labels, value)

		} else if stat.MType == commons.MetricTypeGauge {
			makeOtelGaugeMetric(meter, ctx, qualifiedName, desc, labels, stat.Value)
		}

		// Add stat to previous-process-map
		previousSysInfoStats[stat.GetMetricMapKey()] = stat
	}

}

func calcSysInfoStatValueToUse(metricName string, stat systeminfo.SystemInfoStat) float64 {
	value := stat.Value

	// if previous value exists, then set value as DIFFerence ( current_value , previous_value)
	prevStatState, ok := previousRefreshStats[stat.GetMetricMapKey()]
	// only if this is a stat and not a config, TODO:, how to check if system metric is a config
	if ok {
		value = stat.Value - prevStatState.Value
	}

	return value
}

func makeOtelCounterMetric(meter metric.Meter, ctx context.Context, metricName string, desc string, labels []attribute.KeyValue, value float64) {

	if strings.Contains(metricName, "uptime") {
		fmt.Println(" found Uptime metricName is ", metricName)
	}

	ometric, _ := meter.Float64Counter(
		metricName,
		metric.WithDescription(desc),
	)

	ometric.Add(ctx, value, metric.WithAttributes(labels...))

}

func makeOtelGaugeMetric(meter metric.Meter, ctx context.Context, metricName string, desc string, labels []attribute.KeyValue, value float64) {

	// _, ok := mapGaugeMetricObjects[metricName]
	ometric, _ := meter.Float64ObservableGauge(
		metricName,
		metric.WithDescription(desc),
	)
	_, err := meter.RegisterCallback(func(_ context.Context, o metric.Observer) error {
		a := value
		o.ObserveFloat64(ometric, a, metric.WithAttributes(labels...))
		return nil
	}, ometric)

	handleErr(err, "makeOtelGaugeMetric() Error while creating object for stat "+metricName)

}

func handleErr(err error, message string) {
	if err != nil {
		log.Fatalf("%s: %v", message, err)
	}
}
