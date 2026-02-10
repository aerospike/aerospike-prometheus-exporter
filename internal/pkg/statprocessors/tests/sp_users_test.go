package statprocessors

import (
	"fmt"
	"testing"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/dataprovider"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/statprocessors"
	"github.com/stretchr/testify/assert"
)

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
	sharedState := statprocessors.NewStatProcessorSharedState()
	usersWatcher := statprocessors.NewUserStatsProcessor(sharedState)

	// append common keys
	infoKeys := []string{sharedState.Infokey_ClusterName, sharedState.Infokey_Service, sharedState.Infokey_Build}

	arrRawMetrics, err := dataprovider.GetProvider("mock").RequestInfo(infoKeys)
	assert.Nil(t, err, "Error while usersWatcher.PassTwokeys ")
	assert.NotEmpty(t, arrRawMetrics, "Error while usersWatcher.PassTwokeys, RawMetrics is EMPTY ")

	sharedState.ClusterName = arrRawMetrics[sharedState.Infokey_ClusterName]
	sharedState.Build = arrRawMetrics[sharedState.Infokey_Build]
	sharedState.Service = arrRawMetrics[sharedState.Infokey_Service]

	canFetchUsers, users, err := dataprovider.GetProvider("mock").FetchUsersDetails()
	assert.True(t, canFetchUsers, "Error while usersWatcher.FetchUsersDetails ")
	assert.Nil(t, err, "Error while usersWatcher.FetchUsersDetails ")
	assert.NotEmpty(t, users, "Error while usersWatcher.FetchUsersDetails, users is EMPTY ")

	// check the output with usersWatcher
	usersMetrics, err := usersWatcher.Refresh(users)
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
	sharedState := statprocessors.NewStatProcessorSharedState()
	usersWatcher := statprocessors.NewUserStatsProcessor(sharedState)

	// append common keys
	infoKeys := []string{sharedState.Infokey_ClusterName, sharedState.Infokey_Service, sharedState.Infokey_Build}

	arrRawMetrics, err := dataprovider.GetProvider("mock").RequestInfo(infoKeys)
	assert.Nil(t, err, "Error while usersWatcher.PassTwokeys ")
	assert.NotEmpty(t, arrRawMetrics, "Error while usersWatcher.PassTwokeys, RawMetrics is EMPTY ")

	sharedState.ClusterName = arrRawMetrics[sharedState.Infokey_ClusterName]
	sharedState.Build = arrRawMetrics[sharedState.Infokey_Build]
	sharedState.Service = arrRawMetrics[sharedState.Infokey_Service]

	// check the output with usersWatcher
	canFetchUsers, users, err := dataprovider.GetProvider("mock").FetchUsersDetails()
	assert.True(t, canFetchUsers, "Error while usersWatcher.FetchUsersDetails ")
	assert.Nil(t, err, "Error while usersWatcher.FetchUsersDetails ")
	assert.NotEmpty(t, users, "Error while usersWatcher.FetchUsersDetails, users is EMPTY ")

	usersMetrics, err := usersWatcher.Refresh(users)
	assert.Nil(t, err, "Error while usersWatcher.Refresh with user-roles ")
	assert.NotEmpty(t, usersMetrics, "Error while usersWatcher.Refresh, usersWatcher is EMPTY ")

}
