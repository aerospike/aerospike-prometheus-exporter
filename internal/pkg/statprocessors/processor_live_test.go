//go:build integration

package statprocessors

import (
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	aero "github.com/aerospike/aerospike-client-go/v8"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLiveServerInfoFeedsNamespaceAndNodeProcessors(t *testing.T) {
	state := useProcessorConfig(t)
	config.Cfg.Aerospike.IndexPressureStatsFetchInterval = 1

	conn := dialAerospike(t)
	defer conn.Close()

	base := requestInfo(t, conn, "build", "cluster-name", "namespaces", "service-clear-std")
	require.NotContains(t, strings.ToLower(base["build"]), "error")
	state.Build = strings.TrimSpace(base["build"])
	state.ClusterName = strings.TrimSpace(base["cluster-name"])
	state.Service = strings.TrimSpace(base["service-clear-std"])
	require.NotEmpty(t, state.Build)
	require.Equal(t, "ape-processor-it", state.ClusterName)

	namespaces := splitNames(base["namespaces"])
	assert.ElementsMatch(t, []string{"test_only_mem", "test", "bar", "bar_pmem", "employees_flash"}, namespaces)

	latencyRaw := prepareLiveData(t, conn)

	assertLiveNode(t, conn, state)
	assertLiveNamespaces(t, conn, state, namespaces)
	assertLiveSets(t, conn, state)
	assertLiveSindex(t, conn, state)
	assertLiveLatencies(t, conn, state, latencyRaw)
}

// prepareLiveData creates one secondary index and reads and writes records.
// The server image has no asbench binary. The same read and write traffic is what fills latencies: and the set and sindex stats.
func prepareLiveData(t *testing.T, conn *aero.Connection) map[string]string {
	t.Helper()

	for _, command := range []string{
		"set-config:context=namespace;id=test;enable-benchmarks-read=true",
		"set-config:context=namespace;id=test;enable-benchmarks-write=true",
		"sindex-create:ns=test;set=demo;indexname=name_idx;bin=name;type=string",
	} {
		raw, err := conn.RequestInfo(command)
		require.NoError(t, err)
		require.NotContains(t, strings.ToLower(raw[command]), "error:", command)
	}
	waitForSindex(t, conn)

	hostName, port := aerospikeEndpoint(t)
	policy := aero.NewClientPolicy()
	policy.Timeout = 30 * time.Second
	client, err := aero.NewClientWithPolicy(policy, hostName, port)
	require.NoError(t, err)
	t.Cleanup(client.Close)

	writePolicy := aero.NewWritePolicy(0, 0)
	readPolicy := aero.NewPolicy()
	for i := 0; i < 50; i++ {
		key, keyErr := aero.NewKey("test", "demo", i)
		require.NoError(t, keyErr)
		require.NoError(t, client.Put(writePolicy, key, aero.BinMap{"name": "user"}))
		_, getErr := client.Get(readPolicy, key)
		require.NoError(t, getErr)
	}

	// Benchmark detail is on latencies:hist=, and the slice closes after the reads.
	histogramKeys := []string{
		"latencies:",
		"latencies:hist={test}-read",
		"latencies:hist={test}-benchmarks-read",
		"latencies:hist={test}-write",
		"latencies:hist={test}-benchmarks-write",
	}
	deadline := time.Now().Add(12 * time.Second)
	var latencyRaw map[string]string
	for time.Now().Before(deadline) {
		for i := 0; i < 20; i++ {
			key, keyErr := aero.NewKey("test", "demo", i)
			require.NoError(t, keyErr)
			_, getErr := client.Get(readPolicy, key)
			require.NoError(t, getErr)
		}
		raw, infoErr := conn.RequestInfo(histogramKeys...)
		require.NoError(t, infoErr)
		latencyRaw = raw
		for _, key := range histogramKeys {
			if histogramHasOps(raw[key]) {
				return raw
			}
		}
		time.Sleep(time.Second)
	}
	t.Fatalf("latencies had no histogram after traffic: %#v", latencyRaw)
	return nil
}

func waitForSindex(t *testing.T, conn *aero.Connection) {
	t.Helper()

	deadline := time.Now().Add(20 * time.Second)
	var last string
	for time.Now().Before(deadline) {
		raw, err := conn.RequestInfo(KEY_SINDEX_COMMAND)
		require.NoError(t, err)
		last = raw[KEY_SINDEX_COMMAND]
		if strings.Contains(last, "indexname=name_idx") && strings.Contains(last, "state=RW") {
			return
		}
		time.Sleep(time.Second)
	}
	t.Fatalf("sindex name_idx was not readable: %s", last)
}

func assertLiveSets(t *testing.T, conn *aero.Connection, state *StatProcessorSharedState) {
	t.Helper()

	processor := NewSetsStatsProcessor(state)
	keys := processor.PassTwoKeys(nil)
	require.Equal(t, []string{KEY_SETS}, keys)

	raw := requestInfo(t, conn, keys...)
	metrics, err := processor.Refresh(keys, raw)
	require.NoError(t, err)
	objects, ok := findStatWithLabel(metrics, "objects", commons.METRIC_LABEL_SET, "demo")
	require.True(t, ok, raw[KEY_SETS])
	assert.GreaterOrEqual(t, objects.Value, 50.0)
	assertLabel(t, objects, commons.METRIC_LABEL_NS, "test")
}

func assertLiveSindex(t *testing.T, conn *aero.Connection, state *StatProcessorSharedState) {
	t.Helper()

	processor := NewSindexStatsProcessor(state)
	passOne := requestInfo(t, conn, processor.PassOneKeys()...)
	require.Contains(t, passOne[KEY_SINDEX_COMMAND], "indexname=name_idx")

	passTwo := processor.PassTwoKeys(passOne)
	require.Contains(t, passTwo, "sindex-stat:namespace=test;indexname=name_idx")

	raw := requestInfo(t, conn, passTwo...)
	metrics, err := processor.Refresh(passTwo, raw)
	require.NoError(t, err)
	entries, ok := findStat(metrics, "entries")
	require.True(t, ok, raw["sindex-stat:namespace=test;indexname=name_idx"])
	assert.GreaterOrEqual(t, entries.Value, 50.0)
	assertLabel(t, entries, commons.METRIC_LABEL_NS, "test")
	assertLabel(t, entries, commons.METRIC_LABEL_SINDEX, "name_idx")
}

func assertLiveLatencies(t *testing.T, conn *aero.Connection, state *StatProcessorSharedState, captured map[string]string) {
	t.Helper()

	processor := NewLatencyStatsProcessor(state)
	keys := processor.PassTwoKeys(nil)
	require.Contains(t, keys, "latencies:")

	raw, err := conn.RequestInfo(keys...)
	require.NoError(t, err)
	for key, value := range captured {
		if strings.Contains(value, "msec,") || strings.Contains(value, "usec,") {
			raw[key] = value
		}
	}
	if body := raw["latencies:"]; !strings.Contains(body, "msec,") && !strings.Contains(body, "usec,") {
		for _, value := range captured {
			if strings.Contains(value, "msec,") || strings.Contains(value, "usec,") {
				raw["latencies:"] = value
				break
			}
		}
	}

	metrics, refreshErr := processor.Refresh(keys, raw)
	require.NoError(t, refreshErr)
	read, ok := findLatencyBucket(metrics, "read")
	if !ok {
		read, ok = findLatencyBucket(metrics, "write")
	}
	require.True(t, ok, raw["latencies:"])
	assert.Greater(t, read.Value, 0.0)
	assertLabel(t, read, commons.METRIC_LABEL_NS, "test")
}

func histogramHasOps(body string) bool {
	for _, unit := range []string{"msec,", "usec,"} {
		rest := body
		for {
			pos := strings.Index(rest, unit)
			if pos < 0 {
				break
			}
			rest = rest[pos+len(unit):]
			end := strings.IndexAny(rest, ",;")
			if end < 0 {
				end = len(rest)
			}
			value, err := strconv.ParseFloat(rest[:end], 64)
			if err == nil && value > 0 {
				return true
			}
		}
	}
	return false
}

func findLatencyBucket(stats []AerospikeStat, operation string) (AerospikeStat, bool) {
	for _, stat := range stats {
		if !strings.Contains(stat.Name, operation) || !strings.Contains(stat.Name, "_bucket") || stat.Value <= 0 {
			continue
		}
		for i, label := range stat.Labels {
			if label == commons.METRIC_LABEL_LE && stat.LabelValues[i] == "+Inf" {
				return stat, true
			}
		}
	}
	return AerospikeStat{}, false
}

func assertLiveNode(t *testing.T, conn *aero.Connection, state *StatProcessorSharedState) {
	t.Helper()

	processor := NewNodeStatsProcessor(state)
	passOne := processor.PassOneKeys()
	require.Equal(t, []string{KEY_SERVICE_LOGS}, passOne)

	first := requestInfo(t, conn, passOne...)
	passTwo := processor.PassTwoKeys(first)
	assert.Contains(t, passTwo, KEY_SERVICE_CONFIG)
	assert.Contains(t, passTwo, KEY_SERVICE_STATISTICS)
	assert.Contains(t, passTwo, "log/0")

	newer, err := isBuildVersionGreaterThanOrEqual(state.Build, "8.2.0")
	require.NoError(t, err)
	if newer {
		assert.Contains(t, passTwo, CMD_INFOKEY_USER_AGENTS)
		assert.Contains(t, passTwo, CMD_SMD_INFO)
		assert.Contains(t, passTwo, CMD_INFOKEY_CHECKPOINT_STATUS)
	}

	raw := requestInfo(t, conn, passTwo...)
	for key, value := range first {
		raw[key] = value
	}

	metrics, err := processor.Refresh(passTwo, raw)
	require.NoError(t, err)
	assertEveryNumericStat(t, raw[KEY_SERVICE_STATISTICS], metrics)
	assertEveryNumericStat(t, raw[KEY_SERVICE_CONFIG], metrics)

	fdMax, ok := findStat(metrics, "proto-fd-max")
	require.True(t, ok)
	assert.Equal(t, 15000.0, fdMax.Value)
	assert.Equal(t, commons.MetricTypeGauge, fdMax.MType)

	_, ok = findStat(metrics, "pseudo_log_debug")
	assert.True(t, ok)
	_, ok = findStat(metrics, "pseudo_log_detail")
	assert.True(t, ok)
}

func assertLiveNamespaces(t *testing.T, conn *aero.Connection, state *StatProcessorSharedState, namespaces []string) {
	t.Helper()

	processor := NewNamespaceStatsProcessor(state)
	require.Equal(t, []string{KEY_NS_METADATA}, processor.PassOneKeys())

	passOne := map[string]string{KEY_NS_METADATA: strings.Join(namespaces, ";")}
	passTwo := processor.PassTwoKeys(passOne)
	for _, ns := range namespaces {
		assert.Contains(t, passTwo, "namespace/"+ns)
	}
	assert.NotContains(t, passTwo, KEY_NS_INDEX_PRESSURE)

	raw := requestInfo(t, conn, passTwo...)
	for _, ns := range namespaces {
		infoKey := "namespace/" + ns
		metrics, err := processor.Refresh([]string{infoKey}, raw)
		require.NoError(t, err)
		require.NotEmpty(t, metrics, infoKey)
		assertEveryNumericStat(t, raw[infoKey], metrics)

		sample, ok := findStatWithLabel(metrics, metrics[0].Name, commons.METRIC_LABEL_NS, ns)
		if !ok {
			sample = metrics[0]
		}
		assertLabel(t, sample, commons.METRIC_LABEL_NS, ns)
	}

	memory, err := processor.Refresh([]string{"namespace/test_only_mem"}, raw)
	require.NoError(t, err)
	memoryStat, ok := firstWithLabel(memory, commons.METRIC_LABEL_STORAGE_ENGINE, "memory")
	require.True(t, ok)
	assert.Equal(t, "memory", labelValue(memoryStat, commons.METRIC_LABEL_STORAGE_ENGINE))

	require.True(t, processor.isFlashStatSentByServer)
	processor.idxPressurePreviousFetchTime = time.Now().Add(-2 * time.Minute)
	secondPass := processor.PassTwoKeys(passOne)
	assert.Contains(t, secondPass, KEY_NS_INDEX_PRESSURE)

	pressureRaw := requestInfo(t, conn, KEY_NS_INDEX_PRESSURE)
	pressure, err := processor.Refresh([]string{KEY_NS_INDEX_PRESSURE}, pressureRaw)
	require.NoError(t, err)
	total, ok := findStatWithLabel(pressure, "index_pressure_total_memory", commons.METRIC_LABEL_NS, "employees_flash")
	require.True(t, ok)
	assert.GreaterOrEqual(t, total.Value, 0.0)
}

// assertEveryNumericStat fails when the processor drops a numeric or boolean info field.
// Array stats are emitted under the rewritten storage-engine_file_<stat> name.
func assertEveryNumericStat(t *testing.T, body string, metrics []AerospikeStat) {
	t.Helper()

	stats := commons.ParseStats(body, ";")
	names := map[string]struct{}{}
	for _, metric := range metrics {
		names[metric.Name] = struct{}{}
	}

	for name, value := range stats {
		if _, err := commons.TryConvert(value); err != nil {
			continue
		}
		want := emittedStatName(name)
		if want == "" {
			continue
		}
		if _, ok := names[want]; !ok {
			t.Errorf("info stat %s (emitted as %s) is missing from processor output", name, want)
		}
	}
}

func emittedStatName(serverStat string) string {
	for _, prefix := range []string{commons.STORAGE_ENGINE, commons.INDEX_TYPE, commons.SINDEX_TYPE} {
		if strings.HasPrefix(serverStat, prefix) && strings.Contains(serverStat, "[") {
			parts := strings.SplitN(strings.TrimPrefix(serverStat, prefix+"."), ".", 2)
			if len(parts) != 2 {
				return ""
			}
			kind, _, found := strings.Cut(parts[0], "[")
			if !found {
				return ""
			}
			statName := strings.TrimPrefix(parts[1], "]")
			statName = strings.TrimPrefix(statName, ".")
			return prefix + "_" + kind + "_" + statName
		}
	}
	return serverStat
}

func aerospikeEndpoint(t *testing.T) (string, int) {
	t.Helper()

	hostName := os.Getenv("AEROSPIKE_HOST")
	if hostName == "" {
		hostName = "aerospike"
	}
	port := 3000
	if raw := os.Getenv("AEROSPIKE_PORT"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		require.NoError(t, err)
		port = parsed
	}
	return hostName, port
}

func dialAerospike(t *testing.T) *aero.Connection {
	t.Helper()

	hostName, port := aerospikeEndpoint(t)

	policy := aero.NewClientPolicy()
	policy.Timeout = 30 * time.Second
	policy.User = os.Getenv("AEROSPIKE_USER")
	policy.Password = os.Getenv("AEROSPIKE_PASSWORD")

	var conn *aero.Connection
	var dialErr error
	for attempt := 0; attempt < 10; attempt++ {
		conn, dialErr = aero.NewConnection(policy, aero.NewHost(hostName, port))
		if dialErr != nil {
			time.Sleep(time.Second)
			continue
		}
		if policy.RequiresAuthentication() {
			if loginErr := conn.Login(policy); loginErr != nil {
				conn.Close()
				dialErr = loginErr
				time.Sleep(time.Second)
				continue
			}
		}
		// A zero deadline keeps the socket usable for the whole test. Timeout still applies per call.
		if timeoutErr := conn.SetTimeout(time.Time{}, policy.Timeout); timeoutErr != nil {
			conn.Close()
			dialErr = timeoutErr
			time.Sleep(time.Second)
			continue
		}
		return conn
	}
	require.NoError(t, dialErr)
	return conn
}

func requestInfo(t *testing.T, conn *aero.Connection, keys ...string) map[string]string {
	t.Helper()

	raw, err := conn.RequestInfo(keys...)
	require.NoError(t, err)
	for _, key := range keys {
		value, ok := raw[key]
		require.True(t, ok, key)
		// user-agents is empty when no client has connected. That is a real response.
		require.NotContains(t, strings.ToLower(value), "error:", key)
	}
	return raw
}

func splitNames(raw string) []string {
	parts := strings.Split(raw, ";")
	names := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			names = append(names, part)
		}
	}
	return names
}

func firstWithLabel(metrics []AerospikeStat, label string, value string) (AerospikeStat, bool) {
	for _, metric := range metrics {
		if labelValue(metric, label) == value {
			return metric, true
		}
	}
	return AerospikeStat{}, false
}

func labelValue(metric AerospikeStat, label string) string {
	for i, name := range metric.Labels {
		if name == label {
			return metric.LabelValues[i]
		}
	}
	return ""
}
