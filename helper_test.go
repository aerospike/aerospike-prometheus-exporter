package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gobwas/glob"
)

var DEFAULT_APE_TOML = "tests/default_ape.toml"
var LABELS_APE_TOML = "tests/labels_ape.toml"
var NS_ALLOWLIST_APE_TOML = "tests/ns_allowlist_ape.toml"
var NS_BLOCKLIST_APE_TOML = "tests/ns_blocklist_ape.toml"

var TESTCASE_MODE = "TESTCASE_MODE"
var TESTCASE_MODE_TRUE = "true"
var TESTCASE_MODE_FALSE = "false"

// var g_ns_metric_allow_list = []string{"aerospike_namespace_master_objects", "aerospike_namespace_memory_used_bytes"}

func extractNamespaceFromLabel(label string) string {
	// [name:"cluster_name" value:""  name:"ns" value:"bar"  name:"service" value:"" ]
	nsFromLabel := label
	nsFromLabel = nsFromLabel[(strings.Index(nsFromLabel, "ns"))+11:]
	nsFromLabel = nsFromLabel[0:(strings.Index(nsFromLabel, "\""))]

	return nsFromLabel
}

func extractMetricNameFromDesc(desc string) string {
	// Desc{fqName: "aerospike_namespac_memory_free_pct", help: "memory free pct", constLabels: {}, variableLabels: [cluster_name service ns]}
	// fmt.Println("description given: ===> ", desc)
	metricNameFromDesc := desc[0 : (strings.Index(desc, ","))-1]
	metricNameFromDesc = metricNameFromDesc[(strings.Index(metricNameFromDesc, ":"))+3:]

	return strings.Trim(metricNameFromDesc, " ")
}

func makeKeyname(a string, b string, combineBoth bool) string {
	if combineBoth {
		return a + "/" + b
	}
	return a
}

func splitAndRetrieveStats(s, sep string) map[string]string {
	stats := make(map[string]string, strings.Count(s, sep)+1)
	s2 := strings.Split(s, sep)
	for _, s := range s2 {
		list := strings.SplitN(s, "=", 2)
		switch len(list) {
		case 0, 1:
		case 2:
			stats[list[0]] = list[1]
		default:
			stats[list[0]] = strings.Join(list[1:], "=")
		}
	}

	return stats
}

func convertValue(s string) (float64, error) {
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f, nil
	}

	if b, err := strconv.ParseBool(s); err == nil {
		if b {
			return 1, nil
		}
		return 0, nil
	}

	// fmt.Println("input string is ", s, " **** returning 0")
	return 0, fmt.Errorf("invalid value `%s`. Only Float or Boolean are accepted", s)
}

func isHelperAllowedMetric(rawStatName string, allowlist []string) bool {
	// consider, disabled-allowlist and empty-allowlist as same, stat is accepted and allowed
	if len(allowlist) == 0 {
		return true
	}

	for _, cfgStatPattern := range allowlist {
		if globbingPattern.MatchString(cfgStatPattern) {
			ge := glob.MustCompile(cfgStatPattern)

			if ge.Match(rawStatName) {
				return true
			}
		} else {
			if rawStatName == cfgStatPattern {
				return true
			}
		}
	}

	return false
}

func isHelperBlockedMetric(rawMetricName string, blocklist []string) bool {
	if len(blocklist) == 0 {
		return false
	}

	for _, stat := range blocklist {
		if globbingPattern.MatchString(stat) {
			ge := glob.MustCompile(stat)

			if ge.Match(rawMetricName) {
				fmt.Println("isHelperBlockedMetric: given stat ", rawMetricName, " is matching block-list-pattern: ", stat)
				return true
			}
		} else {
			if rawMetricName == stat {
				return true
			}
		}
	}

	return false
}
