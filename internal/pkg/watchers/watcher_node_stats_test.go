package watchers

import (
	"fmt"
	"testing"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/data"
	tests_utils "github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/tests_utils"
	"github.com/stretchr/testify/assert"
)

func Test_Node_PassOneKeys(t *testing.T) {

	fmt.Println("initializing config ... Test_Node_PassOneKeys")

	// Check passoneKeys
	nodeWatcher := &NodeStatsWatcher{}
	nwPassOneKeys := nodeWatcher.PassOneKeys()

	udh := &tests_utils.UnittestDataHandler{}
	ndv := udh.GetUnittestValidator("node")
	passOneOutputs := ndv.GetPassOneKeys(*udh)

	assert.Nil(t, nwPassOneKeys, passOneOutputs)

}

func Test_Node_PassTwoKeys(t *testing.T) {

	fmt.Println("initializing config ... Test_Node_PassTwoKeys")

	// initialize config and gauge-lists
	config.InitConfig(tests_utils.GetConfigfileLocation(tests_utils.TESTS_DEFAULT_CONFIG_FILE))

	// Check passoneKeys
	nodeWatcher := &NodeStatsWatcher{}
	nwPassOneKeys := nodeWatcher.PassOneKeys()
	passOneOutput, _ := data.GetProvider().RequestInfo(nwPassOneKeys)
	fmt.Println("TestPassTwoKeys: passOneOutput: ", passOneOutput)
	passTwoOutputs := nodeWatcher.PassTwoKeys(passOneOutput)

	udh := &tests_utils.UnittestDataHandler{}
	ndv := udh.GetUnittestValidator("node")
	expectedPassTwoOutputs := ndv.GetPassTwoKeys(*udh)

	assert.NotEmpty(t, passTwoOutputs)
	assert.NotEmpty(t, expectedPassTwoOutputs)

	for idx := range expectedPassTwoOutputs {
		// assert each element returned by NamespaceWatcher exists in expected outputs
		assert.Contains(t, passTwoOutputs, expectedPassTwoOutputs[idx], " value exists!")
	}

}

func Test_Node_RefreshDefault(t *testing.T) {

	fmt.Println("initializing config ... Test_Node_RefreshDefault")

	// initialize config and gauge-lists
	config.InitConfig(tests_utils.GetConfigfileLocation(tests_utils.TESTS_DEFAULT_CONFIG_FILE))

	node_runTestcase(t)
}

/**
* complete logic to call watcher, generate-mock data and asset is part of this function
 */
func node_runTestcase(t *testing.T) {

	// Check passoneKeys
	nodeWatcher := &NodeStatsWatcher{}
	nwPassOneKeys := nodeWatcher.PassOneKeys()
	passOneOutput, _ := data.GetProvider().RequestInfo(nwPassOneKeys)
	fmt.Println("TestPassTwoKeys: passOneOutput: ", passOneOutput)
	passTwoOutputs := nodeWatcher.PassTwoKeys(passOneOutput)

	// append common keys
	infoKeys := []string{Infokey_ClusterName, Infokey_Service, Infokey_Build}
	passTwoOutputs = append(passTwoOutputs, infoKeys...)

	arrRawMetrics, err := data.GetProvider().RequestInfo(passTwoOutputs)
	assert.Nil(t, err, "Error while NodeStatsWatcher.PassTwokeys ")
	assert.NotEmpty(t, arrRawMetrics, "Error while NamespaceWatcher.PassTwokeys, RawMetrics is EMPTY ")

	// check the output with NodeStatsWatcher
	nodeMetrics, err := nodeWatcher.Refresh(passTwoOutputs, arrRawMetrics)
	assert.Nil(t, err, "Error while NodeStatsWatcher.Refresh with passTwoOutputs ")
	assert.NotEmpty(t, nodeMetrics, "Error while NodeStatsWatcher.Refresh, NodeStatsWatcher is EMPTY ")

	// udh := &tests_utils.UnittestDataHandler{}
	// ndv := udh.GetUnittestValidator("node")
	// expectedPassTwoOutputs := ndv.GetPassTwoKeys(*udh)

	// check the WatcherMetrics if all stats & configs coming with required labels
	// below block of code is used when we create the baseline mock data, which is stored in exporter_mock_results.txt for test verification/assertion
	// do-not-remove below code, use when to dump the output
	for k := range nodeMetrics {
		str := fmt.Sprintf("%#v", nodeMetrics[k])
		fmt.Println(str)
	}
}
