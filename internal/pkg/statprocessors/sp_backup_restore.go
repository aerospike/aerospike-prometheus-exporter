package statprocessors

import (
	"strings"
	"time"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"

	log "github.com/sirupsen/logrus"
)

const (
	KEY_BACKUP_STATUS  = "backup-status"
	KEY_RESTORE_STATUS = "restore-status"

	// STAT_JOB is emitted as its own gauge so jobs can be counted per state:
	// aerospike_backup_job / aerospike_restore_job, labeled with job_id and state.
	// The job_id stays a label on every other metric.
	STAT_JOB = "job"
)

// BackupRestoreStatsProcessor exports the per-node status of in-server backup and restore jobs.
// Both info commands are enterprise-only and gated behind --preview backup-restore, so on any
// other node the response is an error and nothing is emitted.
type BackupRestoreStatsProcessor struct {
	backupMetrics  map[string]AerospikeStat
	restoreMetrics map[string]AerospikeStat
	sharedState    *StatProcessorSharedState
}

func NewBackupRestoreStatsProcessor(state *StatProcessorSharedState) *BackupRestoreStatsProcessor {
	return &BackupRestoreStatsProcessor{
		sharedState:    state,
		backupMetrics:  make(map[string]AerospikeStat),
		restoreMetrics: make(map[string]AerospikeStat),
	}
}

func (bw *BackupRestoreStatsProcessor) PassOneKeys() []string {
	log.Tracef("backup-restore-passonekeys:nil")

	return nil
}

func (bw *BackupRestoreStatsProcessor) PassTwoKeys(passOneStats map[string]string) []string {
	if bw.sharedState.CheckpointSaveInProgress {
		log.Tracef("backup-restore-passtwokeys:nil")
		return nil
	}

	// in-server backup and restore is only available from 8.2.1 onwards
	ge, err := isBuildVersionGreaterThanOrEqual(bw.sharedState.Build, "8.2.1")

	if err != nil {
		return nil
	}

	if !ge {
		log.Tracef("backup-restore-passtwokeys:nil")
		return nil
	}

	log.Tracef("backup-restore-passtwokeys:%s", []string{KEY_BACKUP_STATUS, KEY_RESTORE_STATUS})

	return []string{KEY_BACKUP_STATUS, KEY_RESTORE_STATUS}
}

func (bw *BackupRestoreStatsProcessor) Refresh(infoKeys []string, rawMetrics map[string]string) ([]AerospikeStat, error) {

	var allMetricsToSend = []AerospikeStat{}

	allMetricsToSend = append(allMetricsToSend,
		bw.refreshJobs(commons.CTX_BACKUP, bw.backupMetrics, rawMetrics[KEY_BACKUP_STATUS])...)

	allMetricsToSend = append(allMetricsToSend,
		bw.refreshJobs(commons.CTX_RESTORE, bw.restoreMetrics, rawMetrics[KEY_RESTORE_STATUS])...)

	return allMetricsToSend, nil
}

