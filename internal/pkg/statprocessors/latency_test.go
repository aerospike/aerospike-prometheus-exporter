package statprocessors

import (
	"testing"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLatencyStatsProcessorChoosesTheCommandForTheBuild(t *testing.T) {
	state := useProcessorConfig(t)
	state.Build = "5.0.0.0"
	processor := NewLatencyStatsProcessor(state)

	assert.Nil(t, processor.PassOneKeys())
	assert.Equal(t, []string{"latency:"}, processor.PassTwoKeys(nil))

	state.Build = "5.1.0.0"
	state.NamespaceLatencyBenchmarks["test"] = map[string]string{"read": "{test}-benchmarks-read"}
	state.ServiceLatencyBenchmarks["enable-hist-info"] = "info"
	keys := processor.PassTwoKeys(nil)
	assert.Contains(t, keys, "latencies:")
	assert.Contains(t, keys, "latencies:hist={test}-benchmarks-read")
	assert.Contains(t, keys, "latencies:hist=info")

	config.Cfg.Aerospike.DisableLatenciesMetrics = true
	assert.Nil(t, processor.PassTwoKeys(nil))

	config.Cfg.Aerospike.DisableLatenciesMetrics = false
	state.CheckpointSaveInProgress = true
	assert.Nil(t, processor.PassOneKeys())
	assert.Nil(t, processor.PassTwoKeys(nil))
}

func TestLatencyStatsProcessorRefreshParsesBuckets(t *testing.T) {
	state := useProcessorConfig(t)
	config.Cfg.Aerospike.LatencyBucketsCount = 1
	config.Cfg.Aerospike.LatenciesMetricsAllowlistEnabled = true
	config.Cfg.Aerospike.LatenciesMetricsAllowlist = []string{"read"}

	processor := NewLatencyStatsProcessor(state)
	raw := "{test}-read:msec,100,0,50;{test}-write:msec,20,0;"
	stats, err := processor.Refresh([]string{"latencies:"}, map[string]string{
		"latencies:": raw,
	})
	require.NoError(t, err)

	assert.Zero(t, countNamed(stats, "write_ms_bucket"))

	inf, ok := findStatWithLabel(stats, "read_ms_bucket", commons.METRIC_LABEL_LE, "+Inf")
	require.True(t, ok)
	assert.Equal(t, 100.0, inf.Value)
	assert.Equal(t, commons.CTX_LATENCIES, inf.Context)
	assert.Equal(t, commons.MetricTypeGauge, metricTypeOf(inf))
	assertLabel(t, inf, commons.METRIC_LABEL_NS, "test")

	bounded, ok := findStatWithLabel(stats, "read_ms_bucket", commons.METRIC_LABEL_LE, "1")
	require.True(t, ok)
	assert.Equal(t, 100.0, bounded.Value)

	count, ok := findStat(stats, "read_ms_count")
	require.True(t, ok)
	assert.Equal(t, 100.0, count.Value)
	assert.Equal(t, 1, countNamed(stats, "read_ms_count"))
}

func TestLatencyStatsProcessorRefreshDropsBlockedHistogram(t *testing.T) {
	state := useProcessorConfig(t)
	config.Cfg.Aerospike.LatencyBucketsCount = 1
	config.Cfg.Aerospike.LatenciesMetricsAllowlistEnabled = true
	config.Cfg.Aerospike.LatenciesMetricsAllowlist = []string{"read", "write"}
	config.Cfg.Aerospike.LatenciesMetricsBlocklist = []string{"read"}

	processor := NewLatencyStatsProcessor(state)
	stats, err := processor.Refresh([]string{"latencies:"}, map[string]string{
		"latencies:": "{test}-read:msec,100,0;{test}-write:msec,20,0;",
	})
	require.NoError(t, err)

	assert.Zero(t, countNamed(stats, "read_ms_bucket"))
	write, ok := findStatWithLabel(stats, "write_ms_bucket", commons.METRIC_LABEL_LE, "+Inf")
	require.True(t, ok)
	assert.Equal(t, 20.0, write.Value)
}
