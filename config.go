package main

import (
	"io/ioutil"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
	aslog "github.com/aerospike/aerospike-client-go/logger"
	log "github.com/sirupsen/logrus"
)

type Config struct {
	AeroProm struct {
		CertFile string `toml:"cert_file"`
		KeyFile  string `toml:"key_file"`
		// UseLetsEncrypt bool   `toml:"use_lets_encrypt"`

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
	} `toml:"Aerospike"`

	LogFile *os.File
}

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

func InitConfig(configFile string, config *Config) {
	// to print everything out regarding reading the config in app init
	log.SetLevel(log.DebugLevel)

	log.Info("Reading config file...")
	blob, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Fatalln(err)
	}

	md, err := toml.Decode(string(blob), &config)
	if err != nil {
		log.Fatalln(err)
	}

	config.Aerospike.NamespaceMetricsAllowlistEnabled = md.IsDefined("Aerospike", "namespace_metrics_allowlist")
	config.Aerospike.SetMetricsAllowlistEnabled = md.IsDefined("Aerospike", "set_metrics_allowlist")
	config.Aerospike.NodeMetricsAllowlistEnabled = md.IsDefined("Aerospike", "node_metrics_allowlist")
	config.Aerospike.XdrMetricsAllowlistEnabled = md.IsDefined("Aerospike", "xdr_metrics_allowlist")

	config.Aerospike.NamespaceMetricsBlocklistEnabled = md.IsDefined("Aerospike", "namespace_metrics_blocklist")
	config.Aerospike.SetMetricsBlocklistEnabled = md.IsDefined("Aerospike", "set_metrics_blocklist")
	config.Aerospike.NodeMetricsBlocklistEnabled = md.IsDefined("Aerospike", "node_metrics_blocklist")
	config.Aerospike.XdrMetricsBlocklistEnabled = md.IsDefined("Aerospike", "xdr_metrics_blocklist")

	config.LogFile = setLogFile(config.AeroProm.LogFile)

	aslog.Logger.SetLogger(log.StandardLogger())
	setLogLevel(config.AeroProm.LogLevel)
}

func setLogFile(filepath string) *os.File {
	if len(strings.TrimSpace(filepath)) > 0 {
		out, err := os.OpenFile(filepath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("error opening file: %v", err)
		}
		log.SetOutput(out)
		return out
	}

	return nil
}

func setLogLevel(level string) {
	level = strings.ToLower(level)
	log.SetLevel(log.InfoLevel)
	aslog.Logger.SetLevel(aslog.INFO)

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
	default:
		log.SetLevel(log.InfoLevel)
		aslog.Logger.SetLevel(aslog.INFO)
	}
}
