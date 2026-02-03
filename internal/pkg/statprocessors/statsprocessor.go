package statprocessors

var (
	// Node service endpoint, cluster name and build version
	Service, ClusterName, Build string
)

var ServiceLatencyBenchmarks = make(map[string]string)
var NamespaceLatencyBenchmarks = make(map[string]map[string]string)

type StatProcessor interface {
	PassOneKeys() []string
	PassTwoKeys(passOneStats map[string]string) []string
	Refresh(infoKeys []string, rawMetrics map[string]string) ([]AerospikeStat, error)
}

// stat-processors are created only once per process
var statprocessors = []StatProcessor{
	&NamespaceStatsProcessor{},
	&NodeStatsProcessor{},
	&SetsStatsProcessor{},
	&SindexStatsProcessor{},
	&XdrStatsProcessor{},
	&LatencyStatsProcessor{},
	&UserStatsProcessor{},
}

func GetStatsProcessors() []StatProcessor {
	return statprocessors
}
