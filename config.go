package main

import (
	"io/ioutil"
	"os"
	"strings"

	"github.com/BurntSushi/toml"

	aslog "github.com/aerospike/aerospike-client-go/v5/logger"
	log "github.com/sirupsen/logrus"
)

// Config represents the aerospike-prometheus-exporter configuration
type Config struct {
	AeroProm struct {
		CertFile          string `toml:"cert_file"`
		KeyFile           string `toml:"key_file"`
		RootCA            string `toml:"root_ca"`
		KeyFilePassphrase string `toml:"key_file_passphrase"`

		MetricLabels map[string]string `toml:"labels"`

		Bind    string `toml:"bind"`
		Timeout uint8  `toml:"timeout"`

		LogFile  string `toml:"log_file"`
		LogLevel string `toml:"log_level"`

		BasicAuthUsername string `toml:"basic_auth_username"`
		BasicAuthPassword string `toml:"basic_auth_password"`
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

		NamespaceMetricsAllowlist []string `toml:"namespace_metrics_allowlist"`
		SetMetricsAllowlist       []string `toml:"set_metrics_allowlist"`
		NodeMetricsAllowlist      []string `toml:"node_metrics_allowlist"`
		XdrMetricsAllowlist       []string `toml:"xdr_metrics_allowlist"`

		NamespaceMetricsAllowlistEnabled bool
		SetMetricsAllowlistEnabled       bool
		NodeMetricsAllowlistEnabled      bool
		XdrMetricsAllowlistEnabled       bool

		NamespaceMetricsBlocklist []string `toml:"namespace_metrics_blocklist"`
		SetMetricsBlocklist       []string `toml:"set_metrics_blocklist"`
		NodeMetricsBlocklist      []string `toml:"node_metrics_blocklist"`
		XdrMetricsBlocklist       []string `toml:"xdr_metrics_blocklist"`

		NamespaceMetricsBlocklistEnabled bool
		SetMetricsBlocklistEnabled       bool
		NodeMetricsBlocklistEnabled      bool
		XdrMetricsBlocklistEnabled       bool

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
func (c *Config) validateAndUpdate() {
	if c.AeroProm.Bind == "" {
		c.AeroProm.Bind = ":9145"
	}

	if c.AeroProm.Timeout == 0 {
		c.AeroProm.Timeout = 5
	}

	if c.Aerospike.AuthMode == "" {
		c.Aerospike.AuthMode = "internal"
	}

	if c.Aerospike.Timeout == 0 {
		c.Aerospike.Timeout = 5
	}
}

// Initialize exporter configuration
func initConfig(configFile string, config *Config) {
	// to print everything out regarding reading the config in app init
	log.SetLevel(log.DebugLevel)

	log.Infof("Loading configuration file %s", configFile)
	blob, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Fatalln(err)
	}

	md, err := toml.Decode(string(blob), &config)
	if err != nil {
		log.Fatalln(err)
	}

	initAllowlistAndBlocklistConfigs(config, md)

	config.LogFile = setLogFile(config.AeroProm.LogFile)

	aslog.Logger.SetLogger(log.StandardLogger())
	setLogLevel(config.AeroProm.LogLevel)
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
func initAllowlistAndBlocklistConfigs(config *Config, md toml.MetaData) {
	// Initialize AllowlistEnabled config
	config.Aerospike.NamespaceMetricsAllowlistEnabled = md.IsDefined("Aerospike", "namespace_metrics_allowlist")
	config.Aerospike.SetMetricsAllowlistEnabled = md.IsDefined("Aerospike", "set_metrics_allowlist")
	config.Aerospike.NodeMetricsAllowlistEnabled = md.IsDefined("Aerospike", "node_metrics_allowlist")
	config.Aerospike.XdrMetricsAllowlistEnabled = md.IsDefined("Aerospike", "xdr_metrics_allowlist")

	// Initialize BlocklistEnabled config
	config.Aerospike.NamespaceMetricsBlocklistEnabled = md.IsDefined("Aerospike", "namespace_metrics_blocklist")
	config.Aerospike.SetMetricsBlocklistEnabled = md.IsDefined("Aerospike", "set_metrics_blocklist")
	config.Aerospike.NodeMetricsBlocklistEnabled = md.IsDefined("Aerospike", "node_metrics_blocklist")
	config.Aerospike.XdrMetricsBlocklistEnabled = md.IsDefined("Aerospike", "xdr_metrics_blocklist")

	// Tolerate older whitelist and blacklist configurations for a while.
	// If whitelist and blacklist configs are defined copy them into allowlist and blocklist.
	// Error out if both configurations are used at the same time.
	if md.IsDefined("Aerospike", "namespace_metrics_whitelist") {
		if config.Aerospike.NamespaceMetricsAllowlistEnabled {
			log.Fatalf("namespace_metrics_whitelist and namespace_metrics_allowlist are mutually exclusive!")
		}

		config.Aerospike.NamespaceMetricsAllowlistEnabled = true
		config.Aerospike.NamespaceMetricsAllowlist = config.Aerospike.NamespaceMetricsWhitelist
	}

	if md.IsDefined("Aerospike", "set_metrics_whitelist") {
		if config.Aerospike.SetMetricsAllowlistEnabled {
			log.Fatalf("set_metrics_whitelist and set_metrics_allowlist are mutually exclusive!")
		}

		config.Aerospike.SetMetricsAllowlistEnabled = true
		config.Aerospike.SetMetricsAllowlist = config.Aerospike.SetMetricsWhitelist
	}

	if md.IsDefined("Aerospike", "node_metrics_whitelist") {
		if config.Aerospike.NodeMetricsAllowlistEnabled {
			log.Fatalf("node_metrics_whitelist and node_metrics_allowlist are mutually exclusive!")
		}

		config.Aerospike.NodeMetricsAllowlistEnabled = true
		config.Aerospike.NodeMetricsAllowlist = config.Aerospike.NodeMetricsWhitelist
	}

	if md.IsDefined("Aerospike", "xdr_metrics_whitelist") {
		if config.Aerospike.XdrMetricsAllowlistEnabled {
			log.Fatalf("xdr_metrics_whitelist and xdr_metrics_allowlist are mutually exclusive!")
		}

		config.Aerospike.XdrMetricsAllowlistEnabled = true
		config.Aerospike.XdrMetricsAllowlist = config.Aerospike.XdrMetricsWhitelist
	}

	if md.IsDefined("Aerospike", "namespace_metrics_blacklist") {
		if config.Aerospike.NamespaceMetricsBlocklistEnabled {
			log.Fatalf("namespace_metrics_blacklist and namespace_metrics_blocklist are mutually exclusive!")
		}

		config.Aerospike.NamespaceMetricsBlocklistEnabled = true
		config.Aerospike.NamespaceMetricsBlocklist = config.Aerospike.NamespaceMetricsBlacklist
	}

	if md.IsDefined("Aerospike", "set_metrics_blacklist") {
		if config.Aerospike.SetMetricsBlocklistEnabled {
			log.Fatalf("set_metrics_blacklist and set_metrics_blocklist are mutually exclusive!")
		}

		config.Aerospike.SetMetricsBlocklistEnabled = true
		config.Aerospike.SetMetricsBlocklist = config.Aerospike.SetMetricsBlacklist
	}

	if md.IsDefined("Aerospike", "node_metrics_blacklist") {
		if config.Aerospike.NodeMetricsBlocklistEnabled {
			log.Fatalf("node_metrics_blacklist and node_metrics_blocklist are mutually exclusive!")
		}

		config.Aerospike.NodeMetricsBlocklistEnabled = true
		config.Aerospike.NodeMetricsBlocklist = config.Aerospike.NodeMetricsBlacklist
	}

	if md.IsDefined("Aerospike", "xdr_metrics_blacklist") {
		if config.Aerospike.XdrMetricsBlocklistEnabled {
			log.Fatalf("xdr_metrics_blacklist and xdr_metrics_blocklist are mutually exclusive!")
		}

		config.Aerospike.XdrMetricsBlocklistEnabled = true
		config.Aerospike.XdrMetricsBlocklist = config.Aerospike.XdrMetricsBlacklist
	}
}
