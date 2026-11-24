package statprocessors

import (
	"testing"
	"time"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBackupRestorePassTwoKeysBuildGate(t *testing.T) {
	tests := []struct {
		name     string
		build    string
		expected bool
	}{
		{"before 8.2.1", "8.2.0.9", false},
		{"8.2.0", "8.2.0.0", false},
		{"exactly 8.2.1", "8.2.1.0", true},
		{"after 8.2.1", "8.2.1.5", true},
		{"major upgrade", "9.0.0.0", true},
		{"empty build", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sharedState := useProcessorConfig(t)
			sharedState.Build = tt.build
			bw := NewBackupRestoreStatsProcessor(sharedState)

			keys := bw.PassTwoKeys(nil)

			if tt.expected {
				assert.Equal(t, []string{KEY_BACKUP_STATUS, KEY_RESTORE_STATUS}, keys)
			} else {
				assert.Nil(t, keys)
			}
		})
	}
}

func TestBackupRestoreRefreshParsesJobs(t *testing.T) {
	sharedState := useProcessorConfig(t)
	sharedState.ClusterName = "c1"
	sharedState.Service = "10.0.0.1:3000"
	sharedState.Build = "8.2.1.0"
	bw := NewBackupRestoreStatsProcessor(sharedState)

	raw := map[string]string{
		KEY_BACKUP_STATUS: "job-id=b1:ns=test:state=RUNNING:progress-pct=42.50:change-stream-active=true:start-time=20260728T161500.250Z:finish-time=-:achieved-consistency=-;" +
			"job-id=b2:ns=test:state=ABORTED:start-time=2026-11-10 06.19.51.659 UTC:finish-time=2026-11-10 06.19.52.836 UTC:error-reason=job-id reuse prefix already holds a finished backup:abort-origin=server:achieved-consistency=strict",
		KEY_RESTORE_STATUS: "ns=test:ns-state=NORMAL;job-id=r1:ns=test:kind=cold:ns-state=RESTORING:state=COMPLETE:progress-pct=100.00:applied=7:joiner=false:done=false:backup-id=daily1:cluster-key=3F6AF2E3A109",
	}

	metrics, err := bw.Refresh(nil, raw)
	require.NoError(t, err)

	byKey := map[string]AerospikeStat{}
	for _, m := range metrics {
		byKey[string(m.Context)+"/"+m.Name+"/"+m.LabelValues[3]] = m
	}

	backupPct, ok := byKey["backup/progress-pct/b1"]
	require.True(t, ok)
	assert.Equal(t, 42.5, backupPct.Value)
	assert.Equal(t, []string{"b1"}, backupPct.LabelValues[3:])

	// a running job carries only the state; error-reason is absent so it is not a label
	backupJob, ok := byKey["backup/job/b1"]
	assert.Equal(t, commons.MetricTypeGauge, backupJob.MType)
	require.True(t, ok)
	assert.Equal(t, 1.0, backupJob.Value)
	assert.Equal(t, []string{commons.METRIC_LABEL_STATE}, backupJob.Labels[4:])
	assert.Equal(t, []string{"c1", "10.0.0.1:3000", "test", "b1", "RUNNING"}, backupJob.LabelValues)

	// achieved-consistency is its own gauge: 0 while "-", 2 for strict, with the level as a label
	pending, ok := byKey["backup/achieved-consistency/b1"]
	require.True(t, ok)
	assert.Equal(t, commons.MetricTypeGauge, pending.MType)
	assert.Equal(t, 0.0, pending.Value)
	assert.Equal(t, []string{"-"}, pending.LabelValues[4:])

	reached, ok := byKey["backup/achieved-consistency/b2"]
	require.True(t, ok)
	assert.Equal(t, 2.0, reached.Value)
	assert.Equal(t, []string{commons.METRIC_LABEL_ACHIEVED_CONSISTENCY}, reached.Labels[4:])
	assert.Equal(t, []string{"strict"}, reached.LabelValues[4:])

	// error-reason and abort-origin are not job labels for now, even on an aborted job
	abortedJob, ok := byKey["backup/job/b2"]
	require.True(t, ok)
	assert.Equal(t, []string{commons.METRIC_LABEL_STATE}, abortedJob.Labels[4:])
	assert.Equal(t, []string{"ABORTED"}, abortedJob.LabelValues[4:])

	restoreJob, ok := byKey["restore/job/r1"]
	assert.Equal(t, commons.MetricTypeGauge, restoreJob.MType)
	require.True(t, ok)
	assert.Equal(t, 1.0, restoreJob.Value)
	assert.Equal(t, []string{commons.METRIC_LABEL_STATE, commons.METRIC_LABEL_KIND, commons.METRIC_LABEL_NS_STATE, commons.METRIC_LABEL_BACKUP_ID, commons.METRIC_LABEL_CLUSTER_KEY}, restoreJob.Labels[4:])
	assert.Equal(t, []string{"c1", "10.0.0.1:3000", "test", "r1", "COMPLETE", "cold", "RESTORING", "daily1", "3F6AF2E3A109"}, restoreJob.LabelValues)

	// backup-id and the hex cluster-key are labels, never metrics
	_, hasBackupID := byKey["restore/backup-id/r1"]
	assert.False(t, hasBackupID)
	_, hasClusterKey := byKey["restore/cluster-key/r1"]
	assert.False(t, hasClusterKey)

	cs, ok := byKey["backup/change-stream-active/b1"]
	require.True(t, ok)
	assert.Equal(t, 1.0, cs.Value)
	assert.Equal(t, commons.MetricTypeGauge, cs.MType)

	// boolean flags are gauges, even when the name has no hyphen and would otherwise be a counter
	done, ok := byKey["restore/done/r1"]
	require.True(t, ok)
	assert.Equal(t, 0.0, done.Value)
	assert.Equal(t, commons.MetricTypeGauge, done.MType)

	joiner, ok := byKey["restore/joiner/r1"]
	require.True(t, ok)
	assert.Equal(t, commons.MetricTypeGauge, joiner.MType)

	restoreApplied, ok := byKey["restore/applied/r1"]
	require.True(t, ok)
	assert.Equal(t, 7.0, restoreApplied.Value)
	// every backup/restore stat is a gauge for now, including names without a hyphen
	assert.Equal(t, commons.MetricTypeGauge, restoreApplied.MType)
	assert.Equal(t, []string{"c1", "10.0.0.1:3000", "test", "r1"}, restoreApplied.LabelValues)

	// timestamps become unix milliseconds; finish-time is "-" while the job is still running and is exported as 0
	start, ok := byKey["backup/start-time/b1"]
	require.True(t, ok)
	assert.Equal(t, float64(time.Date(2026, 7, 28, 16, 15, 0, int(250*time.Millisecond), time.UTC).UnixMilli()), start.Value)

	runningFinish, ok := byKey["backup/finish-time/b1"]
	require.True(t, ok)
	assert.Equal(t, 0.0, runningFinish.Value)

	// the live server prints "2026-11-10 06.19.51.659 UTC"
	liveStart, ok := byKey["backup/start-time/b2"]
	require.True(t, ok)
	assert.Equal(t, float64(time.Date(2026, 11, 10, 6, 19, 51, int(659*time.Millisecond), time.UTC).UnixMilli()), liveStart.Value)

	// the bare ns row has no job-id, so it must not produce any metric
	_, hasNsState := byKey["restore/ns-state"]
	assert.False(t, hasNsState)
}

