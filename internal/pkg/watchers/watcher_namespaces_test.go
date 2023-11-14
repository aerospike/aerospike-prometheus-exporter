package watchers

import (
	"fmt"
	"testing"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/data"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/unittests"
	"github.com/stretchr/testify/assert"
)

func TestPassOneKeys(t *testing.T) {

	// Check passoneKeys
	nsWatcher := &NamespaceWatcher{}
	nsPassOneKeys := nsWatcher.PassOneKeys()

	udp := &unittests.UnittestDataProvider{}
	ndv := udp.GetUnittestValidator("namespace")
	passOneOutputs := ndv.GetPassOneKeys(*udp)

	// fmt.Println("TestPassOneKeys: ", passOneOutputs)
	var expectedOutputs []string
	expectedOutputs = append(expectedOutputs, passOneOutputs["namespaces"])

	assert.Equal(t, nsPassOneKeys, expectedOutputs)

}

func TestPassTwoKeys(t *testing.T) {

	// initialize config and gauge-lists
	config.InitConfig(unittests.GetConfigfileLocation(unittests.DEFAULT_CONFIG_FILE))

	// rawMetrics := getRawMetrics()
	nsWatcher := new(NamespaceWatcher)

	// simulate, as if we are sending requestInfo to AS and get the namespaces, these are coming from mock-data-generator
	passOneKeys := nsWatcher.PassOneKeys()
	passOneOutput, _ := data.GetProvider().RequestInfo(passOneKeys)
	fmt.Println("TestPassTwoKeys: passOneOutput: ", passOneOutput)
	passTwokeyOutputs := nsWatcher.PassTwoKeys(passOneOutput)

	// expectedOutputs := []string{"namespace/bar", "namespace/test"}
	udp := &unittests.UnittestDataProvider{}
	ndv := udp.GetUnittestValidator("namespace")
	expectedOutputs := ndv.GetPassTwoKeys(*udp)

	assert := assert.New(t)

	assert.NotEmpty(passTwokeyOutputs)
	assert.NotEmpty(expectedOutputs)

	for idx := range expectedOutputs {
		// assert each element returned by NamespaceWatcher exists in expected outputs
		// fmt.Println("expected outputs: key & value", idx, expectedOutputs[idx])
		assert.Contains(passTwokeyOutputs, expectedOutputs[idx], " value exists!")
	}
}

func TestNamespaceRefreshDefault(t *testing.T) {

	fmt.Println("initializing config ... TestNamespaceRefreshDefault")
	// Initialize and validate config

	// initialize config and gauge-lists
	config.InitConfig(unittests.GetConfigfileLocation(unittests.DEFAULT_CONFIG_FILE))

	runTestcase(t)
}

/**
* complete logic to call watcher, generate-mock data and asset is part of this function
 */
func runTestcase(t *testing.T) {

	// initialize gauges list
	config.InitGaugeStats(unittests.GetConfigfileLocation(unittests.DEFAULT_GAUGE_LIST_FILE))

	// rawMetrics := getRawMetrics()
	nsWatcher := &NamespaceWatcher{}

	// simulate, as if we are sending requestInfo to AS and get the namespaces, these are coming from mock-data-generator
	passOneKeys := nsWatcher.PassOneKeys()
	passOneOutput, _ := data.GetProvider().RequestInfo(passOneKeys)
	passTwokeyOutputs := nsWatcher.PassTwoKeys(passOneOutput)
	// append common keys
	infoKeys := []string{commons.Infokey_ClusterName, commons.Infokey_Service, commons.Infokey_Build}
	passTwokeyOutputs = append(passTwokeyOutputs, infoKeys...)

	arrRawMetrics, err := data.GetProvider().RequestInfo(passTwokeyOutputs)
	assert.Nil(t, err, "Error while NamespaceWatcher.PassTwokeys ")
	assert.NotEmpty(t, arrRawMetrics, "Error while NamespaceWatcher.PassTwokeys, RawMetrics is EMPTY ")

	// check the output with NamespaceWatcher
	nsWatcherMetrics, err := nsWatcher.Refresh(passTwokeyOutputs, arrRawMetrics)
	assert.Nil(t, err, "Error while NamespaceWatcher.Refresh with passTwokeyOutputs ")
	assert.NotEmpty(t, nsWatcherMetrics, "Error while NamespaceWatcher.Refresh, WatcherMetrics is EMPTY ")

	// check the WatcherMetrics if all stats & configs coming with required labels
	// fmt.Println(nsWatcherMetrics)
	for k := range nsWatcherMetrics {
		fmt.Println(nsWatcherMetrics[k])
	}

	// check for defined pattern, namespace metrics
	// context, name, labels: cluster, service, namespace,
}
