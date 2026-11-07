package statprocessors

import (
	"strings"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"

	log "github.com/sirupsen/logrus"
)

const (
	KEY_BACKUP_STATUS  = "backup-status"
	KEY_RESTORE_STATUS = "restore-status"
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

		for stat, value := range stats {
			// identifiers are used as labels, not as metrics
			if stat == "job-id" || stat == "ns" {
				continue
			}

			pv, err := commons.TryConvert(value)

			if err != nil {
				continue
			}

			asMetric, exists := metrics[stat]

			if !exists {
				allowed := isMetricAllowed(context, stat)
				asMetric = NewAerospikeStat(context, stat, allowed)
				metrics[stat] = asMetric
			}

			labelValues := []string{bw.sharedState.ClusterName, bw.sharedState.Service, stats["ns"], jobID}

			asMetric.updateValues(pv, labels, labelValues)
			allMetricsToSend = append(allMetricsToSend, asMetric)
		}
	}

	return allMetricsToSend
}
