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

var TEST_XDR_PASSONE_KEYS = []string{"get-config:context=xdr"}

func TestXdr_PassOneKeys(t *testing.T) {
	fmt.Println("initializing config ... TestXdr_PassOneKeys")

	mas := new(MockAerospikeServer)
	mas.initialize()

	watcher := new(XdrWatcher)
	// Check passoneKeys
	passOneKeys := watcher.passOneKeys()
	assert.Equal(t, passOneKeys, TEST_XDR_PASSONE_KEYS)

}

func TestXdr_PassTwoKeys(t *testing.T) {
	fmt.Println("initializing config ... TestXdr_PassTwoKeys")

	mas := new(MockAerospikeServer)
	mas.initialize()

	watcher := new(XdrWatcher)

	// simulate, as if we are sending requestInfo to AS and get the XdrStats, these are coming from mock-data-generator
	passOneKeyOutputs := mas.createXdrPassOneKeys()
	outputs := watcher.passTwoKeys(passOneKeyOutputs)

	expectedOutputs := mas.createXdrPassTwoExpectedOutputs(passOneKeyOutputs)

	for _, xdr := range outputs {
		assert.Contains(t, expectedOutputs, xdr)
	}
}

func TestXdr_RefreshDefault(t *testing.T) {
	fmt.Println("initializing config ... TestXdr_RefreshDefault")
	// Initialize and validate config
	config = new(Config)
	initConfig(DEFAULT_APE_TOML, config)
	config.validateAndUpdate()

	xdr_runTestCase(t)

}

func TestXdr_RefreshWithLabelsConfig(t *testing.T) {
	fmt.Println("initializing config ... TestXdr_RefreshWithLabelsConfig")

	mas := new(MockAerospikeServer)
	mas.initialize()
	rawMetrics := mas.fetchRawMetrics()

	// Initialize and validate config
	config = new(Config)
	initConfig(LABELS_APE_TOML, config)
	config.validateAndUpdate()

	watcher := new(XdrWatcher)

	gaugeStatHandler = new(GaugeStats)

	initGaugeStats(METRICS_CONFIG_FILE, gaugeStatHandler)
	lObserver := &Observer{}
	ch := make(chan prometheus.Metric, 1000)

	passOneKeyOutputs := mas.createXdrPassOneKeys()
	xdrInfoKeys := watcher.passTwoKeys(passOneKeyOutputs)

	err := watcher.refresh(lObserver, xdrInfoKeys, rawMetrics, ch)

	if err != nil {
		fmt.Println("watcher_xdr_test : Unable to refresh Xdr")
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

// *************************
func TestXdr_Allowlist(t *testing.T) {
	fmt.Println("initializing config ... TestXdr_Allowlist")
	// Initialize and validate config
	config = new(Config)
	initConfig(NS_ALLOWLIST_APE_TOML, config)

	config.validateAndUpdate()

	// run the test-case logic
	xdr_runTestCase(t)
}

/**
* complete logic to call watcher, generate-mock data and asset is part of this function
 */
func xdr_runTestCase(t *testing.T) {

	mas := new(MockAerospikeServer)
	mas.initialize()

	xdrdg := new(MockXdrPromMetricGenerator)

	watcher := new(XdrWatcher)

	gaugeStatHandler = new(GaugeStats)

	initGaugeStats(METRICS_CONFIG_FILE, gaugeStatHandler)
	rawMetrics := mas.fetchRawMetrics()

	lObserver := &Observer{}
	ch := make(chan prometheus.Metric, 10000)

	passOneKeyOutputs := mas.createXdrPassOneKeys()
	xdrInfoKeys := watcher.passTwoKeys(passOneKeyOutputs)

	watcher.passTwoKeys(rawMetrics)
	err := watcher.refresh(lObserver, xdrInfoKeys, rawMetrics, ch)

	if err != nil {
		fmt.Println("watcher_xdr_test : Unable to refresh set stats")
	} else {
		domore := 1

		// map of string ==> map["namespace/metric-name"]["<Label>"]
		//  both used to assert the return values from actual code against calculated values
		lOutputValues := map[string]string{}
		lOutputLabels := map[string]string{}

		arrXdrDcSets := map[string]string{}

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

				// Description: Desc{fqName: "aerospike_xdr_hot_keys", help: "hot keys", constLabels: {a="hello"}, variableLabels: [cluster_name service dc]}
				// Label: [name:"cluster_name" value:"null"  name:"dc" value:"backup_dc_asdev20"  name:"service" value:"172.17.0.3:3000" ]

				metricNameFromDesc := extractMetricNameFromDesc(description)
				dcFromLabel := extractLabelNameValueFromFullLabel(metricLabel, "dc")
				serviceFromLabel := extractLabelNameValueFromFullLabel(metricLabel, "service")
				namespaceFromLabel := extractLabelNameValueFromFullLabel(metricLabel, "ns")
				if len(namespaceFromLabel) == 0 {
					namespaceFromLabel = "no-namespace"
				}

				// key will be like namespace/set/<metric_name>, this we use this check during assertion
				keyName := makeKeyname(dcFromLabel, metricNameFromDesc, true)
				keyName = makeKeyname(namespaceFromLabel, keyName, true)
				keyName = makeKeyname(serviceFromLabel, keyName, true)

				// appends to the xdr array
				namespaceSetKey := makeKeyname(serviceFromLabel, dcFromLabel, true)
				arrXdrDcSets[namespaceSetKey] = namespaceSetKey

				lOutputValues[keyName] = metricValue
				lOutputLabels[keyName] = metricLabel
			case <-time.After(1 * time.Second):
				domore = 0

			} // end select
		}

		// we have only 1 service in our mock-data, however loop thru service array
		for _, xdrDcName := range arrXdrDcSets {

			lExpectedMetricNamedValues, lExpectedMetricLabels := xdrdg.createXdrsWatcherExpectedOutputs(mas, xdrDcName)

			for key := range lOutputValues {
				expectedValues := lExpectedMetricNamedValues[key]
				expectedLabels := lExpectedMetricLabels[key]
				outputMetricValues := lOutputValues[key]
				outpuMetricLabels := lOutputLabels[key]

				// assert - only if the value belongs to the namespace/set we read expected values and processing
				if strings.HasPrefix(key, xdrDcName) {

					assert.Contains(t, expectedValues, outputMetricValues)
					assert.Contains(t, expectedLabels, outpuMetricLabels)
				}
			}
		}

	} // end else-refresh-failure

}
