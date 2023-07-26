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

func TestNodeStats_PassOneKeys(t *testing.T) {
	fmt.Println("initializing config ... TestNodeStats_PassOneKeys")

	watcher := new(StatsWatcher)
	// Check passoneKeys
	passOneKeys := watcher.passOneKeys()
	assert.Nil(t, passOneKeys)

}

func TestNodeStats_PassTwoKeys(t *testing.T) {
	fmt.Println("initializing config ... TestNodeStats_PassTwoKeys")

	watcher := new(StatsWatcher)

	// simulate, as if we are sending requestInfo to AS and get the NodeStats, these are coming from mock-data-generator
	pass2Keys := make(map[string]string)
	outputs := watcher.passTwoKeys(pass2Keys)

	assert.Equal(t, outputs, []string{"get-config:context=service", "statistics"})
}

func TestNodeStats_RefreshDefault(t *testing.T) {
	fmt.Println("initializing config ... TestNodeStats_RefreshDefault")
	// Initialize and validate config
	config = new(Config)
	initConfig(DEFAULT_APE_TOML, config)

	config.validateAndUpdate()

	// run the test-case logic
	nodeStats_runTestCase(t)
}

func TestNodeStats_Allowlist(t *testing.T) {
	fmt.Println("initializing config ... TestNodeStats_Allowlist")
	// Initialize and validate config
	config = new(Config)
	initConfig(NS_ALLOWLIST_APE_TOML, config)

	config.validateAndUpdate()

	// run the test-case logic
	nodeStats_runTestCase(t)
}

func TestNodeStats_Blocklist(t *testing.T) {
	fmt.Println("initializing config ... TestNodeStats_Blocklist")
	// Initialize and validate config
	config = new(Config)
	initConfig(NS_BLOCKLIST_APE_TOML, config)

	config.validateAndUpdate()

	// run the test-case logic
	nodeStats_runTestCase(t)

}

/**
* complete logic to call watcher, generate-mock data and asset is part of this function
 */
func nodeStats_runTestCase(t *testing.T) {

	// mock aerospike server
	mas := new(MockAerospikeServer)
	mas.initialize()

	// mock node-stats prom metric generator
	nstdg := new(MockNodestatPromMetricGenerator)

	gaugeStatHandler = new(GaugeStats)
	initGaugeStats(METRICS_CONFIG_FILE, gaugeStatHandler)
	// rawMetrics := getRawMetrics()
	rawMetrics := mas.fetchRawMetrics()

	lObserver := &Observer{}
	ch := make(chan prometheus.Metric, 10000)
	nodeStatsInfoKeys := []string{}

	watcher := new(StatsWatcher)
	watcher.passTwoKeys(rawMetrics)
	err := watcher.refresh(lObserver, nodeStatsInfoKeys, rawMetrics, ch)

	if err != nil {
		fmt.Println("watcher_node_stats_test : Unable to refresh NodeStats")
	} else {
		domore := 1

		// map of string ==> map["namespace/metric-name"]["<Label>"]
		//  both used to assert the return values from actual code against calculated values
		lOutputValues := map[string]string{}
		lOutputLabels := map[string]string{}

		arrServices := []string{}

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
				// Description: Desc{fqName: "aerospike_node_stats_fabric_rw_recv_rate", help: "fabric rw recv rate", constLabels: {}, variableLabels: [cluster_name service]}
				// Label: [name:"cluster_name" value:"null"  name:"service" value:"172.17.0.3:3000" ]
				metricNameFromDesc := extractMetricNameFromDesc(description)
				serviceFromLabel := extractLabelNameValueFromFullLabel(metricLabel, "service")
				// clusterFromLabel := extractLabelNameValueFromFullLabel(metricLabel, "cluster_name")
				// appends to the service array
				arrServices = append(arrServices, serviceFromLabel)

				// key will be like namespace/<metric_name>, this we use this check during assertion
				keyName := makeKeyname(serviceFromLabel, metricNameFromDesc, true)
				lOutputValues[keyName] = metricValue
				lOutputLabels[keyName] = metricLabel
			case <-time.After(1 * time.Second):
				domore = 0

			} // end select
		}

		// we have only 1 service in our mock-data, however loop thru service array
		for serviceIndex := range arrServices {
			serviceIp := arrServices[serviceIndex]

			lExpectedMetricNamedValues, lExpectedMetricLabels := nstdg.createNodeStatsWatcherExpectedOutputs(mas, serviceIp)

			for key := range lOutputValues {
				expectedValues := lExpectedMetricNamedValues[key]
				expectedLabels := lExpectedMetricLabels[key]
				outputMetricValues := lOutputValues[key]
				outpuMetricLabels := lOutputLabels[key]

				// assert - only if the value belongs to the service we read expected values and processing
				if strings.HasPrefix(key, serviceIp) {

					assert.Contains(t, expectedValues, outputMetricValues)
					assert.Contains(t, expectedLabels, outpuMetricLabels)
				}
			}
		}

	} // end else-refresh-failure

}
