package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/gobwas/glob"
)

var DEFAULT_APE_TOML = "tests/default_ape.toml"
var LABELS_APE_TOML = "tests/labels_ape.toml"
var NS_ALLOWLIST_APE_TOML = "tests/ns_allowlist_ape.toml"
var NS_BLOCKLIST_APE_TOML = "tests/ns_blocklist_ape.toml"

var METRICS_CONFIG_FILE = "gauge_stats_list.toml"

var MOCK_TEST_DATA_FILE = "tests/mock_test_data.txt"

func extractNamespaceFromLabel(label string) string {
	// [name:"cluster_name" value:""  name:"ns" value:"bar"  name:"service" value:"" ]
	nsFromLabel := label
	nsFromLabel = nsFromLabel[(strings.Index(nsFromLabel, "ns"))+11:]
	nsFromLabel = nsFromLabel[0:(strings.Index(nsFromLabel, "\""))]

	return nsFromLabel
}

func stringifyLabel(label string) string {
	labelReplacerFunc := strings.NewReplacer(".", "_", "-", "_", " ", "_", "[", "_", "]", "_", "\"", "_", ":", "_", "/", "_")
	hypenReplacerFunc := strings.NewReplacer("_", "")

	// return labelReplacerFunc.Replace(label)
	return hypenReplacerFunc.Replace(labelReplacerFunc.Replace(label))
}

func extractLabelNameValueFromFullLabel(fullLabel string, reqName string) string {

	// Example Given Original: [name:"cluster_name" value:"null"  name:"service" value:"172.17.0.3:3000" ]
	// Example After Replacing: [name:cluster_name value:null  name:service value:172.17.0.3:3000 ]

	// labels will have individual strings like [ name:cluster_name value:null ] [name:service value:172.17.0.3:3000]
	value := extractNameValuePair(fullLabel, "name:"+reqName)

	return strings.TrimSpace(value)
}

func extractNameValuePair(fullLabel string, reqName string) string {

	// in the label-string, each label is separated by a double-space i.e."  "
	fullLabel = strings.ReplaceAll(fullLabel, "\"", "")
	fullLabel = strings.ReplaceAll(fullLabel, "[", "")
	fullLabel = strings.ReplaceAll(fullLabel, "]", "")

	arrLabels := strings.Split(fullLabel, "  ")
	for idx := range arrLabels {

		element := arrLabels[idx]
		if strings.HasPrefix(element, reqName) {
			// example: name:service value:172.17.0.3:3000
			// name := element[0:len(reqName)]
			// name = name[5:]

			from := len(reqName) + 7
			value := element[from:]

			return strings.TrimSpace(value)
		}
	}

	return ""
}

func extractMetricNameFromDesc(desc string) string {
	// Desc{fqName: "aerospike_namespac_memory_free_pct", help: "memory free pct", constLabels: {}, variableLabels: [cluster_name service ns]}
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
	temp_s := strings.TrimSpace(s)
	if f, err := strconv.ParseFloat(temp_s, 64); err == nil {
		return f, nil
	}

	if b, err := strconv.ParseBool(temp_s); err == nil {
		if b {
			return 1, nil
		}
		return 0, nil
	}

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

// Latency related utility functions

func splitLatencyDetails(latencies []string) (string, string, string) {

	// {test}-read:msec,0.0,0.00,0.00,0.00,0.00,0.00,0.00,0.00,0.00,0.00,0.00,0.00,0.00,0.00,0.00,0.00,0.00,0.00
	// batch-index:
	firstElement := latencies[0]
	firstElement = strings.ReplaceAll(firstElement, "{", "")
	firstElement = strings.ReplaceAll(firstElement, "}", "")
	firstElement = strings.ReplaceAll(firstElement, ":", "-")
	//
	// {test}-read:msec ==> test-read-msec
	// batch-index:     ==> batch-index-
	//
	elements := strings.Split(firstElement, "-")

	return elements[0], elements[1], elements[2]
}

func isLatencyOperationAllowe(operation string, allowlist []string, blocklist []string) bool {

	if len(blocklist) > 0 {
		for idx := range blocklist {
			op := blocklist[idx]
			if strings.EqualFold(op, operation) {
				return false
			}
		}
	}

	if len(allowlist) == 0 {
		return true
	}

	for idx := range allowlist {
		op := allowlist[idx]
		if strings.EqualFold(op, operation) {
			return true
		}
	}

	return false
}

func splitLatencies(latencyRawMetric string) []string {
	// {test}-read:msec,0.0,0.00,0.00,0.00,0.00,0.00,0.00,0.00,0.00,0.00,0.00,0.00,0.00,0.00,0.00,0.00,0.00,0.00
	// batch-index:
	arrLatencies := strings.Split(latencyRawMetric, ",")

	latencies := []string{}

	operation := strings.TrimSpace(arrLatencies[0])

	latencies = append(latencies, operation)
	if len(arrLatencies) > 1 {
		sumOfQueriesStr := arrLatencies[1]
		total, err := convertValue(sumOfQueriesStr)

		sumOfQueriesStr = fmt.Sprintf("%.0f", total)
		latencies = append(latencies, sumOfQueriesStr)
		if err != nil {
			fmt.Println(" unable to convert sumOfQueriesStr to float-value ", sumOfQueriesStr)
			return nil
		}
		for idx := range arrLatencies {
			// process from second element
			if idx >= 2 { // i.e. 3rd element in index
				// value, err := strconv.ParseFloat(arrLatencies[idx], 64)
				value, err := convertValue(arrLatencies[idx])

				// value, err := strconv.ParseInt(arrLatencies[idx], 10, 64)
				if err != nil {
					fmt.Println(" unable to convert latencies value to float-value ", latencies[idx], " at index: ", idx, " -- err: ", err)
					return nil
				}
				// value = total - value
				value = (total - ((value * total) / 100))
				convertedValue := fmt.Sprintf("%.0f", value)
				latencies = append(latencies, convertedValue)
			}
		}
	}

	return latencies
}

func constructLabelElement(labelKey string, labelValue string) string {
	return "  name:" + "\"" + labelKey + "\"" + " value:" + "\"" + labelValue + "\""
}

func extractOperationFromMetric(metricName string) string {
	operationName := strings.ReplaceAll(metricName, "aerospike_latencies_", "")

	operationName = operationName[0:strings.Index(operationName, "_")]

	return operationName
}

func copyConfigLabels() map[string]string {
	cfgLabels := config.AeroProm.MetricLabels

	returnLabelMap := make(map[string]string)
	for key, value := range cfgLabels {
		returnLabelMap[key] = value
	}

	return returnLabelMap
}

func createLabelByNames(labelsMap map[string]string) string {

	arr_label_names := []string{}
	createdLabelString := ""

	for key := range labelsMap {
		arr_label_names = append(arr_label_names, strings.TrimSpace(key))
	}

	// sort the keys
	sort.Strings(arr_label_names)

	for idx := range arr_label_names {
		keyName := arr_label_names[idx]
		value := labelsMap[keyName]
		createdLabelString = createdLabelString + constructLabelElement(keyName, strings.TrimSpace(value))
	}

	return strings.TrimSpace(createdLabelString)
}
