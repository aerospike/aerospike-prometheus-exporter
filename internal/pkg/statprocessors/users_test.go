package statprocessors

import (
	"testing"

	aero "github.com/aerospike/aerospike-client-go/v8"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testUser(name string, conns int) *aero.UserRoles {
	return &aero.UserRoles{
		User:       name,
		ConnsInUse: conns,
		ReadInfo:   []int{100, 2, 3, 0},
		WriteInfo:  []int{80, 1, 4, 0},
	}
}

func TestUserStatsProcessorCanRefresh(t *testing.T) {
	state := useProcessorConfig(t)
	processor := NewUserStatsProcessor(state)

	assert.False(t, processor.canRefreshUserStats(nil, "6.0.0.0"))

	config.Cfg.Aerospike.User = "admin"
	config.Cfg.Aerospike.Password = "secret"
	assert.False(t, processor.canRefreshUserStats(nil, "5.5.0.0"))
	assert.False(t, processor.ShouldFetchUserStatistics)

	processor = NewUserStatsProcessor(state)
	assert.True(t, processor.canRefreshUserStats(nil, "5.6.0.0"))

	config.Cfg.Aerospike.AuthMode = "pki"
	config.Cfg.Aerospike.User = ""
	config.Cfg.Aerospike.Password = ""
	assert.True(t, processor.canRefreshUserStats(nil, "8.0.0.0"))

	state.CheckpointSaveInProgress = true
	assert.False(t, processor.canRefreshUserStats(nil, "8.0.0.0"))
}

func TestUserStatsProcessorRefreshAppliesUserLists(t *testing.T) {
	state := useProcessorConfig(t)
	config.Cfg.Aerospike.UserMetricsUsersAllowlistEnabled = true
	config.Cfg.Aerospike.UserMetricsUsersAllowlist = []string{"reader", "writer"}
	config.Cfg.Aerospike.UserMetricsUsersBlocklist = []string{"writer"}

	processor := NewUserStatsProcessor(state)
	stats, err := processor.Refresh([]*aero.UserRoles{
		testUser("reader", 5),
		testUser("writer", 9),
		testUser("other", 1),
	})
	require.NoError(t, err)

	conns, ok := findStat(stats, "conns_in_use")
	require.True(t, ok)
	assert.Equal(t, 1, countNamed(stats, "conns_in_use"))
	assert.Equal(t, 5.0, conns.Value)
	assert.Equal(t, commons.CTX_USERS, conns.Context)
	assert.Equal(t, commons.MetricTypeGauge, metricTypeOf(conns))
	assertLabel(t, conns, commons.METRIC_LABEL_USER, "reader")

	// writer is on both lists. The blocklist drops that user.
	_, ok = findStatWithLabel(stats, "conns_in_use", commons.METRIC_LABEL_USER, "writer")
	assert.False(t, ok)
	_, ok = findStatWithLabel(stats, "conns_in_use", commons.METRIC_LABEL_USER, "other")
	assert.False(t, ok)

	quota, ok := findStat(stats, "read_quota")
	require.True(t, ok)
	assert.Equal(t, 100.0, quota.Value)
	write, ok := findStat(stats, "write_single_record_tps")
	require.True(t, ok)
	assert.Equal(t, 1.0, write.Value)
}
