package main

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
	log "github.com/Sirupsen/logrus"
	aslog "github.com/aerospike/aerospike-client-go/logger"
)

type Config struct {
	AeroProm struct {
		CertFile string `toml:"cert_file"`
		KeyFile  string `toml:"key_file"`
		// UseLetsEncrypt bool   `toml:"use_lets_encrypt"`

		Tags []string `toml:"tags"`

		Bind    string `toml:"bind"`
		Timeout uint8  `toml:"timeout"`

		LogFile  string `toml:"log_file"`
		LogLevel string `toml:"log_level"`

		tags string
	} `toml:"Agent"`

	Aerospike struct {
		Host string `toml:"db_host"`
		Port uint16 `toml:"db_port"`

		CertFile    string `toml:"cert_file"`
		KeyFile     string `toml:"key_file"`
		NodeTLSName string `toml:"node_tls_name"`
		RootCA      string `toml:"root_ca"`

		AuthMode string `toml:"auth_mode"`
		User     string `toml:"user"`
		Password string `toml:"password"`

		Resolution uint8 `toml:"resolution"`
		Timeout    uint8 `toml:"timeout"`
	} `toml:"Aerospike"`

	serverPool *x509.CertPool
	clientPool []tls.Certificate

	LogFile *os.File
}

func (c *Config) validateAndUpdate() {
	if c.AeroProm.Bind == "" {
		c.AeroProm.Bind = ":9145"
	}

	if c.AeroProm.Timeout == 0 {
		c.AeroProm.Timeout = 5
	}

	if c.Aerospike.Resolution == 0 {
		c.Aerospike.Resolution = 5
	}

	if c.Aerospike.AuthMode == "" {
		c.Aerospike.AuthMode = "internal"
	}

	if c.Aerospike.Timeout == 0 {
		c.Aerospike.Timeout = 5
	}

	c.AeroProm.tags = strings.Join(c.AeroProm.Tags, ",")
}

func InitConfig(configFile string, config *Config) {
	// to print everything out regarding reading the config in app init
	log.SetLevel(log.DebugLevel)

	log.Info("Reading config file...")
	blob, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Fatalln(err)
	}

	if _, err := toml.Decode(string(blob), &config); err != nil {
		log.Fatalln(err)
	}

	config.LogFile = setLogFile(config.AeroProm.LogFile)

	if config.Aerospike.Resolution < 1 {
		config.Aerospike.Resolution = 5
	} else if config.Aerospike.Resolution > 10 {
		config.Aerospike.Resolution = 10
	}

	// Try to load system CA certs, otherwise just make an empty pool
	serverPool, err := x509.SystemCertPool()
	config.serverPool = serverPool

	if config.Aerospike.CertFile != "" && config.Aerospike.KeyFile != "" {
		// Try to load system CA certs and add them to the system cert pool
		cert, err := tls.LoadX509KeyPair(config.Aerospike.CertFile, config.Aerospike.KeyFile)
		if err != nil {
			log.Errorf("FAILED: Adding client certificate %s to the pool failed: %s", config.Aerospike.CertFile, err)
		}

		log.Debugf("Adding client certificate %s to the pool...", config.Aerospike.CertFile)
		config.clientPool = append(config.clientPool, cert)
	}

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
