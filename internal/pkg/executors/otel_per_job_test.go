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

// useOtelNaming resets the config for one test and restores it afterwards, so the metric naming
// set here does not leak into other tests.
func useOtelNaming(t *testing.T) {
	t.Helper()

	original := config.Cfg
	t.Cleanup(func() { config.Cfg = original })

	config.Cfg = config.Config{}
	config.Cfg.Agent.Otel.MetricNamePrefix = "aerospike.server"
	config.Cfg.Agent.Otel.MetricContextSeparator = "period"
}

// newConnectedExecutor builds an executor the way Initialize does, with a connected server.
func newConnectedExecutor() *OtelExecutor {
	return &OtelExecutor{
		gauges:       map[string]*GaugeMetrics{},
		floatGauges:  map[string]*FloatGaugeMetrics{},
		counters:     map[string]*CounterMetrics{},
		sweptKeys:    map[string]struct{}{},
		dataProvider: &otelProvider{connected: true},
	}
}

func jobStat(context commons.ContextType, name string, mType commons.MetricType, jobID string, state string) statprocessors.AerospikeStat {
	return statprocessors.AerospikeStat{
		Context:     context,
		Name:        name,
		MType:       mType,
		IsAllowed:   true,
		Value:       1,
		Labels:      []string{commons.METRIC_LABEL_JOB_ID, commons.METRIC_LABEL_STATE},
		LabelValues: []string{jobID, state},
	}
}

// collectStates returns the "state" attribute of every data point reported for a metric.
func collectStates(t *testing.T, reader *sdkmetric.ManualReader, metricName string) []string {
	t.Helper()

	rm := collectMetrics(t, reader)
	states := []string{}
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != metricName {
				continue
			}

			switch data := m.Data.(type) {
			case metricdata.Gauge[int64]:
				for _, dp := range data.DataPoints {
					v, _ := dp.Attributes.Value(commons.METRIC_LABEL_STATE)
					states = append(states, v.AsString())
				}
			case metricdata.Gauge[float64]:
				for _, dp := range data.DataPoints {
					v, _ := dp.Attributes.Value(commons.METRIC_LABEL_STATE)
					states = append(states, v.AsString())
				}
			case metricdata.Sum[int64]:
				for _, dp := range data.DataPoints {
					v, _ := dp.Attributes.Value(commons.METRIC_LABEL_STATE)
					states = append(states, v.AsString())
				}
			}
		}
	}

	return states
}

// collectGaugeValues returns every value reported for a metric, and the value type of its gauge.
func collectGaugeValues(t *testing.T, reader *sdkmetric.ManualReader, metricName string) (values []float64, valueType string) {
	t.Helper()

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(context.Background(), &rm))

	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != metricName {
				continue
			}

			switch data := m.Data.(type) {
			case metricdata.Gauge[float64]:
				valueType = "float64"
				for _, dp := range data.DataPoints {
					values = append(values, dp.Value)
				}
			case metricdata.Gauge[int64]:
				valueType = "int64"
				for _, dp := range data.DataPoints {
					values = append(values, float64(dp.Value))
				}
			}
		}
	}

	return values, valueType
}

