package statprocessors

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"
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

	state := useProcessorConfig(t)
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

// loadShippedGaugeList loads the gauge_stats_list.toml that ships with the exporter, so a typo in a
// stat name there shows up as a counter here.
func loadShippedGaugeList(t *testing.T) {
	t.Helper()

	cwd, err := os.Getwd()
	require.NoError(t, err)
	config.InitGaugeStats(filepath.Join(cwd, "..", "..", "..", "configs", "gauge_stats_list.toml"))
}

// TestEqualityFilterMetricsWhenTheFilterIsOn covers R11 of the equality query filter spec: with
// enable-sindex-equality-filter on, a scrape has the per-index and the namespace filter metrics.
// The replies are written the way the spec describes them.
func TestEqualityFilterMetricsWhenTheFilterIsOn(t *testing.T) {
	state := useProcessorConfig(t)
	loadShippedGaugeList(t)

	nsKey := "namespace/test"
	namespaces := NewNamespaceStatsProcessor(state)
	namespaces.PassTwoKeys(map[string]string{KEY_NS_METADATA: "test"})
	nsStats, err := namespaces.Refresh([]string{nsKey}, map[string]string{
		nsKey: "objects=5;enable-sindex-equality-filter=true;sindex_equality_filter_bytes=8192",
	})
	require.NoError(t, err)

	flag, ok := findStat(nsStats, "enable-sindex-equality-filter")
	require.True(t, ok, "the setting that turns the filter on is exported with the namespace config")
	assert.Equal(t, 1.0, flag.Value)
	assert.Equal(t, commons.MetricTypeGauge, flag.MType)
	assertLabel(t, flag, commons.METRIC_LABEL_NS, "test")

	total, ok := findStat(nsStats, "sindex_equality_filter_bytes")
	require.True(t, ok, "the namespace total of filter memory is exported")
	assert.Equal(t, 8192.0, total.Value)
	assert.Equal(t, commons.MetricTypeGauge, total.MType)
	assertLabel(t, total, commons.METRIC_LABEL_NS, "test")

	sindexKey := "sindex-stat:namespace=test;indexname=idx"
	sindexes := NewSindexStatsProcessor(state)
	stats, err := sindexes.Refresh([]string{sindexKey}, map[string]string{
		sindexKey: "entries=100;equality_filter_state=ready;equality_filter_bytes=4096;equality_filter_saturated_partitions=2",
	})
	require.NoError(t, err)

	for name, want := range map[string]float64{
		"equality_filter_state":                2,
		"equality_filter_bytes":                4096,
		"equality_filter_saturated_partitions": 2,
	} {
		stat, ok := findStat(stats, name)
		require.True(t, ok, "%s is exported for each index", name)
		assert.Equal(t, want, stat.Value, name)
		assert.Equal(t, commons.MetricTypeGauge, stat.MType, "%s must be a gauge", name)
		assertLabel(t, stat, commons.METRIC_LABEL_NS, "test")
		assertLabel(t, stat, commons.METRIC_LABEL_SINDEX, "idx")
	}
}

// With the filter off the server reports its state and no memory, and the setting is false.
func TestEqualityFilterMetricsWhenTheFilterIsOff(t *testing.T) {
	state := useProcessorConfig(t)
	loadShippedGaugeList(t)

	nsKey := "namespace/test"
	namespaces := NewNamespaceStatsProcessor(state)
	namespaces.PassTwoKeys(map[string]string{KEY_NS_METADATA: "test"})
	nsStats, err := namespaces.Refresh([]string{nsKey}, map[string]string{
		nsKey: "objects=5;enable-sindex-equality-filter=false;sindex_equality_filter_bytes=0",
	})
	require.NoError(t, err)

	flag, ok := findStat(nsStats, "enable-sindex-equality-filter")
	require.True(t, ok)
	assert.Equal(t, 0.0, flag.Value)

	sindexKey := "sindex-stat:namespace=test;indexname=idx"
	stats, err := NewSindexStatsProcessor(state).Refresh([]string{sindexKey}, map[string]string{
		sindexKey: "entries=100;equality_filter_state=off;equality_filter_bytes=0;equality_filter_saturated_partitions=0",
	})
	require.NoError(t, err)

	state0, ok := equalityFilterState(stats)
	require.True(t, ok)
	assert.Equal(t, 0.0, state0.Value)
}