// refreshJobs parses one status response. Each record is one job, records are ';' separated and
// fields within a record are ':' separated, e.g. job-id=<jid>:ns=<name>:state=<STATE>:progress-pct=<f>
func (bw *BackupRestoreStatsProcessor) refreshJobs(context commons.ContextType, metrics map[string]AerospikeStat, response string) []AerospikeStat {

	var allMetricsToSend = []AerospikeStat{}

	if !isValidResponse(response) {
		return allMetricsToSend
	}

	log.Tracef("%s-stats:%v", context, response)

	jobRecords := strings.Split(response, ";")

	for i := range jobRecords {
		stats := commons.ParseStats(jobRecords[i], ":")

		// records without a job-id are not jobs, e.g. restore-status "ns=<name>:ns-state=<STATE>"
		jobID, hasJobID := stats["job-id"]
		if !hasJobID {
			continue
		}

		labels := []string{commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE, commons.METRIC_LABEL_NS, commons.METRIC_LABEL_JOB_ID}
		labelValues := []string{bw.sharedState.ClusterName, bw.sharedState.Service, stats["ns"], jobID}

		// the job is also emitted as its own gauge so the number of jobs in each state can be counted:
		// aerospike_backup_job{job_id="b1",state="BASE_SCAN_ACTIVE"} 1
		// the remaining labels are conditional, so each is added only when the server emitted it
		if state, hasState := stats["state"]; hasState {
			jobLabels := append(labels, commons.METRIC_LABEL_STATE)
			jobLabelValues := append(labelValues, state)

			jobLabels, jobLabelValues = appendLabelIfPresent(jobLabels, jobLabelValues, commons.METRIC_LABEL_ERROR_REASON, stats["error-reason"])
			jobLabels, jobLabelValues = appendLabelIfPresent(jobLabels, jobLabelValues, commons.METRIC_LABEL_ABORT_ORIGIN, stats["abort-origin"])

			// restore rows name their lane, the namespace's restore gate, and — when restoring a
			// list of backups or a frozen layout — which backup and which cluster key
			jobLabels, jobLabelValues = appendLabelIfPresent(jobLabels, jobLabelValues, commons.METRIC_LABEL_KIND, stats["kind"])
			jobLabels, jobLabelValues = appendLabelIfPresent(jobLabels, jobLabelValues, commons.METRIC_LABEL_NS_STATE, stats["ns-state"])
			jobLabels, jobLabelValues = appendLabelIfPresent(jobLabels, jobLabelValues, commons.METRIC_LABEL_BACKUP_ID, stats["backup-id"])
			jobLabels, jobLabelValues = appendLabelIfPresent(jobLabels, jobLabelValues, commons.METRIC_LABEL_CLUSTER_KEY, stats["cluster-key"])

			allMetricsToSend = append(allMetricsToSend, bw.jobMetric(context, metrics, jobLabels, jobLabelValues))
		}

		for stat, value := range stats {
			// identifiers and non-numeric descriptors are used as labels, not as metrics.
			// cluster-key is hex, so an all-digit key would otherwise slip through as a number.
			if stat == "job-id" || stat == "ns" || stat == "cluster-key" || stat == "backup-id" {
				continue
			}

			pv, ok := numericValue(stat, value)
			if !ok {
				continue
			}

			asMetric, exists := metrics[stat]

			if !exists {
				allowed := isMetricAllowed(context, stat)
				asMetric = NewAerospikeStat(context, stat, allowed)
				metrics[stat] = asMetric
			}

			asMetric.updateValues(pv, labels, labelValues)
			allMetricsToSend = append(allMetricsToSend, asMetric)
		}
	}

	return allMetricsToSend
}

// jobMetric emits one gauge per job, valued 1, labeled with job_id and the job's state.
// It is always a gauge: "job" has no hyphen and is not in gauge_stats_list.toml, so it would
// otherwise be typed as a counter.
func (bw *BackupRestoreStatsProcessor) jobMetric(context commons.ContextType, metrics map[string]AerospikeStat, labels []string, labelValues []string) AerospikeStat {

	asMetric, exists := metrics[STAT_JOB]

	if !exists {
		allowed := isMetricAllowed(context, STAT_JOB)
		asMetric = NewAerospikeStat(context, STAT_JOB, allowed)
		asMetric.MType = commons.MetricTypeGauge
		metrics[STAT_JOB] = asMetric
	}

	asMetric.updateValues(1, labels, labelValues)

	return asMetric
}

// appendLabelIfPresent adds a label only when the server emitted a real value for it.
// An absent field, an empty value, or "-" all mean the field was not emitted.
func appendLabelIfPresent(labels []string, labelValues []string, label string, value string) ([]string, []string) {
	if value == "" || value == "-" {
		return labels, labelValues
	}

	return append(labels, label), append(labelValues, value)
}

// serverTimeFormats are the UTC timestamps the status commands emit. The live server prints
// "2026-11-10 06.55.05.903 UTC"; the spec example is the compact "20260728T161500.250Z".
var serverTimeFormats = []string{
	"2006-01-02 15.04.05.000 MST",
	"20060102T150405.000Z",
}

// numericValue returns the metric value for one field. Timestamps become unix milliseconds so a
// job's duration is finish-time minus start-time; "-" means the time was never reached and is skipped.
func numericValue(stat string, value string) (float64, bool) {
	if stat == "start-time" || stat == "finish-time" {
		if value == "-" {
			return 0, false
		}

		t, ok := parseServerTime(value)
		if !ok {
			log.Debugf("backup-restore: unparseable %s %q", stat, value)
			return 0, false
		}

		return float64(t.UnixMilli()), true
	}

	pv, err := commons.TryConvert(value)
	if err != nil {
		return 0, false
	}

	return pv, true
}

func parseServerTime(value string) (time.Time, bool) {
	for _, format := range serverTimeFormats {
		if t, err := time.Parse(format, value); err == nil {
			return t, true
		}
	}

	return time.Time{}, false
}
