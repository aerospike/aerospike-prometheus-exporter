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

func TestLatencies_PassOneKeys(t *testing.T) {
	watcher := new(LatencyWatcher)
	// Check passoneKeys
	passOneKeys := watcher.passOneKeys()

	fmt.Println("TestLatencies_PassOneKeys: outputs: ", passOneKeys)

	// assert.Equal(t, passOneKeys, []string{"build"})
	// modified test-case as watcher-latency no more returns build, this "build" is moved to observer.go
	assert.Nil(t, passOneKeys)
}

func TestLatencies_PassTwoKeys(t *testing.T) {
	fmt.Println("initializing config ... TestLatencies_PassTwoKeys")

	mas := new(MockAerospikeServer)
	mas.initialize()

	// Initialize and validate config
	config = new(Config)
	initConfig(DEFAULT_APE_TOML, config)
	config.validateAndUpdate()

	watcher := new(LatencyWatcher)

	rawMetrics := mas.fetchRawMetrics()
	// simulate, as if we are sending requestInfo to AS and get the Latencies, these are coming from mock-data-generator
	outputs := watcher.passTwoKeys(rawMetrics)

	fmt.Println("TestLatencies_PassTwoKeys: outputs: ", outputs)

	// output is not nil
	assert.NotNil(t, outputs, "build details returns are: ", outputs)
	assert.Equal(t, outputs, []string{"latencies:"})
}

func TestLatencies_RefreshDefault(t *testing.T) {
	fmt.Println("initializing config ... TestLatencies_RefreshDefault")
	// Initialize and validate config
	config = new(Config)
	initConfig(DEFAULT_APE_TOML, config)
	config.validateAndUpdate()

	// run the test-case logic
	latencies_runTestCase(t)

}

func TestLatencies_RefreshWithLabelsConfig(t *testing.T) {
	fmt.Println("initializing config ... TestLatencies_RefreshWithLabelsConfig")

	mas := new(MockAerospikeServer)
	mas.initialize()

	// Initialize and validate config
	config = new(Config)
	initConfig(LABELS_APE_TOML, config)
	config.validateAndUpdate()

	watcher := new(LatencyWatcher)

	gaugeStatHandler = new(GaugeStats)
	initGaugeStats(METRICS_CONFIG_FILE, gaugeStatHandler)
	rawMetrics := mas.fetchRawMetrics()

	lObserver := &Observer{}
	ch := make(chan prometheus.Metric, 1000)
	latenciesInfoKeys := watcher.passTwoKeys(rawMetrics)

	err := watcher.refresh(lObserver, latenciesInfoKeys, rawMetrics, ch)

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
func latencies_runTestCase(t *testing.T) {

	// mock server
	mas := new(MockAerospikeServer)
	mas.initialize()

	latdg := new(MockLatencyPromMetricGenerator)

	gaugeStatHandler = new(GaugeStats)
	initGaugeStats(METRICS_CONFIG_FILE, gaugeStatHandler)
	rawMetrics := mas.fetchRawMetrics()

	watcher := new(LatencyWatcher)
	lObserver := &Observer{}
	ch := make(chan prometheus.Metric, 10000)

	latenciesInfoKeys := watcher.passTwoKeys(rawMetrics)
	err := watcher.refresh(lObserver, latenciesInfoKeys, rawMetrics, ch)

	if err != nil {
		fmt.Println("watcher_latency_test : Unable to refresh Latencies")
	} else {
		domore := 1

		// map of string ==> map["namespace/metric-name"]["<Label>"]
		//  both used to assert the return values from actual code against calculated values
		lOutputValues := map[string]string{}
		lOutputLabels := map[string]string{}

		arrServices := map[string]string{}

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

				// Description: Desc{fqName: "aerospike_latencies_read_ms_bucket", help: "read ms bucket", constLabels: {}, variableLabels: [cluster_name service ns le]}
				// Label: [name:"cluster_name" value:"null"  name:"le" value:"+Inf"  name:"ns" value:"test"  name:"service" value:"172.17.0.3:3000" ]
				metricNameFromDesc := extractMetricNameFromDesc(description)

				operationNameFromMetricName := extractOperationFromMetric(metricNameFromDesc)

				serviceFromLabel := extractLabelNameValueFromFullLabel(metricLabel, "service")
				nsFromLabel := extractLabelNameValueFromFullLabel(metricLabel, "ns")
				leFromLabel := extractLabelNameValueFromFullLabel(metricLabel, "le")
				leFromLabel = strings.ReplaceAll(leFromLabel, "+", "") // replacing +Inf
				// key will be like namespace/<metric_name>, this we use this check during assertion
				keyName := serviceFromLabel + "_" + nsFromLabel + "_" + operationNameFromMetricName + "_" + leFromLabel

				if len(leFromLabel) == 0 { // this is <OPERATION>_count metric
					keyName = serviceFromLabel + "_" + nsFromLabel + "_" + operationNameFromMetricName
				}
				// service_ns_operation_le/
				// appends to the service array
				arrServices[keyName] = keyName

				lOutputValues[keyName] = metricValue
				lOutputLabels[keyName] = metricLabel
			case <-time.After(1 * time.Second):
				domore = 0

			} // end select
		}

		// we have only 1 service in our mock-data, however loop thru service array
		for _, keyValue := range arrServices {

			lExpectedMetricNamedValues, lExpectedMetricLabels := latdg.createLatencysWatcherExpectedOutputs(mas, keyValue)

			for key := range lOutputValues {
				expectedValues := lExpectedMetricNamedValues[key]
				expectedLabels := lExpectedMetricLabels[key]
				outputMetricValues := lOutputValues[key]
				outpuMetricLabels := lOutputLabels[key]

				// assert - only if the value belongs to the service we read expected values and processing
				if strings.HasPrefix(key, keyValue) {
					assert.Contains(t, expectedValues, outputMetricValues)
					assert.Contains(t, expectedLabels, outpuMetricLabels)

				}
			}
		}

	} // end else-refresh-failure

}
