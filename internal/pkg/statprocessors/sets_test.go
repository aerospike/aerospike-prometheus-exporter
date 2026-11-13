package statprocessors

import (
	"testing"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetsStatsProcessorRefresh(t *testing.T) {
	state := useProcessorConfig(t)
	config.Cfg.Aerospike.SetMetricsAllowlist = []string{"objects"}
	processor := NewSetsStatsProcessor(state)

	assert.Nil(t, processor.PassOneKeys())
	assert.Equal(t, []string{KEY_SETS}, processor.PassTwoKeys(nil))

	state.CheckpointSaveInProgress = true
	assert.Nil(t, processor.PassTwoKeys(nil))
	state.CheckpointSaveInProgress = false

	stats, err := processor.Refresh(nil, map[string]string{
		KEY_SETS: "ns=test:set=demo:objects=4:truncate-lut=9;",
	})
	require.NoError(t, err)

	objects, ok := findStatWithLabel(stats, "objects", commons.METRIC_LABEL_SET, "demo")
	require.True(t, ok)
	assert.Equal(t, 4.0, objects.Value)
	assert.Equal(t, commons.CTX_SETS, objects.Context)
	assert.True(t, objects.IsAllowed)
	assertLabel(t, objects, commons.METRIC_LABEL_NS, "test")
	assertLabel(t, objects, commons.METRIC_LABEL_CLUSTER_NAME, "cluster-a")

	truncate, ok := findStat(stats, "truncate-lut")
	require.True(t, ok)
	assert.Equal(t, 9.0, truncate.Value)
	assert.Equal(t, commons.MetricTypeGauge, metricTypeOf(truncate))
	assert.False(t, truncate.IsAllowed, "allowlist omits truncate-lut")

	_, ok = findStat(stats, "set")
	assert.False(t, ok, "the set name is not a numeric metric")
}
