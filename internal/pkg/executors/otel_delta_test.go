package executors

import (
	"context"
	"testing"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/statprocessors"
	"github.com/stretchr/testify/require"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// exportedCounterValue returns what one export of the counter series carries: the data point value, or false
// when the series is not exported at all. A counter is a Sum, or a Gauge when all_metrics_as_gauges is set.
func exportedCounterValue(t *testing.T, reader *sdkmetric.ManualReader, name string) (int64, bool) {
	t.Helper()

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(context.Background(), &rm))

	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != name {
				continue
			}

			switch data := m.Data.(type) {
			case metricdata.Sum[int64]:
				require.Len(t, data.DataPoints, 1)
				return data.DataPoints[0].Value, true
			case metricdata.Gauge[int64]:
				require.Len(t, data.DataPoints, 1)
				return data.DataPoints[0].Value, true
			}
		}
	}

	return 0, false
}

// TestOtelCounterSeriesReturningAfterAGap follows one counter whose series is dropped, because the server stops
// returning it, and then comes back while the exporter stays connected, and shows what each setting exports.
//
// With delta temporality the SDK forgets what it already reported for a series that is not collected, so the first
// export after the return carries the whole cumulative value, not only the increase since the last report. A backend
// that adds up deltas counts everything reported before the gap a second time. The exporter's skip of the first
// exports covers a start and a reconnect, not this case. The test pins that behaviour, which the README describes.
// Cumulative temporality and all_metrics_as_gauges are not affected, as the Docker image's default is.
func TestOtelCounterSeriesReturningAfterAGap(t *testing.T) {
	const counterName = "aerospike.server.namespace.objects"

	tests := []struct {
		name          string
		temporality   string
		allAsGauges   bool
		wantAfterGap  int64
		wantAfterThat int64
	}{
		// the cumulative value is 200 after the gap and 230 after that, the last report before the gap was 150
		{"delta counter", commons.TEMPORALITY_DELTA, false, 200, 30},
		{"cumulative counter", "cumulative", false, 200, 230},
		{"all metrics as gauges", commons.TEMPORALITY_DELTA, true, 200, 230},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			useOtelNaming(t)
			config.Cfg.Agent.Otel.CounterTemporality = tc.temporality
			config.Cfg.Agent.Otel.AllMetricsAsGauge = tc.allAsGauges
			oe := newConnectedExecutor()

			reader := sdkmetric.NewManualReader(sdkmetric.WithTemporalitySelector(oe.getTemporalitySelector))
			provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
			t.Cleanup(func() { require.NoError(t, provider.Shutdown(context.Background())) })
			meter := provider.Meter("ape-otel-delta-test")

			counter := func(value float64) statprocessors.AerospikeStat {
				return statprocessors.AerospikeStat{Context: commons.CTX_NAMESPACE, Name: "objects", MType: commons.MetricTypeCounter, IsAllowed: true, Value: value}
			}
			// a stat that is always returned, so that the refresh that drops the counter still completes
			node := statprocessors.AerospikeStat{Context: commons.CTX_NODE_STATS, Name: "cluster_size", MType: commons.MetricTypeGauge, IsAllowed: true, Value: 3}

			refresh := func(stats ...statprocessors.AerospikeStat) {
				oe.processAndPushStats(meter, nil, append(stats, node), true)
			}

			// the first export of a series is always its whole value
			refresh(counter(100))
			value, ok := exportedCounterValue(t, reader, counterName)
			require.True(t, ok)
			require.Equal(t, int64(100), value)

			// 50 more objects: a delta counter reports 50, the others the cumulative 150
			refresh(counter(150))
			value, ok = exportedCounterValue(t, reader, counterName)
			require.True(t, ok)
			if tc.temporality == commons.TEMPORALITY_DELTA && !tc.allAsGauges {
				require.Equal(t, int64(50), value)
			} else {
				require.Equal(t, int64(150), value)
			}

			// the server stops returning the series
			refresh()
			_, ok = exportedCounterValue(t, reader, counterName)
			require.False(t, ok, "a series that is not returned is not exported")

			// it comes back: only 50 more objects than the 150 last reported, but the delta is the whole 200
			refresh(counter(200))
			value, ok = exportedCounterValue(t, reader, counterName)
			require.True(t, ok)
			require.Equal(t, tc.wantAfterGap, value)

			// and the next export is a normal delta again
			refresh(counter(230))
			value, ok = exportedCounterValue(t, reader, counterName)
			require.True(t, ok)
			require.Equal(t, tc.wantAfterThat, value)
		})
	}
}

