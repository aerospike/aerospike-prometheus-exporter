package statprocessors

import (
	"strings"
	"testing"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	log "github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSindexInfoKey = "sindex-stat:namespace=test;indexname=idx_state"

// refreshSindex runs one Refresh for a single sindex with the given raw sindex-stat response.
// It also returns the log entries written during that Refresh only, not during config setup.
func refreshSindex(t *testing.T, response string) ([]AerospikeStat, []*log.Entry) {
	t.Helper()

	commons.InitConfigurations(commons.GetWatchersConfigFile(commons.TESTS_DEFAULT_CONFIG_FILE))

	state := NewStatProcessorSharedState()
	state.ClusterName = "c1"
	state.Service = "127.0.0.1:3000"

	siw := NewSindexStatsProcessor(state)

	// Debug so the test also sees what is logged below warning level.
	previous := log.GetLevel()
	log.SetLevel(log.DebugLevel)
	hook := logtest.NewGlobal()
	t.Cleanup(func() {
		log.SetLevel(previous)
		hook.Reset()
	})

	stats, err := siw.Refresh([]string{testSindexInfoKey}, map[string]string{testSindexInfoKey: response})
	require.NoError(t, err)

	return stats, hook.AllEntries()
}

// equalityFilterState returns the emitted equality_filter_state series, if any.
func equalityFilterState(stats []AerospikeStat) (AerospikeStat, bool) {
	for _, stat := range stats {
		if stat.Name == STAT_SINDEX_EQUALITY_FILTER_STATE {
			return stat, true
		}
	}

	return AerospikeStat{}, false
}

func TestSindexEqualityFilterStateIsNumeric(t *testing.T) {
	cases := []struct {
		value string
		want  float64
	}{
		{"off", 0},
		{"building", 1},
		{"ready", 2},
		{"READY", 2},
		{"Building", 1},
		{" ready ", 2},
	}

	for _, tc := range cases {
		t.Run(tc.value, func(t *testing.T) {
			stats, _ := refreshSindex(t, "keys=10;equality_filter_state="+tc.value)

			stat, found := equalityFilterState(stats)
			require.True(t, found, "equality_filter_state=%q must emit a series", tc.value)
			assert.Equal(t, tc.want, stat.Value)
			assert.Equal(t, []string{"c1", "127.0.0.1:3000", "test", "idx_state"}, stat.LabelValues)
		})
	}
}

func TestSindexEqualityFilterStateUnknownEmitsNothingAndLogs(t *testing.T) {
	for _, value := range []string{"rebuilding", "unknown", "2", "", "off-ish"} {
		t.Run("value="+value, func(t *testing.T) {
			stats, entries := refreshSindex(t, "keys=10;equality_filter_state="+value)

			_, found := equalityFilterState(stats)
			assert.False(t, found, "an unknown state must not emit a series")

			// Other stats in the same response are not affected.
			var keys bool
			for _, stat := range stats {
				keys = keys || stat.Name == "keys"
			}
			assert.True(t, keys, "numeric stats still emitted")

			var warned bool
			for _, entry := range entries {
				if entry.Level == log.WarnLevel &&
					strings.Contains(entry.Message, "test:idx_state") &&
					strings.Contains(entry.Message, STAT_SINDEX_EQUALITY_FILTER_STATE) &&
					strings.Contains(entry.Message, "Expected one of off, building, ready") {
					warned = true
				}
			}
			assert.True(t, warned, "an unknown state must be logged as a warning with the namespace and index")
		})
	}
}

func TestSindexEqualityFilterStateAbsentIsNotLogged(t *testing.T) {
	stats, entries := refreshSindex(t, "keys=10;entries=4")

	_, found := equalityFilterState(stats)
	assert.False(t, found)

	for _, entry := range entries {
		assert.Greater(t, int(entry.Level), int(log.WarnLevel), "a server without the stat is not a warning: %s", entry.Message)
	}
}

// Stats that are not numbers, other than equality_filter_state, are skipped quietly as before.
func TestSindexOtherNonNumericStatIsNotAWarning(t *testing.T) {
	stats, entries := refreshSindex(t, "keys=10;some_text_stat=hello")

	for _, stat := range stats {
		assert.NotEqual(t, "some_text_stat", stat.Name)
	}

	for _, entry := range entries {
		assert.Greater(t, int(entry.Level), int(log.WarnLevel), entry.Message)
	}
}
