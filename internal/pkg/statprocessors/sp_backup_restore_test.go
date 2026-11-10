package statprocessors

import (
	"testing"
	"time"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/stretchr/testify/assert"
)

func Test_BackupRestore_PassTwoKeysBuildGate(t *testing.T) {
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
			sharedState := NewStatProcessorSharedState()
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

func Test_BackupRestore_RefreshParsesJobs(t *testing.T) {
	sharedState := NewStatProcessorSharedState()
	sharedState.ClusterName = "c1"
	sharedState.Service = "10.0.0.1:3000"
	sharedState.Build = "8.2.1.0"
	bw := NewBackupRestoreStatsProcessor(sharedState)

	raw := map[string]string{
		KEY_BACKUP_STATUS: "job-id=b1:ns=test:state=RUNNING:progress-pct=42.50:change-stream-active=true:start-time=20260728T161500.250Z:finish-time=-;" +
			"job-id=b2:ns=test:state=ABORTED:start-time=2026-11-10 06.19.51.659 UTC:finish-time=2026-11-10 06.19.52.836 UTC:error-reason=job-id reuse prefix already holds a finished backup:abort-origin=server",
		KEY_RESTORE_STATUS: "ns=test:ns-state=NORMAL;job-id=r1:ns=test:kind=cold:ns-state=RESTORING:state=COMPLETE:progress-pct=100.00:applied=7:backup-id=daily1:cluster-key=3F6AF2E3A109",
	}

	metrics, err := bw.Refresh(nil, raw)
	assert.Nil(t, err)

	byKey := map[string]AerospikeStat{}
	for _, m := range metrics {
		byKey[string(m.Context)+"/"+m.Name+"/"+m.LabelValues[3]] = m
	}

	backupPct, ok := byKey["backup/progress-pct/b1"]
	assert.True(t, ok)
	assert.Equal(t, 42.5, backupPct.Value)
	assert.Equal(t, []string{"b1"}, backupPct.LabelValues[3:])

	// a running job carries only the state; error-reason is absent so it is not a label
	backupJob, ok := byKey["backup/job/b1"]
	assert.Equal(t, commons.MetricTypeGauge, backupJob.MType)
	assert.True(t, ok)
	assert.Equal(t, 1.0, backupJob.Value)
	assert.Equal(t, []string{commons.METRIC_LABEL_STATE}, backupJob.Labels[4:])
	assert.Equal(t, []string{"c1", "10.0.0.1:3000", "test", "b1", "RUNNING"}, backupJob.LabelValues)

	// a failed job adds error-reason as a label
	abortedJob, ok := byKey["backup/job/b2"]
	assert.True(t, ok)
	assert.Equal(t, []string{commons.METRIC_LABEL_STATE, commons.METRIC_LABEL_ERROR_REASON, commons.METRIC_LABEL_ABORT_ORIGIN}, abortedJob.Labels[4:])
	assert.Equal(t, []string{"ABORTED", "job-id reuse prefix already holds a finished backup", "server"}, abortedJob.LabelValues[4:])

	restoreJob, ok := byKey["restore/job/r1"]
	assert.Equal(t, commons.MetricTypeGauge, restoreJob.MType)
	assert.True(t, ok)
	assert.Equal(t, 1.0, restoreJob.Value)
	assert.Equal(t, []string{commons.METRIC_LABEL_STATE, commons.METRIC_LABEL_KIND, commons.METRIC_LABEL_NS_STATE, commons.METRIC_LABEL_BACKUP_ID, commons.METRIC_LABEL_CLUSTER_KEY}, restoreJob.Labels[4:])
	assert.Equal(t, []string{"c1", "10.0.0.1:3000", "test", "r1", "COMPLETE", "cold", "RESTORING", "daily1", "3F6AF2E3A109"}, restoreJob.LabelValues)

	// backup-id and the hex cluster-key are labels, never metrics
	_, hasBackupID := byKey["restore/backup-id/r1"]
	assert.False(t, hasBackupID)
	_, hasClusterKey := byKey["restore/cluster-key/r1"]
	assert.False(t, hasClusterKey)

	cs, ok := byKey["backup/change-stream-active/b1"]
	assert.True(t, ok)
	assert.Equal(t, 1.0, cs.Value)

	restoreApplied, ok := byKey["restore/applied/r1"]
	assert.True(t, ok)
	assert.Equal(t, 7.0, restoreApplied.Value)
	assert.Equal(t, []string{"c1", "10.0.0.1:3000", "test", "r1"}, restoreApplied.LabelValues)

	// timestamps become unix milliseconds; finish-time is "-" while the job is still running
	start, ok := byKey["backup/start-time/b1"]
	assert.True(t, ok)
	assert.Equal(t, float64(time.Date(2026, 7, 28, 16, 15, 0, int(250*time.Millisecond), time.UTC).UnixMilli()), start.Value)

	_, hasFinish := byKey["backup/finish-time/b1"]
	assert.False(t, hasFinish)

	// the live server prints "2026-11-10 06.19.51.659 UTC"
	liveStart, ok := byKey["backup/start-time/b2"]
	assert.True(t, ok)
	assert.Equal(t, float64(time.Date(2026, 11, 10, 6, 19, 51, int(659*time.Millisecond), time.UTC).UnixMilli()), liveStart.Value)

	// the bare ns row has no job-id, so it must not produce any metric
	_, hasNsState := byKey["restore/ns-state"]
	assert.False(t, hasNsState)
}
