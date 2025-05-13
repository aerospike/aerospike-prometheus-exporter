package config

import (
	"os"
	"strings"

	"github.com/BurntSushi/toml"

	aslog "github.com/aerospike/aerospike-client-go/v8/logger"
	log "github.com/sirupsen/logrus"
)

var Cfg Config

// Config represents the aerospike-prometheus-exporter configuration
type Config struct {
	Agent struct {
		OtelEnabled       bool `toml:"enable_open_telemetry"`
		PrometheusEnabled bool `toml:"enable_prometheus"`

		MetricLabels map[string]string `toml:"labels"`

		Timeout uint8 `toml:"timeout"`

		RefreshSystemStats bool   `toml:"refresh_system_stats"`
		CloudProvider      string `toml:"cloud_provider"`

		LogFile           string `toml:"log_file"`
		LogLevel          string `toml:"log_level"`
		UseMockDatasource bool   `toml:"use_mock_datasource"`

		Bind              string `toml:"bind"`
		CertFile          string `toml:"cert_file"`
		KeyFile           string `toml:"key_file"`
		RootCA            string `toml:"root_ca"`
		KeyFilePassphrase string `toml:"key_file_passphrase"`
		TlsCipherSuites   string `toml:"tls_cipher_suites"`

		BasicAuthUsername string `toml:"basic_auth_username"`
		BasicAuthPassword string `toml:"basic_auth_password"`

		Otel struct {
			OtelServiceName             string            `toml:"service_name"`
			OtelEndpoint                string            `toml:"endpoint"`
			OtelTlsEnabled              bool              `toml:"endpoint_tls_enabled"`
			OtelHeaders                 map[string]string `toml:"headers"`
			OtelPushInterval            uint8             `toml:"push_interval"`
			OtelServerStatFetchInterval uint8             `toml:"server_stat_fetch_interval"`
		} `toml:"OpenTelemetry"`

		IsKubernetes      bool
		KubernetesPodName string
	} `toml:"Agent"`

	Aerospike struct {
		Host string `toml:"db_host"`
		Port uint16 `toml:"db_port"`

		CertFile          string `toml:"cert_file"`
		KeyFile           string `toml:"key_file"`
		KeyFilePassphrase string `toml:"key_file_passphrase"`
		NodeTLSName       string `toml:"node_tls_name"`
		RootCA            string `toml:"root_ca"`

		AuthMode string `toml:"auth_mode"`
		User     string `toml:"user"`
		Password string `toml:"password"`

		Timeout uint8 `toml:"timeout"`

		LatencyBucketsCount uint8 `toml:"latency_buckets_count"`

		// Order of context ( from observer.go) - namespace, set, latencies, node-stats, xdr, user, jobs, sindex
		// Namespace metrics allow/block
		NamespaceMetricsAllowlist []string `toml:"namespace_metrics_allowlist"`
		NamespaceMetricsBlocklist []string `toml:"namespace_metrics_blocklist"`

		NamespaceMetricsAllowlistEnabled bool

		// Set metrics allow/block
		SetMetricsAllowlist []string `toml:"set_metrics_allowlist"`
		SetMetricsBlocklist []string `toml:"set_metrics_blocklist"`

		SetMetricsAllowlistEnabled bool

		// Latencies metrics allow/block
		LatenciesMetricsAllowlist []string `toml:"latencies_metrics_allowlist"`
		LatenciesMetricsBlocklist []string `toml:"latencies_metrics_blocklist"`

		LatenciesMetricsAllowlistEnabled bool

		// knob to disable latencies metrics collection (for internal use only, will be deprecated)
		DisableLatenciesMetrics bool `toml:"disable_latencies_metrics"`

		// Node metrics allow/block
		NodeMetricsAllowlist []string `toml:"node_metrics_allowlist"`
		NodeMetricsBlocklist []string `toml:"node_metrics_blocklist"`

		NodeMetricsAllowlistEnabled bool

		// Xdr metrics allow/block
		XdrMetricsAllowlist []string `toml:"xdr_metrics_allowlist"`
		XdrMetricsBlocklist []string `toml:"xdr_metrics_blocklist"`

		XdrMetricsAllowlistEnabled bool

		// User metrics allow/block
		UserMetricsUsersAllowlist []string `toml:"user_metrics_users_allowlist"`
		UserMetricsUsersBlocklist []string `toml:"user_metrics_users_blocklist"`

		UserMetricsUsersAllowlistEnabled bool

		// Job metrics allow/block
		JobMetricsAllowlist []string `toml:"job_metrics_allowlist"`
		JobMetricsBlocklist []string `toml:"job_metrics_blocklist"`

		JobMetricsAllowlistEnabled bool

		// knob to disable job metrics collection (for internal use only, will be deprecated)
		DisableJobMetrics bool `toml:"disable_job_metrics"`

		// Sindex metrics allow/block
		SindexMetricsAllowlist []string `toml:"sindex_metrics_allowlist"`
		SindexMetricsBlocklist []string `toml:"sindex_metrics_blocklist"`

		SindexMetricsAllowlistEnabled bool

		// knob to disable sindex metrics collection (for internal use only, will be deprecated)
		DisableSindexMetrics bool `toml:"disable_sindex_metrics"`

		// Tolerate older whitelist and blacklist configurations for a while
		NamespaceMetricsWhitelist []string `toml:"namespace_metrics_whitelist"`
		SetMetricsWhitelist       []string `toml:"set_metrics_whitelist"`
		NodeMetricsWhitelist      []string `toml:"node_metrics_whitelist"`
		XdrMetricsWhitelist       []string `toml:"xdr_metrics_whitelist"`

		NamespaceMetricsBlacklist []string `toml:"namespace_metrics_blacklist"`
		SetMetricsBlacklist       []string `toml:"set_metrics_blacklist"`
		NodeMetricsBlacklist      []string `toml:"node_metrics_blacklist"`
		XdrMetricsBlacklist       []string `toml:"xdr_metrics_blacklist"`
	} `toml:"Aerospike"`

	LogFile *os.File
}

