package config

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func saveConfigState(t *testing.T) {
	t.Helper()

	originalCfg := Cfg
	originalCloud := cloudInfo
	originalGauge := GaugeStatHandler
	originalURL := cloudMetadataURL
	logger := log.StandardLogger()
	prevOut := logger.Out
	prevLevel := logger.GetLevel()
	prevExit := logger.ExitFunc
	logger.SetOutput(io.Discard)

	t.Cleanup(func() {
		if Cfg.LogFile != nil && Cfg.LogFile != originalCfg.LogFile {
			_ = Cfg.LogFile.Close()
		}
		Cfg = originalCfg
		cloudInfo = originalCloud
		GaugeStatHandler = originalGauge
		cloudMetadataURL = originalURL
		logger.SetOutput(prevOut)
		logger.SetLevel(prevLevel)
		logger.ExitFunc = prevExit
	})
}

// rejectText runs fn and returns the fatal log text. log.Fatal exits the process,
// so the test swaps ExitFunc for a panic and recovers it.
func rejectText(t *testing.T, fn func()) (text string) {
	t.Helper()

	logger := log.StandardLogger()
	prevOut := logger.Out
	prevExit := logger.ExitFunc
	var buf bytes.Buffer
	logger.SetOutput(&buf)
	logger.ExitFunc = func(int) { panic("config rejected") }

	defer func() {
		logger.SetOutput(prevOut)
		logger.ExitFunc = prevExit
		text = buf.String()
	}()
	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatalf("expected configuration to be rejected, log: %s", buf.String())
		}
		if recovered != "config rejected" {
			panic(recovered)
		}
	}()

	fn()
	return text
}

func decodeConfig(t *testing.T, src string) (Config, toml.MetaData) {
	t.Helper()

	var cfg Config
	md, err := toml.Decode(src, &cfg)
	require.NoError(t, err)
	return cfg, md
}

func TestValidateAndUpdateAppliesDefaults(t *testing.T) {
	saveConfigState(t)

	cfg, md := decodeConfig(t, `
[Agent]
[Aerospike]
`)
	cfg.ValidateAndUpdate(md)

	assert.Equal(t, ":9145", cfg.Agent.Bind)
	assert.Equal(t, uint8(5), cfg.Agent.Timeout)
	assert.Equal(t, "internal", cfg.Aerospike.AuthMode)
	assert.Equal(t, uint8(5), cfg.Aerospike.Timeout)
	assert.Equal(t, "aerospike-server-metrics-service", cfg.Agent.Otel.ServiceName)
	assert.Equal(t, uint8(60), cfg.Agent.Otel.PushInterval)
	assert.Equal(t, uint8(15), cfg.Agent.Otel.ServerStatFetchInterval)
	assert.True(t, cfg.Agent.PrometheusEnabled)
	assert.False(t, cfg.Agent.OtelEnabled)
	assert.Equal(t, uint8(10), cfg.Aerospike.IndexPressureStatsFetchInterval)
}

func TestValidateAndUpdateKeepsExplicitValues(t *testing.T) {
	saveConfigState(t)

	cfg, md := decodeConfig(t, `
[Agent]
bind = "127.0.0.1:9200"
timeout = 9
enable_prometheus = true

[Agent.OpenTelemetry]
service_name = "custom-service"
push_interval = 7
server_stat_fetch_interval = 3

[Aerospike]
auth_mode = "  PKI  "
timeout = 8
index_pressure_stats_fetch_interval = 0
`)
	cfg.ValidateAndUpdate(md)

	assert.Equal(t, "127.0.0.1:9200", cfg.Agent.Bind)
	assert.Equal(t, uint8(9), cfg.Agent.Timeout)
	assert.Equal(t, "pki", cfg.Aerospike.AuthMode)
	assert.Equal(t, uint8(8), cfg.Aerospike.Timeout)
	assert.Equal(t, "custom-service", cfg.Agent.Otel.ServiceName)
	assert.Equal(t, uint8(7), cfg.Agent.Otel.PushInterval)
	assert.Equal(t, uint8(3), cfg.Agent.Otel.ServerStatFetchInterval)
	assert.True(t, cfg.Agent.PrometheusEnabled)
	assert.Equal(t, uint8(0), cfg.Aerospike.IndexPressureStatsFetchInterval)
}

