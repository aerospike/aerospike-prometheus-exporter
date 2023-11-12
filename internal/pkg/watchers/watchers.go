package watchers

import (
	commons "github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/data"
	log "github.com/sirupsen/logrus"
)

var (
	// Node service endpoint, cluster name and build version
	Service, ClusterName, Build string
)

type WatcherMetric struct {
	Metric      commons.AerospikeStat
	Value       float64
	Labels      []string
	LabelValues []string
}

type Watcher interface {
	PassOneKeys() []string
	PassTwoKeys(rawMetrics map[string]string) []string
	// refresh( o *Observer, infoKeys []string, rawMetrics map[string]string, ch chan<- prometheus.Metric) error
	Refresh(infoKeys []string, rawMetrics map[string]string) ([]WatcherMetric, error)
}

func GetWatchers() []Watcher {
	watchers := []Watcher{
		&NamespaceWatcher{},
		&NodeStatsWatcher{},
		&SetWatcher{},
		&SindexWatcher{},
		&XdrWatcher{},
		&LatencyWatcher{},
	}

	return watchers
}

// public and utility functions

func Refresh() ([]WatcherMetric, error) {

	fullHost := commons.GetFullHost()
	log.Debugf("Refreshing node %s", fullHost)

	// array to accumulate all metrics, which later will be dispatched by various observers
	var all_metrics_to_send = []WatcherMetric{}

	// fetch first set of info keys
	var infoKeys []string
	for _, c := range GetWatchers() {
		if keys := c.PassOneKeys(); len(keys) > 0 {
			infoKeys = append(infoKeys, keys...)
		}
	}
	// append infoKey "build" - this is removed from WatcherLatencies to avoid forced watcher sequence during refresh
	infoKeys = append(infoKeys, "build")

	// info request for first set of info keys, this retrives configs from server
	//   from namespaces,server/node-stats, xdr
	//   if for any context (like jobs, latencies etc.,) no configs, they are not sent to server
	passOneOutput, err := data.GetDataProvider().RequestInfo(infoKeys)
	if err != nil {
		return nil, err
	}

	// fetch second second set of info keys
	infoKeys = []string{commons.Infokey_ClusterName, commons.Infokey_Service, commons.Infokey_Build}
	watcherInfoKeys := make([][]string, len(GetWatchers()))
	for i, c := range GetWatchers() {
		if keys := c.PassTwoKeys(passOneOutput); len(keys) > 0 {
			infoKeys = append(infoKeys, keys...)
			watcherInfoKeys[i] = keys
		}
	}

	// info request for second set of info keys, this retrieves all the stats from server
	rawMetrics, err := data.GetDataProvider().RequestInfo(infoKeys)
	if err != nil {
		return all_metrics_to_send, err
	}

	// set global values
	ClusterName, Service, Build = rawMetrics[commons.Infokey_ClusterName], rawMetrics[commons.Infokey_Service], rawMetrics[commons.Infokey_Build]

	// sanitize the utf8 strings before sending them to watchers
	for k, v := range rawMetrics {
		rawMetrics[k] = commons.SanitizeUTF8(v)
	}

	// sanitize the utf8 strings before sending them to watchers
	for i, c := range GetWatchers() {
		l_watcher_metrics, err := c.Refresh(watcherInfoKeys[i], rawMetrics)
		if err != nil {
			return all_metrics_to_send, err
		}
		all_metrics_to_send = append(all_metrics_to_send, l_watcher_metrics...)
	}

	log.Debugf("Refreshing node was successful")

	return all_metrics_to_send, nil
}
