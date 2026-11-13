package statprocessors

import (
	"testing"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSindexStatsProcessorBuildsCommandsAndMetrics(t *testing.T) {
	state := useProcessorConfig(t)
	processor := NewSindexStatsProcessor(state)

	assert.Equal(t, []string{KEY_SINDEX_COMMAND}, processor.PassOneKeys())

	passOne := map[string]string{
		KEY_SINDEX_COMMAND: "ns=test:indexname=name_idx:set=demo:bin=name:type=string",
	}
	assert.Equal(t, []string{"sindex-stat:namespace=test;indexname=name_idx"}, processor.PassTwoKeys(passOne))

	infoKey := "sindex-stat:namespace=test;indexname=name_idx"
	stats, err := processor.Refresh([]string{infoKey}, map[string]string{
		infoKey: "entries=12;entries_per_rec=1.5",
	})
	require.NoError(t, err)

	entries, ok := findStat(stats, "entries")
	require.True(t, ok)
	assert.Equal(t, 12.0, entries.Value)
	assert.Equal(t, commons.CTX_SINDEX, entries.Context)
	assert.Equal(t, commons.MetricTypeCounter, metricTypeOf(entries))
	assertLabel(t, entries, commons.METRIC_LABEL_NS, "test")
	assertLabel(t, entries, commons.METRIC_LABEL_SINDEX, "name_idx")

	perRec, ok := findStat(stats, "entries_per_rec")
	require.True(t, ok)
	assert.Equal(t, 1.5, perRec.Value)
}

func TestSindexStatsProcessorDisabled(t *testing.T) {
	state := useProcessorConfig(t)
	config.Cfg.Aerospike.DisableSindexMetrics = true
	processor := NewSindexStatsProcessor(state)

	assert.Nil(t, processor.PassOneKeys())
	assert.Nil(t, processor.PassTwoKeys(map[string]string{KEY_SINDEX_COMMAND: "ns=test:indexname=name_idx"}))

	stats, err := processor.Refresh([]string{"sindex-stat:namespace=test;indexname=name_idx"}, map[string]string{
		"sindex-stat:namespace=test;indexname=name_idx": "entries=12",
	})
	require.NoError(t, err)
	assert.Nil(t, stats)
}

func TestSindexStatsProcessorSkipsDuringCheckpoint(t *testing.T) {
	state := useProcessorConfig(t)
	state.CheckpointSaveInProgress = true
	processor := NewSindexStatsProcessor(state)

	assert.Nil(t, processor.PassOneKeys())
	assert.Nil(t, processor.PassTwoKeys(nil))
}
