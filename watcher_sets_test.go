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

func TestSets_PassOneKeys(t *testing.T) {
	fmt.Println("initializing config ... TestSets_PassOneKeys")

	watcher := new(SetWatcher)
	// Check passoneKeys
	passOneKeys := watcher.passOneKeys()
	assert.Nil(t, passOneKeys)

}

func TestSets_PassTwoKeys(t *testing.T) {
	fmt.Println("initializing config ... TestSets_PassTwoKeys")

	watcher := new(SetWatcher)

	// mock aerospike server
	mas := new(MockAerospikeServer)
	mas.initialize()
	rawMetrics := mas.fetchRawMetrics()
	// simulate, as if we are sending requestInfo to AS and get the NodeStats, these are coming from mock-data-generator
	outputs := watcher.passTwoKeys(rawMetrics)

	assert.Equal(t, outputs, []string{"sets"})
}

func TestSets_RefreshDefault(t *testing.T) {
	fmt.Println("initializing config ... TestSets_RefreshDefault")
	// Initialize and validate config
	config = new(Config)
	initConfig(DEFAULT_APE_TOML, config)

	config.validateAndUpdate()

	// run the test-case logic
	sets_runTestCase(t)
}

func TestSets_Allowlist(t *testing.T) {
	fmt.Println("initializing config ... TestSets_Allowlist")
	// Initialize and validate config
	config = new(Config)
	initConfig(NS_ALLOWLIST_APE_TOML, config)

	config.validateAndUpdate()

	// run the test-case logic
	sets_runTestCase(t)
}

func TestSets_RefreshWithLabelsConfig(t *testing.T) {
	fmt.Println("initializing config ... TestSets_RefreshWithLabelsConfig")

	mas := new(MockAerospikeServer)
	mas.initialize()

	// Initialize and validate config
	config = new(Config)
	initConfig(LABELS_APE_TOML, config)
	config.validateAndUpdate()

	watcher := new(SetWatcher)

	gaugeStatHandler = new(GaugeStats)

	initGaugeStats(METRICS_CONFIG_FILE, gaugeStatHandler)
	rawMetrics := mas.fetchRawMetrics()

	lObserver := &Observer{}
	ch := make(chan prometheus.Metric, 10000)
	setsInfoKeys := []string{}

	watcher.passTwoKeys(rawMetrics)
	err := watcher.refresh(lObserver, setsInfoKeys, rawMetrics, ch)

	if err != nil {
		fmt.Println("watcher_latency_test : Unable to refresh Latencies")
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

				// Description: Desc{fqName: "aerospike_latencies_read_ms_bucket", help: "read ms bucket", constLabels: {}, variableLabels: [cluster_name service ns le]}
				// Label: [name:"cluster_name" value:"null"  name:"le" value:"+Inf"  name:"ns" value:"test"  name:"service" value:"172.17.0.3:3000" ]

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
func sets_runTestCase(t *testing.T) {

	// mock aerospike server
	mas := new(MockAerospikeServer)
	mas.initialize()

	msdg := new(MockSetsPromMetricGenerator)

	watcher := new(SetWatcher)

	gaugeStatHandler = new(GaugeStats)

	initGaugeStats(METRICS_CONFIG_FILE, gaugeStatHandler)
	// rawMetrics := getRawMetrics()
	rawMetrics := mas.fetchRawMetrics()

	lObserver := &Observer{}
	ch := make(chan prometheus.Metric, 1000)
	setsInfoKeys := []string{}

	watcher.passTwoKeys(rawMetrics)
	err := watcher.refresh(lObserver, setsInfoKeys, rawMetrics, ch)

	if err != nil {
		fmt.Println("watcher_sets_test : Unable to refresh set stats")
	} else {
		domore := 1

		// map of string ==> map["namespace/metric-name"]["<Label>"]
		//  both used to assert the return values from actual code against calculated values
		lOutputValues := map[string]string{}
		lOutputLabels := map[string]string{}

		arrNamespaceSets := map[string]string{}

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

				// Description: Desc{fqName: "aerospike_sets_truncate_lut", help: "truncate lut", constLabels: {}, variableLabels: [cluster_name service ns set]}
				// Label: [name:"cluster_name" value:"null"  name:"ns" value:"bar"  name:"service" value:"172.17.0.3:3000"  name:"set" value:"west_region" ]
				metricNameFromDesc := extractMetricNameFromDesc(description)
				namespaceFromLabel := extractLabelNameValueFromFullLabel(metricLabel, "ns")
				setFromLabel := extractLabelNameValueFromFullLabel(metricLabel, "set")

				// key will be like namespace/set/<metric_name>, this we use this check during assertion
				keyName := makeKeyname(setFromLabel, metricNameFromDesc, true)
				keyName = makeKeyname(namespaceFromLabel, keyName, true)

				// appends to the sets array
				namespaceSetKey := makeKeyname(namespaceFromLabel, setFromLabel, true)
				arrNamespaceSets[namespaceSetKey] = namespaceSetKey

				lOutputValues[keyName] = metricValue
				lOutputLabels[keyName] = metricLabel
			case <-time.After(1 * time.Second):
				domore = 0

			} // end select
		}

		// we have only 1 service in our mock-data, however loop thru service array
		for _, namespaceWithSetName := range arrNamespaceSets {

			lExpectedMetricNamedValues, lExpectedMetricLabels := msdg.createSetsWatcherExpectedOutputs(mas, namespaceWithSetName)

			for key := range lOutputValues {
				expectedValues := lExpectedMetricNamedValues[key]
				expectedLabels := lExpectedMetricLabels[key]
				outputMetricValues := lOutputValues[key]
				outpuMetricLabels := lOutputLabels[key]

				// assert - only if the value belongs to the namespace/set we read expected values and processing
				if strings.HasPrefix(key, namespaceWithSetName) {
					assert.Contains(t, expectedValues, outputMetricValues)
					assert.Contains(t, expectedLabels, outpuMetricLabels)
				}
			}
		}

	} // end else-refresh-failure

}
