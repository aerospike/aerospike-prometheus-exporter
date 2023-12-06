package statprocessors

import (
	"fmt"
	"testing"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/dataprovider"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/statprocessors"
	"github.com/stretchr/testify/assert"
)

func Test_Sets_PassOneKeys(t *testing.T) {

	fmt.Println("initializing config ... Test_Sets_PassOneKeys")

	// Check passoneKeys
	setsWatcher := &statprocessors.SetsStatsProcessor{}
	nwPassOneKeys := setsWatcher.PassOneKeys()

	udh := &UnittestDataHandler{}
	ndv := udh.GetUnittestValidator("sets")
	passOneOutputs := ndv.GetPassOneKeys()

	assert.Nil(t, nwPassOneKeys, passOneOutputs)

}

func Test_Sets_PassTwoKeys(t *testing.T) {

	fmt.Println("initializing config ... Test_Sets_PassTwoKeys")

	// initialize config and gauge-lists
	commons.InitConfigurations(commons.GetWatchersConfigFile(commons.TESTS_DEFAULT_CONFIG_FILE))

	// Check passoneKeys
	setsWatcher := &statprocessors.SetsStatsProcessor{}
	nwPassOneKeys := setsWatcher.PassOneKeys()
	passOneOutput, _ := dataprovider.GetProvider().RequestInfo(nwPassOneKeys)
	fmt.Println("Test_Sets_PassTwoKeys: passOneOutput: ", passOneOutput)
	passTwoOutputs := setsWatcher.PassTwoKeys(passOneOutput)

	udh := &UnittestDataHandler{}
	ndv := udh.GetUnittestValidator("sets")
	expectedPassTwoOutputs := ndv.GetPassTwoKeys()

	assert.NotEmpty(t, passTwoOutputs)
	assert.NotEmpty(t, expectedPassTwoOutputs)

	for idx := range expectedPassTwoOutputs {
		// assert each element returned by NamespaceWatcher exists in expected outputs
		assert.Contains(t, passTwoOutputs, expectedPassTwoOutputs[idx], " value exists!")
	}

}

func Test_Sets_RefreshDefault(t *testing.T) {

	fmt.Println("initializing config ... Test_Sets_RefreshDefault")

	// initialize config and gauge-lists
	commons.InitConfigurations(commons.GetWatchersConfigFile(commons.TESTS_DEFAULT_CONFIG_FILE))

	sets_runTestcase(t)
}

/**
* complete logic to call watcher, generate-mock data and asset is part of this function
 */
func sets_runTestcase(t *testing.T) {

	// Check passoneKeys
	setsWatcher := &statprocessors.SetsStatsProcessor{}
	nwPassOneKeys := setsWatcher.PassOneKeys()
	passOneOutput, _ := dataprovider.GetProvider().RequestInfo(nwPassOneKeys)
	fmt.Println("TestPassTwoKeys: passOneOutput: ", passOneOutput)
	passTwoOutputs := setsWatcher.PassTwoKeys(passOneOutput)

	// append common keys
	infoKeys := []string{statprocessors.Infokey_ClusterName, statprocessors.Infokey_Service, statprocessors.Infokey_Build}
	passTwoOutputs = append(passTwoOutputs, infoKeys...)

	arrRawMetrics, err := dataprovider.GetProvider().RequestInfo(passTwoOutputs)
	assert.Nil(t, err, "Error while setsWatcher.PassTwokeys ")
	assert.NotEmpty(t, arrRawMetrics, "Error while setsWatcher.PassTwokeys, RawMetrics is EMPTY ")

	// check the output with setsWatcher
	setsMetrics, err := setsWatcher.Refresh(passTwoOutputs, arrRawMetrics)
	assert.Nil(t, err, "Error while setsWatcher.Refresh with passTwoOutputs ")
	assert.NotEmpty(t, setsMetrics, "Error while setsWatcher.Refresh, setsWatcher is EMPTY ")

	udh := &UnittestDataHandler{}
	ndv := udh.GetUnittestValidator("sets")
	expected_results := ndv.GetMetricLabelsWithValues()

	for k := range setsMetrics {
		// convert / serialize to string which can be compared to stored expected mock result
		str_metric := fmt.Sprintf("%#v", setsMetrics[k])
		_, exists := expected_results[str_metric]
		assert.True(t, exists, "Failed, did not find expected result: "+str_metric)
	}

}
