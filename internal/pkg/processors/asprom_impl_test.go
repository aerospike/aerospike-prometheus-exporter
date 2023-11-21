package processors

import (
	"fmt"
	"testing"
	"time"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/data"
	tests_utils "github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/tests_utils"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/watchers"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func Test_RefreshDefault(t *testing.T) {

	fmt.Println("initializing config ... Test_RefreshDefault")

	// initialize config and gauge-lists
	config.InitConfig(tests_utils.GetConfigfileLocation(tests_utils.TESTS_DEFAULT_CONFIG_FILE))

	asMetrics := get_Namespace_Metrics()
	asMetrics = append(asMetrics, get_Node_Metrics()...)
	asMetrics = append(asMetrics, get_Sets_Metrics()...)
	asMetrics = append(asMetrics, get_Sindex_Metrics()...)
	asMetrics = append(asMetrics, get_Users_Metrics()...)
	asMetrics = append(asMetrics, get_Latency_Metrics()...)
	asMetrics = append(asMetrics, get_Xdr_Metrics()...)

	// generate and validate labels
	all_runTestcase(t, asMetrics)
}

/**
* complete logic to call watcher, generate-mock data and asset is part of this function
 */
func all_runTestcase(t *testing.T, asMetrics []watchers.AerospikeStat) {
	ch := make(chan prometheus.Metric, 50000)
	for idx_stats := range asMetrics {
		as := asMetrics[idx_stats]
		PushToPrometheus(as, ch)
	}

	// read from channel and check each metric created
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

			fmt.Println("** description: ", description, "\n\t metricValue: ", metricValue, "\t metricLabel: ", metricLabel)

		case <-time.After(1 * time.Second):
			domore = 0

		} // end select
	}

}

// Data fetch helpers functions
func get_Node_Metrics() []watchers.AerospikeStat {

	// Node
	nodeWatcher := &watchers.NodeStatsWatcher{}
	nwPassOneKeys := nodeWatcher.PassOneKeys()
	passOneOutput, _ := data.GetProvider().RequestInfo(nwPassOneKeys)
	fmt.Println("TestPassTwoKeys: passOneOutput: ", passOneOutput)
	passTwoOutputs := nodeWatcher.PassTwoKeys(passOneOutput)

	// append common keys
	infoKeys := []string{watchers.Infokey_ClusterName, watchers.Infokey_Service, watchers.Infokey_Build}
	passTwoOutputs = append(passTwoOutputs, infoKeys...)

	arrRawMetrics, _ := data.GetProvider().RequestInfo(passTwoOutputs)
	// check the output with NodeStatsWatcher
	nodeMetrics, err := nodeWatcher.Refresh(passTwoOutputs, arrRawMetrics)

	if err != nil {
		return nil
	}

	return nodeMetrics
}

func get_Namespace_Metrics() []watchers.AerospikeStat {
	// initialize gauges list
	config.InitGaugeStats(tests_utils.GetConfigfileLocation(tests_utils.DEFAULT_GAUGE_LIST_FILE))

	// rawMetrics := getRawMetrics()
	nsWatcher := &watchers.NamespaceWatcher{}

	// simulate, as if we are sending requestInfo to AS and get the namespaces, these are coming from mock-data-generator
	passOneKeys := nsWatcher.PassOneKeys()
	passOneOutput, _ := data.GetProvider().RequestInfo(passOneKeys)
	passTwokeyOutputs := nsWatcher.PassTwoKeys(passOneOutput)

	// append common keys
	infoKeys := []string{watchers.Infokey_ClusterName, watchers.Infokey_Service, watchers.Infokey_Build}
	passTwokeyOutputs = append(passTwokeyOutputs, infoKeys...)

	arrRawMetrics, err := data.GetProvider().RequestInfo(passTwokeyOutputs)
	if err != nil {
		return nil
	}

	// check the output with NamespaceWatcher
	nsMetrics, err := nsWatcher.Refresh(passTwokeyOutputs, arrRawMetrics)
	if err != nil {
		return nil
	}

	return nsMetrics
}

func get_Sets_Metrics() []watchers.AerospikeStat {
	// Check passoneKeys
	setsWatcher := &watchers.SetWatcher{}
	nwPassOneKeys := setsWatcher.PassOneKeys()
	passOneOutput, _ := data.GetProvider().RequestInfo(nwPassOneKeys)
	fmt.Println("TestPassTwoKeys: passOneOutput: ", passOneOutput)
	passTwoOutputs := setsWatcher.PassTwoKeys(passOneOutput)

	// append common keys
	infoKeys := []string{watchers.Infokey_ClusterName, watchers.Infokey_Service, watchers.Infokey_Build}
	passTwoOutputs = append(passTwoOutputs, infoKeys...)
	arrRawMetrics, err := data.GetProvider().RequestInfo(passTwoOutputs)
	if err != nil {
		return nil
	}

	// check the output with setsWatcher
	setsMetrics, err := setsWatcher.Refresh(passTwoOutputs, arrRawMetrics)

	if err != nil {
		return nil
	}

	return setsMetrics
}

