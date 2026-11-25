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
		perJobKeys:   map[string]struct{}{},
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
// longer reported, stop being exported, while series from other contexts are left alone.
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
	require.Len(t, collectStates(t, reader, "aerospike.server.namespace.master_objects"), 1,
		"non-job series are never swept")

	// a refresh that does not sweep (system info) must not remove backup series
	oe.processAndPushStats(meter, nil, nil, false)
	require.Equal(t, []string{"BASE_SCAN_ACTIVE"}, collectStates(t, reader, "aerospike.server.backup.job"))

	// no backup jobs left
	oe.processAndPushStats(meter, nil, nil, true)
	require.Empty(t, collectStates(t, reader, "aerospike.server.backup.job"))
	require.Empty(t, oe.perJobKeys)
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
