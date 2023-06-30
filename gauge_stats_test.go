package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetGaugesNotEmpty(t *testing.T) {

	// this is to force-reload the config in the NamespaceWatcher, this is a check on this param in NamespaceWatcher implementation
	os.Setenv(TESTCASE_MODE, TESTCASE_MODE_TRUE)

	fmt.Println("initializing GaugeMetrics ... TestGetGaugesNotEmpty")

	// Initialize and validate Gauge config
	gaugeList := new(GaugeStats)

	initGaugeStats(METRICS_CONFIG_FILE, gaugeList)

	nslist := gaugeList.getGaugeStats(CTX_NAMESPACE)
	nodelist := gaugeList.getGaugeStats(CTX_NODE_STATS)
	assert.NotEmpty(t, nslist)
	assert.NotEmpty(t, nodelist)
}

func TestGetGaugesCounts(t *testing.T) {

	// this is to force-reload the config in the NamespaceWatcher, this is a check on this param in NamespaceWatcher implementation
	os.Setenv(TESTCASE_MODE, TESTCASE_MODE_TRUE)

	fmt.Println("initializing GaugeMetrics ... TestGetGaugesCounts")

	// Initialize and validate Gauge config
	gaugeList := new(GaugeStats)

	initGaugeStats(METRICS_CONFIG_FILE, gaugeList)

	glist := gaugeList.getGaugeStats(CTX_NAMESPACE)
	assert.Equal(t, len(glist), 96)
	//TODO Write checks on Sets, Xdr, Sindedx, Nodestats

	glist = gaugeList.getGaugeStats(CTX_NODE_STATS)
	assert.Equal(t, len(glist), 74)

	glist = gaugeList.getGaugeStats(CTX_SETS)
	assert.Equal(t, len(glist), 7)

	glist = gaugeList.getGaugeStats(CTX_SINDEX)
	assert.Equal(t, len(glist), 13)

	glist = gaugeList.getGaugeStats(CTX_XDR)
	assert.Equal(t, len(glist), 10)

}

func TestIsAGaugeTrue(t *testing.T) {

	// this is to force-reload the config in the NamespaceWatcher, this is a check on this param in NamespaceWatcher implementation
	os.Setenv(TESTCASE_MODE, TESTCASE_MODE_TRUE)

	// Initialize and validate Gauge config
	gaugeList := new(GaugeStats)

	initGaugeStats(METRICS_CONFIG_FILE, gaugeList)

	exists := false

	exists = gaugeList.isGauge(CTX_NAMESPACE, "cache_read_pct")
	assert.Equal(t, exists, true)

	exists = gaugeList.isGauge(CTX_NODE_STATS, "cluster_clock_skew_stop_writes_sec")
	assert.Equal(t, exists, true)

	exists = gaugeList.isGauge(CTX_SINDEX, "entries_per_rec")
	assert.Equal(t, exists, true)

	exists = gaugeList.isGauge(CTX_XDR, "recoveries_pending")
	assert.Equal(t, exists, true)

	exists = gaugeList.isGauge(CTX_SETS, "truncate_lut")
	assert.Equal(t, exists, true)

}

func TestNoGaugeExists(t *testing.T) {

	// this is to force-reload the config in the NamespaceWatcher, this is a check on this param in NamespaceWatcher implementation
	os.Setenv(TESTCASE_MODE, TESTCASE_MODE_TRUE)

	fmt.Println("initializing GaugeMetrics ... TestNoGaugeExists")

	// Initialize and validate Gauge config
	gaugeList := new(GaugeStats)

	initGaugeStats(METRICS_CONFIG_FILE, gaugeList)

	exists := gaugeList.isGauge(CTX_NAMESPACE, "non-existing-key")
	assert.Equal(t, exists, false)
}
