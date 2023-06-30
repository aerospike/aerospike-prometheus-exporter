package main

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
)

func TestSindex_PassOneKeys(t *testing.T) {
	watcher := new(SindexWatcher)

	config = new(Config)
	initConfig(LABELS_APE_TOML, config)
	config.validateAndUpdate()

	// Check passoneKeys
	passOneKeysOutputs := watcher.passOneKeys()
	// assert.Nil(t, passOneKeys)

	assert.Equal(t, passOneKeysOutputs, []string{"sindex"})

}

func TestSindex_PassTwoKeys(t *testing.T) {
	watcher := new(SindexWatcher)

	// simulate, as if we are sending requestInfo to AS and get the NodeStats, these are coming from mock-data-generator
	rawMetrics := getRawMetrics()
	msdg := new(MockSindexDataGen)
	expectedOutputs := msdg.createSindexPassTwoExpectedOutputs(rawMetrics)

	outputs := watcher.passTwoKeys(rawMetrics)

	fmt.Println("TestSindex_PassTwoKeys: outputs: ", outputs)

	assert.Equal(t, outputs, expectedOutputs)
}

func TestSindex_RefreshDefault(t *testing.T) {
	os.Setenv(TESTCASE_MODE, TESTCASE_MODE_TRUE)

	fmt.Println("initializing config ... TestSindex_RefreshDefault")
	// Initialize and validate config
	config = new(Config)
	initConfig(DEFAULT_APE_TOML, config)

	config.validateAndUpdate()

	// run the test-case logic
	sindex_runTestCase(t)

	os.Setenv(TESTCASE_MODE, TESTCASE_MODE_FALSE)
}

func TestSindex_Allowlist(t *testing.T) {
	os.Setenv(TESTCASE_MODE, TESTCASE_MODE_TRUE)

	fmt.Println("initializing config ... TestSets_Allowlist")
	// Initialize and validate config
	config = new(Config)
	initConfig(NS_ALLOWLIST_APE_TOML, config)

	config.validateAndUpdate()

	// run the test-case logic
	sindex_runTestCase(t)

	os.Setenv(TESTCASE_MODE, TESTCASE_MODE_FALSE)
}

func TestSindex_RefreshWithLabelsConfig(t *testing.T) {
	os.Setenv(TESTCASE_MODE, TESTCASE_MODE_TRUE)

	fmt.Println("initializing config ... TestSindex_RefreshWithLabelsConfig")
	// Initialize and validate config
	config = new(Config)
	initConfig(LABELS_APE_TOML, config)
	config.validateAndUpdate()

	watcher := new(SindexWatcher)

	gaugeStatHandler = new(GaugeStats)

	initGaugeStats(METRICS_CONFIG_FILE, gaugeStatHandler)
	rawMetrics := getRawMetrics()

	lObserver := &Observer{}
	ch := make(chan prometheus.Metric, 1000)

	sindexInfoKeys := watcher.passTwoKeys(rawMetrics)

	watcher.passTwoKeys(rawMetrics)
	err := watcher.refresh(lObserver, sindexInfoKeys, rawMetrics, ch)

	if err != nil {
		fmt.Println("watcher_sindex_test : Unable to refresh Sindex")
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

				// Description: Desc{fqName: "aerospike_sindex_entries_per_rec", help: "entries per rec", constLabels: {sample="sample_label_value"}, variableLabels: [cluster_name service ns sindex]}
				// Label: [name:"cluster_name" value:"null"  name:"ns" value:"test"  name:"sample" value:"sample_label_value"  name:"service" value:"172.17.0.3:3000"  name:"sindex" value:"test_sindex1" ]

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
func sindex_runTestCase(t *testing.T) {
	watcher := new(SindexWatcher)

	gaugeStatHandler = new(GaugeStats)

	initGaugeStats(METRICS_CONFIG_FILE, gaugeStatHandler)
	rawMetrics := getRawMetrics()

	lObserver := &Observer{}
	ch := make(chan prometheus.Metric, 1000)
	sindexInfoKeys := watcher.passTwoKeys(rawMetrics)

	watcher.passTwoKeys(rawMetrics)
	err := watcher.refresh(lObserver, sindexInfoKeys, rawMetrics, ch)

	if err != nil {
		fmt.Println("watcher_sets_test : Unable to refresh set stats")
	} else {
		domore := 1

		msdg := new(MockSindexDataGen)

		// map of string ==> map["namespace/metric-name"]["<Label>"]
		//  both used to assert the return values from actual code against calculated values
		lOutputValues := map[string]string{}
		lOutputLabels := map[string]string{}

		arrSindexSets := map[string]string{}

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

				// Description: Desc{fqName: "aerospike_sindex_entries_per_rec", help: "entries per rec", constLabels: {sample="sample_label_value"}, variableLabels: [cluster_name service ns sindex]}
				// Label: [name:"cluster_name" value:"null"  name:"ns" value:"test"  name:"service" value:"172.17.0.3:3000"  name:"sindex" value:"test_sindex1" ]
				//        [name:"cluster_name" value:"null"  name:"ns" value:"test"  name:"service" value:"172.17.0.3:3000"  name:"sindex" value:"test_sindex1" ]
				metricNameFromDesc := extractMetricNameFromDesc(description)
				serviceFromLabel := extractLabelNameValueFromFullLabel(metricLabel, "service")
				namespaceFromLabel := extractLabelNameValueFromFullLabel(metricLabel, "ns")
				sindexFromLabel := extractLabelNameValueFromFullLabel(metricLabel, "sindex")

				// key will be like service/namespace/sindex/<metric_name>, this we use this check during assertion
				keyName := makeKeyname(sindexFromLabel, metricNameFromDesc, true)
				keyName = makeKeyname(namespaceFromLabel, keyName, true)
				keyName = makeKeyname(serviceFromLabel, keyName, true)

				// appends to the sets array
				sindexSetKey := makeKeyname(namespaceFromLabel, sindexFromLabel, true)
				sindexSetKey = makeKeyname(serviceFromLabel, sindexSetKey, true)
				arrSindexSets[sindexSetKey] = sindexSetKey

				lOutputValues[keyName] = metricValue
				lOutputLabels[keyName] = metricLabel
			case <-time.After(1 * time.Second):
				domore = 0

			} // end select
		}

		// we have only 1 service in our mock-data, however loop thru service array
		for _, namespaceWithSindexName := range arrSindexSets {

			lExpectedMetricNamedValues, lExpectedMetricLabels := msdg.createSindexWatcherTestData()

			for key := range lOutputValues {
				expectedValues := lExpectedMetricNamedValues[key]
				expectedLabels := lExpectedMetricLabels[key]
				outputMetricValues := lOutputValues[key]
				outpuMetricLabels := lOutputLabels[key]

				// assert - only if the value belongs to the namespace/set we read expected values and processing
				if strings.HasPrefix(key, namespaceWithSindexName) {
					assert.Contains(t, expectedValues, outputMetricValues)
					assert.Contains(t, expectedLabels, outpuMetricLabels)
				}
			}
		}

	} // end else-refresh-failure

}
