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

const AEROSPIKE_NODE_UP = statprocessors.PREFIX_AEROSPIKE_OTEL + ".node_up"

// this map is used to rename standard labels to OTEL suitable labels
var OTEL_LABEL_NAME_MAPPING = map[string]string{
	commons.METRIC_LABEL_CLUSTER_NAME:              "aerospike_cluster",
	commons.METRIC_LABEL_SERVICE:                   "aerospike_service",
	commons.METRIC_LABEL_NS:                        "ns",
	commons.METRIC_LABEL_SET:                       "set",
	commons.METRIC_LABEL_LE:                        "le",
	commons.METRIC_LABEL_DC_NAME:                   "dc",
	commons.METRIC_LABEL_INDEX:                     "index",
	commons.METRIC_LABEL_SINDEX:                    "sindex",
	commons.METRIC_LABEL_STORAGE_ENGINE:            "storage_engine",
	commons.METRIC_LABEL_USER:                      "user",
	commons.METRIC_LABEL_UA_CLIENT_LIBRARY_VERSION: "client_library_version",
	commons.METRIC_LABEL_UA_CLIENT_APP_ID:          "client_app_id",
}

func (oe *OtelExecutor) sendNodeUp(meter metric.Meter,
	labels []attribute.KeyValue, value int64) {

	metricKey := oe.constructMetricKey(AEROSPIKE_NODE_UP, labels)
	nodeUpGauge := oe.getSendUpGaugeMetric(metricKey, meter, AEROSPIKE_NODE_UP, "Aerospike node active status", labels)
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
		qualifiedName := fmt.Sprintf("%s.%s.%s",
			statprocessors.PREFIX_AEROSPIKE_OTEL,
			string(stat.Context),
			NormalizeMetric(stat.Name),
		)

		desc := NormalizeMetric("description_" + stat.Name)

		labels := []attribute.KeyValue{}

		// label name to value mapped using index
		for idx, label := range stat.Labels {

			labelToSend := label

			if value, ok := OTEL_LABEL_NAME_MAPPING[label]; ok {
				labelToSend = value
			}

			labels = append(labels, attribute.String(labelToSend, stat.LabelValues[idx]))
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
func (oe *OtelExecutor) getSendUpGaugeMetric(key string, meter metric.Meter, metricName string,
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
