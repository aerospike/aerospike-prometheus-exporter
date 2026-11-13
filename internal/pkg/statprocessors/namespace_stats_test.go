package statprocessors

import (
	"testing"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNamespaceStatsProcessorRefresh(t *testing.T) {
	state := useProcessorConfig(t)
	config.Cfg.Aerospike.NamespaceMetricsBlocklist = []string{"data_used_bytes"}
	processor := NewNamespaceStatsProcessor(state)

	assert.Equal(t, []string{KEY_NS_METADATA}, processor.PassOneKeys())
	state.CheckpointSaveInProgress = true
	assert.Nil(t, processor.PassOneKeys())
	state.CheckpointSaveInProgress = false

	passTwo := processor.PassTwoKeys(map[string]string{KEY_NS_METADATA: "test"})
	assert.Contains(t, passTwo, "namespace/test")

	infoKey := "namespace/test"
	stats, err := processor.Refresh([]string{infoKey}, map[string]string{
		infoKey: "objects=5;data_used_bytes=20;storage-engine=memory;index-type=flash;index-type.used=1",
	})
	require.NoError(t, err)

	objects, ok := findStat(stats, "objects")
	require.True(t, ok)
	assert.Equal(t, 5.0, objects.Value)
	assert.Equal(t, commons.CTX_NAMESPACE, objects.Context)
	assert.Equal(t, commons.MetricTypeCounter, metricTypeOf(objects))
	assert.True(t, objects.IsAllowed)
	assertLabel(t, objects, commons.METRIC_LABEL_NS, "test")
	assertLabel(t, objects, commons.METRIC_LABEL_STORAGE_ENGINE, "memory")

	used, ok := findStat(stats, "data_used_bytes")
	require.True(t, ok)
	assert.False(t, used.IsAllowed)

	indexUsed, ok := findStat(stats, "index-type.used")
	require.True(t, ok)
	assert.Equal(t, 1.0, indexUsed.Value)
	assert.Equal(t, commons.MetricTypeGauge, metricTypeOf(indexUsed))
	assertLabel(t, indexUsed, commons.METRIC_LABEL_INDEX, "flash")
	assert.True(t, processor.isFlashStatSentByServer)

	_, ok = findStat(stats, "storage-engine")
	assert.False(t, ok, "a non-numeric storage-engine value is a label, not a metric")
}
