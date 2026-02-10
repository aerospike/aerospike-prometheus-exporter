package statprocessors

type StatProcessor interface {
	PassOneKeys() []string
	PassTwoKeys(passOneStats map[string]string) []string
	Refresh(infoKeys []string, rawMetrics map[string]string) ([]AerospikeStat, error)
}

// Struct to store shared state and values between various processors
type StatProcessorSharedState struct {
	Service, ClusterName, Build string

	Infokey_ClusterName string
	Infokey_Service     string
	Infokey_Build       string

	ServiceLatencyBenchmarks   map[string]string
	NamespaceLatencyBenchmarks map[string]map[string]string
}

func NewStatProcessorSharedState() *StatProcessorSharedState {

	sharedState := &StatProcessorSharedState{
		Infokey_ClusterName: "cluster-name",
		Infokey_Service:     INFOKEY_SERVICE_CLEAR_STD,
		Infokey_Build:       "build",
	}

	sharedState.ServiceLatencyBenchmarks = make(map[string]string)
	sharedState.NamespaceLatencyBenchmarks = make(map[string]map[string]string)

	return sharedState
}
