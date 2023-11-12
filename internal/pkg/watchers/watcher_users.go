package watchers

import (
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
	return nil
}

func (uw *UserWatcher) PassTwoKeys(rawMetrics map[string]string) []string {
	return nil
}

func (uw *UserWatcher) Refresh(infoKeys []string, rawMetrics map[string]string) ([]WatcherMetric, error) {

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
	ok, err := commons.BuildVersionGreaterThanOrEqual(rawMetrics, "5.6.0.0")
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

	shouldFetchUserStatistics, users, err = data.GetDataProvider().FetchUsersDetails()

	var metrics_to_send = []WatcherMetric{}
	// Push metrics to Prometheus or Observability tool
	l_user_metrics_to_send, err := uw.refreshUserStats(infoKeys, rawMetrics, users)
	if err != nil {
		log.Warn("Error while preparing and pushing metrics: ", err)
		return nil, nil

	}

	metrics_to_send = append(metrics_to_send, l_user_metrics_to_send...)

	return metrics_to_send, err
}

func (uw *UserWatcher) refreshUserStats(infoKeys []string, rawMetrics map[string]string, users []*aero.UserRoles) ([]WatcherMetric, error) {
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

	var metrics_to_send = []WatcherMetric{}

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

		// Connections in use
		// pm := makeMetric("aerospike_users", "conns_in_use", mtGauge, config.Cfg.AeroProm.MetricLabels, commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE, commons.METRIC_LABEL_USER)
		// ch <- prometheus.MustNewConstMetric(pm.desc, pm.valueType, float64(user.ConnsInUse), rawMetrics[ikClusterName], rawMetrics[ikService], user.User)

		// labels := []string{commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE, commons.METRIC_LABEL_USER}
		// labelValues := []string{commons.Infokey_ClusterName, commons.Infokey_Service, user.User}
		// asMetric := commons.NewAerospikeStat(commons.CTX_USERS, "conns_in_use")
		var asMetric commons.AerospikeStat
		var labels []string
		var labelValues []string

		asMetric, labels, labelValues = makeAerospikeStat("conns_in_use", user.User)
		metrics_to_send = append(metrics_to_send, WatcherMetric{asMetric, float64(user.ConnsInUse), labels, labelValues})

		if len(user.ReadInfo) >= 4 && len(user.WriteInfo) >= 4 {

			for Idx_Readinfo := 0; Idx_Readinfo < len(user.ReadInfo); Idx_Readinfo++ {
				asMetric, labels, labelValues = makeAerospikeStat(readInfoStats[Idx_Readinfo], user.User)
				metrics_to_send = append(metrics_to_send, WatcherMetric{asMetric, float64(user.ReadInfo[Idx_Readinfo]), labels, labelValues})

			}
			for Idx_Writeinfo := 0; Idx_Writeinfo < len(user.ReadInfo); Idx_Writeinfo++ {
				asMetric, labels, labelValues = makeAerospikeStat(writeInfoStats[Idx_Writeinfo], user.User)
				metrics_to_send = append(metrics_to_send, WatcherMetric{asMetric, float64(user.ReadInfo[Idx_Writeinfo]), labels, labelValues})

			}

			// User read info statistics
			// pm = makeMetric("aerospike_users", readInfoStats[0], mtGauge, config.Cfg.AeroProm.MetricLabels, commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE, commons.METRIC_LABEL_USER)
			// ch <- prometheus.MustNewConstMetric(pm.desc, pm.valueType, float64(user.ReadInfo[0]), rawMetrics[commons.Infokey_ClusterName], rawMetrics[commons.Infokey_Service], user.User)
			// pm = makeMetric("aerospike_users", readInfoStats[1], mtGauge, config.Cfg.AeroProm.MetricLabels, commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE, commons.METRIC_LABEL_USER)
			// ch <- prometheus.MustNewConstMetric(pm.desc, pm.valueType, float64(user.ReadInfo[1]), rawMetrics[commons.Infokey_ClusterName], rawMetrics[commons.Infokey_Service], user.User)
			// pm = makeMetric("aerospike_users", readInfoStats[2], mtGauge, config.Cfg.AeroProm.MetricLabels, commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE, commons.METRIC_LABEL_USER)
			// ch <- prometheus.MustNewConstMetric(pm.desc, pm.valueType, float64(user.ReadInfo[2]), rawMetrics[commons.Infokey_ClusterName], rawMetrics[commons.Infokey_Service], user.User)
			// pm = makeMetric("aerospike_users", readInfoStats[3], mtGauge, config.Cfg.AeroProm.MetricLabels, commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE, commons.METRIC_LABEL_USER)
			// ch <- prometheus.MustNewConstMetric(pm.desc, pm.valueType, float64(user.ReadInfo[3]), rawMetrics[commons.Infokey_ClusterName], rawMetrics[commons.Infokey_Service], user.User)

			// User write info statistics
			// pm = makeMetric("aerospike_users", writeInfoStats[0], mtGauge, config.Cfg.AeroProm.MetricLabels, commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE, commons.METRIC_LABEL_USER)
			// ch <- prometheus.MustNewConstMetric(pm.desc, pm.valueType, float64(user.WriteInfo[0]), rawMetrics[commons.Infokey_ClusterName], rawMetrics[commons.Infokey_Service], user.User)
			// pm = makeMetric("aerospike_users", writeInfoStats[1], mtGauge, config.Cfg.AeroProm.MetricLabels, commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE, commons.METRIC_LABEL_USER)
			// ch <- prometheus.MustNewConstMetric(pm.desc, pm.valueType, float64(user.WriteInfo[1]), rawMetrics[commons.Infokey_ClusterName], rawMetrics[commons.Infokey_Service], user.User)
			// pm = makeMetric("aerospike_users", writeInfoStats[2], mtGauge, config.Cfg.AeroProm.MetricLabels, commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE, commons.METRIC_LABEL_USER)
			// ch <- prometheus.MustNewConstMetric(pm.desc, pm.valueType, float64(user.WriteInfo[2]), rawMetrics[commons.Infokey_ClusterName], rawMetrics[commons.Infokey_Service], user.User)
			// pm = makeMetric("aerospike_users", writeInfoStats[3], mtGauge, config.Cfg.AeroProm.MetricLabels, commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE, commons.METRIC_LABEL_USER)
			// ch <- prometheus.MustNewConstMetric(pm.desc, pm.valueType, float64(user.WriteInfo[3]), rawMetrics[commons.Infokey_ClusterName], rawMetrics[commons.Infokey_Service], user.User)
		}
	}
	return metrics_to_send, nil
}

func makeAerospikeStat(pStatName string, username string) (commons.AerospikeStat, []string, []string) {
	labels := []string{commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE, commons.METRIC_LABEL_USER}
	labelValues := []string{ClusterName, Service, username}
	asMetric := commons.NewAerospikeStat(commons.CTX_USERS, "conns_in_use")

	return asMetric, labels, labelValues
}
