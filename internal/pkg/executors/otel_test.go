package executors

import (
	"context"
	"testing"

	aero "github.com/aerospike/aerospike-client-go/v8"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/dataprovider"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/statprocessors"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestOtelProcessAndPushStats(t *testing.T) {
	original := config.Cfg
	t.Cleanup(func() { config.Cfg = original })

	config.Cfg = config.Config{}
	config.Cfg.Agent.MetricLabels = map[string]string{"zone": "a"}
	config.Cfg.Agent.Otel.MetricNamePrefix = "aerospike"
	config.Cfg.Agent.Otel.MetricContextSeparator = "period"
	config.Cfg.Agent.Otel.RenamedLabels = map[string]string{"ns": "namespace"}

	provider := &otelProvider{}
	executor := &OtelExecutor{
		dataProvider: provider,
		gauges:       map[string]*GaugeMetrics{},
		counters:     map[string]*CounterMetrics{},
	}
	meter, reader := testMeter(t)
	common := executor.getCommonLabels()
	require.Len(t, common, 1)

	stats := []statprocessors.AerospikeStat{
		{
			Context: "namespace", Name: "master-objects", MType: commons.MetricTypeGauge, Value: 7,
			Labels: []string{"ns"}, LabelValues: []string{"test"},
		},
		{
			Context: "namespace", Name: "client_read_success", MType: commons.MetricTypeCounter, Value: 4,
			Labels: []string{"ns"}, LabelValues: []string{"test"},
		},
	}
	executor.processAndPushStats(meter, common, stats)
	stats[0].Value = 9
	executor.processAndPushStats(meter, common, stats[:1])

	require.Len(t, executor.gauges, 1)
	require.Len(t, executor.counters, 1)

	rm := collectMetrics(t, reader)
	_, _, found := metricPoint(t, rm, "aerospike.namespace.master_objects")
	require.False(t, found)

	provider.connected = true
	rm = collectMetrics(t, reader)
	value, attrs, found := metricPoint(t, rm, "aerospike.namespace.master_objects")
	require.True(t, found)
	require.Equal(t, int64(9), value)
	requireAttribute(t, attrs, "namespace", "test")
	requireAttribute(t, attrs, "zone", "a")

	value, _, found = metricPoint(t, rm, "aerospike.namespace.client_read_success")
	require.True(t, found)
	require.Equal(t, int64(4), value)

	config.Cfg.Agent.Otel.MetricContextSeparator = "underscore"
	config.Cfg.Agent.Otel.AllMetricsAsGauge = true
	underscore := &OtelExecutor{
		dataProvider: provider,
		gauges:       map[string]*GaugeMetrics{},
		counters:     map[string]*CounterMetrics{},
	}
	underscoreMeter, underscoreReader := testMeter(t)
	underscore.processAndPushStats(underscoreMeter, nil, stats[1:])
	require.Empty(t, underscore.counters)
	require.Len(t, underscore.gauges, 1)
	rm = collectMetrics(t, underscoreReader)
	value, _, found = metricPoint(t, rm, "aerospike_namespace_client_read_success")
	require.True(t, found)
	require.Equal(t, int64(4), value)

	executor.nodeUpMetricName = "aerospike.node_up"
	executor.sendNodeUp(meter, common, 0)
	provider.connected = false
	rm = collectMetrics(t, reader)
	value, _, found = metricPoint(t, rm, "aerospike.node_up")
	require.True(t, found)
	require.Equal(t, int64(0), value)
}

func TestOtelExportSkipsWarmupWhileConnected(t *testing.T) {
	delegate := &recordingExporter{}
	executor := &OtelExecutor{
		Exporter:     delegate,
		dataProvider: &otelProvider{connected: true},
	}

	for _, scrape := range []int64{0, 1, 2} {
		executor.refreshCounter.Store(scrape)
		require.NoError(t, executor.Export(context.Background(), &metricdata.ResourceMetrics{}))
	}
	require.Zero(t, delegate.calls)

	executor.refreshCounter.Store(3)
	require.NoError(t, executor.Export(context.Background(), &metricdata.ResourceMetrics{}))
	require.Equal(t, 1, delegate.calls)

	executor.dataProvider = &otelProvider{}
	executor.refreshCounter.Store(0)
	require.NoError(t, executor.Export(context.Background(), &metricdata.ResourceMetrics{}))
	require.Equal(t, 2, delegate.calls)
}

func TestOtelTemporalitySelector(t *testing.T) {
	original := config.Cfg
	t.Cleanup(func() { config.Cfg = original })

	config.Cfg = config.Config{}
	executor := &OtelExecutor{}

	config.Cfg.Agent.Otel.CounterTemporality = commons.TEMPORALITY_DELTA
	require.Equal(t, metricdata.DeltaTemporality, executor.getTemporalitySelector(sdkmetric.InstrumentKindObservableCounter))
	require.Equal(t, metricdata.CumulativeTemporality, executor.getTemporalitySelector(sdkmetric.InstrumentKindObservableGauge))

	config.Cfg.Agent.Otel.CounterTemporality = "cumulative"
	require.Equal(t, metricdata.CumulativeTemporality, executor.getTemporalitySelector(sdkmetric.InstrumentKindObservableCounter))
}

func TestOtelReadHeaders(t *testing.T) {
	original := config.Cfg
	t.Cleanup(func() {
		config.Cfg = original
		commons.RegisterSecretAgent(nil)
	})

	config.Cfg = config.Config{}
	t.Setenv("APE_OTEL_TOKEN", "from-env")
	commons.RegisterSecretAgent(&otelFetcher{value: "from-agent"})
	config.Cfg.Agent.Otel.Headers = map[string]string{
		"literal":       "plain-token",
		"env":           "env:APE_OTEL_TOKEN",
		"Authorization": "secrets:APE:ApiKey",
	}

	headers, err := (&OtelExecutor{}).readHeaders()
	require.NoError(t, err)
	require.Equal(t, map[string]string{
		"literal":       "plain-token",
		"env":           "from-env",
		"Authorization": "from-agent",
	}, headers)

	config.Cfg.Agent.Otel.Headers = map[string]string{"api-key": "secrets:APE:Missing"}
	_, err = (&OtelExecutor{}).readHeaders()
	require.ErrorContains(t, err, "OpenTelemetry header api-key")
	require.NotContains(t, err.Error(), "from-agent")
}

func testMeter(t *testing.T) (metric.Meter, *sdkmetric.ManualReader) {
	t.Helper()

	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() {
		require.NoError(t, provider.Shutdown(context.Background()))
	})

	return provider.Meter("ape-otel-test"), reader
}

