package watchers

import (
	"fmt"
	"testing"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/data"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/unittests"
	"github.com/stretchr/testify/assert"
)

func TestPassOneKeys(t *testing.T) {

	// Check passoneKeys
	nsWatcher := &NamespaceWatcher{}
	nsPassOneKeys := nsWatcher.PassOneKeys()

	udp := &unittests.UnittestDataProvider{}
	ndv := udp.GetUnittestValidator("namespace")
	passOneOutputs := ndv.GetPassOneKeys(*udp)

	// fmt.Println("TestPassOneKeys: ", passOneOutputs)
	var expectedOutputs []string
	expectedOutputs = append(expectedOutputs, passOneOutputs["namespaces"])

	assert.Equal(t, nsPassOneKeys, expectedOutputs)

}

func TestPassTwoKeys(t *testing.T) {

	// rawMetrics := getRawMetrics()
	nsWatcher := new(NamespaceWatcher)

	// simulate, as if we are sending requestInfo to AS and get the namespaces, these are coming from mock-data-generator
	passOneKeys := nsWatcher.PassOneKeys()
	passOneOutput, _ := data.GetProvider().RequestInfo(passOneKeys)
	passTwokeyOutputs := nsWatcher.PassTwoKeys(passOneOutput)

	fmt.Println("TestPassTwoKeys: ", passTwokeyOutputs)

	// expectedOutputs := []string{"namespace/bar", "namespace/test"}
	udp := &unittests.UnittestDataProvider{}
	ndv := udp.GetUnittestValidator("namespace")
	expectedOutputs := ndv.GetPassTwoKeys(*udp)

	assert := assert.New(t)

	assert.NotEmpty(passTwokeyOutputs)
	assert.NotEmpty(expectedOutputs)

	for idx := range expectedOutputs {
		// assert each element returned by NamespaceWatcher exists in expected outputs
		fmt.Println("expected outputs: key & value", idx, expectedOutputs[idx])
		assert.Contains(passTwokeyOutputs, expectedOutputs[idx], " value exists!")
	}
}

// func TestNamespaceRefreshDefault(t *testing.T) {

// 	fmt.Println("initializing config ... TestNamespaceRefreshDefault")
// 	// Initialize and validate config
// 	config = new(Config)
// 	initConfig(DEFAULT_APE_TOML, config)

// 	config.validateAndUpdate()

// 	runTestcase(t)
// }

// /**
// * complete logic to call watcher, generate-mock data and asset is part of this function
//  */
// func runTestcase(t *testing.T) {

// 	// mock aerospike server
// 	mas := new(MockAerospikeServer)
// 	mas.initialize()

// 	// mock namespace prom metric generator
// 	nsdg := new(MockNamespacePromMetricGenerator)

// 	// initialize gauges list
// 	gaugeStatHandler = new(GaugeStats)
// 	initGaugeStats(METRICS_CONFIG_FILE, gaugeStatHandler)

// 	// read raw-metrics from mock data gen, create observer and channel prometeus metric ingestion and processing
// 	// rawMetrics := getRawMetrics()
// 	rawMetrics := mas.fetchRawMetrics()

// 	// Actual Watcher-Namespace generator code
// 	nsWatcher := new(NamespaceWatcher)
// 	lObserver := &Observer{}
// 	ch := make(chan prometheus.Metric, 10000)

// 	passOneKeyOutputs := mas.requestInfoNamespaces()
// 	passTwokeyOutputs := nsWatcher.passTwoKeys(passOneKeyOutputs)

// 	err := nsWatcher.refresh(lObserver, passTwokeyOutputs, rawMetrics, ch)

// 	if err == nil {
// 		// map of string ==> map["namespace/metric-name"]["<VALUE>"]
// 		// map of string ==> map["namespace/metric-name"]["<Label>"]
// 		//  both used to assert the return values from actual code against calculated values
// 		lOutputValues := map[string]string{}
// 		lOutputLabels := map[string]string{}

// 		// reads data from the Prom channel and creates a map of strings so we can assert in the below loop
// 		domore := 1

// 		// mock_expected_filedump := []string{}

// 		for domore == 1 {
// 			select {

// 			case nsMetric := <-ch:
// 				description := nsMetric.Desc().String()
// 				var protobuffer dto.Metric
// 				err := nsMetric.Write(&protobuffer)
// 				if err != nil {
// 					fmt.Println(" unable to get metric ", description, " data into protobuf ", err)
// 				}

// 				metricValue := ""
// 				metricLabel := fmt.Sprintf("%s", protobuffer.Label)
// 				if protobuffer.Gauge != nil {
// 					metricValue = fmt.Sprintf("%.0f", *protobuffer.Gauge.Value)
// 				} else if protobuffer.Counter != nil {
// 					metricValue = fmt.Sprintf("%.0f", *protobuffer.Counter.Value)
// 				}

// 				// Desc{fqName: "aerospike_namespac_memory_free_pct", help: "memory free pct", constLabels: {}, variableLabels: [cluster_name service ns]}
// 				metricNameFromDesc := extractMetricNameFromDesc(description)
// 				namespaceFromLabel := extractNamespaceFromLabel(metricLabel)
// 				// namespaceFromLabel := extractLabelNameValueFromFullLabel(metricLabel)
// 				labelString := stringifyLabel(metricLabel)

// 				// key will be like namespace/<metric_name>, this we use this check during assertion
// 				keyName := makeKeyname(namespaceFromLabel, labelString, true)
// 				keyName = makeKeyname(keyName, metricNameFromDesc, true)

// 				lOutputValues[keyName] = metricValue
// 				lOutputLabels[keyName] = metricLabel

// 				// to dump expected outputs to a file, so no-need-to regenerate every-time
// 				// mock_expected_filedump = append(mock_expected_filedump, makeMetricReady2dump(namespaceFromLabel, metricNameFromDesc, description, metricValue))

// 			case <-time.After(1 * time.Second):
// 				domore = 0

// 			} // end select
// 		}

// 		// loop each namespace and compare the label and value
// 		arrNames := strings.Split(passOneKeyOutputs["namespaces"], ";")

// 		for nsIndex := range arrNames {
// 			tnsForNamespace := arrNames[nsIndex]
// 			lExpectedMetricNamedValues, lExpectedMetricLabels := nsdg.createNamespaceWatcherExpectedOutputs(mas, tnsForNamespace, true)

// 			for key := range lOutputValues {
// 				expectedValues := lExpectedMetricNamedValues[key]
// 				expectedLabels := lExpectedMetricLabels[key]
// 				outputMetricValues := lOutputValues[key]
// 				outpuMetricLabels := lOutputLabels[key]
// 				// assert - only if the value belongs to the namespace we read expected values and processing
// 				//  a "/" because namespace can be like test and test_on_shmem,
// 				if strings.HasPrefix(key, tnsForNamespace+"/") {

// 					assert.Contains(t, expectedValues, outputMetricValues)
// 					assert.Contains(t, expectedLabels, outpuMetricLabels)
// 				}
// 			}

// 		}
// 	} else {
// 		fmt.Println(" Failed Refreshing, error: ", err)
// 	}

// }

// // to dump expected outputs to a file, so no-need-to regenerate every-time
// // func makeMetricReady2dump(ns string, metric string, desc string, value string) string {
// // 	return ns + "#" + metric + "#" + desc + "#" + value
// // }
