package statprocessors

import (
	"strings"
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

// namespaceLatencyCommands returns the histogram commands that the latency processor asks for, one per entry.
func namespaceLatencyCommands(state *StatProcessorSharedState) []string {
	return NewLatencyStatsProcessor(state).getLatenciesCommands(nil)
}

func hasCommandFor(commands []string, ns string) bool {
	for _, command := range commands {
		if strings.Contains(command, "{"+ns+"}") {
			return true
		}
	}

	return false
}

// TestRemovedNamespaceStopsGettingLatencyCommands follows a namespace that a restarted server no longer has:
// once a successful namespaces response omits it, no latency command for it is sent any more, while the
// namespaces that remain keep theirs.
func TestRemovedNamespaceStopsGettingLatencyCommands(t *testing.T) {
	state := useProcessorConfig(t)
	state.Build = "8.2.2.0"
	nw := NewNamespaceStatsProcessor(state)

	refreshNamespaces := func(namespaces ...string) {
		keys := nw.PassTwoKeys(map[string]string{KEY_NS_METADATA: strings.Join(namespaces, ";")})
		raw := map[string]string{}
		for _, ns := range namespaces {
			raw["namespace/"+ns] = "strong-consistency=true;master_objects=1"
		}
		_, err := nw.Refresh(keys, raw)
		require.NoError(t, err)
	}

	refreshNamespaces("old", "keep")
	commands := namespaceLatencyCommands(state)
	require.True(t, hasCommandFor(commands, "old"))
	require.True(t, hasCommandFor(commands, "keep"))
	assert.Contains(t, commands, "latencies:hist={old}-re-repl")
	assert.Contains(t, commands, "latencies:hist={old}-object-store-put")
	assert.True(t, nw.namespaceSCstatus["old"])

	// an error answer, or an empty one, does not say which namespaces exist
	for _, answer := range []string{"", "ERROR:4:unrecognized command"} {
		nw.PassTwoKeys(map[string]string{KEY_NS_METADATA: answer})
		assert.True(t, hasCommandFor(namespaceLatencyCommands(state), "old"), "answer %q must keep the namespaces", answer)
	}

	// a checkpoint save asks for nothing, so it cannot confirm anything either
	state.CheckpointSaveInProgress = true
	nw.PassTwoKeys(map[string]string{KEY_NS_METADATA: "keep"})
	state.CheckpointSaveInProgress = false
	assert.True(t, hasCommandFor(namespaceLatencyCommands(state), "old"))

	// the restarted server has the new namespace and not the old one
	refreshNamespaces("keep", "new")
	commands = namespaceLatencyCommands(state)
	assert.False(t, hasCommandFor(commands, "old"), "no latency command for old remains: %v", commands)
	assert.True(t, hasCommandFor(commands, "keep"))
	assert.True(t, hasCommandFor(commands, "new"))
	assert.NotContains(t, state.NamespaceLatencyBenchmarks, "old")
	assert.NotContains(t, nw.namespaceSCstatus, "old")
}
