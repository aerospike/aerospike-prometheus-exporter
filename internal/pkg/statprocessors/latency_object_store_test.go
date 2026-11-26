package statprocessors

import (
	"fmt"
	"strings"
	"testing"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"
	"github.com/stretchr/testify/assert"
)

// Test_ObjectStore_NamespaceRequestsHistograms checks that the namespace processor registers the four
// object-store histograms for each namespace, the same way it registers re-repl.
func Test_ObjectStore_NamespaceRequestsHistograms(t *testing.T) {
	sharedState := NewStatProcessorSharedState()
	sharedState.Build = "8.2.1.0"
	nw := NewNamespaceStatsProcessor(sharedState)

	// PassTwoKeys creates the per-namespace map before Refresh runs, as in the real refresh cycle
	nw.PassTwoKeys(map[string]string{KEY_NS_METADATA: "test"})
	_, err := nw.Refresh([]string{"namespace/test"}, map[string]string{
		"namespace/test": "strong-consistency=false:master_objects=1",
	})
	assert.NoError(t, err)

	benchmarks := sharedState.NamespaceLatencyBenchmarks["test"]
	assert.Equal(t, "{test}-object-store-put", benchmarks["object-store-put"])
	assert.Equal(t, "{test}-object-store-get", benchmarks["object-store-get"])
	assert.Equal(t, "{test}-object-store-list", benchmarks["object-store-list"])
	assert.Equal(t, "{test}-object-store-delete", benchmarks["object-store-delete"])
}

// Test_ObjectStore_NotRequestedOnOlderServers checks that servers older than the gate never get the
// object-store commands: they answer "bad hist name", so each would be a wasted info command per
// namespace on every scrape. It runs the real namespace refresh and then the latency command pass.
func Test_ObjectStore_NotRequestedOnOlderServers(t *testing.T) {
	tests := []struct {
		name     string
		build    string
		expected bool
	}{
		{"5.1 server", "5.1.0.0", false},
		{"8.1.1 server", "8.1.1.0", false},
		{"unknown build", "", false},
		{"unparseable build", "not-a-version", false},
		{"exactly the gate", "8.2.1.0", true},
		{"newer server", "8.3.0.0", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sharedState := NewStatProcessorSharedState()
			sharedState.Build = tt.build

			nw := NewNamespaceStatsProcessor(sharedState)
			nw.PassTwoKeys(map[string]string{KEY_NS_METADATA: "test"})
			_, err := nw.Refresh([]string{"namespace/test"}, map[string]string{
				"namespace/test": "strong-consistency=false:master_objects=1",
			})
			assert.NoError(t, err)

			// the latency pass only looks at the build to choose the command style, so give it a valid one
			latencyState := NewStatProcessorSharedState()
			latencyState.Build = "8.2.1.0"
			latencyState.NamespaceLatencyBenchmarks = sharedState.NamespaceLatencyBenchmarks
			commands := NewLatencyStatsProcessor(latencyState).PassTwoKeys(nil)

			hasObjectStore := false
			for _, command := range commands {
				if strings.Contains(command, "object-store") {
					hasObjectStore = true
				}
			}

			assert.Equal(t, tt.expected, hasObjectStore)
			// re-repl is unaffected by the gate
			assert.Contains(t, commands, "latencies:hist={test}-re-repl")
		})
	}
}

// Test_ObjectStore_RemovedWhenServerIsOlder checks that entries registered earlier are dropped if the
// build turns out to be below the gate, so state stays consistent.
func Test_ObjectStore_RemovedWhenServerIsOlder(t *testing.T) {
	sharedState := NewStatProcessorSharedState()
	sharedState.Build = "8.2.1.0"
	nw := NewNamespaceStatsProcessor(sharedState)

	refresh := func() {
		nw.PassTwoKeys(map[string]string{KEY_NS_METADATA: "test"})
		_, err := nw.Refresh([]string{"namespace/test"}, map[string]string{
			"namespace/test": "strong-consistency=false:master_objects=1",
		})
		assert.NoError(t, err)
	}

	refresh()
	assert.Contains(t, sharedState.NamespaceLatencyBenchmarks["test"], "object-store-put")

	sharedState.Build = "8.1.1.0"
	refresh()
	assert.NotContains(t, sharedState.NamespaceLatencyBenchmarks["test"], "object-store-put")
}

