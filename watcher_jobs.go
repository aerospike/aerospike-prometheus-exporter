package main

import (
	"strings"

	"github.com/prometheus/client_golang/prometheus"

	log "github.com/sirupsen/logrus"
)

// Jobs raw metrics
var jobsRawMetrics = map[string]metricType{
	"priority":           mtGauge,
	"net-io-time":        mtGauge,
	"n-pids-requested":   mtGauge,
	"rps":                mtGauge,
	"active-threads":     mtGauge,
	"job-progress":       mtGauge,
	"run-time":           mtGauge,
	"time-since-done":    mtGauge,
	"recs-throttled":     mtGauge,
	"recs-filtered-meta": mtGauge,
	"recs-filtered-bins": mtGauge,
	"recs-succeeded":     mtGauge,
	"recs-failed":        mtGauge,
	"net-io-bytes":       mtGauge,
	"socket-timeout":     mtGauge,
	"udf-active":         mtGauge,
	"ops-active":         mtGauge,
}

type JobsWatcher struct{}

func (jw *JobsWatcher) describe(ch chan<- *prometheus.Desc) {}

func (jw *JobsWatcher) passOneKeys() []string {
	// "build" info key should be returned here,
	// but it is also being sent by LatencyWatcher.passOneKeys(),
	// hence skipping here.
	return nil
}

func (jw *JobsWatcher) passTwoKeys(rawMetrics map[string]string) (jobsCommands []string) {
	if config.Aerospike.DisableJobMetrics {
		// disabled
		return nil
	}

	jobsCommands = []string{"jobs:", "scan-show:", "query-show:"}

	ok, err := buildVersionGreaterThanOrEqual(rawMetrics, "6.0.0.0-0")
	if err != nil {
		log.Warn(err)
		return jobsCommands
	}

	if ok {
		return []string{"query-show:"}
	}

	ok, err = buildVersionGreaterThanOrEqual(rawMetrics, "5.7.0.0")
	if err != nil {
		log.Warn(err)
		return jobsCommands
	}

	if ok {
		return []string{"scan-show:", "query-show:"}
	}

	return []string{"jobs:"}
}

var jobMetrics map[string]metricType

func (jw *JobsWatcher) refresh(o *Observer, infoKeys []string, rawMetrics map[string]string, ch chan<- prometheus.Metric) error {
	if config.Aerospike.DisableJobMetrics {
		// disabled
		return nil
	}

	var jobStats []string

	if rawMetrics["scan-show:"] != "" || rawMetrics["query-show:"] != "" {
		jobStats = strings.Split(rawMetrics["scan-show:"], ";")
		jobStats = append(jobStats, strings.Split(rawMetrics["query-show:"], ";")...)
	} else {
		jobStats = strings.Split(rawMetrics["jobs:"], ";")
	}
	log.Tracef("job-stats:%v", jobStats)

	if jobMetrics == nil {
		jobMetrics = getFilteredMetrics(jobsRawMetrics, config.Aerospike.JobMetricsAllowlist, config.Aerospike.JobMetricsAllowlistEnabled, config.Aerospike.JobMetricsBlocklist, config.Aerospike.JobMetricsBlocklistEnabled)
	}

	for i := range jobStats {
		jobObserver := make(MetricMap, len(jobMetrics))
		for m, t := range jobMetrics {
			jobObserver[m] = makeMetric("aerospike_jobs", m, t, config.AeroProm.MetricLabels, "cluster_name", "service", "ns", "set", "module", "trid")
		}

		stats := parseStats(jobStats[i], ":")
		for stat, pm := range jobObserver {
			v, exists := stats[stat]
			if !exists {
				// not found
				continue
			}

			pv, err := tryConvert(v)
			if err != nil {
				continue
			}

			ch <- prometheus.MustNewConstMetric(pm.desc, pm.valueType, pv, rawMetrics[ikClusterName], rawMetrics[ikService], stats["ns"], stats["set"], stats["module"], stats["trid"])
		}
	}

	return nil
}
