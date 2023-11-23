package statprocessors

import (
	"fmt"
	"testing"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/dataprovider"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/statprocessors"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/testutils"
	"github.com/stretchr/testify/assert"
)

func Test_Latency_PassOneKeys(t *testing.T) {

	fmt.Println("initializing config ... Test_Latency_PassOneKeys")

	// Check passoneKeys
	latencyStatsProcessor := &statprocessors.LatencyStatsProcessor{}
	nwPassOneKeys := latencyStatsProcessor.PassOneKeys()

	udh := &testutils.UnittestDataHandler{}
	ndv := udh.GetUnittestValidator("latency")
	passOneOutputs := ndv.GetPassOneKeys()

	assert.Nil(t, nwPassOneKeys, passOneOutputs)

}

func Test_Latency_PassTwoKeys(t *testing.T) {

	fmt.Println("initializing config ... Test_Latency_PassTwoKeys")

	// initialize config and gauge-lists
	testutils.InitConfigurations(testutils.TESTS_DEFAULT_CONFIG_FILE)

	// Check passoneKeys
	latencyStatsProcessor := &statprocessors.LatencyStatsProcessor{}
	nwPassOneKeys := latencyStatsProcessor.PassOneKeys()
	passOneOutput, _ := dataprovider.GetProvider().RequestInfo(nwPassOneKeys)
	fmt.Println("Test_Latency_PassTwoKeys: passOneOutput: ", passOneOutput)
	passTwoOutputs := latencyStatsProcessor.PassTwoKeys(passOneOutput)

	udh := &testutils.UnittestDataHandler{}
	ndv := udh.GetUnittestValidator("latency")
	expectedPassTwoOutputs := ndv.GetPassTwoKeys()

	assert.NotEmpty(t, passTwoOutputs)
	assert.NotEmpty(t, expectedPassTwoOutputs)

	for idx := range expectedPassTwoOutputs {
		// assert each element returned by NamespaceWStatsProcessor exists in expected outputs
		assert.Contains(t, passTwoOutputs, expectedPassTwoOutputs[idx], " value exists!")
	}

}

func Test_Latency_RefreshDefault(t *testing.T) {

	fmt.Println("initializing config ... Test_Latency_RefreshDefault")

	// initialize config and gauge-lists
	testutils.InitConfigurations(testutils.TESTS_DEFAULT_CONFIG_FILE)

	latency_runTestcase(t)
}

/**
* complete logic to call watcher, generate-mock data and asset is part of this function
 */
func latency_runTestcase(t *testing.T) {

	// Check passoneKeys
	latencyWatcher := &statprocessors.LatencyStatsProcessor{}
	nwPassOneKeys := latencyWatcher.PassOneKeys()
	passOneOutput, _ := dataprovider.GetProvider().RequestInfo(nwPassOneKeys)
	fmt.Println("TestPassTwoKeys: passOneOutput: ", passOneOutput)
	passTwoOutputs := latencyWatcher.PassTwoKeys(passOneOutput)

	// append common keys
	infoKeys := []string{statprocessors.Infokey_ClusterName, statprocessors.Infokey_Service, statprocessors.Infokey_Build}
	passTwoOutputs = append(passTwoOutputs, infoKeys...)

	arrRawMetrics, err := dataprovider.GetProvider().RequestInfo(passTwoOutputs)
	assert.Nil(t, err, "Error while latencyWatcher.PassTwokeys ")
	assert.NotEmpty(t, arrRawMetrics, "Error while latencyWatcher.PassTwokeys, RawMetrics is EMPTY ")

	// check the output with setsWatcher
	latencyMetrics, err := latencyWatcher.Refresh(passTwoOutputs, arrRawMetrics)
	assert.Nil(t, err, "Error while latencyWatcher.Refresh with passTwoOutputs ")
	assert.NotEmpty(t, latencyMetrics, "Error while latencyWatcher.Refresh, latencyWatcher is EMPTY ")

	// // check the WatcherMetrics if all stats & configs coming with required labels
	// // below block of code is used when we create the baseline mock data, which is stored in exporter_mock_results.txt for test verification/assertion
	// // do-not-remove below code, use when to dump the output
	// for k := range latencyMetrics {
	// 	str := fmt.Sprintf("%#v", latencyMetrics[k])
	// 	fmt.Println(str)
	// }

	udh := &testutils.UnittestDataHandler{}
	ndv := udh.GetUnittestValidator("latency")
	expected_results := ndv.GetMetricLabelsWithValues()

	for k := range latencyMetrics {
		// convert / serialize to string which can be compared to stored expected mock result
		str_metric := fmt.Sprintf("%#v", latencyMetrics[k])
		_, exists := expected_results[str_metric]
		assert.True(t, exists, "Failed, did not find expected result: "+str_metric)
	}

}
