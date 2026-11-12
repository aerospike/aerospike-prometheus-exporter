package dataprovider

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// usesProcFixture points PROC_PATH at a temp directory and restores it after the test.
// Package-level variables are shared, so these tests must not run in parallel.
func usesProcFixture(t *testing.T, files map[string]string) string {
	t.Helper()

	root := t.TempDir()
	original := PROC_PATH
	PROC_PATH = root
	t.Cleanup(func() { PROC_PATH = original })

	for rel, content := range files {
		path := filepath.Join(root, rel)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	}
	return root
}

// parseSysInfoFloat reads a value the provider formatted with fmt.Sprint, which can use exponent notation.
func parseSysInfoFloat(t *testing.T, value string) float64 {
	t.Helper()
	f, err := strconv.ParseFloat(value, 64)
	require.NoError(t, err, "value %q is not numeric", value)
	return f
}

func TestSystemInfoProviderGetFileFD(t *testing.T) {
	usesProcFixture(t, map[string]string{
		"sys/fs/file-nr": "12345\t0\t9223372036854775807\n",
	})

	stats := SystemInfoProvider{}.GetFileFD()

	assert.Equal(t, map[string]string{"allocated": "12345"}, stats)
}

func TestSystemInfoProviderGetFileFDMissingFile(t *testing.T) {
	usesProcFixture(t, map[string]string{})

	stats := SystemInfoProvider{}.GetFileFD()

	assert.Empty(t, stats)
}

func TestSystemInfoProviderGetMemInfoStats(t *testing.T) {
	usesProcFixture(t, map[string]string{
		"meminfo": "MemTotal:       16000000 kB\n" +
			"MemFree:         1000000 kB\n" +
			"SwapCached:         2048 kB\n" +
			"Shmem:              4096 kB\n",
	})

	stats := SystemInfoProvider{}.GetMemInfoStats()

	require.Contains(t, stats, "MemTotal")
	require.Contains(t, stats, "Shmem")
	require.Contains(t, stats, "Swap_Cached")
	assert.Equal(t, float64(16000000*ONE_KILO_BYTE), parseSysInfoFloat(t, stats["MemTotal"]))
	assert.Equal(t, float64(4096*ONE_KILO_BYTE), parseSysInfoFloat(t, stats["Shmem"]))
	assert.Equal(t, float64(2048*ONE_KILO_BYTE), parseSysInfoFloat(t, stats["Swap_Cached"]))
	assert.NotContains(t, stats, "MemFree", "MemFree is intentionally not exported")
}

func TestSystemInfoProviderGetMemInfoStatsMissingFile(t *testing.T) {
	usesProcFixture(t, map[string]string{})

	assert.Empty(t, SystemInfoProvider{}.GetMemInfoStats())
}

func TestSystemInfoProviderGetNetStatInfoAcceptsOnlyListedCounters(t *testing.T) {
	usesProcFixture(t, map[string]string{
		"net/snmp": "Tcp: ActiveOpens RetransSegs CurrEstab OutRsts\n" +
			"Tcp: 100 5 7 9\n" +
			"Udp: InDatagrams NoPorts\n" +
			"Udp: 40 2\n",
	})

	stats := SystemInfoProvider{}.GetNetStatInfo()

	assert.Equal(t, map[string]string{
		"tcp_activeopens": "100",
		"tcp_retranssegs": "5",
		"tcp_currestab":   "7",
	}, stats)
}

func TestSystemInfoProviderGetNetStatInfoMissingFile(t *testing.T) {
	usesProcFixture(t, map[string]string{})

	assert.Empty(t, SystemInfoProvider{}.GetNetStatInfo())
}

func TestSystemInfoProviderGetNetDevStats(t *testing.T) {
	usesProcFixture(t, map[string]string{
		"net/dev": "Inter-|   Receive                                                |  Transmit\n" +
			" face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed\n" +
			"  eth0: 1000 10 0 0 0 0 0 0 2000 20 0 0 0 0 0 0\n" +
			"    lo: 500 5 0 0 0 0 0 0 600 6 0 0 0 0 0 0\n",
	})

	receive, transfer := SystemInfoProvider{}.GetNetDevStats()

	require.Len(t, receive, 2)
	require.Len(t, transfer, 2)

	byDeviceReceive := map[string]map[string]string{}
	for _, r := range receive {
		byDeviceReceive[r["device_name"]] = r
	}
	byDeviceTransfer := map[string]map[string]string{}
	for _, tr := range transfer {
		byDeviceTransfer[tr["device_name"]] = tr
	}

	assert.Equal(t, "1000", byDeviceReceive["eth0"]["receive_bytes_total"])
	assert.Equal(t, "2000", byDeviceTransfer["eth0"]["transfer_bytes_total"])
	assert.Equal(t, "500", byDeviceReceive["lo"]["receive_bytes_total"])
	assert.Equal(t, "600", byDeviceTransfer["lo"]["transfer_bytes_total"])
}

func TestSystemInfoProviderGetNetDevStatsMissingFile(t *testing.T) {
	usesProcFixture(t, map[string]string{})

	receive, transfer := SystemInfoProvider{}.GetNetDevStats()

	assert.Empty(t, receive)
	assert.Empty(t, transfer)
}

func TestSystemInfoProviderGetSharedMemoryStatsKeepsOnlyAerospikeSegments(t *testing.T) {
	usesProcFixture(t, map[string]string{
		"sysvipc/shm": "       key      shmid perms       size  cpid  lpid nattch   uid   gid  cuid  cgid      atime      dtime      ctime                   rss                  swap\n" +
			"-1375727360          1   600 1073741824  1234  1235      3  1000  1000  1000  1000 1700000000          0 1700000000           1048576                     0\n" +
			"     12345          2   600      4096     1     1      1     0     0     0     0          0          0          0         4096                     0\n",
	})

	stats := SystemInfoProvider{}.GetSharedMemoryStats()

	require.Len(t, stats, 1, "only the Aerospike key should be kept")
	segment := stats[0]
	assert.Equal(t, "-1375727360", segment["key"])
	assert.Equal(t, "1", segment["shmid"])
	assert.Equal(t, "600", segment["perms"])
	assert.Equal(t, "1073741824", segment["size"])
	assert.Equal(t, "1048576", segment["rss"])
	assert.Equal(t, "0xae001100", segment["hexKey"])
	assert.Equal(t, "0xae", segment["prefix"])
	assert.Equal(t, "pi", segment["typeid"])
}

func TestSystemInfoProviderGetSharedMemoryStatsSkipsMalformedRows(t *testing.T) {
	usesProcFixture(t, map[string]string{
		"sysvipc/shm": "key shmid perms size\n" +
			"-1375727360 not-a-number 600 1073741824 1234 1235 3 1000 1000 1000 1000 1700000000 0 1700000000 1048576 0\n" +
			"\n",
	})

	assert.Empty(t, SystemInfoProvider{}.GetSharedMemoryStats())
}

func TestSystemInfoProviderGetSharedMemoryStatsMissingFile(t *testing.T) {
	usesProcFixture(t, map[string]string{})

	assert.Empty(t, SystemInfoProvider{}.GetSharedMemoryStats())
}
