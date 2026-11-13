package statprocessors

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/dataprovider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func useSysinfoFixture(t *testing.T) {
	t.Helper()

	root := t.TempDir()
	files := map[string]string{
		"sys/fs/file-nr": "12345\t0\t9223372036854775807\n",
		"meminfo":        "MemTotal: 2048 kB\nShmem: 512 kB\nSwapCached: 0 kB\n",
		"net/snmp":       "Tcp: ActiveOpens RetransSegs CurrEstab\nTcp: 10 2 4\n",
		"net/dev": "Inter-|   Receive                                                |  Transmit\n" +
			" face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed\n" +
			"  eth0: 1000 10 0 0 0 0 0 0 2000 20 0 0 0 0 0 0\n",
		"sysvipc/shm": "key shmid perms size\n",
	}
	for rel, content := range files {
		path := filepath.Join(root, rel)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	}

	original := dataprovider.PROC_PATH
	dataprovider.PROC_PATH = root
	t.Cleanup(func() { dataprovider.PROC_PATH = original })
}

func TestHostSystemInfoProcessorSkipsWhenDisabled(t *testing.T) {
	state := useProcessorConfig(t)
	processor := NewHostSystemInfoProcessor(state)

	stats, err := processor.RefreshSystemInfo()
	require.NoError(t, err)
	assert.Empty(t, stats)
}

func TestHostSystemInfoProcessorRefreshReadsProc(t *testing.T) {
	state := useProcessorConfig(t)
	useSysinfoFixture(t)
	config.Cfg.Agent.RefreshSystemStats = true
	config.GaugeStatHandler.SysInfoStats["memtotal_bytes"] = true

	processor := NewHostSystemInfoProcessor(state)
	stats, err := processor.RefreshSystemInfo()
	require.NoError(t, err)

	allocated, ok := findStat(stats, "allocated")
	require.True(t, ok)
	assert.Equal(t, 12345.0, allocated.Value)
	assert.Equal(t, commons.CTX_SYSINFO_FILEFD_STATS, allocated.Context)
	assertLabel(t, allocated, commons.METRIC_LABEL_CLUSTER_NAME, "cluster-a")

	total, ok := findStat(stats, "memtotal_bytes")
	require.True(t, ok)
	assert.Equal(t, float64(2048*1024), total.Value)
	assert.Equal(t, commons.CTX_SYSINFO_MEMORY_STATS, total.Context)
	assert.Equal(t, commons.MetricTypeGauge, metricTypeOf(total))

	opens, ok := findStat(stats, "tcp_activeopens")
	require.True(t, ok)
	assert.Equal(t, 10.0, opens.Value)
	assert.Equal(t, commons.CTX_SYSINFO_NET_STATS, opens.Context)

	receive, ok := findStatWithLabel(stats, "receive_bytes_total", commons.METRIC_LABEL_DEVICE, "eth0")
	require.True(t, ok)
	assert.Equal(t, 1000.0, receive.Value)
	assert.Equal(t, commons.CTX_SYSINFO_NETWORK_STATS, receive.Context)
}
