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
	CTX_SETS       ContextType = "sets"
	CTX_SINDEX     ContextType = "sindex"
	CTX_XDR        ContextType = "xdr"
	CTX_LATENCIES  ContextType = "latencies"
)

// below constant represent the labels we send along with metrics to Prometheus or something
const METRIC_LABEL_CLUSTER_NAME = "cluster_name"
const METRIC_LABEL_SERVICE = "service"
const METRIC_LABEL_NS = "ns"
const METRIC_LABEL_SET = "set"
const METRIC_LABEL_LE = "le"
const METRIC_LABEL_DC_NAME = "dc"
const METRIC_LABEL_SINDEX = "sindex"

/**
 * Constructs Prometheus parameters required which are needed to push metrics to Prometheus
 */

func (as *AerospikeStat) makePromeMetric(pLabels ...string) (*prometheus.Desc, prometheus.ValueType) {

	qualifiedName := as.qualifyMetricContext() + "_" + normalizeMetric(as.name)
	promDesc := prometheus.NewDesc(
		qualifiedName,
		normalizeDesc(as.name),
		pLabels,
		config.AeroProm.MetricLabels,
	)

	if as.mType == mtGauge {
		return promDesc, prometheus.GaugeValue
	}

	return promDesc, prometheus.CounterValue
}

/**
 * prefixs a Context with Aerospike qualifier
 */
func (as *AerospikeStat) qualifyMetricContext() string {
	return PREFIX_AEROSPIKE + string(as.context)
}

/*
*
* Utility, constructs a new AerospikeStat object with required checks like is-allowed, metric-type
 */
func newAerospikeStat(pContext ContextType, pStatName string) AerospikeStat {

	isAllowed := config.isMetricAllowed(pContext, pStatName)
	mType := getMetricType(pContext, pStatName)

	return AerospikeStat{pContext, pStatName, mType, isAllowed}
}
