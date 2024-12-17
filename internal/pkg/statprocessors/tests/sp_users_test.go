package statprocessors

import (
	"fmt"
	"testing"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/dataprovider"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/statprocessors"
	"github.com/stretchr/testify/assert"
)

func Test_Users_PassOneKeys(t *testing.T) {

	fmt.Println("initializing config ... Test_Users_PassOneKeys")

	// Check passoneKeys
	usersWatcher := &statprocessors.UserStatsProcessor{}
	nwPassOneKeys := usersWatcher.PassOneKeys()

	udh := &UnittestDataHandler{}
	ndv := udh.GetUnittestValidator("users")
	passOneOutputs := ndv.GetPassOneKeys()

	assert.Nil(t, nwPassOneKeys, passOneOutputs)

}

func Test_Users_PassTwoKeys(t *testing.T) {

	fmt.Println("initializing config ... Test_Users_PassTwoKeys")

	// initialize config and gauge-lists
	commons.InitConfigurations(commons.GetWatchersConfigFile(commons.TESTS_USERS_CONFIG_FILE))

	// Check passoneKeys
	usersWatcher := &statprocessors.UserStatsProcessor{}
	nwPassOneKeys := usersWatcher.PassOneKeys()
	passOneOutput, _ := dataprovider.GetProvider().RequestInfo(nwPassOneKeys)
	fmt.Println("Test_Users_PassTwoKeys: passOneOutput: ", passOneOutput)
	passTwoOutputs := usersWatcher.PassTwoKeys(passOneOutput)

	udh := &UnittestDataHandler{}
	ndv := udh.GetUnittestValidator("users")
	expectedPassTwoOutputs := ndv.GetPassTwoKeys()

	assert.Nil(t, passTwoOutputs)
	assert.Nil(t, expectedPassTwoOutputs)

}

func Test_Users_RefreshDefault(t *testing.T) {

	fmt.Println("initializing config ... Test_Users_RefreshDefault")

	// initialize config and gauge-lists
	commons.InitConfigurations(commons.GetWatchersConfigFile(commons.TESTS_USERS_CONFIG_FILE))

	users_runTestcase(t)
}

/**
* complete logic to call watcher, generate-mock data and asset is part of this function
 */
func users_runTestcase(t *testing.T) {

	// Check passoneKeys
	usersWatcher := &statprocessors.UserStatsProcessor{}
	nwPassOneKeys := usersWatcher.PassOneKeys()
	passOneOutput, _ := dataprovider.GetProvider().RequestInfo(nwPassOneKeys)
	fmt.Println("users_runTestcase: passOneOutput: ", passOneOutput)
	passTwoOutputs := usersWatcher.PassTwoKeys(passOneOutput)

	// append common keys
	infoKeys := []string{statprocessors.Infokey_ClusterName, statprocessors.Infokey_Service, statprocessors.Infokey_Build}
	passTwoOutputs = append(passTwoOutputs, infoKeys...)

	arrRawMetrics, err := dataprovider.GetProvider().RequestInfo(passTwoOutputs)
	assert.Nil(t, err, "Error while usersWatcher.PassTwokeys ")
	assert.NotEmpty(t, arrRawMetrics, "Error while usersWatcher.PassTwokeys, RawMetrics is EMPTY ")

	statprocessors.ClusterName = arrRawMetrics[statprocessors.Infokey_ClusterName]
	statprocessors.Build = arrRawMetrics[statprocessors.Infokey_Build]
	statprocessors.Service = arrRawMetrics[statprocessors.Infokey_Service]

	// check the output with usersWatcher
	usersMetrics, err := usersWatcher.Refresh(passTwoOutputs, arrRawMetrics)
	assert.Nil(t, err, "Error while usersWatcher.Refresh with passTwoOutputs ")
	assert.NotEmpty(t, usersMetrics, "Error while usersWatcher.Refresh, usersWatcher is EMPTY ")

	udh := &UnittestDataHandler{}
	ndv := udh.GetUnittestValidator("users")
	expected_results := ndv.GetMetricLabelsWithValues()

	for k := range usersMetrics {
		// convert / serialize to string which can be compared to stored expected mock result
		str_metric := fmt.Sprintf("%#v", usersMetrics[k])
		_, exists := expected_results[str_metric]
		assert.True(t, exists, "Failed, did not find expected result: "+str_metric)
	}

}

func Test_Users_TestPKI(t *testing.T) {

	fmt.Println("initializing config ... Test_Users_TestPKI")

	// initialize config and gauge-lists
	commons.InitConfigurations(commons.GetWatchersConfigFile(commons.TESTS_USERS_PKI_CONFIG_FILE))

	users_runTestcase(t)
}

func Test_Users_Not_Configured(t *testing.T) {

	fmt.Println("initializing config ... Test_Users_Not_Configured")

	// initialize config and gauge-lists
	commons.InitConfigurations(commons.GetWatchersConfigFile(commons.TESTS_DEFAULT_CONFIG_FILE))

	// Check passoneKeys
	usersWatcher := &statprocessors.UserStatsProcessor{}
	nwPassOneKeys := usersWatcher.PassOneKeys()
	passOneOutput, _ := dataprovider.GetProvider().RequestInfo(nwPassOneKeys)
	fmt.Println("users_runTestcase: passOneOutput: ", passOneOutput)
	passTwoOutputs := usersWatcher.PassTwoKeys(passOneOutput)

	// append common keys
	infoKeys := []string{statprocessors.Infokey_ClusterName, statprocessors.Infokey_Service, statprocessors.Infokey_Build}
	passTwoOutputs = append(passTwoOutputs, infoKeys...)

	arrRawMetrics, err := dataprovider.GetProvider().RequestInfo(passTwoOutputs)
	assert.Nil(t, err, "Error while usersWatcher.PassTwokeys ")
	assert.NotEmpty(t, arrRawMetrics, "Error while usersWatcher.PassTwokeys, RawMetrics is EMPTY ")

	statprocessors.ClusterName = arrRawMetrics[statprocessors.Infokey_ClusterName]
	statprocessors.Build = arrRawMetrics[statprocessors.Infokey_Build]
	statprocessors.Service = arrRawMetrics[statprocessors.Infokey_Service]

	// check the output with usersWatcher
	usersMetrics, err := usersWatcher.Refresh(passTwoOutputs, arrRawMetrics)
	assert.Nil(t, err, "Error while usersWatcher.Refresh with passTwoOutputs ")
	assert.Empty(t, usersMetrics, "Error while usersWatcher.Refresh, usersWatcher is EMPTY ")

}