func get_Sindex_Metrics() []watchers.AerospikeStat {
	// Check passoneKeys
	infoKeys := []string{watchers.Infokey_ClusterName, watchers.Infokey_Service, watchers.Infokey_Build}

	sindexWatcher := &watchers.SindexWatcher{}
	nwPassOneKeys := sindexWatcher.PassOneKeys()
	passOneOutput, _ := data.GetProvider().RequestInfo(nwPassOneKeys)
	fmt.Println("sindex_runTestcase: passOneOutput: ", passOneOutput)
	passTwoOutputs := sindexWatcher.PassTwoKeys(passOneOutput)

	// append common keys
	passTwoOutputs = append(passTwoOutputs, infoKeys...)

	arrRawMetrics, err := data.GetProvider().RequestInfo(passTwoOutputs)
	if err != nil {
		return nil
	}
	// check the output with sindexWatcher
	sindexMetrics, err := sindexWatcher.Refresh(passTwoOutputs, arrRawMetrics)

	if err != nil {
		return nil
	}

	return sindexMetrics
}

func get_Users_Metrics() []watchers.AerospikeStat {
	// Check passoneKeys
	infoKeys := []string{watchers.Infokey_ClusterName, watchers.Infokey_Service, watchers.Infokey_Build}

	usersWatcher := &watchers.UserWatcher{}
	nwPassOneKeys := usersWatcher.PassOneKeys()
	passOneOutput, _ := data.GetProvider().RequestInfo(nwPassOneKeys)
	fmt.Println("users_runTestcase: passOneOutput: ", passOneOutput)
	passTwoOutputs := usersWatcher.PassTwoKeys(passOneOutput)

	// append common keys
	passTwoOutputs = append(passTwoOutputs, infoKeys...)

	arrRawMetrics, err := data.GetProvider().RequestInfo(passTwoOutputs)

	// check the output with usersWatcher
	usersMetrics, err := usersWatcher.Refresh(passTwoOutputs, arrRawMetrics)
	if err != nil {
		return nil
	}

	if err != nil {
		return nil
	}

	return usersMetrics
}

func get_Latency_Metrics() []watchers.AerospikeStat {
	// Check passoneKeys
	infoKeys := []string{watchers.Infokey_ClusterName, watchers.Infokey_Service, watchers.Infokey_Build}

	latencyWatcher := &watchers.LatencyWatcher{}
	nwPassOneKeys := latencyWatcher.PassOneKeys()
	passOneOutput, _ := data.GetProvider().RequestInfo(nwPassOneKeys)
	fmt.Println("TestPassTwoKeys: passOneOutput: ", passOneOutput)
	passTwoOutputs := latencyWatcher.PassTwoKeys(passOneOutput)

	// append common keys
	passTwoOutputs = append(passTwoOutputs, infoKeys...)

	arrRawMetrics, err := data.GetProvider().RequestInfo(passTwoOutputs)
	if err != nil {
		return nil
	}
	// check the output with setsWatcher
	latencyMetrics, err := latencyWatcher.Refresh(passTwoOutputs, arrRawMetrics)

	if err != nil {
		return nil
	}

	return latencyMetrics
}

func get_Xdr_Metrics() []watchers.AerospikeStat {
	// Check passoneKeys
	xdrWatcher := &watchers.XdrWatcher{}
	xdrPassOneKeys := xdrWatcher.PassOneKeys()
	// append common keys
	infoKeys := []string{watchers.Infokey_ClusterName, watchers.Infokey_Service, watchers.Infokey_Build, "namespaces"}
	xdrPassOneKeys = append(xdrPassOneKeys, infoKeys...)

	passOneOutput, _ := data.GetProvider().RequestInfo(xdrPassOneKeys)
	passTwoOutputs := xdrWatcher.PassTwoKeys(passOneOutput)

	passTwoOutputs = append(passTwoOutputs, infoKeys...)
	arrRawMetrics, err := data.GetProvider().RequestInfo(passTwoOutputs)

	if err != nil {
		return nil
	}

	// check the output with NodeStatsWatcher
	xdrMetrics, err := xdrWatcher.Refresh(passTwoOutputs, arrRawMetrics)

	if err != nil {
		return nil
	}

	return xdrMetrics
}
