package main

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"

	dto "github.com/prometheus/client_model/go"
)

func TestPassOneKeys(t *testing.T) {
	nsWatcher := new(NamespaceWatcher)
	// Check passoneKeys
	nsPassOneKeys := nsWatcher.passOneKeys()
	assert.Equal(t, nsPassOneKeys, []string{"namespaces"})

}

func TestPassTwoKeys(t *testing.T) {
	rawMetrics := getRawMetrics()
	nsWatcher := new(NamespaceWatcher)

	// simulate, as if we are sending requestInfo to AS and get the namespaces, these are coming from mock-data-generator
	pass2Keys := requestInfoNamespaces(rawMetrics)
	nsOutputs := nsWatcher.passTwoKeys(pass2Keys)

	// nsExpected := []string{"namespace/bar", "namespace/test"}
	nsExpected := createNamespacePassTwoExpectedOutputs(rawMetrics)

	assert := assert.New(t)

	for idx := range nsExpected {
		// assert each element returned by NamespaceWatcher exists in expected outputs
		assert.Contains(nsOutputs, nsExpected[idx], " value exists!")
	}
}

func TestNamespaceRefreshDefault(t *testing.T) {

	// this is to force-reload the config in the NamespaceWatcher, this is a check on this param in NamespaceWatcher implementation
	os.Setenv(TESTCASE_MODE, TESTCASE_MODE_TRUE)

	fmt.Println("initializing config ... TestNamespaceRefreshDefault")
	// Initialize and validate config
	config = new(Config)
	initConfig(DEFAULT_APE_TOML, config)

	config.validateAndUpdate()

	runTestcase(t)
	os.Setenv(TESTCASE_MODE, TESTCASE_MODE_FALSE)
}

func TestNamespaceRefreshLabels(t *testing.T) {

	// this is to force-reload the config in the NamespaceWatcher, this is a check on this param in NamespaceWatcher implementation
	os.Setenv(TESTCASE_MODE, TESTCASE_MODE_TRUE)

	fmt.Println("initializing config ... TestNamespaceRefreshLabels")
	// Initialize and validate config
	config = new(Config)
	initConfig(LABELS_APE_TOML, config)

	config.validateAndUpdate()

	runTestcase(t)
	os.Setenv(TESTCASE_MODE, TESTCASE_MODE_FALSE)
}

func TestNamespaceRefreshAllowlist(t *testing.T) {

	// this is to force-reload the config in the NamespaceWatcher, this is a check on this param in NamespaceWatcher implementation
	os.Setenv(TESTCASE_MODE, TESTCASE_MODE_TRUE)

	fmt.Println("initializing config ... TestNamespaceRefreshAllowlist")
	// Initialize and validate config
	config = new(Config)
	initConfig(NS_ALLOWLIST_APE_TOML, config)

	config.validateAndUpdate()

	runTestcase(t)

}

func TestNamespaceRefreshBlocklist(t *testing.T) {

	// this is to force-reload the config in the NamespaceWatcher, this is a check on this param in NamespaceWatcher implementation
	os.Setenv(TESTCASE_MODE, TESTCASE_MODE_TRUE)

	fmt.Println("initializing config ... TestNamespaceRefreshBlocklist")
	// Initialize and validate config
	config = new(Config)
	initConfig(NS_BLOCKLIST_APE_TOML, config)

	config.validateAndUpdate()

	runTestcase(t)

}

/**
* complete logic to call watcher, generate-mock data and asset is part of this function
 */
func runTestcase(t *testing.T) {

	gaugeStatHandler = new(GaugeStats)

	initGaugeStats(METRICS_CONFIG_FILE, gaugeStatHandler)

	// read raw-metrics from mock data gen, create observer and channel prometeus metric ingestion and processing
	rawMetrics := getRawMetrics()
	nsWatcher := new(NamespaceWatcher)
	lObserver := &Observer{}
	ch := make(chan prometheus.Metric, 1000)
	pass2Metrics := requestInfoNamespaces(rawMetrics)

	nsWatcher.passTwoKeys(rawMetrics)

	nsInfoKeys := createNamespacePassTwoExpectedOutputs(rawMetrics)

	// outputs := nsWatcher.passTwoKeys(pass2Metrics)
	// assert.Equal(t, outputs, expectedOutputs)

	err := nsWatcher.refresh(lObserver, nsInfoKeys, rawMetrics, ch)

	if err == nil {
		// map of string ==> map["namespace/metric-name"]["<VALUE>"]
		// map of string ==> map["namespace/metric-name"]["<Label>"]
		//  both used to assert the return values from actual code against calculated values
		lOutputValues := map[string]string{}
		lOutputLabels := map[string]string{}

		// reads data from the Prom channel and creates a map of strings so we can assert in the below loop
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

				metricValue := ""
				metricLabel := fmt.Sprintf("%s", protobuffer.Label)
				if protobuffer.Gauge != nil {
					metricValue = fmt.Sprintf("%.0f", *protobuffer.Gauge.Value)
				} else if protobuffer.Counter != nil {
					metricValue = fmt.Sprintf("%.0f", *protobuffer.Counter.Value)
				}

				// Desc{fqName: "aerospike_namespac_memory_free_pct", help: "memory free pct", constLabels: {}, variableLabels: [cluster_name service ns]}
				metricNameFromDesc := extractMetricNameFromDesc(description)
				namespaceFromLabel := extractNamespaceFromLabel(metricLabel)

				// key will be like namespace/<metric_name>, this we use this check during assertion
				keyName := makeKeyname(namespaceFromLabel, metricNameFromDesc, true)
				lOutputValues[keyName] = metricValue
				lOutputLabels[keyName] = metricLabel

			case <-time.After(1 * time.Second):
				domore = 0

			} // end select
		}

		// loop each namespace and compare the label and value
		arrNames := strings.Split(pass2Metrics["namespaces"], ";")

		for nsIndex := range arrNames {
			tnsForNamespace := arrNames[nsIndex]
			fmt.Println("Running test data assertion for namespace : ", tnsForNamespace)
			lExpectedMetricNamedValues, lExpectedMetricLabels := createNamespaceWatcherExpectedOutputs(tnsForNamespace, true)

			for key := range lOutputValues {
				expectedValues := lExpectedMetricNamedValues[key]
				expectedLabels := lExpectedMetricLabels[key]
				outputMetricValues := lOutputValues[key]
				outpuMetricLabels := lOutputLabels[key]

				// assert - only if the value belongs to the namespace we read expected values and processing
				if strings.HasPrefix(key, tnsForNamespace) {
					assert.Contains(t, expectedValues, outputMetricValues)
					assert.Contains(t, expectedLabels, outpuMetricLabels)
				}
			}

		}
	} else {
		fmt.Println(" Failed Refreshing, error: ", err)
	}
}
