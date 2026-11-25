package statprocessors

import (
	"os"
	"path/filepath"
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
			sharedState := backupState(t)
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
	sharedState := backupState(t)
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

// TestBackupRestoreUnderscoreFieldNames covers 8.2.2, which prints stats with underscores (start_time,
// progress_pct) and keeps hyphens on identifiers (job-id). Both spellings must give the same metrics.
func TestBackupRestoreUnderscoreFieldNames(t *testing.T) {
	sharedState := backupState(t)
	bw := NewBackupRestoreStatsProcessor(sharedState)

	raw := map[string]string{
		KEY_BACKUP_STATUS: "job-id=b1:ns=test:state=BASE_SCAN_ACTIVE:backup-rps=0:recs_backed_up=8989:progress_pct=29.52:" +
			"change_stream_active=false:start_time=2026-11-24 14.51.31.499 UTC:finish_time=-:achieved_consistency=strict",
		KEY_RESTORE_STATUS: "job-id=r1:ns=test:kind=cold:ns_state=RESTORING:state=ACTIVE:applied=7:backup_id=daily1:cluster_key=123456",
	}

	metrics, err := bw.Refresh(nil, raw)
	require.NoError(t, err)

	byKey := map[string]AerospikeStat{}
	for _, m := range metrics {
		byKey[string(m.Context)+"/"+m.Name] = m
	}

	start, ok := byKey["backup/start_time"]
	require.True(t, ok, "start_time must be read as a timestamp, like start-time")
	assert.Equal(t, float64(time.Date(2026, 11, 24, 14, 51, 31, int(499*time.Millisecond), time.UTC).UnixMilli()), start.Value)

	finish, ok := byKey["backup/finish_time"]
	require.True(t, ok, "finish_time=- is exported as 0, like finish-time")
	assert.Equal(t, 0.0, finish.Value)

	pct, ok := byKey["backup/progress_pct"]
	require.True(t, ok)
	assert.Equal(t, 29.52, pct.Value)
	assert.Equal(t, commons.MetricTypeGauge, pct.MType)

	// recs_backed_up only grows during a job, so it is a counter; backup-rps is a setting, so a gauge
	backedUp, ok := byKey["backup/recs_backed_up"]
	require.True(t, ok)
	assert.Equal(t, commons.MetricTypeCounter, backedUp.MType)
	rps, ok := byKey["backup/backup-rps"]
	require.True(t, ok)
	assert.Equal(t, commons.MetricTypeGauge, rps.MType)

	// restore stats stay gauges until their types are confirmed
	applied, ok := byKey["restore/applied"]
	require.True(t, ok)
	assert.Equal(t, commons.MetricTypeGauge, applied.MType)

	consistency, ok := byKey["backup/achieved_consistency"]
	require.True(t, ok, "the consistency gauge keeps the server's spelling")
	assert.Equal(t, 2.0, consistency.Value)
	assertLabel(t, consistency, commons.METRIC_LABEL_ACHIEVED_CONSISTENCY, "strict")
	assert.Equal(t, 1, countNamed(metrics, "achieved_consistency"), "exported once, as the consistency gauge")
	assert.Zero(t, countNamed(metrics, STAT_ACHIEVED_CONSISTENCY))

	restoreJob, ok := byKey["restore/job"]
	require.True(t, ok)
	assert.Equal(t, []string{commons.METRIC_LABEL_STATE, commons.METRIC_LABEL_KIND, commons.METRIC_LABEL_NS_STATE, commons.METRIC_LABEL_BACKUP_ID, commons.METRIC_LABEL_CLUSTER_KEY}, restoreJob.Labels[4:])
	assert.Equal(t, []string{"ACTIVE", "cold", "RESTORING", "daily1", "123456"}, restoreJob.LabelValues[4:])

	_, ok = byKey["restore/cluster_key"]
	assert.False(t, ok, "an all-digit cluster_key is a label, never a metric")
	_, ok = byKey["restore/backup_id"]
	assert.False(t, ok)
}

// TestBackupRestoreErrorTextInsideJob checks that a job whose error-reason contains "error:" does not
// make the whole response look like a failed command, while a real command error still emits nothing.
func TestBackupRestoreErrorTextInsideJob(t *testing.T) {
	bw := NewBackupRestoreStatsProcessor(backupState(t))

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
	sharedState := backupState(t)
	config.Cfg.Aerospike.BackupMetricsAllowlist = []string{"recs_*", "job", "achieved_consistency"}
	config.Cfg.Aerospike.BackupMetricsBlocklist = []string{"recs_filtered"}
	config.Cfg.Aerospike.RestoreMetricsAllowlist = []string{"applied"}
	sharedState.ClusterName = "c1"
	sharedState.Service = "10.0.0.1:3000"
	bw := NewBackupRestoreStatsProcessor(sharedState)

	raw := map[string]string{
		KEY_BACKUP_STATUS:  "job-id=b1:ns=test:state=RUNNING:recs_backed_up=5:recs_filtered=2:bytes_backed_up=9:achieved_consistency=-",
		KEY_RESTORE_STATUS: "job-id=r1:ns=test:state=ACTIVE:applied=7:failed=3",
	}

	metrics, err := bw.Refresh(nil, raw)
	require.NoError(t, err)

	allowed := map[string]bool{}
	for _, m := range metrics {
		allowed[string(m.Context)+"/"+m.Name] = m.IsAllowed
	}

	// patterns use the server's spelling: recs_* matches the allowlist, but recs_filtered is blocked;
	// bytes_backed_up matches neither
	assert.True(t, allowed["backup/recs_backed_up"])
	assert.False(t, allowed["backup/recs_filtered"])
	assert.False(t, allowed["backup/bytes_backed_up"])
	assert.True(t, allowed["backup/achieved_consistency"])

	// the job gauge is filtered too, since it is built through the same check
	assert.True(t, allowed["backup/job"])

	// restore has its own lists: applied is allowed, failed is not
	assert.True(t, allowed["restore/applied"])
	assert.False(t, allowed["restore/failed"])
	assert.False(t, allowed["restore/job"])
}

// TestBackupRestoreMetricType checks the types come from the shipped gauge_stats_list.toml: listed stats are
// gauges and unlisted ones are counters, under either spelling. Restore is list-driven too, so an unlisted
// restore stat is a counter.
func TestBackupRestoreMetricType(t *testing.T) {
	backupState(t)

	for _, stat := range []string{"recs_backed_up", "recs-backed-up", "bytes_base", "partitions_scanned", "partitions-flushed", "backpressure_activations", "upload_permanent_failures"} {
		assert.Equal(t, commons.MetricTypeCounter, backupRestoreMetricType(commons.CTX_BACKUP, stat), stat)
	}

	for _, stat := range []string{"progress_pct", "progress-pct", "total_buffers_allocated", "partitions_owned", "recs_per_sec", "upload_q_depth", "backup-rps", "start_time", "finish_time", "change_stream_active"} {
		assert.Equal(t, commons.MetricTypeGauge, backupRestoreMetricType(commons.CTX_BACKUP, stat), stat)
	}

	assert.Equal(t, commons.MetricTypeGauge, backupRestoreMetricType(commons.CTX_RESTORE, "applied"))
	assert.Equal(t, commons.MetricTypeGauge, backupRestoreMetricType(commons.CTX_RESTORE, "progress_pct"))
	assert.Equal(t, commons.MetricTypeCounter, backupRestoreMetricType(commons.CTX_RESTORE, "failed"),
		"a restore stat that is not listed is a counter")
}

// TestBackupRestoreShippedGaugeList checks the gauge_stats_list.toml that ships with the exporter, which
// customers can edit, lists the stats the processor is expected to export as gauges.
func TestBackupRestoreShippedGaugeList(t *testing.T) {
	backupState(t)

	assert.True(t, isGauge(commons.CTX_BACKUP, "progress-pct"))
	assert.True(t, isGauge(commons.CTX_BACKUP, "start-time"), "start_time is written with underscores in the file")
	assert.True(t, isGauge(commons.CTX_RESTORE, "progress-pct"))
	assert.False(t, isGauge(commons.CTX_BACKUP, "recs-backed-up"), "the counters are not listed")
	assert.False(t, isGauge(commons.CTX_BACKUP, "partitions-scanned"))
}

// TestBackupRestoreSkippedDuringCheckpointSave checks that backup-status and restore-status are not
// requested while a checkpoint save runs, like the other processors.
func TestBackupRestoreSkippedDuringCheckpointSave(t *testing.T) {
	state := backupState(t)
	state.Build = "8.2.2.0"
	bw := NewBackupRestoreStatsProcessor(state)

	assert.Equal(t, []string{KEY_BACKUP_STATUS, KEY_RESTORE_STATUS}, bw.PassTwoKeys(nil))

	state.CheckpointSaveInProgress = true
	assert.Nil(t, bw.PassTwoKeys(nil))
}

// TestBackupRestoreLiveBackupRecord feeds the processor a whole backup-status record captured from an
// 8.2.2.0 server. Every numeric stat must be exported under the server's spelling, with the types
// agreed so far, and the text fields must not leak out as metrics.
func TestBackupRestoreLiveBackupRecord(t *testing.T) {
	state := backupState(t)
	bw := NewBackupRestoreStatsProcessor(state)

	metrics, err := bw.Refresh(nil, map[string]string{KEY_BACKUP_STATUS: backupStatus822})
	require.NoError(t, err)

	got := map[string]AerospikeStat{}
	for _, m := range metrics {
		require.Equal(t, commons.CTX_BACKUP, m.Context)
		got[m.Name] = m
	}

	// every field of the record is a metric, except the identifiers and the text fields
	notMetrics := map[string]bool{"job-id": true, "ns": true, "state": true, "backup-policy": true}
	for name, value := range commons.ParseStats(backupStatus822, ":") {
		if notMetrics[name] {
			continue
		}
		_, exported := got[name]
		assert.True(t, exported, "%s=%s is missing from the output", name, value)
	}
	assert.Equal(t, 38, len(got), "37 stats plus the job gauge")
	_, ok := got["backup-policy"]
	assert.False(t, ok, "a text setting is not a metric")

	counters := 0
	for name, m := range got {
		if name == STAT_JOB || name == "achieved_consistency" {
			continue
		}
		listed := isGauge(commons.CTX_BACKUP, fieldKey(name))
		assert.Equal(t, listed, m.MType == commons.MetricTypeGauge, "%s is typed by the gauge list", name)
		if m.MType == commons.MetricTypeCounter {
			counters++
		}
	}
	assert.Equal(t, 18, counters, "the 8.2.2 record has 18 counters")

	for _, name := range []string{"progress_pct", "total_buffers_allocated", "recs_per_sec", "upload_q_depth", "backup-rps", "change_stream_active"} {
		assert.Equal(t, commons.MetricTypeGauge, got[name].MType, name)
	}

	assert.Equal(t, 29.52, got["progress_pct"].Value)
	assert.Equal(t, 144.0, got["total_buffers_allocated"].Value)
	assert.Equal(t, 0.0, got["backpressure_active"].Value)
	assert.Equal(t, 1795531891499.0, got["start_time"].Value)
	assertLabel(t, got[STAT_JOB], commons.METRIC_LABEL_STATE, "BASE_SCAN_ACTIVE")
	assertLabel(t, got["recs_backed_up"], commons.METRIC_LABEL_JOB_ID, "261124T145131-hhj8")
}

// backupState resets the processor config, then loads the gauge list that ships with the exporter, so the
// backup and restore types are the ones customers get.
func backupState(t *testing.T) *StatProcessorSharedState {
	t.Helper()

	state := useProcessorConfig(t)

	original := config.GaugeStatHandler
	t.Cleanup(func() { config.GaugeStatHandler = original })

	cwd, err := os.Getwd()
	require.NoError(t, err)
	config.InitGaugeStats(filepath.Join(cwd, "..", "..", "..", "configs", "gauge_stats_list.toml"))

	return state
}

// TestBackupRestoreSourceConsistencyIsLabel checks restore-status from the 8.2.1+ build, where source_consistency
// is the consistency a restore was asked for. Like achieved_consistency it is a gauge valued by the level, with
// the level as a label. The record also covers the restore settings, flags and levels that the shipped list types as gauges.
func TestBackupRestoreSourceConsistencyIsLabel(t *testing.T) {
	state := backupState(t)
	bw := NewBackupRestoreStatsProcessor(state)

	record := "job-id=261125T073858-k16l:ns=test:kind=cold:state=ACTIVE:restore-rps=0:restore-worker-threads=8:" +
		"restore-download-threads=8:ns_state=RESTORING:start_time=2026-11-25 07.38.58.774 UTC:joiner=false:" +
		"placeholder=false:pids_in_scope=4096:progress_pct=66.74:missing_layout_nodes=0:n_done_nodes=0:" +
		"source_consistency=strict:ref_time=533288338:cluster_key=BF60EE7585F1:node_done=false:done=false:" +
		"clear=none:clear_zeroed_bytes=134217728:clear_ms=6875:total=2846:applied=2598:deletes=0:filtered=248:" +
		"filtered_not_ours=0:filter_exits=0:exit_shadowed=0:exit_cap_hit=false:failed=0:recs_per_sec=117:" +
		"pending_q_depth=0:pending_q_cap=16:last_lap_ms=24"

	metrics, err := bw.Refresh(nil, map[string]string{KEY_RESTORE_STATUS: record})
	require.NoError(t, err)

	byKey := map[string]AerospikeStat{}
	for _, m := range metrics {
		byKey[string(m.Context)+"/"+m.Name] = m
	}

	source, ok := byKey["restore/source_consistency"]
	require.True(t, ok, "source_consistency is exported under the server's spelling")
	assert.Equal(t, 2.0, source.Value, "strict is level 2")
	assert.Equal(t, commons.MetricTypeGauge, source.MType)
	assertLabel(t, source, commons.METRIC_LABEL_SOURCE_CONSISTENCY, "strict")
	assert.Zero(t, countNamed(metrics, "source-consistency"))
	assert.Equal(t, 1, countNamed(metrics, "source_consistency"), "exported once, as the consistency gauge")

	// clear=none is text, so it is not a metric and not a label
	_, ok = byKey["restore/clear"]
	assert.False(t, ok)

	for _, name := range []string{"restore-rps", "restore-worker-threads", "restore-download-threads", "start_time",
		"pids_in_scope", "placeholder", "node_done", "exit_cap_hit", "recs_per_sec", "last_lap_ms"} {
		stat, ok := byKey["restore/"+name]
		require.True(t, ok, "%s is exported", name)
		assert.Equal(t, commons.MetricTypeGauge, stat.MType, "%s is a gauge in the shipped list", name)
	}
}

// TestBackupRestoreSourceConsistencyLevels checks the level for each source_consistency value, including "-".
func TestBackupRestoreSourceConsistencyLevels(t *testing.T) {
	state := backupState(t)
	bw := NewBackupRestoreStatsProcessor(state)

	cases := map[string]float64{"-": 0, "relaxed": 1, "strict": 2}
	for value, level := range cases {
		metrics, err := bw.Refresh(nil, map[string]string{
			KEY_RESTORE_STATUS: "job-id=r1:ns=test:state=ACTIVE:source_consistency=" + value,
		})
		require.NoError(t, err)

		source, ok := findStat(metrics, "source_consistency")
		require.True(t, ok, "value %s", value)
		assert.Equal(t, level, source.Value, "value %s", value)
		assertLabel(t, source, commons.METRIC_LABEL_SOURCE_CONSISTENCY, value)
	}
}

// TestBackupRestoreFinishedRestoreJob checks a restore-status record after the job completed. finish_time is
// now present, so it is exported like the backup finish_time (a gauge, unix milliseconds). The job is a gauge
// in state COMPLETE, and done, node_done and the source consistency level are exported.
func TestBackupRestoreFinishedRestoreJob(t *testing.T) {
	state := backupState(t)
	bw := NewBackupRestoreStatsProcessor(state)

	record := "job-id=261125T073858-k16l:ns=test:kind=cold:state=COMPLETE:restore-rps=0:restore-worker-threads=8:" +
		"restore-download-threads=8:ns_state=NONE:start_time=2026-11-25 07.38.58.774 UTC:" +
		"finish_time=2026-11-25 07.48.16.841 UTC:joiner=false:placeholder=false:pids_in_scope=4096:" +
		"progress_pct=100.00:missing_layout_nodes=0:n_done_nodes=1:source_consistency=strict:ref_time=533288338:" +
		"cluster_key=BF60EE7585F1:node_done=true:done=true:clear=none:clear_zeroed_bytes=134217728:clear_ms=6875:" +
		"total=9839:applied=8989:deletes=0:filtered=850:filtered_not_ours=0:filter_exits=0:exit_shadowed=0:" +
		"exit_cap_hit=false:failed=0:recs_per_sec=95:pending_q_depth=0:pending_q_cap=0"

	metrics, err := bw.Refresh(nil, map[string]string{KEY_RESTORE_STATUS: record})
	require.NoError(t, err)

	byKey := map[string]AerospikeStat{}
	for _, m := range metrics {
		byKey[string(m.Context)+"/"+m.Name] = m
	}

	finish, ok := byKey["restore/finish_time"]
	require.True(t, ok, "finish_time is exported once the job has finished")
	assert.Equal(t, commons.MetricTypeGauge, finish.MType)
	assert.Equal(t, float64(time.Date(2026, 11, 25, 7, 48, 16, int(841*time.Millisecond), time.UTC).UnixMilli()), finish.Value)

	done, ok := byKey["restore/done"]
	require.True(t, ok)
	assert.Equal(t, 1.0, done.Value)

	job, ok := byKey["restore/job"]
	require.True(t, ok)
	assert.Equal(t, []string{"COMPLETE", "cold", "NONE", "BF60EE7585F1"}, job.LabelValues[4:])
}
