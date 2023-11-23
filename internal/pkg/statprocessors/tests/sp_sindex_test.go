package statprocessors

import (
	"fmt"
	"testing"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/dataprovider"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/statprocessors"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/testutils"
	"github.com/stretchr/testify/assert"
)

func Test_Sindex_PassOneKeys(t *testing.T) {

	fmt.Println("initializing config ... Test_Sindex_PassOneKeys")

	// Check passoneKeys
	sindexWatcher := &statprocessors.SindexStatsProcessor{}
	nwPassOneKeys := sindexWatcher.PassOneKeys()

	udh := &testutils.UnittestDataHandler{}
	ndv := udh.GetUnittestValidator("sindex")
	passOneOutputs := ndv.GetPassOneKeys()

	var expectedOutputs []string
	expectedOutputs = append(expectedOutputs, passOneOutputs["sindex"])

	assert.Equal(t, nwPassOneKeys, expectedOutputs)

}

func Test_Sindex_PassTwoKeys(t *testing.T) {

	fmt.Println("initializing config ... Test_Sindex_PassTwoKeys")

	// initialize config and gauge-lists
	testutils.InitConfigurations(testutils.GetWatchersConfigFile(testutils.TESTS_DEFAULT_CONFIG_FILE))

	// Check passoneKeys
	sindexWatcher := &statprocessors.SindexStatsProcessor{}
	nwPassOneKeys := sindexWatcher.PassOneKeys()
	passOneOutput, _ := dataprovider.GetProvider().RequestInfo(nwPassOneKeys)
	fmt.Println("Test_Sindex_PassTwoKeys: passOneOutput: ", passOneOutput)
	passTwoOutputs := sindexWatcher.PassTwoKeys(passOneOutput)

	udh := &testutils.UnittestDataHandler{}
	ndv := udh.GetUnittestValidator("sindex")
	expectedPassTwoOutputs := ndv.GetPassTwoKeys()

	assert.NotEmpty(t, passTwoOutputs)
	assert.NotEmpty(t, expectedPassTwoOutputs)

	for idx := range expectedPassTwoOutputs {
		// assert each element returned by NamespaceWatcher exists in expected outputs
		assert.Contains(t, passTwoOutputs, expectedPassTwoOutputs[idx], " value exists!")
	}

}

func Test_Sindex_RefreshDefault(t *testing.T) {

	fmt.Println("initializing config ... Test_Sindex_RefreshDefault")

	// initialize config and gauge-lists
	testutils.InitConfigurations(testutils.GetWatchersConfigFile(testutils.TESTS_DEFAULT_CONFIG_FILE))

	sindex_runTestcase(t)
}

/**
* complete logic to call watcher, generate-mock data and asset is part of this function
 */
func sindex_runTestcase(t *testing.T) {

	// common keys
	infoKeys := []string{statprocessors.Infokey_ClusterName, statprocessors.Infokey_Service, statprocessors.Infokey_Build}

	// Check passoneKeys
	sindexWatcher := &statprocessors.SindexStatsProcessor{}
	nwPassOneKeys := sindexWatcher.PassOneKeys()
	passOneOutput, _ := dataprovider.GetProvider().RequestInfo(nwPassOneKeys)
	fmt.Println("sindex_runTestcase: passOneOutput: ", passOneOutput)
	passTwoOutputs := sindexWatcher.PassTwoKeys(passOneOutput)

	// append common keys
	passTwoOutputs = append(passTwoOutputs, infoKeys...)

	arrRawMetrics, err := dataprovider.GetProvider().RequestInfo(passTwoOutputs)
	assert.Nil(t, err, "Error while sindexMetrics.PassTwokeys ")
	assert.NotEmpty(t, arrRawMetrics, "Error while sindexMetrics.PassTwokeys, RawMetrics is EMPTY ")

	// check the output with sindexWatcher
	sindexMetrics, err := sindexWatcher.Refresh(passTwoOutputs, arrRawMetrics)
	assert.Nil(t, err, "Error while sindexMetrics.Refresh with passTwoOutputs ")
	assert.NotEmpty(t, sindexMetrics, "Error while sindexMetrics.Refresh, sindexMetrics is EMPTY ")

	// // check the WatcherMetrics if all stats & configs coming with required labels
	// // below block of code is used when we create the baseline mock data, which is stored in exporter_mock_results.txt for test verification/assertion
	// // do-not-remove below code, use when to dump the output
	// for k := range sindexMetrics {
	// 	str := fmt.Sprintf("%#v", sindexMetrics[k])
	// 	fmt.Println(str)
	// }

	udh := &testutils.UnittestDataHandler{}
	ndv := udh.GetUnittestValidator("sindex")
	expected_results := ndv.GetMetricLabelsWithValues()

	for k := range sindexMetrics {
		// convert / serialize to string which can be compared to stored expected mock result
		str_metric := fmt.Sprintf("%#v", sindexMetrics[k])
		_, exists := expected_results[str_metric]
		assert.True(t, exists, "Failed, did not find expected result: "+str_metric)
	}

}