// Validate and update exporter configuration
func (c *Config) ValidateAndUpdate(md toml.MetaData) {

	if c.Agent.Bind == "" {
		c.Agent.Bind = ":9145"
	}

	if c.Agent.Timeout == 0 {
		c.Agent.Timeout = 5
	}

	if c.Aerospike.AuthMode == "" {
		c.Aerospike.AuthMode = "internal"
	}
	c.Aerospike.AuthMode = strings.ToLower(strings.TrimSpace(c.Aerospike.AuthMode))

	if c.Aerospike.Timeout == 0 {
		c.Aerospike.Timeout = 5
	}

	if md.IsDefined("Agent", "use_mock_datasource") && c.Agent.UseMockDatasource {
		c.Agent.UseMockDatasource = true
	} else {
		c.Agent.UseMockDatasource = false
	}

	if len(c.Agent.Otel.OtelServiceName) == 0 {
		c.Agent.Otel.OtelServiceName = "aerospike-server-metrics-service"
	}

	if c.Agent.Otel.OtelPushInterval == 0 {
		c.Agent.Otel.OtelPushInterval = 60
	}

	if c.Agent.Otel.OtelServerStatFetchInterval == 0 {
		c.Agent.Otel.OtelPushInterval = 15
	}

	if !md.IsDefined("Agent", "enable_prometheus") {
		log.Infof("Defaulting to Prometheus Exporting mode")
		c.Agent.PrometheusEnabled = true
	}

	// key-file and cert-file either exist or not-exist together
	if len(Cfg.Aerospike.KeyFile) == 0 && len(Cfg.Aerospike.CertFile) != 0 {
		log.Fatalf("In Aerospike section, key_file is not present")
	}

	if len(Cfg.Aerospike.KeyFile) != 0 && len(Cfg.Aerospike.CertFile) == 0 {
		log.Fatalf("In Aerospike section, cert_file is not present")
	}

	// validate Aerospike root-ca and cert-file configs
	if len(Cfg.Aerospike.RootCA) == 0 && len(Cfg.Aerospike.CertFile) != 0 {
		log.Fatalf("In Aerospike section, root_ca cannot be null when cert_file and key_file are configured")
	}

}

