package statprocessors

import (
	commons "github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/dataprovider"
	log "github.com/sirupsen/logrus"
)

// public and utility functions

func Refresh() ([]AerospikeStat, error) {

	// Reset global variables
	LatencyBenchmarks = make(map[string]float64)

	fullHost := commons.GetFullHost()
	log.Debugf("Refreshing node %s", fullHost)

	// array to accumulate all metrics, which later will be dispatched by various observers
	var all_metrics_to_send = []AerospikeStat{}

	// list of all the StatsProcessor
	all_statsprocessor_list := GetStatsProcessors()

	// fetch first set of info keys
	var infoKeys []string
	for _, c := range all_statsprocessor_list {
		if keys := c.PassOneKeys(); len(keys) > 0 {
			infoKeys = append(infoKeys, keys...)
		}
	}
	// append infoKey "build" - this is removed from LatenciesStatsProcessor to avoid forced StatsProcessor sequence during refresh
	infoKeys = append(infoKeys, "build")

	// info request for first set of info keys, this retrives configs from server
	//   from namespaces,server/node-stats, xdr
	//   if for any context (like jobs, latencies etc.,) no configs, they are not sent to server
	passOneOutput, err := dataprovider.GetProvider().RequestInfo(infoKeys)
	if err != nil {
		return nil, err
	}

	// fetch second second set of info keys
	// check and load this only once, to avoid multiple file-reads, so this Infokey assignment will happen only once during restart
	if Infokey_Service != INFOKEY_SERVICE_TLS_STD {
		serverPool, clientPool := commons.LoadServerOrClientCertificates()
		// we need to have atleast one certificate configured and read successfully
		if serverPool != nil && clientPool != nil {
			Infokey_Service = INFOKEY_SERVICE_TLS_STD
			log.Debugf("TLS Mode is enabled, setting infokey-service as  'service-tls-std' for further fetching from server.")
		}
	}

	infoKeys = []string{Infokey_ClusterName, Infokey_Service, Infokey_Build}
	statprocessorInfoKeys := make([][]string, len(all_statsprocessor_list))
	for i, c := range all_statsprocessor_list {
		if keys := c.PassTwoKeys(passOneOutput); len(keys) > 0 {
			infoKeys = append(infoKeys, keys...)
			// fmt.Println("\nkeys: ", keys)
			statprocessorInfoKeys[i] = keys
		}
	}

	// fmt.Println("\n-----------------\nwatcherInfoKeys: ", watcherInfoKeys)

	// info request for second set of info keys, this retrieves all the stats from server
	rawMetrics, err := dataprovider.GetProvider().RequestInfo(infoKeys)
	if err != nil {
		return all_metrics_to_send, err
	}

	// set global values
	ClusterName, Service, Build = rawMetrics[Infokey_ClusterName], rawMetrics[Infokey_Service], rawMetrics[Infokey_Build]

	// sanitize the utf8 strings before sending them to watchers
	for k, v := range rawMetrics {
		rawMetrics[k] = commons.SanitizeUTF8(v)
	}

	// sanitize the utf8 strings before sending them to watchers
	for i, c := range all_statsprocessor_list {
		// fmt.Println("\nSending... ", watcherInfoKeys[i], " keys to each Refresh ...")
		l_refreshed_metrics, err := c.Refresh(statprocessorInfoKeys[i], rawMetrics)
		if err != nil {
			return all_metrics_to_send, err
		}
		all_metrics_to_send = append(all_metrics_to_send, l_refreshed_metrics...)
	}

	log.Debugf("Refreshing node was successful")

	return all_metrics_to_send, nil
}