// TestOtelPerJobSeriesAreSwept checks that a backup job's previous state, and a job that is no
// longer reported, stop being exported, like any other series that a completed refresh no longer returns.
func TestOtelPerJobSeriesAreSwept(t *testing.T) {
	useOtelNaming(t)
	meter, reader := testMeter(t)
	oe := newConnectedExecutor()

	nsStat := jobStat(commons.CTX_NAMESPACE, "master_objects", commons.MetricTypeGauge, "-", "-")

	// first refresh: the job is in INIT, and a restore counter and a namespace stat are reported
	oe.processAndPushStats(meter, nil, []statprocessors.AerospikeStat{
		jobStat(commons.CTX_BACKUP, "job", commons.MetricTypeGauge, "b1", "INIT"),
		jobStat(commons.CTX_RESTORE, "applied", commons.MetricTypeCounter, "r1", "ACTIVE"),
		nsStat,
	}, true)

	require.Equal(t, []string{"INIT"}, collectStates(t, reader, "aerospike.server.backup.job"))
	require.Equal(t, []string{"ACTIVE"}, collectStates(t, reader, "aerospike.server.restore.applied"))

	// second refresh: the job moved on, the restore job aged out, and the namespace stat is absent once
	oe.processAndPushStats(meter, nil, []statprocessors.AerospikeStat{
		jobStat(commons.CTX_BACKUP, "job", commons.MetricTypeGauge, "b1", "BASE_SCAN_ACTIVE"),
	}, true)

	require.Equal(t, []string{"BASE_SCAN_ACTIVE"}, collectStates(t, reader, "aerospike.server.backup.job"),
		"the INIT series must stop reporting once the job changes state")
	require.Empty(t, collectStates(t, reader, "aerospike.server.restore.applied"),
		"a job that is no longer reported must stop being exported")
	require.Empty(t, collectStates(t, reader, "aerospike.server.namespace.master_objects"),
		"a namespace stat that is no longer returned is swept like a job series")

	// a refresh that does not sweep (system info) must not remove backup series
	oe.processAndPushStats(meter, nil, nil, false)
	require.Equal(t, []string{"BASE_SCAN_ACTIVE"}, collectStates(t, reader, "aerospike.server.backup.job"))

	// no backup jobs left
	oe.processAndPushStats(meter, nil, nil, true)
	require.Empty(t, collectStates(t, reader, "aerospike.server.backup.job"))
	require.Empty(t, oe.sweptKeys)
}

// TestOtelBlockedStatsAreNotExported checks that a stat filtered out by an allowlist or blocklist
// (IsAllowed=false) never becomes an OTel series, matching the Prometheus path.
func TestOtelBlockedStatsAreNotExported(t *testing.T) {
	useOtelNaming(t)
	meter, reader := testMeter(t)
	oe := newConnectedExecutor()

	allowed := jobStat(commons.CTX_NAMESPACE, "master_objects", commons.MetricTypeGauge, "-", "-")
	blocked := jobStat(commons.CTX_NAMESPACE, "client_read_success", commons.MetricTypeCounter, "-", "-")
	blocked.IsAllowed = false
	blockedJob := jobStat(commons.CTX_BACKUP, "job", commons.MetricTypeGauge, "b1", "INIT")
	blockedJob.IsAllowed = false

	oe.processAndPushStats(meter, nil, []statprocessors.AerospikeStat{allowed, blocked, blockedJob}, true)

	require.Len(t, collectStates(t, reader, "aerospike.server.namespace.master_objects"), 1)
	require.Empty(t, collectStates(t, reader, "aerospike.server.namespace.client_read_success"),
		"a blocked stat must not be exported")
	require.Empty(t, collectStates(t, reader, "aerospike.server.backup.job"),
		"a blocked backup stat must not be exported")
	require.Len(t, oe.gauges, 1)
	require.Empty(t, oe.floatGauges)
	require.Empty(t, oe.counters)
}

