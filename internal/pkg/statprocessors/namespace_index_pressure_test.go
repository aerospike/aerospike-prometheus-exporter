package statprocessors

import (
	"testing"
	"time"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestIndexPressureFollowsConfiguredInterval(t *testing.T) {
	original := config.Cfg
	t.Cleanup(func() { config.Cfg = original })

	config.Cfg = config.Config{}
	config.Cfg.Aerospike.IndexPressureStatsFetchInterval = 3

	nw := NewNamespaceStatsProcessor(NewStatProcessorSharedState())
	assert.Equal(t, 3.0, nw.idxPressureFetchInterval)

	passOne := map[string]string{KEY_NS_METADATA: "test"}
	nw.isFlashStatSentByServer = false
	nw.idxPressurePreviousFetchTime = time.Now().Add(-4 * time.Minute)
	assert.False(t, nw.canSendIndexPressureInfoKey())
	assert.NotContains(t, nw.PassTwoKeys(passOne), KEY_NS_INDEX_PRESSURE)

	nw.isFlashStatSentByServer = true
	nw.idxPressurePreviousFetchTime = time.Now().Add(-4 * time.Minute)
	assert.True(t, nw.canSendIndexPressureInfoKey())
	assert.Contains(t, nw.PassTwoKeys(passOne), KEY_NS_INDEX_PRESSURE)
	assert.NotContains(t, nw.PassTwoKeys(passOne), KEY_NS_INDEX_PRESSURE)

	config.Cfg.Aerospike.IndexPressureStatsFetchInterval = 0
	nw.idxPressurePreviousFetchTime = time.Now().Add(-4 * time.Minute)
	assert.False(t, nw.canSendIndexPressureInfoKey())
	assert.NotContains(t, nw.PassTwoKeys(passOne), KEY_NS_INDEX_PRESSURE)
}

// TestIsIntervalFetchedStat checks that every stat the index-pressure refresh builds is recognised as fetched
// only every few minutes, and that a regular namespace stat is not.
func TestIsIntervalFetchedStat(t *testing.T) {
	nw := NewNamespaceStatsProcessor(NewStatProcessorSharedState())

	stats := nw.refreshIndexPressure(KEY_NS_INDEX_PRESSURE, map[string]string{KEY_NS_INDEX_PRESSURE: "test:100:40;bar:0:0"})
	assert.NotEmpty(t, stats)
	for _, stat := range stats {
		assert.True(t, IsIntervalFetchedStat(stat), stat.Name)
	}

	assert.False(t, IsIntervalFetchedStat(AerospikeStat{Context: commons.CTX_NAMESPACE, Name: "master_objects"}))
	assert.False(t, IsIntervalFetchedStat(AerospikeStat{Context: commons.CTX_NODE_STATS, Name: "index_pressure_total_memory"}),
		"only the namespace context carries index-pressure")
}

// TestIndexPressureFetchedOnlyByAValidAnswer checks the confirmation that a consumer uses to drop the series of
// a namespace that is gone: a valid answer gives it, an error or an empty answer does not and gives no stats.
func TestIndexPressureFetchedOnlyByAValidAnswer(t *testing.T) {
	for _, answer := range []string{"", "ERROR:4:unrecognized command"} {
		state := NewStatProcessorSharedState()
		nw := NewNamespaceStatsProcessor(state)

		stats := nw.refreshIndexPressure(KEY_NS_INDEX_PRESSURE, map[string]string{KEY_NS_INDEX_PRESSURE: answer})
		assert.Empty(t, stats, "answer %q", answer)
		assert.False(t, state.IndexPressureFetched, "answer %q is no confirmation", answer)
	}

	state := NewStatProcessorSharedState()
	nw := NewNamespaceStatsProcessor(state)

	stats := nw.refreshIndexPressure(KEY_NS_INDEX_PRESSURE, map[string]string{KEY_NS_INDEX_PRESSURE: "bar:7:3"})
	assert.NotEmpty(t, stats)
	assert.True(t, state.IndexPressureFetched)
}
