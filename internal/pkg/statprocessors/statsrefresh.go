package statprocessors

import (
	aero "github.com/aerospike/aerospike-client-go/v8"
	commons "github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/dataprovider"
	log "github.com/sirupsen/logrus"
)

// StatsRefresher is the main struct that refreshes the stats from the server
// responsible for refreshing the stats from the server and dispatching them to the appropriate processors
// uses shared state to store the latency benchmarks and namespace latency benchmarks
type StatsRefresher struct {
	dataProvider dataprovider.DataProvider
	sharedState  *StatProcessorSharedState

	namespaceStatsProcessor *NamespaceStatsProcessor
	nodeStatsProcessor      *NodeStatsProcessor
	setsStatsProcessor      *SetsStatsProcessor
	sindexStatsProcessor    *SindexStatsProcessor
	xdrStatsProcessor       *XdrStatsProcessor
	latencyStatsProcessor   *LatencyStatsProcessor

	userStatsProcessor *UserStatsProcessor
}

func NewStatsRefresher(dataProvider dataprovider.DataProvider,
	sharedState *StatProcessorSharedState) *StatsRefresher {

	log.Debugf("Creating new StatsRefresher with dataProvider: %p", dataProvider)

	statsRefresher := &StatsRefresher{}

	statsRefresher.dataProvider = dataProvider
	statsRefresher.sharedState = sharedState

	statsRefresher.namespaceStatsProcessor = NewNamespaceStatsProcessor(statsRefresher.sharedState)
	statsRefresher.nodeStatsProcessor = NewNodeStatsProcessor(statsRefresher.sharedState)
	statsRefresher.setsStatsProcessor = NewSetsStatsProcessor(statsRefresher.sharedState)
	statsRefresher.sindexStatsProcessor = NewSindexStatsProcessor(statsRefresher.sharedState)
	statsRefresher.xdrStatsProcessor = NewXdrStatsProcessor(statsRefresher.sharedState)
	statsRefresher.latencyStatsProcessor = NewLatencyStatsProcessor(statsRefresher.sharedState)
	statsRefresher.userStatsProcessor = NewUserStatsProcessor(statsRefresher.sharedState)

	return statsRefresher
}

func (sr *StatsRefresher) GetStatsProcessors() []StatProcessor {
	var statprocessors = []StatProcessor{
		sr.namespaceStatsProcessor,
		sr.nodeStatsProcessor,
		sr.setsStatsProcessor,
		sr.sindexStatsProcessor,
		sr.xdrStatsProcessor,
		sr.latencyStatsProcessor,
		// Did not include users processor, as it process UserRoles directly and not the stats from server
	}

	return statprocessors
}

func (sr *StatsRefresher) Refresh() ([]AerospikeStat, error) {

	fullHost := commons.GetFullHost()
	log.Debugf("Refreshing node %s", fullHost)

	// array to accumulate all metrics, which later will be dispatched by various observers
	var allStatsToSend = []AerospikeStat{}

	// list of all the StatsProcessor
	allStatsprocessorList := sr.GetStatsProcessors()

	// fetch first set of info keys
	var infoKeys []string
	for _, c := range allStatsprocessorList {
		if keys := c.PassOneKeys(); len(keys) > 0 {
			infoKeys = append(infoKeys, keys...)
		}
	}

	// append infoKey "build" - this is removed from LatenciesStatsProcessor to avoid forced StatsProcessor sequence during refresh
	infoKeys = append(infoKeys, "build")

	// info request for first set of info keys, this retrives configs from server
	//   from namespaces,server/node-stats, xdr
	//   if for any context (like jobs, latencies etc.,) no configs, they are not sent to server
	passOneOutput, err := sr.dataProvider.RequestInfo(infoKeys)

	if err != nil {
		return nil, err
	}

	// fetch second second set of info keys
	// check and load this only once, to avoid multiple file-reads, so this Infokey assignment will happen only once during restart
	if sr.sharedState.Infokey_Service != INFOKEY_SERVICE_TLS_STD {
		serverPool, clientPool := commons.LoadServerOrClientCertificates()
		// we need to have atleast one certificate configured and read successfully
		if serverPool != nil || clientPool != nil {
			sr.sharedState.Infokey_Service = INFOKEY_SERVICE_TLS_STD
			log.Debugf("TLS Mode is enabled, setting infokey-service as  'service-tls-std' for further fetching from server.")
		}
	}

	infoKeys = []string{sr.sharedState.Infokey_ClusterName, sr.sharedState.Infokey_Service, sr.sharedState.Infokey_Build}
	statprocessorInfoKeys := make([][]string, len(allStatsprocessorList))

	for i, c := range allStatsprocessorList {

		if keys := c.PassTwoKeys(passOneOutput); len(keys) > 0 {
			infoKeys = append(infoKeys, keys...)
			statprocessorInfoKeys[i] = keys
		}
	}

	// info request for second set of info keys, this retrieves all the stats from server
	passTwoResponse, err := sr.dataProvider.RequestInfo(infoKeys)

	if err != nil {
		return allStatsToSend, err
	}

	// set global values
	sr.sharedState.ClusterName = passTwoResponse[sr.sharedState.Infokey_ClusterName]
	sr.sharedState.Service = passTwoResponse[sr.sharedState.Infokey_Service]
	sr.sharedState.Build = passTwoResponse[sr.sharedState.Infokey_Build]

	// Servce is IP of Aerospike Server, in Kubernetes we need pod-name instead of IP.
	if config.Cfg.Agent.IsKubernetes {
		sr.sharedState.Service = config.Cfg.Agent.KubernetesPodName
	}

	// sanitize the utf8 strings before sending them to watchers
	for k, v := range passTwoResponse {
		passTwoResponse[k] = commons.SanitizeUTF8(v)
	}

	// sanitize the utf8 strings before sending them to watchers
	for i, c := range allStatsprocessorList {

		tmpRefreshedMetrics, err := c.Refresh(statprocessorInfoKeys[i], passTwoResponse)

		if err != nil {
			return allStatsToSend, err
		}

		allStatsToSend = append(allStatsToSend, tmpRefreshedMetrics...)
	}

	// Refresh user info if supported by the server
	if sr.userStatsProcessor.canRefreshUserStats(passTwoResponse) {
		userMetrics, err := sr.RefreshUserStats()

		if err != nil {
			return allStatsToSend, err
		}

		allStatsToSend = append(allStatsToSend, userMetrics...)
	}

	// Get User metrics
	log.Debugf("Refreshing node was successful")

	return allStatsToSend, nil
}

// User stats are not different stats, metrics are creating using the user info
// user-role info are normalized as aerospike-stats
func (sr *StatsRefresher) RefreshUserStats() ([]AerospikeStat, error) {

	var err error
	var users []*aero.UserRoles

	sr.userStatsProcessor.ShouldFetchUserStatistics, users, err = sr.dataProvider.FetchUsersDetails()

	if err != nil {
		return nil, err
	}

	userMetrics, err := sr.userStatsProcessor.Refresh(users)

	if err != nil {
		return nil, err
	}

	return userMetrics, nil
}
