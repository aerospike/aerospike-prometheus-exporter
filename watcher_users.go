package main

import (
	"fmt"
	"time"

	"github.com/aerospike/aerospike-client-go/v6/types"
	"github.com/prometheus/client_golang/prometheus"

	aero "github.com/aerospike/aerospike-client-go/v6"
	log "github.com/sirupsen/logrus"
)

var (
	shouldFetchUserStatistics bool = true
)

type UserWatcher struct{}

func (uw *UserWatcher) describe(ch chan<- *prometheus.Desc) {}

func (uw *UserWatcher) passOneKeys() []string {
	// "build" info key should be returned here,
	// but it is also being sent by LatencyWatcher.passOneKeys(),
	// hence skipping here.
	return nil
}

func (uw *UserWatcher) passTwoKeys(rawMetrics map[string]string) []string {
	return nil
}

func (uw *UserWatcher) refresh(o *Observer, infoKeys []string, rawMetrics map[string]string, ch chan<- prometheus.Metric) error {

	// check if security configurations are enabled
	if config.Aerospike.User == "" && config.Aerospike.Password == "" {
		return nil
	}

	// check if we should fetch user metrics
	if !shouldFetchUserStatistics {
		log.Debug("Fetching user statistics is disabled")
		return nil
	}

	// validate aerospike build version
	// support for user statistics is added in aerospike 5.6
	var err error
	ok, err := buildVersionGreaterThanOrEqual(rawMetrics, "5.6.0.0")
	if err != nil {
		// just log warning. don't send an error back
		log.Warn(err)
		return nil
	}

	if !ok {
		// disable user statisitcs if build version is not >= 5.6.0.0
		log.Debug("Aerospike version doesn't support user statistics")
		shouldFetchUserStatistics = false
		return nil
	}

	// read the data from Aerospike Server
	var users = fetchUsersDetails(o)

	// Push metrics to Prometheus or Observability tool
	err = uw.refreshUserStats(o, infoKeys, rawMetrics, ch, users)
	if err != nil {
		log.Warn("Error while preparing and pushing metrics: ", err)
		return nil

	}

	return err
}

func fetchUsersDetails(o *Observer) []*aero.UserRoles {
	admPlcy := aero.NewAdminPolicy()
	admPlcy.Timeout = time.Duration(config.Aerospike.Timeout) * time.Second
	admCmd := aero.NewAdminCommand(nil)

	var users []*aero.UserRoles
	var aeroErr aero.Error
	var err error

	for i := 0; i < retryCount; i++ {
		// Validate existing connection
		if o.conn == nil || !o.conn.IsConnected() {
			// Create new connection
			o.conn, err = o.newConnection()
			if err != nil {
				log.Debug(err)
				continue
			}
		}

		// query users
		users, aeroErr = admCmd.QueryUsers(o.conn, admPlcy)

		if aeroErr != nil {
			// Do not retry if there's role violation.
			// This could be a permanent error leading to unnecessary errors on server end.
			if aeroErr.Matches(types.ROLE_VIOLATION) {
				shouldFetchUserStatistics = false
				log.Debugf("Unable to fetch user statistics: %s", aeroErr.Error())
				break
			}

			err = fmt.Errorf(aeroErr.Error())
			if err != nil {
				log.Warnf("Error while querying users: %s", err)
				continue
			}
		}

		break
	}

	return users
}