func (c *Config) FetchCloudInfo(md toml.MetaData) {
	if !md.IsDefined("Agent", "cloud_provider") {
		return
	}

	if Cfg.Agent.CloudProvider != "" && len(strings.Trim(Cfg.Agent.CloudProvider, " ")) > 0 {
		cloudLabels := CollectCloudDetails()
		log.Debug("Adding Cloud Info to Metric Labels ", cloudLabels)

		for k, v := range cloudLabels {
			if v == "" || len(v) == 0 {
				v = "null"
			}
			Cfg.Agent.MetricLabels[k] = v
		}
	}
}

func (c *Config) FetchKubernetesInfo(md toml.MetaData) {
	// use kubectl to fetch required Kubernetes context and find the required Kubenetes environment variables
	envKubeServiceHost := os.Getenv("KUBERNETES_SERVICE_HOST")

	Cfg.Agent.IsKubernetes = false

	if envKubeServiceHost != "" && len(strings.TrimSpace(envKubeServiceHost)) > 0 {
		Cfg.Agent.IsKubernetes = true
		log.Info("Exporter is running in Kubernetes")

		// get host-name
		var err error
		Cfg.Agent.KubernetesPodName, err = os.Hostname()
		if err != nil {
			log.Errorln(err)
			return
		}

	}
}

// Initialize exporter configuration
func InitConfig(configFile string) {
	// to print everything out regarding reading the config in app init
	log.SetLevel(log.DebugLevel)

	log.Infof("Loading configuration file %s", configFile)
	blob, err := os.ReadFile(configFile)
	if err != nil {
		log.Fatalln(err)
	}

	md, err := toml.Decode(string(blob), &Cfg)
	if err != nil {
		log.Fatalln(err)
	}

	initAllowlistAndBlocklistConfigs(md)

	Cfg.LogFile = setLogFile(Cfg.Agent.LogFile)

	aslog.Logger.SetLogger(log.StandardLogger())
	setLogLevel(Cfg.Agent.LogLevel)

	Cfg.ValidateAndUpdate(md)
	Cfg.FetchCloudInfo(md)

	Cfg.FetchKubernetesInfo(md)
}

// Set log file path
func setLogFile(filepath string) *os.File {
	if len(strings.TrimSpace(filepath)) > 0 {
		out, err := os.OpenFile(filepath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("Error opening file: %v", err)
		}
		log.SetOutput(out)
		return out
	}

	return nil
}

// Set logging level
func setLogLevel(level string) {
	level = strings.ToLower(level)

	switch level {
	case "info":
		log.SetLevel(log.InfoLevel)
		aslog.Logger.SetLevel(aslog.INFO)
	case "warning", "warn":
		log.SetLevel(log.WarnLevel)
		aslog.Logger.SetLevel(aslog.WARNING)
	case "error", "err":
		log.SetLevel(log.ErrorLevel)
		aslog.Logger.SetLevel(aslog.ERR)
	case "debug":
		log.SetLevel(log.DebugLevel)
		aslog.Logger.SetLevel(aslog.DEBUG)
	case "trace":
		log.SetLevel(log.TraceLevel)
		aslog.Logger.SetLevel(aslog.DEBUG)
	default:
		log.SetLevel(log.InfoLevel)
		aslog.Logger.SetLevel(aslog.INFO)
	}
}