// Test_ObjectStore_LatenciesCommandsIncludeHistograms checks that the histograms are asked for by name
// through latencies:hist=, since plain latencies: does not list them.
func Test_ObjectStore_LatenciesCommandsIncludeHistograms(t *testing.T) {
	sharedState := NewStatProcessorSharedState()
	sharedState.Build = "8.2.1.0"
	sharedState.NamespaceLatencyBenchmarks["test"] = map[string]string{
		"object-store-put": "{test}-object-store-put",
	}

	commands := NewLatencyStatsProcessor(sharedState).PassTwoKeys(nil)

	assert.Contains(t, commands, "latencies:")
	assert.Contains(t, commands, "latencies:hist={test}-object-store-put")
}

// Test_ObjectStore_HistogramIsMilliseconds checks that a histogram reply is parsed with the unit the
// server gives in its header (always msec for these), and exported as a msec bucket metric.
func Test_ObjectStore_HistogramIsMilliseconds(t *testing.T) {
	sharedState := NewStatProcessorSharedState()
	sharedState.ClusterName = "c1"
	sharedState.Service = "10.0.0.1:3000"
	lw := NewLatencyStatsProcessor(sharedState)

	raw := map[string]string{
		"latencies:":                             "placeholder",
		"latencies:hist={test}-object-store-put": "{test}-object-store-put:msec,4234.9,28.75,7.40,1.63,0.26,0.03,0.00,0.00,0.00,0.00,0.00,0.00,0.00,0.00,0.00,0.00,0.00,0.00;",
	}

	metrics, err := lw.Refresh([]string{"latencies:hist={test}-object-store-put"}, raw)
	assert.Nil(t, err)

	var bucketSeen, countSeen bool
	for _, m := range metrics {
		assert.Equal(t, commons.CTX_LATENCIES, m.Context)

		// the unit is trimmed to "ms" in metric names, as for every other latency histogram
		switch m.Name {
		case "object-store-put_ms_bucket":
			bucketSeen = true
		case "object-store-put_ms_count":
			countSeen = true
		}
	}

	assert.True(t, bucketSeen, "expected an ms bucket series for object-store-put")
	assert.True(t, countSeen, "expected an ms count series for object-store-put")
}

// Test_ObjectStore_UnavailableHistogramIsSilent checks that a histogram the server cannot return,
// such as "bad hist name" on a node where histograms were never initialised, yields no series and no panic.
func Test_ObjectStore_UnavailableHistogramIsSilent(t *testing.T) {
	for _, reply := range []string{
		"bad hist name",
		"bad hist name;",
		"",
	} {
		parsed := parseLatencyInfo(reply, 0)
		assert.Empty(t, parsed, "reply %q must produce no histogram", reply)
	}
}

// Test_ObjectStore_BlocklistFiltersOneHistogram checks that latencies_metrics_blocklist removes only the
// named object-store histogram, and the others still export.
func Test_ObjectStore_BlocklistFiltersOneHistogram(t *testing.T) {
	config.Cfg.Aerospike.LatenciesMetricsBlocklist = []string{"object-store-put"}
	defer func() { config.Cfg.Aerospike.LatenciesMetricsBlocklist = nil }()

	sharedState := NewStatProcessorSharedState()
	lw := NewLatencyStatsProcessor(sharedState)

	reply := "{test}-object-store-%s:msec,4234.9,28.75,7.40,1.63,0.26,0.03,0.00,0.00,0.00,0.00,0.00,0.00,0.00,0.00,0.00,0.00,0.00,0.00;"
	raw := map[string]string{
		"latencies:":                             "placeholder",
		"latencies:hist={test}-object-store-put": fmt.Sprintf(reply, "put"),
		"latencies:hist={test}-object-store-get": fmt.Sprintf(reply, "get"),
	}

	metrics, err := lw.Refresh([]string{
		"latencies:hist={test}-object-store-put",
		"latencies:hist={test}-object-store-get",
	}, raw)
	assert.Nil(t, err)

	names := map[string]bool{}
	for _, m := range metrics {
		names[m.Name] = true
	}

	assert.False(t, names["object-store-put_ms_bucket"], "blocked histogram must not export")
	assert.False(t, names["object-store-put_ms_count"], "blocked histogram must not export")
	assert.True(t, names["object-store-get_ms_bucket"], "unblocked histogram must still export")
}
