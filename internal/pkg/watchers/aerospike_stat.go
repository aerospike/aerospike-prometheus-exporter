package watchers

import "github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"

// this is used as a prefix to qualify a metric while pushing to Prometheus or something
var PREFIX_AEROSPIKE = "aerospike_"

type AerospikeStat struct {
	// type of metric, name and allow/disallow
	Context   commons.ContextType
	Name      string
	MType     commons.MetricType
	IsAllowed bool

	// Value, Label and Label values
	Value       float64
	Labels      []string
	LabelValues []string
}

/**
 * prefixs a Context with Aerospike qualifier
 */
func (as *AerospikeStat) QualifyMetricContext() string {
	return PREFIX_AEROSPIKE + string(as.Context)
}

/*
Utility, constructs a new AerospikeStat object with required checks like is-allowed, metric-type
*/
func NewAerospikeStat(pContext commons.ContextType, pStatName string) AerospikeStat {

	isAllowed := isMetricAllowed(pContext, pStatName)
	mType := GetMetricType(pContext, pStatName)

	return AerospikeStat{pContext, pStatName, mType, isAllowed, 0.0, nil, nil}
}

func (as *AerospikeStat) updateValues(value float64, labels []string, labelValues []string) {
	as.resetValues() // resetting values, labels & label-values to nil to avoid any old values re-used/ re-shared

	as.Value = value
	as.Labels = labels
	as.LabelValues = labelValues
}

func (as *AerospikeStat) resetValues() string {
	return PREFIX_AEROSPIKE + string(as.Context)
}
