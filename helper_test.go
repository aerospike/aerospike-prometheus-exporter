package main

import (
	"fmt"
	"strconv"
	"strings"
)

var DEFAULT_APE_TOML = "tests/default_ape.toml"
var NS_ALLOWLIST_APE_TOML = "tests/ns_allow_block_list_ape.toml"

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
