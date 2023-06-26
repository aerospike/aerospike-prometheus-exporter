package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetGuagesNotEmpty(t *testing.T) {

	// this is to force-reload the config in the NamespaceWatcher, this is a check on this param in NamespaceWatcher implementation
	os.Setenv(TESTCASE_MODE, TESTCASE_MODE_TRUE)

	fmt.Println("initializing GuageMetrics ... TestGetGuages")

	// Initialize and validate Guage config
	guageList := new(GaugeStats)
	METRICS_CONFIG_FILE := "gauge_metrics_list.toml"

	initGaugeStats(METRICS_CONFIG_FILE, guageList)

	nslist := guageList.getGaugeStats(CTX_NAMESPACE)
	nodelist := guageList.getGaugeStats(CTX_NODE_STATS)
	// fmt.Println(glist)
	assert.NotEmpty(t, nslist)
	assert.NotEmpty(t, nodelist)
}

func TestGetGuagesCounts(t *testing.T) {

	// this is to force-reload the config in the NamespaceWatcher, this is a check on this param in NamespaceWatcher implementation
	os.Setenv(TESTCASE_MODE, TESTCASE_MODE_TRUE)

	fmt.Println("initializing GuageMetrics ... TestGetGuagesCounts")

	// Initialize and validate Guage config
	guageList := new(GaugeStats)
	METRICS_CONFIG_FILE := "gauge_metrics_list.toml"

	initGaugeStats(METRICS_CONFIG_FILE, guageList)

	glist := guageList.getGaugeStats(CTX_NAMESPACE)
	// fmt.Println(glist)
	assert.Equal(t, len(glist), 193)
}

func TestIsAGuageTrue(t *testing.T) {

	// this is to force-reload the config in the NamespaceWatcher, this is a check on this param in NamespaceWatcher implementation
	os.Setenv(TESTCASE_MODE, TESTCASE_MODE_TRUE)

	fmt.Println("initializing GuageMetrics ... TestIsAGuageTrue")

	// Initialize and validate Guage config
	guageList := new(GaugeStats)
	METRICS_CONFIG_FILE := "gauge_metrics_list.toml"

	initGaugeStats(METRICS_CONFIG_FILE, guageList)

	exists := guageList.isGauge(CTX_NAMESPACE, "cache_read_pct")
	assert.Equal(t, exists, true)
}

func TestNoGuageExists(t *testing.T) {

	// this is to force-reload the config in the NamespaceWatcher, this is a check on this param in NamespaceWatcher implementation
	os.Setenv(TESTCASE_MODE, TESTCASE_MODE_TRUE)

	fmt.Println("initializing GuageMetrics ... TestNoGuageExists")

	// Initialize and validate Guage config
	guageList := new(GaugeStats)
	METRICS_CONFIG_FILE := "gauge_metrics_list.toml"

	initGaugeStats(METRICS_CONFIG_FILE, guageList)

	exists := guageList.isGauge(CTX_NAMESPACE, "non-existing-key")
	assert.Equal(t, exists, false)
}
