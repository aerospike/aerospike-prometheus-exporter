package commons

import (
	"os"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/gobwas/glob"

	aslog "github.com/aerospike/aerospike-client-go/v6/logger"
	log "github.com/sirupsen/logrus"
)

var Cfg Config

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
func (c *Config) ValidateAndUpdate() {
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

	Cfg.LogFile = setLogFile(Cfg.AeroProm.LogFile)

	aslog.Logger.SetLogger(log.StandardLogger())
	setLogLevel(Cfg.AeroProm.LogLevel)
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

/**
 * this function check is a given stat is allowed or blocked against given patterns
 * these patterns are defined within ape.toml
 *
 * NOTE: when a stat falls within intersection of allow-list & block-list, we block that stat
 *
 *             | empty         | no-pattern-match-found | pattern-match-found
 *  allow-list | allowed/true  |   not-allowed/ false   |    allowed/true
 *  block-list | blocked/false |   not-blocked/ false   |    blocked/true
 *
 *  by checking the blocklist first,
 *     we avoid processing the allow-list for some of the metrics
 *
 */
func (cfg *Config) isMetricAllowed(pContextType ContextType, pRawStatName string) bool {

	pAllowlist := []string{}
	pBlocklist := []string{}

	switch pContextType {
	case CTX_NAMESPACE:
		pAllowlist = cfg.Aerospike.NamespaceMetricsAllowlist
		pBlocklist = cfg.Aerospike.NamespaceMetricsBlocklist
	case CTX_NODE_STATS:
		pAllowlist = cfg.Aerospike.NodeMetricsAllowlist
		pBlocklist = cfg.Aerospike.NodeMetricsBlocklist
	case CTX_SETS:
		pAllowlist = cfg.Aerospike.SetMetricsAllowlist
		pBlocklist = cfg.Aerospike.SetMetricsBlocklist
	case CTX_SINDEX:
		pAllowlist = cfg.Aerospike.SindexMetricsAllowlist
		pBlocklist = cfg.Aerospike.SindexMetricsBlocklist
	case CTX_XDR:
		pAllowlist = cfg.Aerospike.XdrMetricsAllowlist
		pBlocklist = cfg.Aerospike.XdrMetricsBlocklist

	}

	/**
		* is this stat is in blocked list
	    *    if is-block-list array not-defined or is-empty, then false (i.e. STAT-IS-NOT-BLOCKED)
		*    else
		*       match stat with "all-block-list-patterns",
		*             if-any-pattern-match-found,
		*                    return true (i.e. STAT-IS-BLOCKED)
		* if stat-is-not-blocked
		*    if is-allow-list array not-defined or is-empty, then true (i.e. STAT-IS-ALLOWED)
		*    else
		*      match stat with "all-allow-list-patterns"
		*             if-any-pattern-match-found,
		*                    return true (i.e. STAT-IS-ALLOWED)
	*/
	if len(pBlocklist) > 0 {
		isBlocked := cfg.loopPatterns(pRawStatName, pBlocklist)
		if isBlocked {
			return false
		}
	}

	// as it is already blocked, we dont need to check in allow-list,
	// i.e. when a stat falls within intersection of allow-list & block-list, we block that stat
	//

	if len(pAllowlist) == 0 {
		return true
	}

	return cfg.loopPatterns(pRawStatName, pAllowlist)
}

/**
 *  this function is used to loop thru any given regex-pattern-list, [ master_objects or *master* ]
 *
 *             | empty         | no-pattern-match-found | pattern-match-found
 *  allow-list | allowed/true  |   not-allowed/ false   |    allowed/true
 *  block-list | blocked/false |   not-blocked/ false   |    blocked/true
 *
 *
 */

func (cfg *Config) loopPatterns(pRawStatName string, pPatternList []string) bool {

	for _, statPattern := range pPatternList {
		if len(statPattern) > 0 {

			ge := glob.MustCompile(statPattern)

			if ge.Match(pRawStatName) {
				return true
			}
		}
	}

	return false
}
