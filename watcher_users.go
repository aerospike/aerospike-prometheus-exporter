package main

import (
	"fmt"
	"time"

	"github.com/aerospike/aerospike-client-go/v5/types"
	"github.com/prometheus/client_golang/prometheus"

	aero "github.com/aerospike/aerospike-client-go/v5"
	log "github.com/sirupsen/logrus"
)

var (
	shouldFetchUserStatistics bool = true
)

type UserWatcher struct{}

func (uw *UserWatcher) describe(ch chan<- *prometheus.Desc) {}

func (uw *UserWatcher) passOneKeys() []string {
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

	admPlcy := aero.NewAdminPolicy()
	admPlcy.Timeout = time.Duration(config.Aerospike.Timeout) * time.Second
	admCmd := aero.NewAdminCommand(nil)

	var users []*aero.UserRoles
	var aeroErr aero.Error

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
			log.Debugf("Error while querying users: %s", aeroErr.Error())
			continue
		}

		break
	}

	allowedUsersList := make(map[string]struct{})
	blockedUsersList := make(map[string]struct{})

	if config.Aerospike.UserMetricsUsersAllowlistEnabled {
		for _, allowedUser := range config.Aerospike.UserMetricsUsersAllowlist {
			allowedUsersList[allowedUser] = struct{}{}
		}
	}

	if config.Aerospike.UserMetricsUsersBlocklistEnabled {
		for _, blockedUser := range config.Aerospike.UserMetricsUsersBlocklist {
			blockedUsersList[blockedUser] = struct{}{}
		}
	}

	for _, user := range users {
		// check is user allowed
		if config.Aerospike.UserMetricsUsersAllowlistEnabled {
			if _, ok := allowedUsersList[user.User]; !ok {
				continue
			}
		}

		// check if user blocked
		if config.Aerospike.UserMetricsUsersBlocklistEnabled {
			if _, ok := blockedUsersList[user.User]; ok {
				continue
			}
		}

		// Connections in use
		pm := makeMetric("aerospike_users", "conns_in_use", mtGauge, config.AeroProm.MetricLabels, "cluster_name", "service", "user")
		ch <- prometheus.MustNewConstMetric(pm.desc, pm.valueType, float64(user.ConnsInUse), rawMetrics[ikClusterName], rawMetrics[ikService], user.User)

		if len(user.ReadInfo) >= 4 && len(user.WriteInfo) >= 4 {
			// User read info statistics
			pm = makeMetric("aerospike_users", "read_quota", mtGauge, config.AeroProm.MetricLabels, "cluster_name", "service", "user")
			ch <- prometheus.MustNewConstMetric(pm.desc, pm.valueType, float64(user.ReadInfo[0]), rawMetrics[ikClusterName], rawMetrics[ikService], user.User)
			pm = makeMetric("aerospike_users", "read_single_record_tps", mtGauge, config.AeroProm.MetricLabels, "cluster_name", "service", "user")
			ch <- prometheus.MustNewConstMetric(pm.desc, pm.valueType, float64(user.ReadInfo[1]), rawMetrics[ikClusterName], rawMetrics[ikService], user.User)
			pm = makeMetric("aerospike_users", "read_scan_query_rps", mtGauge, config.AeroProm.MetricLabels, "cluster_name", "service", "user")
			ch <- prometheus.MustNewConstMetric(pm.desc, pm.valueType, float64(user.ReadInfo[2]), rawMetrics[ikClusterName], rawMetrics[ikService], user.User)
			pm = makeMetric("aerospike_users", "limitless_read_scan_query", mtGauge, config.AeroProm.MetricLabels, "cluster_name", "service", "user")
			ch <- prometheus.MustNewConstMetric(pm.desc, pm.valueType, float64(user.ReadInfo[3]), rawMetrics[ikClusterName], rawMetrics[ikService], user.User)

			// User write info statistics
			pm = makeMetric("aerospike_users", "write_quota", mtGauge, config.AeroProm.MetricLabels, "cluster_name", "service", "user")
			ch <- prometheus.MustNewConstMetric(pm.desc, pm.valueType, float64(user.WriteInfo[0]), rawMetrics[ikClusterName], rawMetrics[ikService], user.User)
			pm = makeMetric("aerospike_users", "write_single_record_tps", mtGauge, config.AeroProm.MetricLabels, "cluster_name", "service", "user")
			ch <- prometheus.MustNewConstMetric(pm.desc, pm.valueType, float64(user.WriteInfo[1]), rawMetrics[ikClusterName], rawMetrics[ikService], user.User)
			pm = makeMetric("aerospike_users", "write_scan_query_rps", mtGauge, config.AeroProm.MetricLabels, "cluster_name", "service", "user")
			ch <- prometheus.MustNewConstMetric(pm.desc, pm.valueType, float64(user.WriteInfo[2]), rawMetrics[ikClusterName], rawMetrics[ikService], user.User)
			pm = makeMetric("aerospike_users", "limitless_write_scan_query", mtGauge, config.AeroProm.MetricLabels, "cluster_name", "service", "user")
			ch <- prometheus.MustNewConstMetric(pm.desc, pm.valueType, float64(user.WriteInfo[3]), rawMetrics[ikClusterName], rawMetrics[ikService], user.User)
		}
	}

	return err
}