// TestOtelBackupRestoreGaugesKeepFractions checks that a fractional backup or restore gauge such as
// progress-pct=42.50 is exported as 42.5 and not truncated to 42, while a gauge from an existing context
// stays an int64 so its exported value type does not change.
func TestOtelBackupRestoreGaugesKeepFractions(t *testing.T) {
	useOtelNaming(t)
	meter, reader := testMeter(t)
	oe := newConnectedExecutor()

	progress := jobStat(commons.CTX_BACKUP, "progress-pct", commons.MetricTypeGauge, "b1", "BASE_SCAN_ACTIVE")
	progress.Value = 42.50
	restoreProgress := jobStat(commons.CTX_RESTORE, "progress-pct", commons.MetricTypeGauge, "r1", "ACTIVE")
	restoreProgress.Value = 0.25
	nsStat := jobStat(commons.CTX_NAMESPACE, "master_objects", commons.MetricTypeGauge, "-", "-")
	nsStat.Value = 7

	oe.processAndPushStats(meter, nil, []statprocessors.AerospikeStat{progress, restoreProgress, nsStat}, true)

	values, valueType := collectGaugeValues(t, reader, "aerospike.server.backup.progress_pct")
	require.Equal(t, "float64", valueType)
	require.Equal(t, []float64{42.5}, values, "the fractional part must not be dropped")

	values, valueType = collectGaugeValues(t, reader, "aerospike.server.restore.progress_pct")
	require.Equal(t, "float64", valueType)
	require.Equal(t, []float64{0.25}, values)

	values, valueType = collectGaugeValues(t, reader, "aerospike.server.namespace.master_objects")
	require.Equal(t, "int64", valueType, "gauges of existing contexts keep their value type")
	require.Equal(t, []float64{7}, values)

	// the value follows the latest refresh, and a restore job that is no longer reported stops being exported
	progress.Value = 87.75
	oe.processAndPushStats(meter, nil, []statprocessors.AerospikeStat{progress}, true)

	values, _ = collectGaugeValues(t, reader, "aerospike.server.backup.progress_pct")
	require.Equal(t, []float64{87.75}, values)

	values, _ = collectGaugeValues(t, reader, "aerospike.server.restore.progress_pct")
	require.Empty(t, values, "a restore job that is no longer reported must stop being exported")

	oe.processAndPushStats(meter, nil, nil, true)
	values, _ = collectGaugeValues(t, reader, "aerospike.server.backup.progress_pct")
	require.Empty(t, values)
	require.Empty(t, oe.floatGauges)
}

// latencyStat is a latency histogram series for one namespace, as the latency processor builds them.
func latencyStat(name string, value float64) statprocessors.AerospikeStat {
	return statprocessors.AerospikeStat{
		Context:     commons.CTX_LATENCIES,
		Name:        name,
		MType:       commons.MetricTypeGauge,
		IsAllowed:   true,
		Value:       value,
		Labels:      []string{commons.METRIC_LABEL_NS},
		LabelValues: []string{"test"},
	}
}

// requireNoPoint checks that a metric currently reports no data point at all.
func requireNoPoint(t *testing.T, reader *sdkmetric.ManualReader, name string, msg string) {
	t.Helper()

	_, _, ok := metricPoint(t, collectMetrics(t, reader), name)
	require.False(t, ok, msg)
}

// requirePoint checks the value a metric currently reports.
func requirePoint(t *testing.T, reader *sdkmetric.ManualReader, name string, expected int64) {
	t.Helper()

	value, _, ok := metricPoint(t, collectMetrics(t, reader), name)
	require.True(t, ok, "%s must be reported", name)
	require.Equal(t, expected, value, name)
}

// TestOtelObjectStoreLatencySeriesAreSwept follows the object-store histograms from available to unavailable,
// as after a server restart where the histograms are not registered yet, and back again. Without the sweep
// the cached series keeps exporting its previous value for as long as the server is connected.
func TestOtelObjectStoreLatencySeriesAreSwept(t *testing.T) {
	useOtelNaming(t)
	meter, reader := testMeter(t)
	oe := newConnectedExecutor()

	const put = "aerospike.server.latencies.object_store_put_ms_count"
	const get = "aerospike.server.latencies.object_store_get_ms_count"
	const read = "aerospike.server.latencies.benchmarks_read_ms_count"

	// the histograms are available, next to a regular latency histogram
	oe.processAndPushStats(meter, nil, []statprocessors.AerospikeStat{
		latencyStat("object-store-put_ms_count", 7),
		latencyStat("object-store-get_ms_count", 3),
		latencyStat("benchmarks-read_ms_count", 11),
	}, true)

	requirePoint(t, reader, put, 7)
	requirePoint(t, reader, get, 3)
	requirePoint(t, reader, read, 11)

	// a refresh that does not sweep (system info) never removes them
	oe.processAndPushStats(meter, nil, nil, false)
	requirePoint(t, reader, put, 7)

	// the server restarted: a completed refresh returns no object-store histogram, but the regular one
	oe.processAndPushStats(meter, nil, []statprocessors.AerospikeStat{
		latencyStat("benchmarks-read_ms_count", 12),
	}, true)

	requireNoPoint(t, reader, put, "an unavailable object-store histogram must stop exporting its previous value")
	requireNoPoint(t, reader, get, "an unavailable object-store histogram must stop exporting its previous value")
	requirePoint(t, reader, read, 12)
	require.Len(t, oe.gauges, 1, "only the regular latency series stays registered")
	require.Len(t, oe.sweptKeys, 1)

	// the histograms come back with fresh values, not the ones from before the restart
	oe.processAndPushStats(meter, nil, []statprocessors.AerospikeStat{
		latencyStat("object-store-put_ms_count", 1),
		latencyStat("object-store-get_ms_count", 1),
		latencyStat("benchmarks-read_ms_count", 13),
	}, true)

	requirePoint(t, reader, put, 1)
	requirePoint(t, reader, get, 1)

	// only one of them goes away: the other keeps reporting
	oe.processAndPushStats(meter, nil, []statprocessors.AerospikeStat{
		latencyStat("object-store-put_ms_count", 2),
		latencyStat("benchmarks-read_ms_count", 14),
	}, true)

	requirePoint(t, reader, put, 2)
	requireNoPoint(t, reader, get, "get is no longer returned")
	requirePoint(t, reader, read, 14)
}