func TestValidateAndUpdateOtelDefaults(t *testing.T) {
	saveConfigState(t)

	cfg, md := decodeConfig(t, `
[Agent]
enable_prometheus = false
enable_open_telemetry = true

[Agent.OpenTelemetry]
http_endpoint = "http://collector:4318/v1/metrics"

[Aerospike]
`)
	cfg.ValidateAndUpdate(md)

	assert.False(t, cfg.Agent.PrometheusEnabled)
	assert.Equal(t, "delta", cfg.Agent.Otel.CounterTemporality)
	assert.True(t, cfg.Agent.Otel.AllMetricsAsGauge)
	assert.Equal(t, "aerospike.server", cfg.Agent.Otel.MetricNamePrefix)
	assert.Equal(t, "period", cfg.Agent.Otel.MetricContextSeparator)
	assert.Equal(t, map[string]string{
		"cluster_name": "aerospike_cluster",
		"service":      "aerospike_service",
	}, cfg.Agent.Otel.RenamedLabels)
}

func TestValidateAndUpdateKeepsExplicitOtelChoices(t *testing.T) {
	saveConfigState(t)

	cfg, md := decodeConfig(t, `
[Agent]
enable_open_telemetry = true

[Agent.OpenTelemetry]
http_endpoint = "http://collector:4318/v1/metrics"
counter_temporality = "Cumulative"
all_metrics_as_gauges = false
metric_name_prefix = "ape"
metric_context_separator = "Underscore"

[Agent.OpenTelemetry.renamed_labels]
cluster_name = "cluster"

[Aerospike]
`)
	cfg.ValidateAndUpdate(md)

	assert.Equal(t, "cumulative", cfg.Agent.Otel.CounterTemporality)
	assert.False(t, cfg.Agent.Otel.AllMetricsAsGauge)
	assert.Equal(t, "ape", cfg.Agent.Otel.MetricNamePrefix)
	assert.Equal(t, "underscore", cfg.Agent.Otel.MetricContextSeparator)
	assert.Equal(t, map[string]string{"cluster_name": "cluster"}, cfg.Agent.Otel.RenamedLabels)
}

func TestValidateAndUpdateCopiesDeprecatedOtelEndpoint(t *testing.T) {
	saveConfigState(t)

	cfg, md := decodeConfig(t, `
[Agent]
enable_open_telemetry = true

[Agent.OpenTelemetry]
endpoint = "collector:4317"

[Aerospike]
`)
	cfg.ValidateAndUpdate(md)

	assert.Equal(t, "collector:4317", cfg.Agent.Otel.GrpcEndpoint)
}

func TestValidateAndUpdateRejectsInvalidExporterConfig(t *testing.T) {
	saveConfigState(t)

	cases := []struct {
		name    string
		src     string
		message string
	}{
		{
			name: "cert without key",
			src: `
[Agent]
[Aerospike]
cert_file = "client.pem"
`,
			message: "key_file is not present",
		},
		{
			name: "key without cert",
			src: `
[Agent]
[Aerospike]
key_file = "client.key"
`,
			message: "cert_file is not present",
		},
		{
			name: "client cert without root ca",
			src: `
[Agent]
[Aerospike]
cert_file = "client.pem"
key_file = "client.key"
`,
			message: "root_ca cannot be null",
		},
		{
			name: "neither exporter enabled",
			src: `
[Agent]
enable_prometheus = false
enable_open_telemetry = false
[Aerospike]
`,
			message: "Atleast one of Prometheus or OpenTelemetry should be enabled",
		},
		{
			name: "deprecated endpoint and grpc endpoint",
			src: `
[Agent]
enable_open_telemetry = true
[Agent.OpenTelemetry]
endpoint = "collector:4317"
grpc_endpoint = "collector:4317"
[Aerospike]
`,
			message: "ONLY  endpoint or grpc_endpoint",
		},
		{
			name: "grpc and http endpoints",
			src: `
[Agent]
enable_open_telemetry = true
[Agent.OpenTelemetry]
grpc_endpoint = "collector:4317"
http_endpoint = "http://collector:4318/v1/metrics"
[Aerospike]
`,
			message: "grpc_endpoint or http_endpoint",
		},
		{
			name: "otel without an endpoint",
			src: `
[Agent]
enable_open_telemetry = true
[Agent.OpenTelemetry]
[Aerospike]
`,
			message: "neither is configured",
		},
		{
			name: "unknown temporality",
			src: `
[Agent]
enable_open_telemetry = true
[Agent.OpenTelemetry]
http_endpoint = "http://collector:4318/v1/metrics"
counter_temporality = "deltaX"
[Aerospike]
`,
			message: "counter_temporality must be either delta or cumulative",
		},
		{
			name: "unknown metric separator",
			src: `
[Agent]
enable_open_telemetry = true
[Agent.OpenTelemetry]
http_endpoint = "http://collector:4318/v1/metrics"
metric_context_separator = "dot"
[Aerospike]
`,
			message: "metric_context_separator must be either period or underscore",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, md := decodeConfig(t, tc.src)
			text := rejectText(t, func() { cfg.ValidateAndUpdate(md) })
			assert.Contains(t, text, tc.message)
		})
	}
}

