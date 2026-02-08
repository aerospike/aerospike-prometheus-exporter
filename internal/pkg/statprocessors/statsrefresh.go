package statprocessors

import (
	aero "github.com/aerospike/aerospike-client-go/v8"
	commons "github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/dataprovider"
	log "github.com/sirupsen/logrus"
)

type StatsRefresher struct {
	dataProvider dataprovider.DataProvider
	ExecutorMode string

	namespaceStatsProcessor *NamespaceStatsProcessor
	nodeStatsProcessor      *NodeStatsProcessor
	setsStatsProcessor      *SetsStatsProcessor
	sindexStatsProcessor    *SindexStatsProcessor
	xdrStatsProcessor       *XdrStatsProcessor
	latencyStatsProcessor   *LatencyStatsProcessor

	userStatsProcessor *UserStatsProcessor
}

func NewStatsRefresher(executorMode string) *StatsRefresher {

	return &StatsRefresher{
		dataProvider:            dataprovider.GetProvider(executorMode),
		ExecutorMode:            executorMode,
		namespaceStatsProcessor: &NamespaceStatsProcessor{},
		nodeStatsProcessor:      &NodeStatsProcessor{},
		setsStatsProcessor:      &SetsStatsProcessor{},
		sindexStatsProcessor:    &SindexStatsProcessor{},
		xdrStatsProcessor:       &XdrStatsProcessor{},
		latencyStatsProcessor:   &LatencyStatsProcessor{},
		userStatsProcessor:      &UserStatsProcessor{ShouldFetchUserStatistics: true},
	}
}

func (sr *StatsRefresher) GetStatsProcessors() []StatProcessor {
	var statprocessors = []StatProcessor{
		sr.namespaceStatsProcessor,
		sr.nodeStatsProcessor,
		sr.setsStatsProcessor,
		sr.sindexStatsProcessor,
		sr.xdrStatsProcessor,
		sr.latencyStatsProcessor,
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
	if Infokey_Service != INFOKEY_SERVICE_TLS_STD {
		serverPool, clientPool := commons.LoadServerOrClientCertificates()
		// we need to have atleast one certificate configured and read successfully
		if serverPool != nil || clientPool != nil {
			Infokey_Service = INFOKEY_SERVICE_TLS_STD
			log.Debugf("TLS Mode is enabled, setting infokey-service as  'service-tls-std' for further fetching from server.")
		}
	}

	infoKeys = []string{Infokey_ClusterName, Infokey_Service, Infokey_Build}
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
	ClusterName, Service, Build = passTwoResponse[Infokey_ClusterName], passTwoResponse[Infokey_Service], passTwoResponse[Infokey_Build]
	if config.Cfg.Agent.IsKubernetes {
		Service = config.Cfg.Agent.KubernetesPodName
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