// TestOtelRemovedStatsStopReporting checks the sweep on series that have nothing to do with backup and restore,
// the way a config change shows up: latency histograms switched from milliseconds to microseconds are a new set of
// series, and the old ones must stop reporting, while a series that is still returned keeps its latest value.
func TestOtelRemovedStatsStopReporting(t *testing.T) {
	useOtelNaming(t)
	meter, reader := testMeter(t)
	oe := newConnectedExecutor()

	const clusterSize = "aerospike.server.node_stats.cluster_size"
	const readMs = "aerospike.server.latencies.benchmarks_read_ms_count"
	const readUs = "aerospike.server.latencies.benchmarks_read_us_count"

	nodeStat := statprocessors.AerospikeStat{Context: commons.CTX_NODE_STATS, Name: "cluster_size", MType: commons.MetricTypeGauge, IsAllowed: true, Value: 3}

	oe.processAndPushStats(meter, nil, []statprocessors.AerospikeStat{nodeStat, latencyStat("benchmarks-read_ms_count", 10)}, true)
	requirePoint(t, reader, clusterSize, 3)
	requirePoint(t, reader, readMs, 10)

	// microsecond-histograms is switched on: the same histogram is now reported in us
	nodeStat.Value = 4
	oe.processAndPushStats(meter, nil, []statprocessors.AerospikeStat{nodeStat, latencyStat("benchmarks-read_us_count", 20)}, true)

	requirePoint(t, reader, clusterSize, 4)
	requirePoint(t, reader, readUs, 20)
	requireNoPoint(t, reader, readMs, "the ms series is no longer returned and must stop reporting")

	// and switched off again: the ms series comes back with a fresh value, the us one stops
	oe.processAndPushStats(meter, nil, []statprocessors.AerospikeStat{nodeStat, latencyStat("benchmarks-read_ms_count", 1)}, true)

	requirePoint(t, reader, readMs, 1)
	requireNoPoint(t, reader, readUs, "the us series is no longer returned and must stop reporting")
	requirePoint(t, reader, clusterSize, 4)
}