func TestAllowAndBlockListsFollowToml(t *testing.T) {
	saveConfigState(t)

	cfg, md := decodeConfig(t, `
[Aerospike]
namespace_metrics_allowlist = ["ns_objects"]
set_metrics_allowlist = ["objects"]
node_metrics_allowlist = ["client_connections"]
xdr_metrics_allowlist = ["lag"]
user_metrics_users_allowlist = ["reader"]
job_metrics_allowlist = ["scan"]
sindex_metrics_allowlist = ["entries"]
latencies_metrics_allowlist = ["reads"]
namespace_metrics_blocklist = ["ns_dead"]
`)
	Cfg = cfg
	initAllowlistAndBlocklistConfigs(md)

	assert.True(t, Cfg.Aerospike.NamespaceMetricsAllowlistEnabled)
	assert.True(t, Cfg.Aerospike.SetMetricsAllowlistEnabled)
	assert.True(t, Cfg.Aerospike.NodeMetricsAllowlistEnabled)
	assert.True(t, Cfg.Aerospike.XdrMetricsAllowlistEnabled)
	assert.True(t, Cfg.Aerospike.UserMetricsUsersAllowlistEnabled)
	assert.True(t, Cfg.Aerospike.JobMetricsAllowlistEnabled)
	assert.True(t, Cfg.Aerospike.SindexMetricsAllowlistEnabled)
	assert.True(t, Cfg.Aerospike.LatenciesMetricsAllowlistEnabled)
	assert.Equal(t, []string{"ns_objects"}, Cfg.Aerospike.NamespaceMetricsAllowlist)
	assert.Equal(t, []string{"ns_dead"}, Cfg.Aerospike.NamespaceMetricsBlocklist)
}

func TestLegacyWhitelistAndBlacklistAreCopied(t *testing.T) {
	saveConfigState(t)

	cases := []struct {
		name  string
		src   string
		allow []string
		block []string
		read  func() ([]string, []string, bool)
	}{
		{
			name:  "namespace",
			src:   "[Aerospike]\nnamespace_metrics_whitelist = [\"ns_objects\"]\nnamespace_metrics_blacklist = [\"ns_dead\"]\n",
			allow: []string{"ns_objects"},
			block: []string{"ns_dead"},
			read: func() ([]string, []string, bool) {
				return Cfg.Aerospike.NamespaceMetricsAllowlist, Cfg.Aerospike.NamespaceMetricsBlocklist, Cfg.Aerospike.NamespaceMetricsAllowlistEnabled
			},
		},
		{
			name:  "set",
			src:   "[Aerospike]\nset_metrics_whitelist = [\"objects\"]\nset_metrics_blacklist = [\"dead\"]\n",
			allow: []string{"objects"},
			block: []string{"dead"},
			read: func() ([]string, []string, bool) {
				return Cfg.Aerospike.SetMetricsAllowlist, Cfg.Aerospike.SetMetricsBlocklist, Cfg.Aerospike.SetMetricsAllowlistEnabled
			},
		},
		{
			name:  "node",
			src:   "[Aerospike]\nnode_metrics_whitelist = [\"uptime\"]\nnode_metrics_blacklist = [\"dead\"]\n",
			allow: []string{"uptime"},
			block: []string{"dead"},
			read: func() ([]string, []string, bool) {
				return Cfg.Aerospike.NodeMetricsAllowlist, Cfg.Aerospike.NodeMetricsBlocklist, Cfg.Aerospike.NodeMetricsAllowlistEnabled
			},
		},
		{
			name:  "xdr",
			src:   "[Aerospike]\nxdr_metrics_whitelist = [\"lag\"]\nxdr_metrics_blacklist = [\"dead\"]\n",
			allow: []string{"lag"},
			block: []string{"dead"},
			read: func() ([]string, []string, bool) {
				return Cfg.Aerospike.XdrMetricsAllowlist, Cfg.Aerospike.XdrMetricsBlocklist, Cfg.Aerospike.XdrMetricsAllowlistEnabled
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, md := decodeConfig(t, tc.src)
			Cfg = cfg
			initAllowlistAndBlocklistConfigs(md)

			allow, block, enabled := tc.read()
			assert.True(t, enabled)
			assert.Equal(t, tc.allow, allow)
			assert.Equal(t, tc.block, block)
		})
	}
}