func (uw *UserWatcher) refreshUserStats(o *Observer, infoKeys []string, rawMetrics map[string]string, ch chan<- prometheus.Metric, users []*aero.UserRoles) error {
	allowedUsersList := make(map[string]struct{})
	blockedUsersList := make(map[string]struct{})

	// let us not cache the user-info (like user-role, any permisison etc.,), as this can be changed at server-level without any restarts
	//
	if config.Aerospike.UserMetricsUsersAllowlistEnabled {
		for _, allowedUser := range config.Aerospike.UserMetricsUsersAllowlist {
			allowedUsersList[allowedUser] = struct{}{}
		}
	}

	if len(config.Aerospike.UserMetricsUsersBlocklist) > 0 {
		for _, blockedUser := range config.Aerospike.UserMetricsUsersBlocklist {
			blockedUsersList[blockedUser] = struct{}{}
		}
	}

	for _, user := range users {
		// check if user is allowed
		if config.Aerospike.UserMetricsUsersAllowlistEnabled {
			if _, ok := allowedUsersList[user.User]; !ok {
				continue
			}
		}

		// check if user is blocked
		if len(config.Aerospike.UserMetricsUsersBlocklist) > 0 {
			if _, ok := blockedUsersList[user.User]; ok {
				continue
			}
		}

		// Order is important as user.ReadInfo returns an int-array - where
		//   0 = read-quota, 1=read_single_record_tps etc., -- this order is fixed from server
		readInfoStats := []string{"read_quota", "read_single_record_tps", "read_scan_query_rps", "limitless_read_scan_query"}
		writeInfoStats := []string{"write_quota", "write_single_record_tps", "write_scan_query_rps", "limitless_write_scan_query"}

		// Connections in use
		pm := makeMetric("aerospike_users", "conns_in_use", mtGauge, config.AeroProm.MetricLabels, METRIC_LABEL_CLUSTER_NAME, METRIC_LABEL_SERVICE, METRIC_LABEL_USER)
		ch <- prometheus.MustNewConstMetric(pm.desc, pm.valueType, float64(user.ConnsInUse), rawMetrics[ikClusterName], rawMetrics[ikService], user.User)

		if len(user.ReadInfo) >= 4 && len(user.WriteInfo) >= 4 {
			// User read info statistics
			pm = makeMetric("aerospike_users", readInfoStats[0], mtGauge, config.AeroProm.MetricLabels, METRIC_LABEL_CLUSTER_NAME, METRIC_LABEL_SERVICE, METRIC_LABEL_USER)
			ch <- prometheus.MustNewConstMetric(pm.desc, pm.valueType, float64(user.ReadInfo[0]), rawMetrics[ikClusterName], rawMetrics[ikService], user.User)
			pm = makeMetric("aerospike_users", readInfoStats[1], mtGauge, config.AeroProm.MetricLabels, METRIC_LABEL_CLUSTER_NAME, METRIC_LABEL_SERVICE, METRIC_LABEL_USER)
			ch <- prometheus.MustNewConstMetric(pm.desc, pm.valueType, float64(user.ReadInfo[1]), rawMetrics[ikClusterName], rawMetrics[ikService], user.User)
			pm = makeMetric("aerospike_users", readInfoStats[2], mtGauge, config.AeroProm.MetricLabels, METRIC_LABEL_CLUSTER_NAME, METRIC_LABEL_SERVICE, METRIC_LABEL_USER)
			ch <- prometheus.MustNewConstMetric(pm.desc, pm.valueType, float64(user.ReadInfo[2]), rawMetrics[ikClusterName], rawMetrics[ikService], user.User)
			pm = makeMetric("aerospike_users", readInfoStats[3], mtGauge, config.AeroProm.MetricLabels, METRIC_LABEL_CLUSTER_NAME, METRIC_LABEL_SERVICE, METRIC_LABEL_USER)
			ch <- prometheus.MustNewConstMetric(pm.desc, pm.valueType, float64(user.ReadInfo[3]), rawMetrics[ikClusterName], rawMetrics[ikService], user.User)

			// User write info statistics
			pm = makeMetric("aerospike_users", writeInfoStats[0], mtGauge, config.AeroProm.MetricLabels, METRIC_LABEL_CLUSTER_NAME, METRIC_LABEL_SERVICE, METRIC_LABEL_USER)
			ch <- prometheus.MustNewConstMetric(pm.desc, pm.valueType, float64(user.WriteInfo[0]), rawMetrics[ikClusterName], rawMetrics[ikService], user.User)
			pm = makeMetric("aerospike_users", writeInfoStats[1], mtGauge, config.AeroProm.MetricLabels, METRIC_LABEL_CLUSTER_NAME, METRIC_LABEL_SERVICE, METRIC_LABEL_USER)
			ch <- prometheus.MustNewConstMetric(pm.desc, pm.valueType, float64(user.WriteInfo[1]), rawMetrics[ikClusterName], rawMetrics[ikService], user.User)
			pm = makeMetric("aerospike_users", writeInfoStats[2], mtGauge, config.AeroProm.MetricLabels, METRIC_LABEL_CLUSTER_NAME, METRIC_LABEL_SERVICE, METRIC_LABEL_USER)
			ch <- prometheus.MustNewConstMetric(pm.desc, pm.valueType, float64(user.WriteInfo[2]), rawMetrics[ikClusterName], rawMetrics[ikService], user.User)
			pm = makeMetric("aerospike_users", writeInfoStats[3], mtGauge, config.AeroProm.MetricLabels, METRIC_LABEL_CLUSTER_NAME, METRIC_LABEL_SERVICE, METRIC_LABEL_USER)
			ch <- prometheus.MustNewConstMetric(pm.desc, pm.valueType, float64(user.WriteInfo[3]), rawMetrics[ikClusterName], rawMetrics[ikService], user.User)
		}
	}
	return nil
}
