package config

import (
	"fmt"
	"os"
	"testing"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"
	"github.com/stretchr/testify/assert"
)

const (
	GAUGES_NAMESPACES_COUNT = 152
	GAUGES_NODE_STATS_COUNT = 103
	GAUGES_SETS_COUNT       = 9
	GAUGES_SINDEX_COUNT     = 13
	GAUGES_XDR_COUNT        = 10
)

var TESTS_DEFAULT_GAUGE_LIST_FILE = "configs/gauge_stats_list.toml"

func initConfigsAndGauges() {
	// Initialize and validate Gauge config
	l_cwd, _ := os.Getwd()
	config.InitGaugeStats(l_cwd + "/../../../../" + TESTS_DEFAULT_GAUGE_LIST_FILE)

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
	assert.Equal(t, gaugeList.NodeStats["cluster_clock_skew_stop_writes_sec"], true)

	assert.Equal(t, gaugeList.SindexStats["entries_per_rec"], true)
	assert.Equal(t, gaugeList.XdrStats["recoveries_pending"], true)
	assert.Equal(t, gaugeList.SetsStats["truncate_lut"], true)

	// 8.1.3 namespace gauges
	assert.True(t, gaugeList.NamespaceStats["index_shmem_alloc_bytes"])
	assert.True(t, gaugeList.NamespaceStats["index_shmem_tail_bytes"])
	assert.True(t, gaugeList.NamespaceStats["repl_wire_compression_delta_hit_pct"])
	assert.True(t, gaugeList.NamespaceStats["set_index_alloc_bytes"])
	assert.True(t, gaugeList.NamespaceStats["sindex_shmem_alloc_bytes"])
	assert.True(t, gaugeList.NamespaceStats["sindex_shmem_tail_bytes"])

	// 8.1.3 node gauges
	assert.True(t, gaugeList.NodeStats["checkpoint_status"])
	assert.True(t, gaugeList.NodeStats["process_rss_bytes"])
	assert.True(t, gaugeList.NodeStats["wire_comp_apply_patch_cpu_pct"])
	assert.True(t, gaugeList.NodeStats["wire_comp_compress_cpu_pct"])
	assert.True(t, gaugeList.NodeStats["wire_comp_cpu_pct"])
	assert.True(t, gaugeList.NodeStats["wire_comp_decompress_cpu_pct"])
	assert.True(t, gaugeList.NodeStats["wire_comp_make_patch_cpu_pct"])

	// 8.1.3 smd-info gauges
	assert.True(t, gaugeList.NodeStats["smd_compression_hit_pct"])
	assert.True(t, gaugeList.NodeStats["smd_initial_sync_done"])
	assert.True(t, gaugeList.NodeStats["smd_mixed_cluster"])
	assert.True(t, gaugeList.NodeStats["smd_n_events"])
	assert.True(t, gaugeList.NodeStats["smd_n_nodes"])
	assert.True(t, gaugeList.NodeStats["smd_n_pending_sets"])
	assert.True(t, gaugeList.NodeStats["smd_evict_settled"])
	assert.True(t, gaugeList.NodeStats["smd_masking_settled"])
	assert.True(t, gaugeList.NodeStats["smd_roster_settled"])
	assert.True(t, gaugeList.NodeStats["smd_security_settled"])
	assert.True(t, gaugeList.NodeStats["smd_sindex_settled"])
	assert.True(t, gaugeList.NodeStats["smd_truncate_settled"])
	assert.True(t, gaugeList.NodeStats["smd_UDF_settled"])
	assert.True(t, gaugeList.NodeStats["smd_XDR_settled"])

}

func TestNoGaugeExists(t *testing.T) {

	fmt.Println("initializing GaugeMetrics ... TestNoGaugeExists")

	// Initialize and validate Gauge config
	initConfigsAndGauges()
	gaugeList := config.GaugeStatHandler

	assert.Equal(t, gaugeList.NamespaceStats["non-existing-key"], false)
}
