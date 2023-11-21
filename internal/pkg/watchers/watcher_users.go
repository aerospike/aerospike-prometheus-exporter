package watchers

import (
	"fmt"

	commons "github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/data"

	aero "github.com/aerospike/aerospike-client-go/v6"
	log "github.com/sirupsen/logrus"
)

var (
	shouldFetchUserStatistics bool = true
)

type UserWatcher struct{}

func (uw *UserWatcher) PassOneKeys() []string {
	// "build" info key should be returned here,
	// but it is also being sent by LatencyWatcher.passOneKeys(),
	// hence skipping here.
	log.Tracef("users-passonekeys:nil")
	return nil
}

func (uw *UserWatcher) PassTwoKeys(rawMetrics map[string]string) []string {
	log.Tracef("users-passtwokeys:nil")
	return nil
}

func (uw *UserWatcher) Refresh(infoKeys []string, rawMetrics map[string]string) ([]AerospikeStat, error) {

	// check if security configurations are enabled
	if config.Cfg.Aerospike.User == "" && config.Cfg.Aerospike.Password == "" {
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

	fmt.Println("... watcher_user: ste...1")

	shouldFetchUserStatistics, users, err = data.GetProvider().FetchUsersDetails()
	if err != nil {
		log.Warn("Error while fetching user statistics: ", err)
		return nil, nil

	}
	fmt.Println("... shouldFetchUserStatistics: ", shouldFetchUserStatistics)

	var metrics_to_send = []AerospikeStat{}
	// Push metrics to Prometheus or Observability tool
	l_user_metrics_to_send, err := uw.refreshUserStats(infoKeys, rawMetrics, users)
	if err != nil {
		log.Warn("Error while preparing and pushing metrics: ", err)
		return nil, nil

	}

	metrics_to_send = append(metrics_to_send, l_user_metrics_to_send...)

	return metrics_to_send, err
}

func (uw *UserWatcher) refreshUserStats(infoKeys []string, rawMetrics map[string]string, users []*aero.UserRoles) ([]AerospikeStat, error) {
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

	var metrics_to_send = []AerospikeStat{}

	for _, user := range users {
		// check if user is allowed
		if config.Cfg.Aerospike.UserMetricsUsersAllowlistEnabled {
			if _, ok := allowedUsersList[user.User]; !ok {
				continue
			}
		}

		// fmt.Println("watcher-user handling user: ", user.User, "\n\t Roles: ", user.Roles,
		// 	"\n\t Conns-in-use: ", user.ConnsInUse,
		// 	"\n\t ReadInfo: ", user.ReadInfo, "\n\t WriteInfo: ", user.WriteInfo)

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

		asMetric, labels, labelValues := internalCreateLocalAerospikeStat(rawMetrics, "conns_in_use", user.User)
		asMetric.updateValues(float64(user.ConnsInUse), labels, labelValues)
		metrics_to_send = append(metrics_to_send, asMetric)

		if len(user.ReadInfo) >= 4 && len(user.WriteInfo) >= 4 {

			for Idx_Readinfo := 0; Idx_Readinfo < len(user.ReadInfo); Idx_Readinfo++ {
				ri_asMetric, ri_labels, ri_labelValues := internalCreateLocalAerospikeStat(rawMetrics, readInfoStats[Idx_Readinfo], user.User)
				ri_asMetric.updateValues(float64(user.ReadInfo[Idx_Readinfo]), ri_labels, ri_labelValues)

				metrics_to_send = append(metrics_to_send, ri_asMetric)

			}
			for Idx_Writeinfo := 0; Idx_Writeinfo < len(user.WriteInfo); Idx_Writeinfo++ {
				wi_asMetric, wi_labels, wi_labelValues := internalCreateLocalAerospikeStat(rawMetrics, writeInfoStats[Idx_Writeinfo], user.User)
				wi_asMetric.updateValues(float64(user.WriteInfo[Idx_Writeinfo]), wi_labels, wi_labelValues)
				metrics_to_send = append(metrics_to_send, wi_asMetric)

			}
		}
	}

	return metrics_to_send, nil
}

func internalCreateLocalAerospikeStat(rawMetrics map[string]string, pStatName string, username string) (AerospikeStat, []string, []string) {
	labels := []string{commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE, commons.METRIC_LABEL_USER}
	labelValues := []string{rawMetrics[Infokey_ClusterName], rawMetrics[Infokey_Service], username}
	asMetric := NewAerospikeStat(commons.CTX_USERS, pStatName)

	return asMetric, labels, labelValues
}