// Initialize Allowlist and Blocklist configurations
func initAllowlistAndBlocklistConfigs(md toml.MetaData) {
	// Initialize AllowlistEnabled config
	Cfg.Aerospike.NamespaceMetricsAllowlistEnabled = md.IsDefined("Aerospike", "namespace_metrics_allowlist")
	Cfg.Aerospike.SetMetricsAllowlistEnabled = md.IsDefined("Aerospike", "set_metrics_allowlist")
	Cfg.Aerospike.NodeMetricsAllowlistEnabled = md.IsDefined("Aerospike", "node_metrics_allowlist")
	Cfg.Aerospike.XdrMetricsAllowlistEnabled = md.IsDefined("Aerospike", "xdr_metrics_allowlist")
	Cfg.Aerospike.UserMetricsUsersAllowlistEnabled = md.IsDefined("Aerospike", "user_metrics_users_allowlist")
	Cfg.Aerospike.JobMetricsAllowlistEnabled = md.IsDefined("Aerospike", "job_metrics_allowlist")
	Cfg.Aerospike.SindexMetricsAllowlistEnabled = md.IsDefined("Aerospike", "sindex_metrics_allowlist")
	Cfg.Aerospike.LatenciesMetricsAllowlistEnabled = md.IsDefined("Aerospike", "latencies_metrics_allowlist")

	// Tolerate older whitelist and blacklist configurations for a while.
	// If whitelist and blacklist configs are defined copy them into allowlist and blocklist.
	// Error out if both configurations are used at the same time.
	if md.IsDefined("Aerospike", "namespace_metrics_whitelist") {
		if Cfg.Aerospike.NamespaceMetricsAllowlistEnabled {
			log.Fatalf("namespace_metrics_whitelist and namespace_metrics_allowlist are mutually exclusive!")
		}

		Cfg.Aerospike.NamespaceMetricsAllowlistEnabled = true
		Cfg.Aerospike.NamespaceMetricsAllowlist = Cfg.Aerospike.NamespaceMetricsWhitelist
	}

	if md.IsDefined("Aerospike", "set_metrics_whitelist") {
		if Cfg.Aerospike.SetMetricsAllowlistEnabled {
			log.Fatalf("set_metrics_whitelist and set_metrics_allowlist are mutually exclusive!")
		}

		Cfg.Aerospike.SetMetricsAllowlistEnabled = true
		Cfg.Aerospike.SetMetricsAllowlist = Cfg.Aerospike.SetMetricsWhitelist
	}

	if md.IsDefined("Aerospike", "node_metrics_whitelist") {
		if Cfg.Aerospike.NodeMetricsAllowlistEnabled {
			log.Fatalf("node_metrics_whitelist and node_metrics_allowlist are mutually exclusive!")
		}

		Cfg.Aerospike.NodeMetricsAllowlistEnabled = true
		Cfg.Aerospike.NodeMetricsAllowlist = Cfg.Aerospike.NodeMetricsWhitelist
	}

	if md.IsDefined("Aerospike", "xdr_metrics_whitelist") {
		if Cfg.Aerospike.XdrMetricsAllowlistEnabled {
			log.Fatalf("xdr_metrics_whitelist and xdr_metrics_allowlist are mutually exclusive!")
		}

		Cfg.Aerospike.XdrMetricsAllowlistEnabled = true
		Cfg.Aerospike.XdrMetricsAllowlist = Cfg.Aerospike.XdrMetricsWhitelist
	}

	if md.IsDefined("Aerospike", "namespace_metrics_blacklist") {
		if len(Cfg.Aerospike.NamespaceMetricsBlocklist) > 0 {
			log.Fatalf("namespace_metrics_blacklist and namespace_metrics_blocklist are mutually exclusive!")
		}

		Cfg.Aerospike.NamespaceMetricsBlocklist = Cfg.Aerospike.NamespaceMetricsBlacklist
	}

	if md.IsDefined("Aerospike", "set_metrics_blacklist") {
		if len(Cfg.Aerospike.SetMetricsBlocklist) > 0 {
			log.Fatalf("set_metrics_blacklist and set_metrics_blocklist are mutually exclusive!")
		}

		Cfg.Aerospike.SetMetricsBlocklist = Cfg.Aerospike.SetMetricsBlacklist
	}

	if md.IsDefined("Aerospike", "node_metrics_blacklist") {
		if len(Cfg.Aerospike.NodeMetricsBlocklist) > 0 {
			log.Fatalf("node_metrics_blacklist and node_metrics_blocklist are mutually exclusive!")
		}

		Cfg.Aerospike.NodeMetricsBlocklist = Cfg.Aerospike.NodeMetricsBlacklist
	}

	if md.IsDefined("Aerospike", "xdr_metrics_blacklist") {
		if len(Cfg.Aerospike.XdrMetricsBlocklist) > 0 {
			log.Fatalf("xdr_metrics_blacklist and xdr_metrics_blocklist are mutually exclusive!")
		}

		Cfg.Aerospike.XdrMetricsBlocklist = Cfg.Aerospike.XdrMetricsBlacklist
	}
}
