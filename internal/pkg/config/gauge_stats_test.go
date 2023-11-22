package config

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func initConfigs() {
	// Initialize and validate Gauge config
	l_cwd, _ := os.Getwd()
	InitGaugeStats(l_cwd + "/../../../configs/gauge_stats_list.toml")

}

func TestGetGaugesNotEmpty(t *testing.T) {
	fmt.Println("initializing GaugeMetrics ... TestGetGaugesNotEmpty")

	// Initialize configs and gauges
	initConfigs()
	gaugeList := GaugeStatHandler

	nslist := gaugeList.NamespaceStats
	nodelist := gaugeList.NodeStats
	assert.NotEmpty(t, nslist)
	assert.NotEmpty(t, nodelist)
}

func TestGetGaugesCounts(t *testing.T) {
	fmt.Println("initializing GaugeMetrics ... TestGetGaugesCounts")

	// Initialize and validate Gauge config
	initConfigs()
	gaugeList := GaugeStatHandler

	glist := gaugeList.NamespaceStats
	assert.Equal(t, len(glist), 99)

	glist = gaugeList.NodeStats
	assert.Equal(t, len(glist), 69)

	glist = gaugeList.SetsStats
	assert.Equal(t, len(glist), 9)

	glist = gaugeList.SindexStats
	assert.Equal(t, len(glist), 13)

	glist = gaugeList.XdrStats
	assert.Equal(t, len(glist), 10)

}

func TestIsAGaugeTrue(t *testing.T) {
	fmt.Println("initializing GaugeMetrics ... TestIsAGaugeTrue")

	// Initialize and validate Gauge config
	initConfigs()
	gaugeList := GaugeStatHandler

	assert.Equal(t, gaugeList.NamespaceStats["cache_read_pct"], true)

	// exists = gaugeList.isGauge(CTX_NODE_STATS, "cluster_clock_skew_stop_writes_sec")
	// assert.Equal(t, exists, true)

	// exists = gaugeList.isGauge(CTX_SINDEX, "entries_per_rec")
	// assert.Equal(t, exists, true)

	// exists = gaugeList.isGauge(CTX_XDR, "recoveries_pending")
	// assert.Equal(t, exists, true)

	// exists = gaugeList.isGauge(CTX_SETS, "truncate_lut")
	// assert.Equal(t, exists, true)

}

// func TestNoGaugeExists(t *testing.T) {

// 	fmt.Println("initializing GaugeMetrics ... TestNoGaugeExists")

// 	// Initialize and validate Gauge config
// 	gaugeList := new(GaugeStats)

// 	initGaugeStats(METRICS_CONFIG_FILE, gaugeList)

// 	exists := gaugeList.isGauge(CTX_NAMESPACE, "non-existing-key")
// 	assert.Equal(t, exists, false)
// }
