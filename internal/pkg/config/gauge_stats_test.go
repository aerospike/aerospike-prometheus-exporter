package config_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	GAUGES_NAMESPACES_COUNT = 155
	GAUGES_NODE_STATS_COUNT = 110
	GAUGES_SETS_COUNT       = 9
	GAUGES_SINDEX_COUNT     = 16
	GAUGES_XDR_COUNT        = 10
)

var TESTS_DEFAULT_GAUGE_LIST_FILE = "configs/gauge_stats_list.toml"

func initConfigsAndGauges() {
	// Initialize and validate Gauge config
	l_cwd, _ := os.Getwd()
	config.InitGaugeStats(l_cwd + "/../../../" + TESTS_DEFAULT_GAUGE_LIST_FILE)

}

func TestGetGaugesNotEmpty(t *testing.T) {
	fmt.Println("initializing GaugeMetrics ... TestGetGaugesNotEmpty")

	// Initialize configs and gauges
	initConfigsAndGauges()
	gaugeList := config.GaugeStatHandler

	nslist := gaugeList.NamespaceStats
	nodelist := gaugeList.NodeStats
	assert.NotEmpty(t, nslist)
	assert.NotEmpty(t, nodelist)
}

func TestGetGaugesCounts(t *testing.T) {
	fmt.Println("initializing GaugeMetrics ... TestGetGaugesCounts")

	// Initialize and validate Gauge config
	initConfigsAndGauges()
	gaugeList := config.GaugeStatHandler

	glist := gaugeList.NamespaceStats
	assert.Equal(t, len(glist), GAUGES_NAMESPACES_COUNT)

	glist = gaugeList.NodeStats
	assert.Equal(t, len(glist), GAUGES_NODE_STATS_COUNT)

	glist = gaugeList.SetsStats
	assert.Equal(t, len(glist), GAUGES_SETS_COUNT)

	glist = gaugeList.SindexStats
	assert.Equal(t, len(glist), GAUGES_SINDEX_COUNT)

	glist = gaugeList.XdrStats
	assert.Equal(t, len(glist), GAUGES_XDR_COUNT)

}

func TestIsAGaugeTrue(t *testing.T) {
	fmt.Println("initializing GaugeMetrics ... TestIsAGaugeTrue")

	// Initialize and validate Gauge config
	initConfigsAndGauges()
	gaugeList := config.GaugeStatHandler

	assert.Equal(t, gaugeList.NamespaceStats["cache_read_pct"], true)
	// no hyphen in the name, so it is a gauge only if the toml key matches the server stat exactly
	assert.True(t, gaugeList.NamespaceStats["sindex_equality_filter_bytes"])
	assert.Equal(t, gaugeList.NodeStats["cluster_clock_skew_stop_writes_sec"], true)

	assert.Equal(t, gaugeList.SindexStats["entries_per_rec"], true)
	assert.Equal(t, gaugeList.XdrStats["recoveries_pending"], true)
	assert.Equal(t, gaugeList.SetsStats["truncate_lut"], true)

	// smd-info group keys are lowercased by the node processor (UDF/XDR from the server)
	assert.True(t, gaugeList.NodeStats["smd_udf_settled"])
	assert.True(t, gaugeList.NodeStats["smd_xdr_settled"])
	assert.False(t, gaugeList.NodeStats["smd_UDF_settled"])
	assert.False(t, gaugeList.NodeStats["smd_XDR_settled"])

	// checkpoint-status exposes cp_status as a label and these values as metrics.
	assert.True(t, gaugeList.NodeStats["checkpoint_files_completed"])
	assert.True(t, gaugeList.NodeStats["checkpoint_files_total"])
	assert.True(t, gaugeList.NodeStats["checkpoint_is_parked"])
	assert.True(t, gaugeList.NodeStats["checkpoint_park_ms"])
	assert.False(t, gaugeList.NodeStats["checkpoint_status"])

}

func TestNoGaugeExists(t *testing.T) {

	fmt.Println("initializing GaugeMetrics ... TestNoGaugeExists")

	// Initialize and validate Gauge config
	initConfigsAndGauges()
	gaugeList := config.GaugeStatHandler

	assert.Equal(t, gaugeList.NamespaceStats["non-existing-key"], false)
}

// TestGaugeStatsBackupRestoreKeysAreNormalised checks that backup_gauge_stats and restore_gauge_stats are read,
// and that an entry written with underscores matches the hyphen spelling the processor looks up.
func TestGaugeStatsBackupRestoreKeysAreNormalised(t *testing.T) {
	original := config.GaugeStatHandler
	t.Cleanup(func() { config.GaugeStatHandler = original })

	path := filepath.Join(t.TempDir(), "gauges.toml")
	require.NoError(t, os.WriteFile(path, []byte(`
backup_gauge_stats = ["progress_pct", "backup-rps", "start_time"]
restore_gauge_stats = ["progress-pct"]
`), 0o600))

	config.InitGaugeStats(path)

	assert.True(t, config.GaugeStatHandler.BackupStats["progress-pct"])
	assert.True(t, config.GaugeStatHandler.BackupStats["backup-rps"])
	assert.True(t, config.GaugeStatHandler.BackupStats["start-time"])
	assert.False(t, config.GaugeStatHandler.BackupStats["progress_pct"], "entries are stored with hyphens")
	assert.True(t, config.GaugeStatHandler.RestoreStats["progress-pct"])
	assert.False(t, config.GaugeStatHandler.RestoreStats["backup-rps"], "a restore entry does not leak into backup")
	assert.Nil(t, config.GaugeStatHandler.Backup, "the read array is emptied once mapped")
}
