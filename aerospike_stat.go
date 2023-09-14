package main

import "github.com/prometheus/client_golang/prometheus"

type AerospikeStat struct {
	context   ContextType
	name      string
	mType     metricType
	isAllowed bool
}

/**
 * Constructs Prometheus parameters required which are needed to push metrics to Prometheus
 */

func (as *AerospikeStat) makePromMetric(pLabels ...string) (*prometheus.Desc, prometheus.ValueType) {

	qualifiedName := as.qualifyMetricContext() + "_" + normalizeMetric(as.name)
	mergedConstLabels := as.fetchMergedConstantLabels()

	promDesc := prometheus.NewDesc(
		qualifiedName,
		normalizeDesc(as.name),
		pLabels,
		// config.AeroProm.MetricLabels,
		mergedConstLabels,
	)

	if as.mType == mtGauge {
		return promDesc, prometheus.GaugeValue
	}

	return promDesc, prometheus.CounterValue
}

/**
* utility function to combine constant labels and predefined labels (context specific ) from config file into 1 map
 */
func (as *AerospikeStat) fetchMergedConstantLabels() map[string]string {
	constLabelMap := config.AeroProm.MetricLabels
	predefLabelMap := map[string]string{}

	switch as.context {
	case CTX_XDR:
		predefLabelMap = config.AeroProm.PredefinexXdrLabels
	}

	// combine const and predefind  label map
	for key, value := range predefLabelMap {
		constLabelMap[key] = value
	}

	return constLabelMap
}

/**
 * prefixs a Context with Aerospike qualifier
 */
func (as *AerospikeStat) qualifyMetricContext() string {
	return PREFIX_AEROSPIKE + string(as.context)
}

/*
Utility, constructs a new AerospikeStat object with required checks like is-allowed, metric-type
*/
func newAerospikeStat(pContext ContextType, pStatName string) AerospikeStat {

	isAllowed := config.isMetricAllowed(pContext, pStatName)
	mType := getMetricType(pContext, pStatName)

	return AerospikeStat{pContext, pStatName, mType, isAllowed}
}