func collectMetrics(t *testing.T, reader *sdkmetric.ManualReader) metricdata.ResourceMetrics {
	t.Helper()

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(context.Background(), &rm))

	return rm
}

func metricPoint(t *testing.T, rm metricdata.ResourceMetrics, name string) (int64, attribute.Set, bool) {
	t.Helper()

	for _, scope := range rm.ScopeMetrics {
		for _, item := range scope.Metrics {
			if item.Name != name {
				continue
			}
			switch data := item.Data.(type) {
			case metricdata.Gauge[int64]:
				if len(data.DataPoints) == 0 {
					return 0, attribute.Set{}, false
				}
				return data.DataPoints[0].Value, data.DataPoints[0].Attributes, true
			case metricdata.Sum[int64]:
				if len(data.DataPoints) == 0 {
					return 0, attribute.Set{}, false
				}
				require.True(t, data.IsMonotonic)
				return data.DataPoints[0].Value, data.DataPoints[0].Attributes, true
			default:
				t.Fatalf("metric %s has type %T", name, item.Data)
			}
		}
	}

	return 0, attribute.Set{}, false
}

func requireAttribute(t *testing.T, attrs attribute.Set, key, value string) {
	t.Helper()

	got, ok := attrs.Value(attribute.Key(key))
	require.True(t, ok, key)
	require.Equal(t, value, got.AsString())
}

type otelProvider struct {
	connected bool
}

func (p *otelProvider) RequestInfo([]string) (map[string]string, error) {
	return nil, nil
}

func (p *otelProvider) FetchUsersDetails() (bool, []*aero.UserRoles, error) {
	return false, nil, nil
}

func (p *otelProvider) IsServerConnected() bool {
	return p.connected
}

type recordingExporter struct {
	calls int
}

func (r *recordingExporter) Temporality(sdkmetric.InstrumentKind) metricdata.Temporality {
	return metricdata.CumulativeTemporality
}

func (r *recordingExporter) Aggregation(kind sdkmetric.InstrumentKind) sdkmetric.Aggregation {
	return sdkmetric.DefaultAggregationSelector(kind)
}

func (r *recordingExporter) Export(context.Context, *metricdata.ResourceMetrics) error {
	r.calls++
	return nil
}

func (r *recordingExporter) ForceFlush(context.Context) error { return nil }

func (r *recordingExporter) Shutdown(context.Context) error { return nil }

type otelFetcher struct {
	value string
}

func (f *otelFetcher) Fetch(_ context.Context, resource, key string) ([]byte, error) {
	if resource == "APE" && key == "ApiKey" {
		return []byte(f.value), nil
	}
	return nil, errMissingOtelSecret
}

type errMissingOtelSecretType struct{}

func (errMissingOtelSecretType) Error() string { return "not found" }

var errMissingOtelSecret error = errMissingOtelSecretType{}

var _ dataprovider.DataProvider = (*otelProvider)(nil)
