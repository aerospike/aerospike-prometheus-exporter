package main

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
)

func TestUsers_PassOneKeys(t *testing.T) {
	fmt.Println("initializing config ... TestXdr_PassOneKeys")

	watcher := new(UserWatcher)
	// Check passoneKeys
	passOneKeys := watcher.passOneKeys()
	assert.Nil(t, passOneKeys)

}

func TestUsers_PassTwoKeys(t *testing.T) {
	watcher := new(UserWatcher)

	mas := new(MockAerospikeServer)
	mas.initialize()

	rawMetrics := mas.fetchRawMetrics()

	// simulate, as if we are sending requestInfo to AS and get the NodeStats, these are coming from mock-data-generator
	outputs := watcher.passTwoKeys(rawMetrics)

	assert.Nil(t, outputs)
}

func TestUsers_Allowlist(t *testing.T) {
	fmt.Println("initializing config ... TestUsers_Allowlist")
	// Initialize and validate config
	config = new(Config)
	initConfig(NS_ALLOWLIST_APE_TOML, config)
	config.validateAndUpdate()

	// this is required as UserWatcher cheks for user/password in the properties file
	config.Aerospike.User = "admin"
	config.Aerospike.Password = "admin"

	gaugeStatHandler = new(GaugeStats)

	initGaugeStats(METRICS_CONFIG_FILE, gaugeStatHandler)

	users_runTestCase(t)
}

func TestUsers_RefreshWithLabelsConfig(t *testing.T) {
	fmt.Println("initializing config ... TestUsers_RefreshWithLabelsConfig")

	mas := new(MockAerospikeServer)
	mas.initialize()

	// Initialize and validate config
	config = new(Config)
	initConfig(LABELS_APE_TOML, config)
	config.validateAndUpdate()

	watcher := new(UserWatcher)

	// this is required as UserWatcher cheks for user/password in the properties file
	config.Aerospike.User = "admin"
	config.Aerospike.Password = "admin"

	gaugeStatHandler = new(GaugeStats)
	initGaugeStats(METRICS_CONFIG_FILE, gaugeStatHandler)
	rawMetrics := mas.fetchRawMetrics()

	lObserver := &Observer{}
	ch := make(chan prometheus.Metric, 1000)
	userInfoKeys := watcher.passTwoKeys(rawMetrics)

	// get user data from mock
	mUsers := new(MockUsersPromMetricGenerator)
	users := mUsers.createDummyUserRoles()

	watcher.passTwoKeys(rawMetrics)
	err := watcher.refreshUserStats(lObserver, userInfoKeys, rawMetrics, ch, users)

	if err != nil {
		fmt.Println("watcher_users_test : Unable to refresh Users")
	} else {
		domore := 1

		for domore == 1 {
			select {

			case nsMetric := <-ch:
				description := nsMetric.Desc().String()
				var protobuffer dto.Metric
				err := nsMetric.Write(&protobuffer)
				if err != nil {
					fmt.Println(" unable to get metric ", description, " data into protobuf ", err)
				}

				metricLabel := fmt.Sprintf("%s", protobuffer.Label)

				// Desc{fqName: "aerospike_users_write_single_record_tps", help: "write single record tps", constLabels: {sample="sample_label_value"}, variableLabels: [cluster_name service user]}
				// [name:"cluster_name" value:"null"  name:"sample" value:"sample_label_value"  name:"service" value:"172.17.0.3:3000"  name:"user" value:"" ]

				for eachConfigMetricLabel := range config.AeroProm.MetricLabels {
					modifiedConfigMetricLabels := strings.ReplaceAll(eachConfigMetricLabel, "=", ":")

					assert.Contains(t, metricLabel, modifiedConfigMetricLabels)
				}

			case <-time.After(1 * time.Second):
				domore = 0

			} // end select
		}

	} // end else-refresh-failure

}

/**
* complete logic to call watcher, generate-mock data and asset is part of this function
 */
