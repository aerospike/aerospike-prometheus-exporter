package main

import "github.com/prometheus/client_golang/prometheus"

type AerospikeStat struct {
	context   ContextType
	name      string
	mType     metricType
	isAllowed bool
}

// this is used as a prefix to qualify a metric while pushing to Prometheus or something
var PREFIX_AEROSPIKE = "aerospike_"

// used to define context of stat types (like namespace, set, xdr etc.,)
type ContextType string

const (
	CTX_USERS      ContextType = "users"
	CTX_NAMESPACE  ContextType = "namespace"
	CTX_NODE_STATS ContextType = "node_stats"
	CTX_SETS       ContextType = "node_stats"
	CTX_SINDEX     ContextType = "sindex"
	CTX_XDR        ContextType = "xdr"
	CTX_LATENCIES  ContextType = "latencies"
)

// below constant represent the labels we send along with metrics to Prometheus or something
const METRIC_LABEL_CLUSTER_NAME = "cluster_name"
const METRIC_LABEL_SERVICE = "service"
const METRIC_LABEL_NS = "ns"

/**
 * Constructs Prometheus parameters required which are needed to push metrics to Prometheus
 */

func (as *AerospikeStat) makePromeMetric(pContext ContextType, pName string, pMetricType metricType, pConstLabels map[string]string, pLabels ...string) (string, *prometheus.Desc, prometheus.ValueType) {
	qualifiedName := as.qualifyMetricContext(pContext) + "_" + normalizeMetric(pName)
	promDesc := prometheus.NewDesc(
		qualifiedName,
		normalizeDesc(pName),
		pLabels,
		pConstLabels,
	)

	if pMetricType == mtGauge {
		return qualifiedName, promDesc, prometheus.GaugeValue
	}

	return qualifiedName, promDesc, prometheus.CounterValue
}

/**
 * prefixs a Context with Aerospike qualifier
 */
func (as *AerospikeStat) qualifyMetricContext(pContext ContextType) string {
	return PREFIX_AEROSPIKE + string(pContext)
}

/*
*
* Utility, constructs a new AerospikeStat object with required checks like is-allowed, metric-type
TODO: move to common place
*/
func newAerospikeStat(pContext ContextType, pStatName string, pAllowList []string, pBlockList []string) AerospikeStat {

	isAllowed := isMetricAllowed(pStatName, pAllowList, pBlockList)
	mType := getMetricType(pContext, pStatName)
	nsMetric := AerospikeStat{pContext, pStatName, mType, isAllowed}

	return nsMetric
}
