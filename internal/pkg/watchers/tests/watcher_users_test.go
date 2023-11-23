package watchers

import (
	"fmt"
	"testing"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/data"
	tests_utils "github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/tests_utils"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/watchers"
	"github.com/stretchr/testify/assert"
)

func Test_Users_PassOneKeys(t *testing.T) {

	fmt.Println("initializing config ... Test_Users_PassOneKeys")

	// Check passoneKeys
	usersWatcher := &watchers.UserWatcher{}
	nwPassOneKeys := usersWatcher.PassOneKeys()

	udh := &tests_utils.UnittestDataHandler{}
	ndv := udh.GetUnittestValidator("users")
	passOneOutputs := ndv.GetPassOneKeys()

	assert.Nil(t, nwPassOneKeys, passOneOutputs)

}

func Test_Users_PassTwoKeys(t *testing.T) {

	fmt.Println("initializing config ... Test_Users_PassTwoKeys")

	// initialize config and gauge-lists
	tests_utils.InitConfigurations(tests_utils.TESTS_USERS_CONFIG_FILE)

	// Check passoneKeys
	usersWatcher := &watchers.UserWatcher{}
	nwPassOneKeys := usersWatcher.PassOneKeys()
	passOneOutput, _ := data.GetProvider().RequestInfo(nwPassOneKeys)
	fmt.Println("Test_Users_PassTwoKeys: passOneOutput: ", passOneOutput)
	passTwoOutputs := usersWatcher.PassTwoKeys(passOneOutput)

	udh := &tests_utils.UnittestDataHandler{}
	ndv := udh.GetUnittestValidator("users")
	expectedPassTwoOutputs := ndv.GetPassTwoKeys()

	assert.Nil(t, passTwoOutputs)
	assert.Nil(t, expectedPassTwoOutputs)

}

func Test_Users_RefreshDefault(t *testing.T) {

	fmt.Println("initializing config ... Test_Users_RefreshDefault")

	// initialize config and gauge-lists
	tests_utils.InitConfigurations(tests_utils.TESTS_USERS_CONFIG_FILE)

	users_runTestcase(t)
}

/**
* complete logic to call watcher, generate-mock data and asset is part of this function
 */
func users_runTestcase(t *testing.T) {

	// Check passoneKeys
	usersWatcher := &watchers.UserWatcher{}
	nwPassOneKeys := usersWatcher.PassOneKeys()
	passOneOutput, _ := data.GetProvider().RequestInfo(nwPassOneKeys)
	fmt.Println("users_runTestcase: passOneOutput: ", passOneOutput)
	passTwoOutputs := usersWatcher.PassTwoKeys(passOneOutput)

	// append common keys
	infoKeys := []string{watchers.Infokey_ClusterName, watchers.Infokey_Service, watchers.Infokey_Build}
	passTwoOutputs = append(passTwoOutputs, infoKeys...)

	arrRawMetrics, err := data.GetProvider().RequestInfo(passTwoOutputs)
	assert.Nil(t, err, "Error while usersWatcher.PassTwokeys ")
	assert.NotEmpty(t, arrRawMetrics, "Error while usersWatcher.PassTwokeys, RawMetrics is EMPTY ")

	// check the output with usersWatcher
	usersMetrics, err := usersWatcher.Refresh(passTwoOutputs, arrRawMetrics)
	assert.Nil(t, err, "Error while usersWatcher.Refresh with passTwoOutputs ")
	assert.NotEmpty(t, usersMetrics, "Error while usersWatcher.Refresh, usersWatcher is EMPTY ")

	// // check the WatcherMetrics if all stats & configs coming with required labels
	// // below block of code is used when we create the baseline mock data, which is stored in exporter_mock_results.txt for test verification/assertion
	// // do-not-remove below code, use when to dump the output
	// for k := range usersMetrics {
	// 	str := fmt.Sprintf("%#v", usersMetrics[k])
	// 	fmt.Println(str)
	// }

	udh := &tests_utils.UnittestDataHandler{}
	ndv := udh.GetUnittestValidator("users")
	expected_results := ndv.GetMetricLabelsWithValues()

	for k := range usersMetrics {
		// convert / serialize to string which can be compared to stored expected mock result
		str_metric := fmt.Sprintf("%#v", usersMetrics[k])
		_, exists := expected_results[str_metric]
		assert.True(t, exists, "Failed, did not find expected result: "+str_metric)
	}

}
