package statprocessors

import (
	commons "github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/dataprovider"

	aero "github.com/aerospike/aerospike-client-go/v8"
	log "github.com/sirupsen/logrus"
)

var (
	shouldFetchUserStatistics bool = true
)

type UserStatsProcessor struct{}

func (uw *UserStatsProcessor) PassOneKeys() []string {
	// "build" info key should be returned here,
	// but it is also being sent by LatencyStatsProcessor.passOneKeys(),
	// hence skipping here.
	log.Tracef("users-passonekeys:nil")
	return nil
}

func (uw *UserStatsProcessor) PassTwoKeys(rawMetrics map[string]string) []string {
	log.Tracef("users-passtwokeys:nil")
	return nil
}

func (uw *UserStatsProcessor) Refresh(infoKeys []string, rawMetrics map[string]string) ([]AerospikeStat, error) {

	// check if security configurations are enabled
	if config.Cfg.Aerospike.AuthMode != "pki" &&
		(config.Cfg.Aerospike.User == "" && config.Cfg.Aerospike.Password == "") {
		return nil, nil
	}

	// check if we should fetch user metrics
	if !shouldFetchUserStatistics {
		log.Debug("Fetching user statistics is disabled")
		return nil, nil
	}

	// validate aerospike build version
	// support for user statistics is added in aerospike 5.6
	var err error
	ok, err := BuildVersionGreaterThanOrEqual(rawMetrics, "5.6.0.0")
	if err != nil {
		// just log warning. don't send an error back
		log.Warn(err)
		return nil, nil
	}

	if !ok {
		// disable user statisitcs if build version is not >= 5.6.0.0
		log.Debug("Aerospike version doesn't support user statistics")
		shouldFetchUserStatistics = false
		return nil, nil
	}

	// read the data from Aerospike Server
	var users []*aero.UserRoles

	shouldFetchUserStatistics, users, err = dataprovider.GetProvider().FetchUsersDetails()
	if err != nil {
		log.Warn("Error while fetching user statistics: ", err)
		return nil, nil

	}

	var allMetricsToSend = []AerospikeStat{}
	// Push metrics to Prometheus or Observability tool
	lUserMetricsToSend, err := uw.refreshUserStats(infoKeys, rawMetrics, users)
	if err != nil {
		log.Warn("Error while preparing and pushing metrics: ", err)
		return nil, nil

	}

	allMetricsToSend = append(allMetricsToSend, lUserMetricsToSend...)

	return allMetricsToSend, err
}

func (uw *UserStatsProcessor) refreshUserStats(infoKeys []string, rawMetrics map[string]string, users []*aero.UserRoles) ([]AerospikeStat, error) {
	allowedUsersList := make(map[string]struct{})
	blockedUsersList := make(map[string]struct{})

	// let us not cache the user-info (like user-role, any permisison etc.,), as this can be changed at server-level without any restarts
	//
	if config.Cfg.Aerospike.UserMetricsUsersAllowlistEnabled {
		for _, allowedUser := range config.Cfg.Aerospike.UserMetricsUsersAllowlist {
			allowedUsersList[allowedUser] = struct{}{}
		}
	}

	if len(config.Cfg.Aerospike.UserMetricsUsersBlocklist) > 0 {
		for _, blockedUser := range config.Cfg.Aerospike.UserMetricsUsersBlocklist {
			blockedUsersList[blockedUser] = struct{}{}
		}
	}

	var allMetricsToSend = []AerospikeStat{}

	for _, user := range users {
		// check if user is allowed
		if config.Cfg.Aerospike.UserMetricsUsersAllowlistEnabled {
			if _, ok := allowedUsersList[user.User]; !ok {
				continue
			}
		}

		// check if user is blocked
		if len(config.Cfg.Aerospike.UserMetricsUsersBlocklist) > 0 {
			if _, ok := blockedUsersList[user.User]; ok {
				continue
			}
		}

		// Order is important as user.ReadInfo returns an int-array - where
		//   0 = read-quota, 1=read_single_record_tps etc., -- this order is fixed from server
		readInfoStats := []string{"read_quota", "read_single_record_tps", "read_scan_query_rps", "limitless_read_scan_query"}
		writeInfoStats := []string{"write_quota", "write_single_record_tps", "write_scan_query_rps", "limitless_write_scan_query"}

		asMetric, labels, labelValues := internalCreateLocalAerospikeStat("conns_in_use", user.User)
		asMetric.updateValues(float64(user.ConnsInUse), labels, labelValues)
		allMetricsToSend = append(allMetricsToSend, asMetric)

		if len(user.ReadInfo) >= 4 && len(user.WriteInfo) >= 4 {

			for idxReadinfo := 0; idxReadinfo < len(user.ReadInfo); idxReadinfo++ {
				riAeroMetric, riLabels, riLabelValues := internalCreateLocalAerospikeStat(readInfoStats[idxReadinfo], user.User)
				riAeroMetric.updateValues(float64(user.ReadInfo[idxReadinfo]), riLabels, riLabelValues)

				allMetricsToSend = append(allMetricsToSend, riAeroMetric)

			}
			for idxWriteinfo := 0; idxWriteinfo < len(user.WriteInfo); idxWriteinfo++ {
				wiAeroMetric, wiLabels, wiLabelValues := internalCreateLocalAerospikeStat(writeInfoStats[idxWriteinfo], user.User)
				wiAeroMetric.updateValues(float64(user.WriteInfo[idxWriteinfo]), wiLabels, wiLabelValues)
				allMetricsToSend = append(allMetricsToSend, wiAeroMetric)

			}
		}
	}

	return allMetricsToSend, nil
}

func internalCreateLocalAerospikeStat(pStatName string, username string) (AerospikeStat, []string, []string) {
	labels := []string{commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE, commons.METRIC_LABEL_USER}
	labelValues := []string{ClusterName, Service, username}
	allowed := isMetricAllowed(commons.CTX_USERS, pStatName)
	asMetric := NewAerospikeStat(commons.CTX_USERS, pStatName, allowed)

	return asMetric, labels, labelValues
}
