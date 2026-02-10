package statprocessors

import (
	"fmt"
	"testing"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/dataprovider"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/statprocessors"
	"github.com/stretchr/testify/assert"
)

func Test_Xdr_PassOneKeys(t *testing.T) {

	fmt.Println("initializing config ... Test_Xdr_PassOneKeys")

	// Check passoneKeys
	sharedState := statprocessors.NewStatProcessorSharedState()
	xdrWatcher := statprocessors.NewXdrStatsProcessor(sharedState)
	xdrPassOneKeys := xdrWatcher.PassOneKeys()

	udh := &UnittestDataHandler{}
	ndv := udh.GetUnittestValidator("xdr")
	passOneOutputs := ndv.GetPassOneKeys()

	var expectedOutputs []string
	expectedOutputs = append(expectedOutputs, passOneOutputs["xdr"])

	assert.Equal(t, xdrPassOneKeys, expectedOutputs)

}

func Test_Xdr_PassTwoKeys(t *testing.T) {

	fmt.Println("initializing config ... Test_Xdr_PassTwoKeys")

	// initialize config and gauge-lists
	commons.InitConfigurations(commons.GetWatchersConfigFile(commons.TESTS_DEFAULT_CONFIG_FILE))

	// Check passoneKeys
	sharedState := statprocessors.NewStatProcessorSharedState()
	xdrWatcher := statprocessors.NewXdrStatsProcessor(sharedState)
	xdrPassOneKeys := xdrWatcher.PassOneKeys()
	// append common keys
	infoKeys := []string{sharedState.Infokey_ClusterName, sharedState.Infokey_Service, sharedState.Infokey_Build, "namespaces"}
	xdrPassOneKeys = append(xdrPassOneKeys, infoKeys...)

	passOneOutput, _ := dataprovider.GetProvider("mock").RequestInfo(xdrPassOneKeys)

	passTwoOutputs := xdrWatcher.PassTwoKeys(passOneOutput)

	udh := &UnittestDataHandler{}
	ndv := udh.GetUnittestValidator("xdr")
	expectedPassTwoOutputs := ndv.GetPassTwoKeys()

	assert.NotEmpty(t, passTwoOutputs)
	assert.NotEmpty(t, expectedPassTwoOutputs)

	for idx := range expectedPassTwoOutputs {
		// assert each element returned by NamespaceWatcher exists in expected outputs
		assert.Contains(t, passTwoOutputs, expectedPassTwoOutputs[idx], " value exists!")
	}

}

func Test_Xdr_RefreshDefault(t *testing.T) {

	fmt.Println("initializing config ... Test_Xdr_RefreshDefault")

	// initialize config and gauge-lists
	commons.InitConfigurations(commons.GetWatchersConfigFile(commons.TESTS_DEFAULT_CONFIG_FILE))

	xdr_runTestcase(t)
}

/**
* complete logic to call watcher, generate-mock data and asset is part of this function
 */
func xdr_runTestcase(t *testing.T) {

	// Check passoneKeys
	sharedState := statprocessors.NewStatProcessorSharedState()
	xdrWatcher := statprocessors.NewXdrStatsProcessor(sharedState)
	xdrPassOneKeys := xdrWatcher.PassOneKeys()
	// append common keys
	infoKeys := []string{sharedState.Infokey_ClusterName, sharedState.Infokey_Service, sharedState.Infokey_Build, "namespaces"}
	xdrPassOneKeys = append(xdrPassOneKeys, infoKeys...)

	passOneOutput, _ := dataprovider.GetProvider("mock").RequestInfo(xdrPassOneKeys)
	passTwoOutputs := xdrWatcher.PassTwoKeys(passOneOutput)

	passTwoOutputs = append(passTwoOutputs, infoKeys...)
	arrRawMetrics, err := dataprovider.GetProvider("mock").RequestInfo(passTwoOutputs)

	sharedState.ClusterName = passOneOutput[sharedState.Infokey_ClusterName]
	sharedState.Build = passOneOutput[sharedState.Infokey_Build]
	sharedState.Service = passOneOutput[sharedState.Infokey_Service]

	assert.Nil(t, err, "Error while XdrWatcher.PassTwokeys ")
	assert.NotEmpty(t, arrRawMetrics, "Error while XdrWatcher.PassTwokeys, RawMetrics is EMPTY ")

	// check the output with NodeStatsWatcher
	xdrMetrics, err := xdrWatcher.Refresh(passTwoOutputs, arrRawMetrics)
	assert.Nil(t, err, "Error while XdrWatcher.Refresh with passTwoOutputs ")
	assert.NotEmpty(t, xdrMetrics, "Error while XdrWatcher.Refresh, XdrWatcher is EMPTY ")

	udh := &UnittestDataHandler{}
	ndv := udh.GetUnittestValidator("xdr")
	expected_results := ndv.GetMetricLabelsWithValues()

	for k := range xdrMetrics {
		// convert / serialize to string which can be compared to stored expected mock result
		str_metric := fmt.Sprintf("%#v", xdrMetrics[k])
		_, exists := expected_results[str_metric]
		assert.True(t, exists, "Failed, did not find expected result: "+str_metric)
	}
}
