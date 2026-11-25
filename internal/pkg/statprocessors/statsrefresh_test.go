package statprocessors

import (
	"strings"
	"testing"

	aero "github.com/aerospike/aerospike-client-go/v8"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Info output captured from a live 8.2.2.0 enterprise server. checkpoint-status starts with a node:
// record, and backup-status prints stats with underscores.
const (
	checkpointStatus822 = "node:is_parked=false:park_ms=0;" +
		"test:state=none:files_completed=0:files_total=0;" +
		"test_on_file:state=none:files_completed=0:files_total=0;" +
		"bar:state=none:files_completed=0:files_total=0;" +
		"bar_pmem:state=none:files_completed=0:files_total=0"

	backupStatus822 = "job-id=261124T145131-hhj8:ns=test:state=BASE_SCAN_ACTIVE:backup-rps=0:backup-worker-threads=4:" +
		"backup-upload-threads=8:backup-s3-throughput-target-gbps=10:backup-policy=strict:recs_backed_up=8989:recs_base=8989:" +
		"recs_query_incr=0:recs_change_incr=0:bytes_backed_up=750832:bytes_base=750832:bytes_query_incr=0:bytes_change_incr=0:" +
		"recs_filtered=0:recs_filter_exits=0:recs_degenerate=0:recs_skipped_xdr_tomb=0:recs_read_failed=0:partitions_flushed=2418:" +
		"partitions_owned=4096:partitions_scanned=4096:progress_pct=29.52:change_stream_active=false:start_time=2026-11-24 14.51.31.499 UTC:" +
		"partitions_count_pending=0:count_read_failures=0:backpressure_active=false:backpressure_activations=14:bytes_in_per_sec=0:" +
		"bytes_out_per_sec=0:upload_q_depth=0:upload_q_cap=128:buffers_in_use=0:total_buffers_allocated=144:recs_per_sec=0:" +
		"buffers_scan_cap=144:last_lap_ms=245:upload_permanent_failures=0"

	// the build reply while a checkpoint save runs. The exporter matches on the marker text only.
	buildDuringCheckpointSave = "ERROR::checkpoint-save in progress"
)

// fakeServer answers info requests for a StatsRefresher and records every key it was asked for.
type fakeServer struct {
	responses map[string]string
	asked     [][]string
}

func newFakeServer(build string, checkpointStatus string) *fakeServer {
	return &fakeServer{responses: map[string]string{
		"build":                       build,
		"cluster-name":                "c1",
		"service-clear-std":           "10.0.0.1:3000",
		"node:":                       "BB9",
		KEY_SERVICE_LOGS:              "0:stderr",
		KEY_SERVICE_CONFIG:            "proto-fd-max=15000",
		KEY_SERVICE_STATISTICS:        "client_connections=3",
		CMD_INFOKEY_CHECKPOINT_STATUS: checkpointStatus,
		KEY_BACKUP_STATUS:             backupStatus822,
		KEY_RESTORE_STATUS:            "ERROR:4:no restore jobs",
	}}
}

func (f *fakeServer) RequestInfo(keys []string) (map[string]string, error) {
	f.asked = append(f.asked, keys)

	out := map[string]string{}
	for _, key := range keys {
		out[key] = f.responses[key]
	}

	return out, nil
}

func (f *fakeServer) FetchUsersDetails() (bool, []*aero.UserRoles, error) { return false, nil, nil }
func (f *fakeServer) IsServerConnected() bool                             { return true }

// lastPassTwo returns the keys of the most recent second info request, the one that fetches stats.
func (f *fakeServer) lastPassTwo() []string {
	return f.asked[len(f.asked)-1]
}

func hasLabelValue(stat AerospikeStat, label string, value string) bool {
	for i, name := range stat.Labels {
		if name == label && stat.LabelValues[i] == value {
			return true
		}
	}

	return false
}

// TestStatsRefresherEightTwoTwoFlow runs a full refresh against 8.2.2 output: backup metrics are exported,
// and the node: record in checkpoint-status does not stop the node stats.
func TestStatsRefresherEightTwoTwoFlow(t *testing.T) {
	state := backupState(t)
	server := newFakeServer("8.2.2.0", checkpointStatus822)

	stats, err := NewStatsRefresher(server, state).Refresh()
	require.NoError(t, err)

	keys := server.lastPassTwo()
	assert.Contains(t, keys, KEY_BACKUP_STATUS)
	assert.Contains(t, keys, KEY_RESTORE_STATUS)
	assert.Contains(t, keys, CMD_INFOKEY_CHECKPOINT_STATUS)

	_, ok := findStat(stats, "client_connections")
	assert.True(t, ok, "an idle node: record must not be read as checkpoint-shutdown")

	for _, stat := range stats {
		if strings.HasPrefix(stat.Name, "checkpoint_") {
			assert.False(t, hasLabelValue(stat, commons.METRIC_LABEL_NS, "node"), "node: is not a namespace: %s", stat.Name)
		}
	}
	files, ok := findStatWithLabel(stats, "checkpoint_files_total", commons.METRIC_LABEL_NS, "bar_pmem")
	require.True(t, ok)
	assertLabel(t, files, commons.METRIC_LABEL_CHECKPOINT_STATUS, "none")

	job, ok := findStat(stats, STAT_JOB)
	require.True(t, ok)
	assert.Equal(t, commons.CTX_BACKUP, job.Context)
	assertLabel(t, job, commons.METRIC_LABEL_STATE, "BASE_SCAN_ACTIVE")

	start, ok := findStat(stats, "start_time")
	require.True(t, ok, "start_time is read as a timestamp")
	assert.Equal(t, 1795531891499.0, start.Value)

	for _, stat := range stats {
		assert.NotEqual(t, commons.CTX_RESTORE, stat.Context, "a restore error reply is not a job: %s", stat.Name)
	}
}

// TestStatsRefresherCheckpointSaveFlow follows a server that goes into a checkpoint save after a normal
// refresh: only build and checkpoint-status are requested, so nothing else, including backup and
// restore, is sent to a server that cannot answer it. Everything comes back once the save is over.
func TestStatsRefresherCheckpointSaveFlow(t *testing.T) {
	state := backupState(t)
	server := newFakeServer("8.2.2.0", checkpointStatus822)
	refresher := NewStatsRefresher(server, state)

	_, err := refresher.Refresh()
	require.NoError(t, err)
	assert.Contains(t, server.lastPassTwo(), KEY_BACKUP_STATUS)

	// the save starts: build now reports it, and checkpoint-status shows the namespace saving
	server.responses["build"] = buildDuringCheckpointSave
	server.responses[CMD_INFOKEY_CHECKPOINT_STATUS] = "node:is_parked=false:park_ms=0;" +
		"test:state=saving:files_completed=2:files_total=4;bar:state=none:files_completed=0:files_total=0"

	stats, err := refresher.Refresh()
	require.NoError(t, err)

	keys := server.lastPassTwo()
	assert.Contains(t, keys, CMD_INFOKEY_CHECKPOINT_STATUS)
	for _, notWanted := range []string{KEY_BACKUP_STATUS, KEY_RESTORE_STATUS, KEY_SERVICE_STATISTICS, KEY_SERVICE_CONFIG} {
		assert.NotContains(t, keys, notWanted, "only build and checkpoint-status are requested during a save")
	}

	saving, ok := findStatWithLabel(stats, "checkpoint_files_completed", commons.METRIC_LABEL_NS, "test")
	require.True(t, ok, "checkpoint metrics keep publishing during a save")
	assert.Equal(t, 2.0, saving.Value)
	assertLabel(t, saving, commons.METRIC_LABEL_CHECKPOINT_STATUS, "saving")

	_, ok = findStat(stats, "client_connections")
	assert.False(t, ok, "node stats are not refreshed while a namespace is saving")
	assert.Zero(t, countNamed(stats, STAT_JOB), "no backup metrics during a save")

	// the save is over
	server.responses["build"] = "8.2.2.0"
	server.responses[CMD_INFOKEY_CHECKPOINT_STATUS] = checkpointStatus822

	stats, err = refresher.Refresh()
	require.NoError(t, err)
	assert.Contains(t, server.lastPassTwo(), KEY_BACKUP_STATUS)
	_, ok = findStat(stats, "client_connections")
	assert.True(t, ok)
	assert.Equal(t, 1, countNamed(stats, STAT_JOB))
}

// TestStatsRefresherBeforeBackupRestoreBuild checks that a server too old for backup and restore is never
// asked for it, and still gets its node stats from the older checkpoint-status format, where is_parked is
// on every namespace record and there is no node: record.
func TestStatsRefresherBeforeBackupRestoreBuild(t *testing.T) {
	state := backupState(t)
	server := newFakeServer("8.2.0.0", "test:state=none:files_completed=0:files_total=0:is_parked=false:park_ms=0;"+
		"bar:state=none:files_completed=0:files_total=0:is_parked=false:park_ms=0")

	stats, err := NewStatsRefresher(server, state).Refresh()
	require.NoError(t, err)

	keys := server.lastPassTwo()
	assert.NotContains(t, keys, KEY_BACKUP_STATUS)
	assert.NotContains(t, keys, KEY_RESTORE_STATUS)
	assert.Contains(t, keys, CMD_INFOKEY_CHECKPOINT_STATUS)

	_, ok := findStat(stats, "client_connections")
	assert.True(t, ok)
	_, ok = findStatWithLabel(stats, "checkpoint_park_ms", commons.METRIC_LABEL_NS, "test")
	assert.True(t, ok, "the older format still reports park_ms for each namespace")
	assert.Zero(t, countNamed(stats, STAT_JOB))
}
