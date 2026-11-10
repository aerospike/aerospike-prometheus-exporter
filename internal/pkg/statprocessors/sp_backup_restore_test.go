package statprocessors

import (
	"testing"

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
		KEY_BACKUP_STATUS:  "job-id=b1:ns=test:state=RUNNING:progress-pct=42.50:change-stream-active=true:start-time=20260728T161500.250Z",
		KEY_RESTORE_STATUS: "ns=test:ns-state=NORMAL;job-id=r1:ns=test:kind=cold:state=COMPLETE:progress-pct=100.00:applied=7",
	}

	metrics, err := bw.Refresh(nil, raw)
	assert.Nil(t, err)

	byKey := map[string]AerospikeStat{}
	for _, m := range metrics {
		byKey[string(m.Context)+"/"+m.Name] = m
	}

	backupPct, ok := byKey["backup/progress-pct"]
	assert.True(t, ok)
	assert.Equal(t, 42.5, backupPct.Value)
	assert.Equal(t, []string{"b1"}, backupPct.LabelValues[3:])

	// job-id is its own gauge carrying the state, so jobs can be counted per stage
	backupJob, ok := byKey["backup/job"]
	assert.Equal(t, commons.MetricTypeGauge, backupJob.MType)
	assert.True(t, ok)
	assert.Equal(t, 1.0, backupJob.Value)
	assert.Equal(t, []string{"c1", "10.0.0.1:3000", "test", "b1", "RUNNING"}, backupJob.LabelValues)

	restoreJob, ok := byKey["restore/job"]
	assert.Equal(t, commons.MetricTypeGauge, restoreJob.MType)
	assert.True(t, ok)
	assert.Equal(t, 1.0, restoreJob.Value)
	assert.Equal(t, []string{"c1", "10.0.0.1:3000", "test", "r1", "COMPLETE"}, restoreJob.LabelValues)

	cs, ok := byKey["backup/change-stream-active"]
	assert.True(t, ok)
	assert.Equal(t, 1.0, cs.Value)

	restoreApplied, ok := byKey["restore/applied"]
	assert.True(t, ok)
	assert.Equal(t, 7.0, restoreApplied.Value)
	assert.Equal(t, []string{"c1", "10.0.0.1:3000", "test", "r1"}, restoreApplied.LabelValues)

	// the bare ns row has no job-id, so it must not produce any metric
	_, hasNsState := byKey["restore/ns-state"]
	assert.False(t, hasNsState)
}