func TestLegacyListsRejectBothNames(t *testing.T) {
	saveConfigState(t)

	cases := []struct {
		name    string
		src     string
		message string
	}{
		{
			name:    "namespace allow",
			src:     "[Aerospike]\nnamespace_metrics_whitelist = [\"a\"]\nnamespace_metrics_allowlist = [\"b\"]\n",
			message: "namespace_metrics_whitelist and namespace_metrics_allowlist",
		},
		{
			name:    "set allow",
			src:     "[Aerospike]\nset_metrics_whitelist = [\"a\"]\nset_metrics_allowlist = [\"b\"]\n",
			message: "set_metrics_whitelist and set_metrics_allowlist",
		},
		{
			name:    "node allow",
			src:     "[Aerospike]\nnode_metrics_whitelist = [\"a\"]\nnode_metrics_allowlist = [\"b\"]\n",
			message: "node_metrics_whitelist and node_metrics_allowlist",
		},
		{
			name:    "xdr allow",
			src:     "[Aerospike]\nxdr_metrics_whitelist = [\"a\"]\nxdr_metrics_allowlist = [\"b\"]\n",
			message: "xdr_metrics_whitelist and xdr_metrics_allowlist",
		},
		{
			name:    "namespace block",
			src:     "[Aerospike]\nnamespace_metrics_blacklist = [\"a\"]\nnamespace_metrics_blocklist = [\"b\"]\n",
			message: "namespace_metrics_blacklist and namespace_metrics_blocklist",
		},
		{
			name:    "set block",
			src:     "[Aerospike]\nset_metrics_blacklist = [\"a\"]\nset_metrics_blocklist = [\"b\"]\n",
			message: "set_metrics_blacklist and set_metrics_blocklist",
		},
		{
			name:    "node block",
			src:     "[Aerospike]\nnode_metrics_blacklist = [\"a\"]\nnode_metrics_blocklist = [\"b\"]\n",
			message: "node_metrics_blacklist and node_metrics_blocklist",
		},
		{
			name:    "xdr block",
			src:     "[Aerospike]\nxdr_metrics_blacklist = [\"a\"]\nxdr_metrics_blocklist = [\"b\"]\n",
			message: "xdr_metrics_blacklist and xdr_metrics_blocklist",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, md := decodeConfig(t, tc.src)
			Cfg = cfg
			text := rejectText(t, func() { initAllowlistAndBlocklistConfigs(md) })
			assert.Contains(t, text, tc.message)
		})
	}
}

func TestSecretAgentConfigsCopiesTheBlock(t *testing.T) {
	cfg, _ := decodeConfig(t, `
[Agent.SecretAgent]
address = "127.0.0.1"
port = 4000
timeout = 3
`)

	raw := cfg.SecretAgentConfigs()

	assert.Equal(t, "127.0.0.1", raw.Address)
	assert.Equal(t, uint16(4000), raw.Port)
	assert.Equal(t, uint16(3), raw.TimeoutSeconds)
	assert.Empty(t, raw.SocketPath)
}

func TestFetchKubernetesInfo(t *testing.T) {
	saveConfigState(t)

	t.Run("outside kubernetes", func(t *testing.T) {
		t.Setenv("KUBERNETES_SERVICE_HOST", "")
		cfg := Config{}
		cfg.FetchKubernetesInfo(toml.MetaData{})
		assert.False(t, cfg.Agent.IsKubernetes)
		assert.Empty(t, cfg.Agent.KubernetesPodName)
	})

	t.Run("blank host is ignored", func(t *testing.T) {
		t.Setenv("KUBERNETES_SERVICE_HOST", "   ")
		cfg := Config{}
		cfg.FetchKubernetesInfo(toml.MetaData{})
		assert.False(t, cfg.Agent.IsKubernetes)
	})

	t.Run("service host marks the process as kubernetes", func(t *testing.T) {
		t.Setenv("KUBERNETES_SERVICE_HOST", "10.0.0.1")
		cfg := Config{}
		cfg.FetchKubernetesInfo(toml.MetaData{})
		assert.True(t, cfg.Agent.IsKubernetes)
		hostname, err := os.Hostname()
		require.NoError(t, err)
		assert.Equal(t, hostname, cfg.Agent.KubernetesPodName)
	})
}

