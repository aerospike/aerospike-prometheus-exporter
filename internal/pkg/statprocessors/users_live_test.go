//go:build integration

package statprocessors

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	aero "github.com/aerospike/aerospike-client-go/v8"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLiveUserStatsAfterAsbench checks the user processor against a secured
// server. Compose creates u13user, phani, and u1phani, then asbench runs as
// each of them. The check waits until those sessions are visible.
func TestLiveUserStatsAfterAsbench(t *testing.T) {
	state := useProcessorConfig(t)
	config.Cfg.Aerospike.User = os.Getenv("AEROSPIKE_USER")
	config.Cfg.Aerospike.Password = os.Getenv("AEROSPIKE_PASSWORD")
	require.NotEmpty(t, config.Cfg.Aerospike.User)

	conn := dialAerospike(t)
	defer conn.Close()

	base := requestInfo(t, conn, "build", "cluster-name", "service-clear-std")
	state.Build = strings.TrimSpace(base["build"])
	state.ClusterName = strings.TrimSpace(base["cluster-name"])
	state.Service = strings.TrimSpace(base["service-clear-std"])
	require.NotEmpty(t, state.Build)
	require.Equal(t, "ape-users-it", state.ClusterName)

	processor := NewUserStatsProcessor(state)
	require.True(t, processor.canRefreshUserStats(nil, state.Build))

	hostName, port := aerospikeEndpoint(t)
	policy := aero.NewClientPolicy()
	policy.Timeout = 30 * time.Second
	policy.User = config.Cfg.Aerospike.User
	policy.Password = config.Cfg.Aerospike.Password
	client, err := aero.NewClientWithPolicy(policy, hostName, port)
	require.NoError(t, err)
	t.Cleanup(client.Close)

	want := []string{"u13user", "phani", "u1phani"}
	deadline := time.Now().Add(30 * time.Second)
	var stats []AerospikeStat
	var last string
	for time.Now().Before(deadline) {
		users, queryErr := client.QueryUsers(nil)
		require.NoError(t, queryErr)
		last = describeUsers(users)
		if benchUsersVisible(users, want) {
			var refreshErr error
			stats, refreshErr = processor.Refresh(users)
			require.NoError(t, refreshErr)
			break
		}
		time.Sleep(time.Second)
	}
	require.NotEmpty(t, stats, "user stats were not visible during asbench: %s", last)

	for _, name := range want {
		conns, ok := findStatWithLabel(stats, "conns_in_use", commons.METRIC_LABEL_USER, name)
		require.True(t, ok, name)
		assert.Greater(t, conns.Value, 0.0, name)
		assert.Equal(t, commons.CTX_USERS, conns.Context)
		assertLabel(t, conns, commons.METRIC_LABEL_CLUSTER_NAME, "ape-users-it")
		assertLabel(t, conns, commons.METRIC_LABEL_SERVICE, state.Service)

		readTPS, ok := findStatWithLabel(stats, "read_single_record_tps", commons.METRIC_LABEL_USER, name)
		require.True(t, ok, name)
		assert.Greater(t, readTPS.Value, 0.0, name)
		writeTPS, ok := findStatWithLabel(stats, "write_single_record_tps", commons.METRIC_LABEL_USER, name)
		require.True(t, ok, name)
		assert.Greater(t, writeTPS.Value, 0.0, name)

		// The created roles have no rate quota, so both quotas are 0 and still emitted.
		for _, statName := range []string{"read_quota", "write_quota", "read_scan_query_rps", "write_scan_query_rps", "limitless_read_scan_query", "limitless_write_scan_query"} {
			stat, found := findStatWithLabel(stats, statName, commons.METRIC_LABEL_USER, name)
			require.True(t, found, "%s %s", name, statName)
			assert.Equal(t, 0.0, stat.Value, "%s %s", name, statName)
		}
	}
}

func benchUsersVisible(users []*aero.UserRoles, want []string) bool {
	byName := make(map[string]*aero.UserRoles, len(users))
	for _, user := range users {
		byName[user.User] = user
	}
	for _, name := range want {
		user := byName[name]
		if user == nil || user.ConnsInUse <= 0 || len(user.ReadInfo) < 4 || len(user.WriteInfo) < 4 {
			return false
		}
		if user.ReadInfo[1] <= 0 || user.WriteInfo[1] <= 0 {
			return false
		}
	}
	return true
}

func describeUsers(users []*aero.UserRoles) string {
	parts := make([]string, 0, len(users))
	for _, user := range users {
		parts = append(parts, fmt.Sprintf("%s conns=%d read=%v write=%v", user.User, user.ConnsInUse, user.ReadInfo, user.WriteInfo))
	}
	return strings.Join(parts, "; ")
}
