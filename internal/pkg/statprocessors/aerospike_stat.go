package statprocessors

import (
	"strings"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
)

// this is used as a prefix to qualify a metric while pushing to Prometheus or something
var PREFIX_AEROSPIKE = "aerospike_"

type AerospikeStat struct {
	// type of metric, name and allow/disallow
	Context   commons.ContextType
	Name      string
	MType     commons.MetricType
	IsAllowed bool
	IsConfig  bool

	// Value, Label and Label values
	Value       float64
	Labels      []string
	LabelValues []string

	// ServerStatName string
}

/**
 * prefixs a Context with Aerospike qualifier
 */
func (as *AerospikeStat) QualifyMetricContext() string {
	return PREFIX_AEROSPIKE + string(as.Context)
}

/*
Utility, constructs a new AerospikeStat object with required checks like is-allowed, metric-type
we are sending both stat-name sent by server and massaged stat-name to this, so allowed-regex is applied on original-server-stat-name
very few stat-names will be message and are different from server-stat-name,
like, storage-engine.file[1].defrag_q, storage-engine.stripe[0].defrag_writes
*/
func NewAerospikeStat(pContext commons.ContextType, pStatName string, allowed bool) AerospikeStat {

	// isAllowed := isMetricAllowed(pContext, pServerStatname)
	mType := GetMetricType(pContext, pStatName)

	isConfig := strings.Contains(pStatName, "-")

	return AerospikeStat{pContext, pStatName, mType, allowed, isConfig, 0.0, nil, nil}
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
