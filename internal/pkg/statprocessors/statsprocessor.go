package statprocessors

type StatProcessor interface {
	PassOneKeys() []string
	PassTwoKeys(passOneStats map[string]string) []string
	Refresh(infoKeys []string, rawMetrics map[string]string) ([]AerospikeStat, error)
}

// Struct to store shared state and values between various processors
type StatProcessorSharedState struct {
	Service, ClusterName, Build, NodeId string

	Infokey_ClusterName string
	Infokey_Service     string
	Infokey_Build       string
	Infokey_NodeId      string

	ServiceLatencyBenchmarks   map[string]string
	NamespaceLatencyBenchmarks map[string]map[string]string
}

func NewStatProcessorSharedState() *StatProcessorSharedState {

	sharedState := &StatProcessorSharedState{
		Infokey_Build:       "build",
		Infokey_ClusterName: "cluster-name",
		Infokey_NodeId:      "node:",

		// this value will be set depending on the server mode (tls or clear)
		//   modified in the first Refresh call
		Infokey_Service: INFOKEY_SERVICE_CLEAR_STD,

		ServiceLatencyBenchmarks:   make(map[string]string),
		NamespaceLatencyBenchmarks: make(map[string]map[string]string),
	}

	return sharedState
}