// TestBackupRestoreErrorTextInsideJob checks that a job whose error-reason contains "error:" does not
// make the whole response look like a failed command, while a real command error still emits nothing.
func TestBackupRestoreErrorTextInsideJob(t *testing.T) {
	bw := NewBackupRestoreStatsProcessor(useProcessorConfig(t))

	raw := map[string]string{
		KEY_BACKUP_STATUS: "job-id=b1:ns=test:state=BASE_SCAN_ACTIVE:progress-pct=10.00;" +
			"job-id=b2:ns=test:state=FAILED:progress-pct=0.00:error-reason=S3 error: AccessDenied",
		KEY_RESTORE_STATUS: "ERROR:4:unrecognized command",
	}

	metrics, err := bw.Refresh(nil, raw)
	require.NoError(t, err)

	jobs := map[string]bool{}
	for _, m := range metrics {
		assert.Equal(t, commons.CTX_BACKUP, m.Context, "a command error must not produce restore metrics")
		if m.Name == STAT_JOB {
			jobs[m.LabelValues[3]] = true
		}
	}

	assert.True(t, jobs["b1"], "the running job must survive another job's error text")
	assert.True(t, jobs["b2"])
}

// TestBackupRestoreAllowlistAndBlocklist checks that backup_metrics_* and restore_metrics_* in
// ape.toml filter the stats, the same way the namespace and node lists do.
func TestBackupRestoreAllowlistAndBlocklist(t *testing.T) {
	sharedState := useProcessorConfig(t)
	config.Cfg.Aerospike.BackupMetricsAllowlist = []string{"recs-*", "job"}
	config.Cfg.Aerospike.BackupMetricsBlocklist = []string{"recs-filtered"}
	config.Cfg.Aerospike.RestoreMetricsAllowlist = []string{"applied"}
	sharedState.ClusterName = "c1"
	sharedState.Service = "10.0.0.1:3000"
	bw := NewBackupRestoreStatsProcessor(sharedState)

	raw := map[string]string{
		KEY_BACKUP_STATUS:  "job-id=b1:ns=test:state=RUNNING:recs-backed-up=5:recs-filtered=2:bytes-backed-up=9",
		KEY_RESTORE_STATUS: "job-id=r1:ns=test:state=ACTIVE:applied=7:failed=3",
	}

	metrics, err := bw.Refresh(nil, raw)
	require.NoError(t, err)

	allowed := map[string]bool{}
	for _, m := range metrics {
		allowed[string(m.Context)+"/"+m.Name] = m.IsAllowed
	}

	// recs-* matches the allowlist, but recs-filtered is blocked; bytes-backed-up matches neither
	assert.True(t, allowed["backup/recs-backed-up"])
	assert.False(t, allowed["backup/recs-filtered"])
	assert.False(t, allowed["backup/bytes-backed-up"])

	// the job gauge is filtered too, since it is built through the same check
	assert.True(t, allowed["backup/job"])

	// restore has its own lists: applied is allowed, failed is not
	assert.True(t, allowed["restore/applied"])
	assert.False(t, allowed["restore/failed"])
	assert.False(t, allowed["restore/job"])
}
