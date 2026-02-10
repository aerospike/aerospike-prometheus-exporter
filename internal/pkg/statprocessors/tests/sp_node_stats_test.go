package statprocessors

import (
	"fmt"
	"testing"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/dataprovider"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/statprocessors"
	"github.com/stretchr/testify/assert"
)

func Test_Node_PassOneKeys(t *testing.T) {

	fmt.Println("initializing config ... Test_Node_PassOneKeys")

	sharedState := statprocessors.NewStatProcessorSharedState()
	// Check passoneKeys
	nodeWatcher := statprocessors.NewNodeStatsProcessor(sharedState)
	nwPassOneKeys := nodeWatcher.PassOneKeys()

	udh := &UnittestDataHandler{}
	ndv := udh.GetUnittestValidator("node")
	passOneOutputs := ndv.GetPassOneKeys()

	var expectedOutputs []string
	expectedOutputs = append(expectedOutputs, passOneOutputs["node"])

	assert.Equal(t, nwPassOneKeys, expectedOutputs)

}

func Test_Node_PassTwoKeys(t *testing.T) {

	fmt.Println("initializing config ... Test_Node_PassTwoKeys")

	// initialize config and gauge-lists
	commons.InitConfigurations(commons.GetWatchersConfigFile(commons.TESTS_DEFAULT_CONFIG_FILE))

	sharedState := statprocessors.NewStatProcessorSharedState()
	// Check passoneKeys
	nodeWatcher := statprocessors.NewNodeStatsProcessor(sharedState)
	nwPassOneKeys := nodeWatcher.PassOneKeys()
	passOneOutput, _ := dataprovider.GetProvider("mock").RequestInfo(nwPassOneKeys)
	passTwoOutputs := nodeWatcher.PassTwoKeys(passOneOutput)

	udh := &UnittestDataHandler{}
	ndv := udh.GetUnittestValidator("node")
	expectedPassTwoOutputs := ndv.GetPassTwoKeys()

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
	commons.InitConfigurations(commons.GetWatchersConfigFile(commons.TESTS_DEFAULT_CONFIG_FILE))

	node_runTestcase(t)
}

/**
* complete logic to call watcher, generate-mock data and asset is part of this function
 */
func node_runTestcase(t *testing.T) {

	// Check passoneKeys
	sharedState := statprocessors.NewStatProcessorSharedState()
	nodeWatcher := statprocessors.NewNodeStatsProcessor(sharedState)
	nwPassOneKeys := nodeWatcher.PassOneKeys()
	passOneOutput, _ := dataprovider.GetProvider("mock").RequestInfo(nwPassOneKeys)
	passTwoOutputs := nodeWatcher.PassTwoKeys(passOneOutput)

	// append common keys
	infoKeys := []string{statprocessors.Infokey_ClusterName, statprocessors.Infokey_Service, statprocessors.Infokey_Build}
	passTwoOutputs = append(passTwoOutputs, infoKeys...)

	arrRawMetrics, err := dataprovider.GetProvider("mock").RequestInfo(passTwoOutputs)
	assert.Nil(t, err, "Error while NodeStatsWatcher.PassTwokeys ")
	assert.NotEmpty(t, arrRawMetrics, "Error while NamespaceWatcher.PassTwokeys, RawMetrics is EMPTY ")

	sharedState.ClusterName = arrRawMetrics[statprocessors.Infokey_ClusterName]
	sharedState.Build = arrRawMetrics[statprocessors.Infokey_Build]
	sharedState.Service = arrRawMetrics[statprocessors.Infokey_Service]

	// check the output with NodeStatsWatcher
	nodeMetrics, err := nodeWatcher.Refresh(passTwoOutputs, arrRawMetrics)
	assert.Nil(t, err, "Error while NodeStatsWatcher.Refresh with passTwoOutputs ")
	assert.NotEmpty(t, nodeMetrics, "Error while NodeStatsWatcher.Refresh, NodeStatsWatcher is EMPTY ")

	udh := &UnittestDataHandler{}
	ndv := udh.GetUnittestValidator("node")
	expected_results := ndv.GetMetricLabelsWithValues()

	for k := range nodeMetrics {
		// convert / serialize to string which can be compared to stored expected mock result
		str_metric := fmt.Sprintf("%#v", nodeMetrics[k])
		_, exists := expected_results[str_metric]
		assert.True(t, exists, "Failed, did not find expected result: "+str_metric)
	}

}
