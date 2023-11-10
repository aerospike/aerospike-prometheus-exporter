package commons

type AerospikeStat struct {
	Context   ContextType
	Name      string
	MType     metricType
	IsAllowed bool
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
func NewAerospikeStat(pContext ContextType, pStatName string) AerospikeStat {

	isAllowed := IsMetricAllowed(pContext, pStatName)
	mType := GetMetricType(pContext, pStatName)

	return AerospikeStat{pContext, pStatName, mType, isAllowed}
}
