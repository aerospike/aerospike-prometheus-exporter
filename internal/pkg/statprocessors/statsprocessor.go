package statprocessors

type StatProcessor interface {
	PassOneKeys() []string
	PassTwoKeys(passOneStats map[string]string) []string
	Refresh(infoKeys []string, rawMetrics map[string]string) ([]AerospikeStat, error)
}

// Struct to store shared state and values between various processors
type StatProcessorSharedState struct {
	Service, ClusterName, Build, NodeId string

	Infokey_ClusterName      string
	Infokey_Service          string
	Infokey_Build            string
	Infokey_NodeId           string
	Infokey_CheckpointStatus string

	ServiceLatencyBenchmarks   map[string]string
	NamespaceLatencyBenchmarks map[string]map[string]string

	// set when server reports checkpoint-save in progress, only build and checkpoint-status are requested until it clears
	CheckpointSaveInProgress bool

	// PartialRefresh is set for the whole of a Refresh once either of its two passes saw a checkpoint save.
	// The first pass decides which commands the second one asks for, so a save that ends between the passes
	// still leaves a refresh that is missing most of its stats, although CheckpointSaveInProgress is false again.
	PartialRefresh bool

	// IndexPressureFetched is set when a Refresh ran the index-pressure command and got a valid answer.
	// Index-pressure is fetched only every few minutes, so only such a refresh says which namespaces are gone.
	IndexPressureFetched bool
}

// IsPartialRefresh reports whether the latest refresh was cut short by a checkpoint save, so that its
// result is missing stats that the server still has.
func (s *StatProcessorSharedState) IsPartialRefresh() bool {
	return s != nil && (s.CheckpointSaveInProgress || s.PartialRefresh)
}

func NewStatProcessorSharedState() *StatProcessorSharedState {

	sharedState := &StatProcessorSharedState{
		Infokey_Build:            "build",
		Infokey_ClusterName:      "cluster-name",
		Infokey_NodeId:           "node:",
		Infokey_CheckpointStatus: "checkpoint-status",

		// this value will be set depending on the server mode (tls or clear)
		//   modified in the first Refresh call
		Infokey_Service: INFOKEY_SERVICE_CLEAR_STD,

		ServiceLatencyBenchmarks:   make(map[string]string),
		NamespaceLatencyBenchmarks: make(map[string]map[string]string),
		CheckpointSaveInProgress:   false,
	}

	return sharedState
}
