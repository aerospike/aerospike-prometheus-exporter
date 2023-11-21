package watchers

import (
	"fmt"
	"testing"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"
	tests_utils "github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/tests_utils"
	"github.com/stretchr/testify/assert"
)

func Test_Xdr_PassOneKeys(t *testing.T) {

	fmt.Println("initializing config ... Test_Xdr_PassOneKeys")

	// Check passoneKeys
	xdrWatcher := &XdrWatcher{}
	xdrPassOneKeys := xdrWatcher.PassOneKeys()

	udh := &tests_utils.UnittestDataHandler{}
	ndv := udh.GetUnittestValidator("xdr")
	passOneOutputs := ndv.GetPassOneKeys(*udh)

	// fmt.Println("TestPassOneKeys: ", passOneOutputs)
	var expectedOutputs []string
	expectedOutputs = append(expectedOutputs, passOneOutputs["xdr"])

	assert.Equal(t, xdrPassOneKeys, expectedOutputs)

}

func Test_Xdr_PassTwoKeys(t *testing.T) {

	fmt.Println("initializing config ... Test_Xdr_PassTwoKeys")

	// initialize config and gauge-lists
	config.InitConfig(tests_utils.GetConfigfileLocation(tests_utils.TESTS_DEFAULT_CONFIG_FILE))

	// // Check passoneKeys
	// nodeWatcher := &NodeStatsWatcher{}
	// nwPassOneKeys := nodeWatcher.PassOneKeys()
	// passOneOutput, _ := data.GetProvider().RequestInfo(nwPassOneKeys)
	// fmt.Println("TestPassTwoKeys: passOneOutput: ", passOneOutput)
	// passTwoOutputs := nodeWatcher.PassTwoKeys(passOneOutput)

	// udh := &tests_utils.UnittestDataHandler{}
	// ndv := udh.GetUnittestValidator("node")
	// expectedPassTwoOutputs := ndv.GetPassTwoKeys(*udh)

	// assert.NotEmpty(t, passTwoOutputs)
	// assert.NotEmpty(t, expectedPassTwoOutputs)

	// for idx := range expectedPassTwoOutputs {
	// 	// assert each element returned by NamespaceWatcher exists in expected outputs
	// 	assert.Contains(t, passTwoOutputs, expectedPassTwoOutputs[idx], " value exists!")
	// }

}

func Test_Xdr_RefreshDefault(t *testing.T) {

	fmt.Println("initializing config ... Test_Xdr_RefreshDefault")

	// initialize config and gauge-lists
	config.InitConfig(tests_utils.GetConfigfileLocation(tests_utils.TESTS_DEFAULT_CONFIG_FILE))

	xdr_runTestcase(t)
}

/**
* complete logic to call watcher, generate-mock data and asset is part of this function
 */
func xdr_runTestcase(t *testing.T) {

	// // Check passoneKeys
	// nodeWatcher := &NodeStatsWatcher{}
	// nwPassOneKeys := nodeWatcher.PassOneKeys()
	// passOneOutput, _ := data.GetProvider().RequestInfo(nwPassOneKeys)
	// fmt.Println("TestPassTwoKeys: passOneOutput: ", passOneOutput)
	// passTwoOutputs := nodeWatcher.PassTwoKeys(passOneOutput)

	// // append common keys
	// infoKeys := []string{Infokey_ClusterName, Infokey_Service, Infokey_Build}
	// passTwoOutputs = append(passTwoOutputs, infoKeys...)

	// arrRawMetrics, err := data.GetProvider().RequestInfo(passTwoOutputs)
	// assert.Nil(t, err, "Error while NodeStatsWatcher.PassTwokeys ")
	// assert.NotEmpty(t, arrRawMetrics, "Error while NamespaceWatcher.PassTwokeys, RawMetrics is EMPTY ")

	// // check the output with NodeStatsWatcher
	// nodeMetrics, err := nodeWatcher.Refresh(passTwoOutputs, arrRawMetrics)
	// assert.Nil(t, err, "Error while NodeStatsWatcher.Refresh with passTwoOutputs ")
	// assert.NotEmpty(t, nodeMetrics, "Error while NodeStatsWatcher.Refresh, NodeStatsWatcher is EMPTY ")

	// // // check the WatcherMetrics if all stats & configs coming with required labels
	// // // below block of code is used when we create the baseline mock data, which is stored in exporter_mock_results.txt for test verification/assertion
	// // // do-not-remove below code, use when to dump the output
	// // for k := range nodeMetrics {
	// // 	str := fmt.Sprintf("%#v", nodeMetrics[k])
	// // 	fmt.Println(str)
	// // }

	// udh := &tests_utils.UnittestDataHandler{}
	// ndv := udh.GetUnittestValidator("node")
	// expected_results := ndv.GetMetricLabelsWithValues(*udh)

	// for k := range nodeMetrics {
	// 	// convert / serialize to string which can be compared to stored expected mock result
	// 	str_metric := fmt.Sprintf("%#v", nodeMetrics[k])
	// 	_, exists := expected_results[str_metric]
	// 	assert.True(t, exists, "Failed, did not find expected result: "+str_metric)
	// }

}
