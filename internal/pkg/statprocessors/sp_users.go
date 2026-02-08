package statprocessors

import (
	commons "github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"

	aero "github.com/aerospike/aerospike-client-go/v8"
	log "github.com/sirupsen/logrus"
)

type UserStatsProcessor struct {
	ShouldFetchUserStatistics bool
}

func (uw *UserStatsProcessor) canRefreshUserStats(rawMetrics map[string]string) bool {
	// check if security configurations are enabled
	if config.Cfg.Aerospike.AuthMode != "pki" &&
		(config.Cfg.Aerospike.User == "" && config.Cfg.Aerospike.Password == "") {
		return false
	}

	// check if we should fetch user metrics
	if !uw.ShouldFetchUserStatistics {
		log.Debug("Fetching user statistics is disabled")
		return false
	}

	// validate aerospike build version
	// support for user statistics is added in aerospike 5.6
	ge, err := isBuildVersionGreaterThanOrEqual(rawMetrics["build"], "5.6.0.0")

	if err != nil {
		return false
	}

	if !ge {
		// disable user statisitcs if build version is not >= 5.6.0.0
		log.Debug("Aerospike version doesn't support user statistics")
		uw.ShouldFetchUserStatistics = false
		return false
	}

	return true
}

func (uw *UserStatsProcessor) Refresh(users []*aero.UserRoles) ([]AerospikeStat, error) {

	var allMetricsToSend = []AerospikeStat{}
	// Push metrics to Prometheus or Observability tool
	lUserMetricsToSend, err := uw.refreshUserStats(users)
	if err != nil {
		log.Warn("Error while preparing and pushing metrics: ", err)
		return nil, nil

	}

	allMetricsToSend = append(allMetricsToSend, lUserMetricsToSend...)

	return allMetricsToSend, err
}

func (uw *UserStatsProcessor) refreshUserStats(users []*aero.UserRoles) ([]AerospikeStat, error) {
	allowedUsersList := make(map[string]struct{})
	blockedUsersList := make(map[string]struct{})

	// let us not cache the user-info (like user-role, any permisison etc.,), as this can be changed at
	// server-level without any restarts
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

		asMetric, labels, labelValues := uw.makeAerospikeStat("conns_in_use", user.User)
		asMetric.updateValues(float64(user.ConnsInUse), labels, labelValues)
		allMetricsToSend = append(allMetricsToSend, asMetric)

		if len(user.ReadInfo) >= 4 && len(user.WriteInfo) >= 4 {

			for idxReadinfo := 0; idxReadinfo < len(user.ReadInfo); idxReadinfo++ {
				riAeroMetric, riLabels, riLabelValues := uw.makeAerospikeStat(readInfoStats[idxReadinfo], user.User)
				riAeroMetric.updateValues(float64(user.ReadInfo[idxReadinfo]), riLabels, riLabelValues)

				allMetricsToSend = append(allMetricsToSend, riAeroMetric)

			}
			for idxWriteinfo := 0; idxWriteinfo < len(user.WriteInfo); idxWriteinfo++ {
				wiAeroMetric, wiLabels, wiLabelValues := uw.makeAerospikeStat(writeInfoStats[idxWriteinfo], user.User)
				wiAeroMetric.updateValues(float64(user.WriteInfo[idxWriteinfo]), wiLabels, wiLabelValues)
				allMetricsToSend = append(allMetricsToSend, wiAeroMetric)

			}
		}
	}

	return allMetricsToSend, nil
}

func (uw *UserStatsProcessor) makeAerospikeStat(pStatName string, username string) (AerospikeStat, []string, []string) {
	labels := []string{commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE, commons.METRIC_LABEL_USER}
	labelValues := []string{ClusterName, Service, username}
	allowed := isMetricAllowed(commons.CTX_USERS, pStatName)
	asMetric := NewAerospikeStat(commons.CTX_USERS, pStatName, allowed)

	return asMetric, labels, labelValues
}