// TestOtelSystemInfoSeriesAreNotSwept checks that the host system info, a separate refresh that never sweeps,
// keeps its series when a server refresh does not return them, and that a server refresh does not claim them.
func TestOtelSystemInfoSeriesAreNotSwept(t *testing.T) {
	useOtelNaming(t)
	meter, reader := testMeter(t)
	oe := newConnectedExecutor()

	const memory = "aerospike.server.system_info.memory_total"
	const clusterSize = "aerospike.server.node_stats.cluster_size"

	sysStat := statprocessors.AerospikeStat{Context: "system_info", Name: "memory_total", MType: commons.MetricTypeGauge, IsAllowed: true, Value: 64}
	nodeStat := statprocessors.AerospikeStat{Context: commons.CTX_NODE_STATS, Name: "cluster_size", MType: commons.MetricTypeGauge, IsAllowed: true, Value: 3}

	oe.processAndPushStats(meter, nil, []statprocessors.AerospikeStat{sysStat}, false)
	oe.processAndPushStats(meter, nil, []statprocessors.AerospikeStat{nodeStat}, true)

	// the server refresh that returned only node stats does not touch the system info series
	oe.processAndPushStats(meter, nil, []statprocessors.AerospikeStat{nodeStat}, true)
	requirePoint(t, reader, memory, 64)
	requirePoint(t, reader, clusterSize, 3)
	require.Len(t, oe.sweptKeys, 1, "only the series of server refreshes are tracked")
}

// TestOtelIntervalFetchedStatsAreNotSwept checks that the index-pressure stats, which the server is only asked
// for every few minutes, keep their last value on the refreshes in between instead of flapping.
func TestOtelIntervalFetchedStatsAreNotSwept(t *testing.T) {
	useOtelNaming(t)
	meter, reader := testMeter(t)
	oe := newConnectedExecutor()

	const pressure = "aerospike.server.namespace.index_pressure_total_memory"

	pressureStat := statprocessors.AerospikeStat{Context: commons.CTX_NAMESPACE, Name: "index_pressure_total_memory", MType: commons.MetricTypeGauge, IsAllowed: true, Value: 100}
	nodeStat := statprocessors.AerospikeStat{Context: commons.CTX_NODE_STATS, Name: "cluster_size", MType: commons.MetricTypeGauge, IsAllowed: true, Value: 3}

	oe.processAndPushStats(meter, nil, []statprocessors.AerospikeStat{pressureStat, nodeStat}, true)
	requirePoint(t, reader, pressure, 100)

	// the next refreshes do not fetch index-pressure
	oe.processAndPushStats(meter, nil, []statprocessors.AerospikeStat{nodeStat}, true)
	oe.processAndPushStats(meter, nil, []statprocessors.AerospikeStat{nodeStat}, true)
	requirePoint(t, reader, pressure, 100)
}

// TestOtelCheckpointSaveRefreshDoesNotSweep checks that a refresh cut short by a checkpoint save, which returns
// only the checkpoint stats, does not drop every other series, and that the sweep works again afterwards.
func TestOtelCheckpointSaveRefreshDoesNotSweep(t *testing.T) {
	useOtelNaming(t)
	meter, reader := testMeter(t)
	oe := newConnectedExecutor()
	oe.sharedState = statprocessors.NewStatProcessorSharedState()

	const clusterSize = "aerospike.server.node_stats.cluster_size"
	const checkpoint = "aerospike.server.namespace.checkpoint_files_total"

	nodeStat := statprocessors.AerospikeStat{Context: commons.CTX_NODE_STATS, Name: "cluster_size", MType: commons.MetricTypeGauge, IsAllowed: true, Value: 3}
	checkpointStat := statprocessors.AerospikeStat{Context: commons.CTX_NAMESPACE, Name: "checkpoint_files_total", MType: commons.MetricTypeGauge, IsAllowed: true, Value: 4}

	oe.processAndPushStats(meter, nil, []statprocessors.AerospikeStat{nodeStat}, true)
	requirePoint(t, reader, clusterSize, 3)

	// the save: only checkpoint stats come back
	oe.sharedState.CheckpointSaveInProgress = true
	oe.processAndPushStats(meter, nil, []statprocessors.AerospikeStat{checkpointStat}, true)
	requirePoint(t, reader, clusterSize, 3)
	requirePoint(t, reader, checkpoint, 4)

	// the save is over, and the next completed refresh no longer returns the checkpoint stat
	oe.sharedState.CheckpointSaveInProgress = false
	oe.processAndPushStats(meter, nil, []statprocessors.AerospikeStat{nodeStat}, true)
	requirePoint(t, reader, clusterSize, 3)
	requireNoPoint(t, reader, checkpoint, "a series first reported during the save is swept by the next completed refresh")
}