func users_runTestCase(t *testing.T) {

	mas := new(MockAerospikeServer)
	mas.initialize()

	watcher := new(UserWatcher)

	gaugeStatHandler = new(GaugeStats)

	initGaugeStats(METRICS_CONFIG_FILE, gaugeStatHandler)
	rawMetrics := mas.fetchRawMetrics()

	lObserver := &Observer{}
	ch := make(chan prometheus.Metric, 1000)
	userInfoKeys := watcher.passTwoKeys(rawMetrics)

	// get user data from mock
	mockUserGen := new(MockUsersPromMetricGenerator)
	userRoles := mockUserGen.createDummyUserRoles()

	// get user data from mock

	watcher.passTwoKeys(rawMetrics)
	err := watcher.refreshUserStats(lObserver, userInfoKeys, rawMetrics, ch, userRoles)

	if err != nil {
		fmt.Println("users_runTestCase : Unable to refresh User stats")
	} else {
		domore := 1

		// map of string ==> map["namespace/metric-name"]["<Label>"]
		//     both used to assert the return values from actual code against calculated values
		lOutputValues := map[string]string{}
		lOutputLabels := map[string]string{}

		arrUserRoles := map[string]string{}

		for domore == 1 {
			select {

			case nsMetric := <-ch:
				description := nsMetric.Desc().String()
				var protobuffer dto.Metric
				err := nsMetric.Write(&protobuffer)
				if err != nil {
					fmt.Println(" unable to get metric ", description, " data into protobuf ", err)
				}

				metricValue := ""
				metricLabel := fmt.Sprintf("%s", protobuffer.Label)
				if protobuffer.Gauge != nil {
					metricValue = fmt.Sprintf("%.0f", *protobuffer.Gauge.Value)
				} else if protobuffer.Counter != nil {
					metricValue = fmt.Sprintf("%.0f", *protobuffer.Counter.Value)
				}

				// Desc{fqName: "aerospike_users_write_single_record_tps", help: "write single record tps", constLabels: {sample="sample_label_value"}, variableLabels: [cluster_name service user]}
				// [name:"cluster_name" value:"null"  name:"sample" value:"sample_label_value"  name:"service" value:"172.17.0.3:3000"  name:"user" value:"" ]

				metricNameFromDesc := extractMetricNameFromDesc(description)
				userFromLabel := extractLabelNameValueFromFullLabel(metricLabel, "user")
				serviceFromLabel := extractLabelNameValueFromFullLabel(metricLabel, "service")

				// key will be like namespace/set/<metric_name>, this we use this check during assertion
				keyName := makeKeyname(userFromLabel, metricNameFromDesc, true)
				keyName = makeKeyname(serviceFromLabel, keyName, true)

				// appends to the sets array
				namespaceSetKey := makeKeyname(serviceFromLabel, userFromLabel, true)
				arrUserRoles[namespaceSetKey] = namespaceSetKey

				lOutputValues[keyName] = metricValue
				lOutputLabels[keyName] = metricLabel
			case <-time.After(1 * time.Second):
				domore = 0

			} // end select
		}

		// we have only 1 service in our mock-data, however loop thru service array
		for _, userRole := range arrUserRoles {

			lExpectedMetricNamedValues, lExpectedMetricLabels := mockUserGen.createMockUserData(mas)

			for key := range lOutputValues {
				expectedValues := lExpectedMetricNamedValues[key]
				expectedLabels := lExpectedMetricLabels[key]
				outputMetricValues := lOutputValues[key]
				outpuMetricLabels := lOutputLabels[key]

				// assert - only if the value belongs to the namespace/set we read expected values and processing
				if strings.HasPrefix(key, userRole) {
					assert.Contains(t, expectedValues, outputMetricValues)
					assert.Contains(t, expectedLabels, outpuMetricLabels)
				}
			}
		}

	} // end else-refresh-failure

}
