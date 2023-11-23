package config

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	GAUGES_NAMESPACES_COUNT = 99
	GAUGES_NODE_STATS_COUNT = 69
	GAUGES_SETS_COUNT       = 9
	GAUGES_SINDEX_COUNT     = 13
	GAUGES_XDR_COUNT        = 10
)

func initConfigsAndGauges() {
	// Initialize and validate Gauge config
	l_cwd, _ := os.Getwd()
	InitGaugeStats(l_cwd + "/../../../configs/gauge_stats_list.toml")

}

func TestGetGaugesNotEmpty(t *testing.T) {
	fmt.Println("initializing GaugeMetrics ... TestGetGaugesNotEmpty")

	// Initialize configs and gauges
	initConfigsAndGauges()
	gaugeList := GaugeStatHandler

	nslist := gaugeList.NamespaceStats
	nodelist := gaugeList.NodeStats
	assert.NotEmpty(t, nslist)
	assert.NotEmpty(t, nodelist)
}

func TestGetGaugesCounts(t *testing.T) {
	fmt.Println("initializing GaugeMetrics ... TestGetGaugesCounts")

	// Initialize and validate Gauge config
	initConfigsAndGauges()
	gaugeList := GaugeStatHandler

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
	gaugeList := GaugeStatHandler

	assert.Equal(t, gaugeList.NamespaceStats["cache_read_pct"], true)
	assert.Equal(t, gaugeList.NodeStats["cluster_clock_skew_stop_writes_sec"], true)

	assert.Equal(t, gaugeList.SindexStats["entries_per_rec"], true)
	assert.Equal(t, gaugeList.XdrStats["recoveries_pending"], true)
	assert.Equal(t, gaugeList.SetsStats["truncate_lut"], true)

}

func TestNoGaugeExists(t *testing.T) {

	fmt.Println("initializing GaugeMetrics ... TestNoGaugeExists")

	// Initialize and validate Gauge config
	initConfigsAndGauges()
	gaugeList := GaugeStatHandler

	assert.Equal(t, gaugeList.NamespaceStats["non-existing-key"], false)
}
