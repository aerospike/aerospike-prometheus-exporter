package statprocessors

var (
	// Node service endpoint, cluster name and build version
	Service, ClusterName, Build string
)

type StatProcessor interface {
	PassOneKeys() []string
	PassTwoKeys(passOneStats map[string]string) []string
	Refresh(infoKeys []string, rawMetrics map[string]string) ([]AerospikeStat, error)
}

// Struct to store shared state and values between various processors in a thread-safe manner
type StatProcessorSharedState struct {
	ServiceLatencyBenchmarks   map[string]string
	NamespaceLatencyBenchmarks map[string]map[string]string
}

func NewStatProcessorSharedState() *StatProcessorSharedState {
	return &StatProcessorSharedState{
		ServiceLatencyBenchmarks:   make(map[string]string),
		NamespaceLatencyBenchmarks: make(map[string]map[string]string),
	}
}
