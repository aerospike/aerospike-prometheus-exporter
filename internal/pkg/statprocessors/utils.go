package statprocessors

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"
	"github.com/gobwas/glob"

	goversion "github.com/hashicorp/go-version"
)

const (
	INFOKEY_SERVICE_CLEAR_STD = "service-clear-std"
	INFOKEY_SERVICE_TLS_STD   = "service-tls-std"
)

// Default info commands
var (
	Infokey_ClusterName = "cluster-name"
	Infokey_Service     = INFOKEY_SERVICE_CLEAR_STD
	Infokey_Build       = "build"
)

var (
	// Regex for indentifying globbing patterns (or standard wildcards) in the metrics allowlist and blocklist.
	GlobbingPattern = regexp.MustCompile(`\[|\]|\*|\?|\{|\}|\\|!`)
)

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

func isMetricAllowed(pContextType commons.ContextType, pRawStatName string) bool {

	pAllowlist := []string{}
	pBlocklist := []string{}

	switch pContextType {
	case commons.CTX_NAMESPACE:
		pAllowlist = config.Cfg.Aerospike.NamespaceMetricsAllowlist
		pBlocklist = config.Cfg.Aerospike.NamespaceMetricsBlocklist
	case commons.CTX_NODE_STATS:
		pAllowlist = config.Cfg.Aerospike.NodeMetricsAllowlist
		pBlocklist = config.Cfg.Aerospike.NodeMetricsBlocklist
	case commons.CTX_SETS:
		pAllowlist = config.Cfg.Aerospike.SetMetricsAllowlist
		pBlocklist = config.Cfg.Aerospike.SetMetricsBlocklist
	case commons.CTX_SINDEX:
		pAllowlist = config.Cfg.Aerospike.SindexMetricsAllowlist
		pBlocklist = config.Cfg.Aerospike.SindexMetricsBlocklist
	case commons.CTX_XDR:
		pAllowlist = config.Cfg.Aerospike.XdrMetricsAllowlist
		pBlocklist = config.Cfg.Aerospike.XdrMetricsBlocklist

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
		isBlocked := loopPatterns(pRawStatName, pBlocklist)
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

	return loopPatterns(pRawStatName, pAllowlist)
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

func loopPatterns(pRawStatName string, pPatternList []string) bool {

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

/**
 * Check if given stat is a Gauge in a given context like Node, Namespace etc.,
 */
func isGauge(pContextType commons.ContextType, pStat string) bool {

	switch pContextType {
	case commons.CTX_NAMESPACE:
		return config.GaugeStatHandler.NamespaceStats[pStat]
	case commons.CTX_NODE_STATS:
		return config.GaugeStatHandler.NodeStats[pStat]
	case commons.CTX_SETS:
		return config.GaugeStatHandler.SetsStats[pStat]
	case commons.CTX_SINDEX:
		return config.GaugeStatHandler.SindexStats[pStat]
	case commons.CTX_XDR:
		return config.GaugeStatHandler.XdrStats[pStat]
	}

	return false
}

/*
Validates if given stat is having - or defined in gauge-stat list, if not, return default metric-type (i.e. Counter)
*/
func GetMetricType(pContext commons.ContextType, pRawMetricName string) commons.MetricType {

	// condition#1 : Config ( which has a - in the stat) is always a Gauge
	// condition#2 : or - it is marked as Gauge in the configuration file
	//
	// If stat is storage-engine related then consider the remaining stat name during below check
	//

	if pContext == commons.CTX_LATENCIES || pContext == commons.CTX_USERS {
		return commons.MetricTypeGauge
	}

	tmpRawMetricName := strings.ReplaceAll(pRawMetricName, commons.STORAGE_ENGINE, "")

	if strings.Contains(tmpRawMetricName, "-") ||
		isGauge(pContext, tmpRawMetricName) {
		return commons.MetricTypeGauge
	}

	return commons.MetricTypeCounter
}

// // Filter metrics
// // Runs the raw metrics through allowlist first and the resulting metrics through blocklist
// func GetFilteredMetrics(rawMetrics map[string]commons.MetricType, allowlist []string, allowlistEnabled bool, blocklist []string) map[string]commons.MetricType {
// 	filteredMetrics := filterAllowedMetrics(rawMetrics, allowlist, allowlistEnabled)
// 	filterBlockedMetrics(filteredMetrics, blocklist)

// 	return filteredMetrics
// }

// // Filter metrics based on configured allowlist.
// func filterAllowedMetrics(rawMetrics map[string]commons.MetricType, allowlist []string, allowlistEnabled bool) map[string]commons.MetricType {
// 	if !allowlistEnabled {
// 		return rawMetrics
// 	}

// 	filteredMetrics := make(map[string]commons.MetricType)

// 	for _, stat := range allowlist {
// 		if GlobbingPattern.MatchString(stat) {
// 			ge := glob.MustCompile(stat)

// 			for k, v := range rawMetrics {
// 				if ge.Match(k) {
// 					filteredMetrics[k] = v
// 				}
// 			}
// 		} else {
// 			if val, ok := rawMetrics[stat]; ok {
// 				filteredMetrics[stat] = val
// 			}
// 		}
// 	}

// 	return filteredMetrics
// }

// // Filter metrics based on configured blocklist.
// func filterBlockedMetrics(filteredMetrics map[string]commons.MetricType, blocklist []string) {
// 	if len(blocklist) == 0 {
// 		return
// 	}

// 	for _, stat := range blocklist {
// 		if GlobbingPattern.MatchString(stat) {
// 			ge := glob.MustCompile(stat)

// 			for k := range filteredMetrics {
// 				if ge.Match(k) {
// 					delete(filteredMetrics, k)
// 				}
// 			}
// 		} else {
// 			delete(filteredMetrics, stat)
// 		}
// 	}
// }

func BuildVersionGreaterThanOrEqual(rawMetrics map[string]string, ref string) (bool, error) {
	if len(rawMetrics["build"]) == 0 {
		return false, fmt.Errorf("couldn't get build version")
	}

	ver := rawMetrics["build"]
	version, err := goversion.NewVersion(ver)
	if err != nil {
		return false, fmt.Errorf("error parsing build version %s: %v", ver, err)
	}

	refVersion, err := goversion.NewVersion(ref)
	if err != nil {
		return false, fmt.Errorf("error parsing reference version %s: %v", ref, err)
	}

	if version.GreaterThanOrEqual(refVersion) {
		return true, nil
	}

	return false, nil
}
