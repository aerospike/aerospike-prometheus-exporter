package statprocessors

import (
	"testing"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNodeStatsProcessorRequestsVersionedInfoKeys(t *testing.T) {
	state := useProcessorConfig(t)
	processor := NewNodeStatsProcessor(state)

	assert.Equal(t, []string{KEY_SERVICE_LOGS}, processor.PassOneKeys())

	state.CheckpointSaveInProgress = true
	assert.Nil(t, processor.PassOneKeys())
	state.CheckpointSaveInProgress = false

	passOne := map[string]string{KEY_SERVICE_LOGS: "0:stderr"}
	keys := processor.PassTwoKeys(passOne)
	assert.Equal(t, []string{KEY_SERVICE_CONFIG, KEY_SERVICE_STATISTICS, "log/0"}, keys)

	state.Build = "8.1.0.0"
	keys = processor.PassTwoKeys(passOne)
	assert.Contains(t, keys, CMD_INFOKEY_USER_AGENTS)
	assert.NotContains(t, keys, CMD_SMD_INFO)

	state.Build = "8.2.0.1"
	keys = processor.PassTwoKeys(passOne)
	assert.Contains(t, keys, CMD_SMD_INFO)
	assert.Contains(t, keys, CMD_INFOKEY_CHECKPOINT_STATUS)

	state.CheckpointSaveInProgress = true
	assert.Equal(t, []string{CMD_INFOKEY_CHECKPOINT_STATUS}, processor.PassTwoKeys(passOne))
}

func TestNodeStatsProcessorRefreshBuildsNodeMetrics(t *testing.T) {
	state := useProcessorConfig(t)
	config.Cfg.Aerospike.NodeMetricsBlocklist = []string{"client_connections"}
	config.GaugeStatHandler.NodeStats["objects"] = true

	processor := NewNodeStatsProcessor(state)
	processor.PassTwoKeys(map[string]string{KEY_SERVICE_LOGS: "0:stderr;1:/var/log/aerospike/aerospike.log"})

	stats, err := processor.Refresh(nil, map[string]string{
		KEY_SERVICE_CONFIG:     "proto-fd-max=15000;cluster-name=cluster-a",
		KEY_SERVICE_STATISTICS: "client_connections=8;objects=3",
		"log/0":                "0:stderr:DEBUG",
		"log/1":                "1:/var/log/aerospike/aerospike.log:DETAIL",
	})
	require.NoError(t, err)

	fdMax, ok := findStat(stats, "proto-fd-max")
	require.True(t, ok)
	assert.Equal(t, 15000.0, fdMax.Value)
	assert.Equal(t, commons.MetricTypeGauge, metricTypeOf(fdMax))
	assert.True(t, fdMax.IsAllowed)
	assertLabel(t, fdMax, commons.METRIC_LABEL_CLUSTER_NAME, "cluster-a")
	assertLabel(t, fdMax, commons.METRIC_LABEL_SERVICE, "127.0.0.1:3000")

	connections, ok := findStat(stats, "client_connections")
	require.True(t, ok)
	assert.Equal(t, 8.0, connections.Value)
	assert.Equal(t, commons.MetricTypeCounter, metricTypeOf(connections))
	assert.False(t, connections.IsAllowed)

	objects, ok := findStat(stats, "objects")
	require.True(t, ok)
	assert.Equal(t, commons.MetricTypeGauge, metricTypeOf(objects))

	debug, ok := findStat(stats, "pseudo_log_debug")
	require.True(t, ok)
	assert.Equal(t, 1.0, debug.Value)
	detail, ok := findStat(stats, "pseudo_log_detail")
	require.True(t, ok)
	assert.Equal(t, 1.0, detail.Value)

	_, ok = findStat(stats, "cluster-name")
	assert.False(t, ok, "non-numeric config values are skipped")
}

func TestNodeStatsProcessorRecordsEnabledServiceLatency(t *testing.T) {
	state := useProcessorConfig(t)
	processor := NewNodeStatsProcessor(state)

	_, err := processor.Refresh(nil, map[string]string{
		KEY_SERVICE_STATISTICS: "enable-benchmarks-read=1;enable-benchmarks-write=0",
	})
	require.NoError(t, err)

	assert.Equal(t, "benchmarks-read", state.ServiceLatencyBenchmarks["enable-benchmarks-read"])
	_, enabled := state.ServiceLatencyBenchmarks["enable-benchmarks-write"]
	assert.False(t, enabled)
}

func TestNodeStatsProcessorCheckpointStatusShapes(t *testing.T) {
	idle := "test:state=none:files_completed=1:files_total=4:is_parked=false:park_ms=0;" +
		"employees_flash:state=none:files_completed=0:files_total=0:is_parked=false:park_ms=0"
	metrics := refreshCheckpoint(t, idle)

	completed, ok := findStatWithLabel(metrics, "checkpoint_files_completed", commons.METRIC_LABEL_NS, "test")
	require.True(t, ok)
	assert.Equal(t, 1.0, completed.Value)
	assertLabel(t, completed, commons.METRIC_LABEL_CHECKPOINT_STATUS, "none")
	_, ok = findStat(metrics, "checkpoint_state")
	assert.False(t, ok)
	_, ok = findStat(metrics, "client_connections")
	assert.True(t, ok, "an idle per-namespace body still refreshes the other node stats")

	saving := "test:state=saving:files_completed=2:files_total=4:is_parked=false:park_ms=5"
	metrics = refreshCheckpoint(t, saving)
	completed, ok = findStatWithLabel(metrics, "checkpoint_files_completed", commons.METRIC_LABEL_NS, "test")
	require.True(t, ok)
	assert.Equal(t, 2.0, completed.Value)
	assertLabel(t, completed, commons.METRIC_LABEL_CHECKPOINT_STATUS, "saving")
	_, ok = findStat(metrics, "client_connections")
	assert.False(t, ok, "a namespace state other than none stops the rest of the refresh")

	metrics = refreshCheckpoint(t, "is_parked=false:park_ms=0")
	_, ok = findStat(metrics, "checkpoint_files_completed")
	assert.False(t, ok, "an all-flash idle body has no per-namespace records")
	_, ok = findStat(metrics, "client_connections")
	assert.True(t, ok)

	metrics = refreshCheckpoint(t, "is_parked=true:park_ms=12")
	_, ok = findStat(metrics, "checkpoint_park_ms")
	assert.False(t, ok, "the node-level is_parked line is a signal, not a metric")
	_, ok = findStat(metrics, "client_connections")
	assert.False(t, ok, "is_parked=true stops the rest of the refresh")
}

// TestNodeStatsProcessorCheckpointStatusNodeRecord covers 8.2.2, where the node-level fields come as
// their own node: record and the namespace records no longer carry is_parked or park_ms.
func TestNodeStatsProcessorCheckpointStatusNodeRecord(t *testing.T) {
	idle := "node:is_parked=false:park_ms=0;test:state=none:files_completed=0:files_total=0;" +
		"bar:state=none:files_completed=0:files_total=0"
	metrics := refreshCheckpoint(t, idle)

	_, ok := findStat(metrics, "client_connections")
	assert.True(t, ok, "an idle node record must not be read as checkpoint-shutdown")
	_, ok = findStatWithLabel(metrics, "checkpoint_is_parked", commons.METRIC_LABEL_NS, "node")
	assert.False(t, ok, "the node record is a signal, not a namespace")
	completed, ok := findStatWithLabel(metrics, "checkpoint_files_completed", commons.METRIC_LABEL_NS, "bar")
	require.True(t, ok)
	assertLabel(t, completed, commons.METRIC_LABEL_CHECKPOINT_STATUS, "none")

	metrics = refreshCheckpoint(t, "node:is_parked=true:park_ms=12;test:state=none:files_completed=0:files_total=0")
	_, ok = findStat(metrics, "client_connections")
	assert.False(t, ok, "is_parked=true on the node record stops the rest of the refresh")

	metrics = refreshCheckpoint(t, "node:is_parked=false:park_ms=0;test:state=saving:files_completed=2:files_total=4")
	_, ok = findStat(metrics, "client_connections")
	assert.False(t, ok, "a namespace state other than none still stops the rest of the refresh")

	metrics = refreshCheckpoint(t, "node:is_parked=false:park_ms=0;node:state=none:files_completed=3:files_total=4")
	completed, ok = findStatWithLabel(metrics, "checkpoint_files_completed", commons.METRIC_LABEL_NS, "node")
	require.True(t, ok, "a namespace named node still reports, since it has a state")
	assert.Equal(t, 3.0, completed.Value)
	_, ok = findStat(metrics, "client_connections")
	assert.True(t, ok)

	metrics = refreshCheckpoint(t, "node;test:state=none:files_completed=0:files_total=0")
	_, ok = findStat(metrics, "client_connections")
	assert.True(t, ok, "a record without stats is skipped")
}

func refreshCheckpoint(t *testing.T, body string) []AerospikeStat {
	t.Helper()

	state := useProcessorConfig(t)
	processor := NewNodeStatsProcessor(state)
	metrics, err := processor.Refresh(nil, map[string]string{
		CMD_INFOKEY_CHECKPOINT_STATUS: body,
		KEY_SERVICE_STATISTICS:        "client_connections=3",
	})
	require.NoError(t, err)
	return metrics
}