// sentCounterExporter records the value of the counter series in every export that reaches the backend.
type sentCounterExporter struct {
	recordingExporter
	name string
	sent []int64
}

func (s *sentCounterExporter) Export(_ context.Context, rm *metricdata.ResourceMetrics) error {
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if data, ok := m.Data.(metricdata.Sum[int64]); ok && m.Name == s.name {
				for _, dp := range data.DataPoints {
					s.sent = append(s.sent, dp.Value)
				}
			}
		}
	}

	return nil
}

// TestOtelDeltaCounterAcrossAServerOutage follows a delta counter through an outage between the exporter and the
// server, the common way a series goes missing, and a reconnect. While the server is unreachable the refresh fails,
// which resets the refresh counter, and the series is not observed. After the reconnect the SDK reports the whole
// cumulative value as its first delta, and the export warm-up drops that export and the next one. What reaches the
// backend never counts anything twice: the increase during the outage and the warm-up is not reported, which is
// the cost of a delta counter that cannot know what happened while it was not observed.
func TestOtelDeltaCounterAcrossAServerOutage(t *testing.T) {
	const counterName = "aerospike.server.namespace.objects"

	useOtelNaming(t)
	config.Cfg.Agent.Otel.CounterTemporality = commons.TEMPORALITY_DELTA
	config.Cfg.Agent.Otel.AllMetricsAsGauge = false

	provider := &otelProvider{connected: true}
	backend := &sentCounterExporter{name: counterName}
	oe := newConnectedExecutor()
	oe.dataProvider = provider
	oe.Exporter = backend

	reader := sdkmetric.NewManualReader(sdkmetric.WithTemporalitySelector(oe.getTemporalitySelector))
	meterProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() { require.NoError(t, meterProvider.Shutdown(context.Background())) })
	meter := meterProvider.Meter("ape-otel-outage-test")

	counter := func(value float64) statprocessors.AerospikeStat {
		return statprocessors.AerospikeStat{Context: commons.CTX_NAMESPACE, Name: "objects", MType: commons.MetricTypeCounter, IsAllowed: true, Value: value}
	}

	// a refresh that succeeded: the stats are recorded and the refresh counter goes up
	refresh := func(refreshCount int64, value float64) {
		oe.processAndPushStats(meter, nil, []statprocessors.AerospikeStat{counter(value)}, true)
		oe.refreshCounter.Store(refreshCount)
	}

	// one export interval: the SDK collects and the exporter sends, or drops, the result
	export := func() {
		var rm metricdata.ResourceMetrics
		require.NoError(t, reader.Collect(context.Background(), &rm))
		require.NoError(t, oe.Export(context.Background(), &rm))
	}

	// steady state
	refresh(3, 100)
	export()
	refresh(4, 150)
	export()
	require.Equal(t, []int64{100, 50}, backend.sent)

	// the server becomes unreachable: the refresh fails, the counter is reset, and nothing is observed
	provider.connected = false
	oe.refreshCounter.Store(0)
	export()
	require.Equal(t, []int64{100, 50}, backend.sent, "no counter value is exported while the server is unreachable")

	// the server is back and the counter went up by 150 in the meantime: the first deltas carry its whole value
	provider.connected = true
	refresh(1, 300)
	export()
	refresh(2, 330)
	export()
	require.Equal(t, []int64{100, 50}, backend.sent, "the warm-up drops the exports with the large first delta")

	// the warm-up is over, and the deltas are normal again
	refresh(3, 360)
	export()
	require.Equal(t, []int64{100, 50, 30}, backend.sent)

	var total int64
	for _, v := range backend.sent {
		total += v
	}
	require.LessOrEqual(t, total, int64(360), "nothing is counted twice")
}