func TestSetLogLevel(t *testing.T) {
	saveConfigState(t)

	cases := map[string]log.Level{
		"info":    log.InfoLevel,
		"warning": log.WarnLevel,
		"warn":    log.WarnLevel,
		"error":   log.ErrorLevel,
		"err":     log.ErrorLevel,
		"debug":   log.DebugLevel,
		"trace":   log.TraceLevel,
		"TRACE":   log.TraceLevel,
		"":        log.InfoLevel,
		"verbose": log.InfoLevel,
	}

	for input, level := range cases {
		t.Run(input, func(t *testing.T) {
			setLogLevel(input)
			assert.Equal(t, level, log.GetLevel())
		})
	}
}

func TestSetLogFile(t *testing.T) {
	saveConfigState(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "exporter.log")

	file := setLogFile(path)
	require.NotNil(t, file)
	log.Info("config-log-marker")
	require.NoError(t, file.Sync())

	body, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(body), "config-log-marker")

	text := rejectText(t, func() { setLogFile(filepath.Join(dir, "missing", "exporter.log")) })
	assert.Contains(t, text, "Error opening file")
}

func TestInitConfigLoadsAFile(t *testing.T) {
	saveConfigState(t)

	path := filepath.Join(t.TempDir(), "ape.toml")
	body := []byte(`
[Agent]
bind = ":9200"
log_level = "warning"
enable_prometheus = true

[Aerospike]
db_host = "db"
db_port = 3000
auth_mode = " EXTERNAL "
namespace_metrics_allowlist = ["ns_objects"]
`)
	require.NoError(t, os.WriteFile(path, body, 0o600))

	InitConfig(path)

	assert.Equal(t, ":9200", Cfg.Agent.Bind)
	assert.Equal(t, "db", Cfg.Aerospike.Host)
	assert.Equal(t, uint16(3000), Cfg.Aerospike.Port)
	assert.Equal(t, "external", Cfg.Aerospike.AuthMode)
	assert.True(t, Cfg.Aerospike.NamespaceMetricsAllowlistEnabled)
	assert.Equal(t, []string{"ns_objects"}, Cfg.Aerospike.NamespaceMetricsAllowlist)
	assert.Equal(t, log.WarnLevel, log.GetLevel())
	assert.Nil(t, Cfg.LogFile)
}

func TestInitConfigRejectsUnusableFiles(t *testing.T) {
	saveConfigState(t)

	t.Run("missing file", func(t *testing.T) {
		text := rejectText(t, func() {
			InitConfig(filepath.Join(t.TempDir(), "missing.toml"))
		})
		assert.Contains(t, text, "missing.toml")
	})

	t.Run("invalid toml", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "bad.toml")
		require.NoError(t, os.WriteFile(path, []byte("this is not = ["), 0o600))
		text := rejectText(t, func() { InitConfig(path) })
		assert.NotEmpty(t, text)
	})

	t.Run("secret agent address without port", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "sa.toml")
		body := []byte(`
[Agent.SecretAgent]
address = "127.0.0.1"

[Aerospike]
`)
		require.NoError(t, os.WriteFile(path, body, 0o600))
		text := rejectText(t, func() { InitConfig(path) })
		assert.Contains(t, text, "Invalid [Agent.SecretAgent] configuration")
	})
}

func TestInitGaugeStatsBuildsLookupMaps(t *testing.T) {
	saveConfigState(t)

	path := filepath.Join(t.TempDir(), "gauges.toml")
	body := []byte(`
namespace_gauge_stats = ["objects", "objects"]
node_gauge_stats = ["cluster_size"]
sets_gauge_stats = ["truncate_lut"]
sindex_gauge_stats = ["entries"]
xdr_gauge_stats = ["lag"]
system_info_gauge_stats = ["MemTotal"]
`)
	require.NoError(t, os.WriteFile(path, body, 0o600))

	InitGaugeStats(path)

	assert.Equal(t, map[string]bool{"objects": true}, GaugeStatHandler.NamespaceStats)
	assert.True(t, GaugeStatHandler.NodeStats["cluster_size"])
	assert.True(t, GaugeStatHandler.SetsStats["truncate_lut"])
	assert.True(t, GaugeStatHandler.SindexStats["entries"])
	assert.True(t, GaugeStatHandler.XdrStats["lag"])
	assert.True(t, GaugeStatHandler.SysInfoStats["MemTotal"])
	assert.Nil(t, GaugeStatHandler.Namespace)
	assert.Nil(t, GaugeStatHandler.SysInfo)

	text := rejectText(t, func() {
		InitGaugeStats(filepath.Join(t.TempDir(), "missing.toml"))
	})
	assert.Contains(t, text, "missing.toml")
}
