package statprocessors

var (
	// Node service endpoint, cluster name and build version
	Service, ClusterName, Build string
)

var LatencyBenchmarks map[string]float64

type StatProcessor interface {
	PassOneKeys() []string
	PassTwoKeys(rawMetrics map[string]string) []string
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
