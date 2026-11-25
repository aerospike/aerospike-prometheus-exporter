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

// isBackupRestoreContext reports whether a context's series belong to backup or restore jobs.
// Their gauges can carry fractions (progress-pct=42.50), so they are exported as float64, while the gauges of
// every other context stay int64 so their exported value type does not change.
func isBackupRestoreContext(context commons.ContextType) bool {
	return context == commons.CTX_BACKUP || context == commons.CTX_RESTORE
}

// isSweepExempt reports whether a series is left alone by the sweep. The index-pressure stats are fetched
// only every few minutes, so the refreshes in between do not return them by design.
func isSweepExempt(stat statprocessors.AerospikeStat) bool {
	return statprocessors.IsIntervalFetchedStat(stat)
}

// isPartialRefresh reports whether the latest server refresh was cut short. While a checkpoint save runs only
// build and checkpoint-status are requested, so every other series is missing from that refresh, and sweeping
// would drop all of them for the length of the save.
func (oe *OtelExecutor) isPartialRefresh() bool {
	return oe.sharedState != nil && oe.sharedState.CheckpointSaveInProgress
}

// processAndPushStats records the latest values. When sweep is set, the stats are the result of a server
// refresh: every series that refresh returned is remembered, and a series remembered from an earlier refresh
// that this one did not return is unregistered. A job that finished, a job's previous state, a histogram that
// is disabled or renamed (latencies in ms and then in us), or a stat the server no longer reports then stops
// exporting its last value. System info is a separate refresh and is never swept (sweep is false).
func (oe *OtelExecutor) processAndPushStats(meter metric.Meter,
	commonLabels []attribute.KeyValue, refreshStats []statprocessors.AerospikeStat, sweep bool) {

	seenSweptKeys := map[string]struct{}{}

	// create the required metered objects
	for _, stat := range refreshStats {

		// honour the *_metrics_allowlist / *_metrics_blocklist configuration, as the Prometheus path does
		if !stat.IsAllowed {
			continue
		}

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

		if sweep && !isSweepExempt(stat) {
			seenSweptKeys[metricKey] = struct{}{}
			oe.sweptKeys[metricKey] = struct{}{}
		}

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

			// backup and restore gauges keep their fractional part, other gauges stay int64 so the
			// value type of metrics that are already exported does not change
			if isBackupRestoreContext(stat.Context) {
				fMetric := oe.getFloatGaugeMetric(metricKey, meter, qualifiedName, desc, labels)

				if fMetric != nil {
					fMetric.store(stat.Value)
				}

				break
			}

			gMetric := oe.getGaugeMetric(metricKey, meter, qualifiedName, desc, labels)

			if gMetric != nil {
				gMetric.value.Store(int64(stat.Value))
			}

		default:
			log.Errorf("Unknown metric type: %d", stat.MType)
		}
	}

	if sweep && !oe.isPartialRefresh() {
		oe.sweepStaleSeries(seenSweptKeys)
	}
}

// sweepStaleSeries unregisters every series remembered from a server refresh that the latest one did not return.
// Once its callback stops observing, the SDK no longer exports that attribute set.
func (oe *OtelExecutor) sweepStaleSeries(seen map[string]struct{}) {
	for key := range oe.sweptKeys {
		if _, ok := seen[key]; ok {
			continue
		}

		if gd, ok := oe.gauges[key]; ok {
			unregister(key, gd.registration)
			delete(oe.gauges, key)
		}

		if fd, ok := oe.floatGauges[key]; ok {
			unregister(key, fd.registration)
			delete(oe.floatGauges, key)
		}

		if cd, ok := oe.counters[key]; ok {
			unregister(key, cd.registration)
			delete(oe.counters, key)
		}

		delete(oe.sweptKeys, key)
		log.Debugf("otel: stopped reporting series that is no longer returned: %s", key)
	}
}

func unregister(key string, registration metric.Registration) {
	if registration == nil {
		return
	}

	if err := registration.Unregister(); err != nil {
		log.Warnf("otel: failed to unregister callback for %s: %v", key, err)
	}
}

func (oe *OtelExecutor) constructMetricKey(metricName string, labels []attribute.KeyValue) string {
	var b strings.Builder
	b.WriteString(metricName)

	for _, l := range labels {
		b.WriteString("|")
		b.WriteString(string(l.Key))
		b.WriteString("=")
		b.WriteString(l.Value.String())
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
	gd.registration, err = meter.RegisterCallback(func(_ context.Context, o metric.Observer) error {

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

// getFloatGaugeMetric is getGaugeMetric for values that keep their fractional part.
func (oe *OtelExecutor) getFloatGaugeMetric(key string, meter metric.Meter, metricName string,
	desc string, labels []attribute.KeyValue) *FloatGaugeMetrics {

	// Fast path
	if fd, ok := oe.floatGauges[key]; ok {
		return fd
	}

	og, err := meter.Float64ObservableGauge(metricName, metric.WithDescription(desc))

	if err != nil {
		log.Fatalf("Error while creating object for float gauge stat %s: %v", metricName, err)
		return nil
	}

	fd := &FloatGaugeMetrics{
		instrument: og,
		labels:     labels,
	}

	// Register callback ONCE
	fd.registration, err = meter.RegisterCallback(func(_ context.Context, o metric.Observer) error {

		// We send a value only if we are conected and correct value is available
		if oe.dataProvider.IsServerConnected() {
			o.ObserveFloat64(fd.instrument, fd.load(), metric.WithAttributes(labels...))
		}

		return nil
	}, og)

	if err != nil {
		log.Fatalf("Error while RegisterCallback for Float Gauge stat %s: %v", metricName, err)
	}

	oe.floatGauges[key] = fd
	return fd
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
	cd.registration, err = meter.RegisterCallback(func(_ context.Context, o metric.Observer) error {

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
