package tests_utils

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

/*
Dummy Raw Metrics, copied from local Aerospike Server
returns static test data copied from running an Aerospike Server with test namespaces, sets, sindex, jobs, latencies etc.,
we need to update this data for each release to reflect the new metrics, contexts etc.,
this data is passed to the watcher and expected output is also generated
once we have output from watcher-implementations ( like watcher_namespaces.go, watcher_node_stats.go)

	this output is compated with the expected results generated by Test-Cases
*/

var TEST_PROM_DATA_FILE = "tests_data/default_prom_mock_results.txt"
var TEST_WATCHER_DATA_FILE = "tests_data/watcher_mock_results.txt"

// read mock test data from a file
var Is_Unittests_Initialized = 0

type UnittestDataValidator interface {
	Initialize(data []string)
	GetPassOneKeys() map[string]string
	GetPassTwoKeys() map[string]string
	GetMetricLabelsWithValues() map[string]string
}

type UnittestDataHandler struct {
}

func (md *UnittestDataHandler) Initialize() {

	fmt.Println("Unittest Initializing ....: ")
	// avoid multiple initializations
	// if Is_Unittests_Initialized == 1 {
	// 	fmt.Println("Unittest data provider already Initialized: ")
	// 	return
	// }
	// Mark as initialized
	Is_Unittests_Initialized = 1

	// load expected test data from mock files
	md.loadPrometheusData()
	md.loadWatchersData()
}

var (
	namespace_validator  = &NamespaceUnittestValidator{}
	node_stats_validator = &NodeUnittestValidator{}
	xdr_validator        = &XdrUnittestValidator{}
	sets_validator       = &SetsUnittestValidator{}
	sindex_validator     = &SindexUnittestValidator{}
	latency_validator    = &LatencyUnittestValidator{}
	users_validator      = &UsersUnittestValidator{}
	prom_validator       = &PrometheusUnittestValidator{}
)
var validators = map[string]UnittestDataValidator{
	"namespace":  namespace_validator,
	"node":       node_stats_validator,
	"xdr":        xdr_validator,
	"sets":       sets_validator,
	"sindex":     sindex_validator,
	"latency":    latency_validator,
	"users":      users_validator,
	"prometheus": prom_validator,
}

func (md *UnittestDataHandler) GetUnittestValidator(key string) UnittestDataValidator {
	md.Initialize()

	return validators[key]
}

// Internal helper functions
func (md *UnittestDataHandler) loadPrometheusData() {
	filePath := TEST_PROM_DATA_FILE
	cwd, _ := os.Getwd()
	fileLocation := cwd + "/" + filePath
	readFile, err := os.Open(fileLocation)

	if err != nil {
		fmt.Println(err)
	}

	fileScanner := bufio.NewScanner(readFile)
	fileScanner.Split(bufio.ScanLines)
	var fileLines []string

	for fileScanner.Scan() {
		fileLines = append(fileLines, strings.TrimSpace(fileScanner.Text()))
	}

	readFile.Close()

	// Initialize prom_validator data
	prom_validator.Initialize(fileLines)

	// for _, line := range fileLines {
	// 	// fmt.Println("Prometheus_Label_and_Values: ", line)
	// 	if len(line) > 0 && strings.HasPrefix(line, "aerospike_") {
	// 		prom_validator.Metrics = append(prom_validator.Metrics, strings.TrimSpace(line))
	// 	}
	// }
	fmt.Println("loadPrometheusData(): Completed loading test Prometheus Expected Data ")

}

func (md *UnittestDataHandler) loadWatchersData() {

	filePath := TEST_WATCHER_DATA_FILE
	cwd, _ := os.Getwd()
	fileLocation := cwd + "/" + filePath
	// fmt.Println(" current working directory:", cwd)
	// fmt.Println(" using filepath : ", fileLocation)
	readFile, err := os.Open(fileLocation)

	if err != nil {
		fmt.Println(err)
	}

	fileScanner := bufio.NewScanner(readFile)
	fileScanner.Split(bufio.ScanLines)
	var fileLines []string

	for fileScanner.Scan() {
		fileLines = append(fileLines, strings.TrimSpace(fileScanner.Text()))
	}

	readFile.Close()

	// loop all validators and initialize
	for _, validator := range validators {
		validator.Initialize(fileLines)
	}

	fmt.Println("loadWatchersData(): Completed loading test WATCHER Expected Data ")
}

// ===========================
//
// Start Namespace
type NamespaceUnittestValidator struct {
	PassOneOutputs []string
	PassTwoOutputs []string
	Metrics        []string
}

func (unp NamespaceUnittestValidator) Initialize(data []string) {
	for _, line := range data {
		if strings.HasPrefix(line, "#") && strings.HasPrefix(line, "//") {
			// ignore, comments
		} else if len(line) > 0 {
			if strings.HasPrefix(line, "namespace-passonekeys:") {
				namespace_validator.PassOneOutputs = append(namespace_validator.PassOneOutputs, line)
			} else if strings.HasPrefix(line, "namespace-passtwokeys:") {
				namespace_validator.PassTwoOutputs = append(namespace_validator.PassTwoOutputs, line)
			} else if strings.HasPrefix(line, "watchers.AerospikeStat{Context:\"namespace\",") {
				namespace_validator.Metrics = append(namespace_validator.Metrics, line)
			} else if strings.HasPrefix(line, "node-passonekeys:") {
				node_stats_validator.PassOneOutputs = append(node_stats_validator.PassOneOutputs, line)
			} else if strings.HasPrefix(line, "node-passtwokeys:") {
				node_stats_validator.PassTwoOutputs = append(node_stats_validator.PassTwoOutputs, line)
			} else if strings.HasPrefix(line, "watchers.AerospikeStat{Context:\"node_stats\",") {
				node_stats_validator.Metrics = append(node_stats_validator.Metrics, line)
			} else if strings.HasPrefix(line, "xdr-passonekeys:") {
				xdr_validator.PassOneOutputs = append(xdr_validator.PassOneOutputs, line)
			} else if strings.HasPrefix(line, "xdr-passtwokeys:") {
				xdr_validator.PassTwoOutputs = append(xdr_validator.PassTwoOutputs, line)
			} else if strings.HasPrefix(line, "watchers.AerospikeStat{Context:\"xdr\",") {
				xdr_validator.Metrics = append(xdr_validator.Metrics, line)
			} else if strings.HasPrefix(line, "sets-passonekeys:") {
				sets_validator.PassOneOutputs = append(sets_validator.PassOneOutputs, line)
			} else if strings.HasPrefix(line, "sets-passtwokeys:") {
				sets_validator.PassTwoOutputs = append(sets_validator.PassTwoOutputs, line)
			} else if strings.HasPrefix(line, "watchers.AerospikeStat{Context:\"sets\",") {
				sets_validator.Metrics = append(sets_validator.Metrics, line)
			} else if strings.HasPrefix(line, "sindex-passonekeys:") {
				sindex_validator.PassOneOutputs = append(sindex_validator.PassOneOutputs, line)
			} else if strings.HasPrefix(line, "sindex-passtwokeys:") {
				sindex_validator.PassTwoOutputs = append(sindex_validator.PassTwoOutputs, line)
			} else if strings.HasPrefix(line, "watchers.AerospikeStat{Context:\"sindex\",") {
				sindex_validator.Metrics = append(sindex_validator.Metrics, line)
			} else if strings.HasPrefix(line, "latency-passonekeys:") {
				latency_validator.PassOneOutputs = append(latency_validator.PassOneOutputs, line)
			} else if strings.HasPrefix(line, "latency-passtwokeys:") {
				latency_validator.PassTwoOutputs = append(latency_validator.PassTwoOutputs, line)
			} else if strings.HasPrefix(line, "watchers.AerospikeStat{Context:\"latencies\",") {
				latency_validator.Metrics = append(latency_validator.Metrics, line)
			} else if strings.HasPrefix(line, "users-passonekeys:") {
				users_validator.PassOneOutputs = append(users_validator.PassOneOutputs, line)
			} else if strings.HasPrefix(line, "users-passtwokeys:") {
				users_validator.PassTwoOutputs = append(users_validator.PassTwoOutputs, line)
			} else if strings.HasPrefix(line, "watchers.AerospikeStat{Context:\"users\",") {
				users_validator.Metrics = append(users_validator.Metrics, line)
			}
		}
	}
}

func (unp NamespaceUnittestValidator) GetPassOneKeys() map[string]string {
	var outputs = make(map[string]string)
	elements := unp.PassOneOutputs[0]
	elements = strings.Replace(elements, "namespace-passonekeys:", "", 1)
	elements = strings.Replace(elements, "]", "", 1)
	elements = strings.Replace(elements, "[", "", 1)

	outputs["namespaces"] = elements

	return outputs
}

func (unp NamespaceUnittestValidator) GetPassTwoKeys() map[string]string {
	var outputs = make(map[string]string)

	// fmt.Println("GetPassTwoKeys: ", udp.Namespace_PassTwo)

	out_values := unp.PassTwoOutputs[0]
	out_values = strings.Replace(out_values, "namespace-passtwokeys:", "", 1)
	out_values = strings.Replace(out_values, "[", "", 1)
	out_values = strings.Replace(out_values, "]", "", 1)
	elements := strings.Split(out_values, " ")
	for i := 0; i < len(elements); i++ {
		// fmt.Println(" adding namespace: ", elements[i], " - as key to ", i)
		outputs["namespace_"+strconv.Itoa(i)] = elements[i]
	}

	return outputs
}

func (unp NamespaceUnittestValidator) GetMetricLabelsWithValues() map[string]string {

	var outputs = make(map[string]string)
	for k := range unp.Metrics {
		outputs[unp.Metrics[k]] = unp.Metrics[k]
	}

	return outputs
}

// End Namespace

// Start Node
type NodeUnittestValidator struct {
	PassOneOutputs []string
	PassTwoOutputs []string
	Metrics        []string
}

func (unp NodeUnittestValidator) Initialize(data []string) {
	for _, line := range data {
		if strings.HasPrefix(line, "#") && strings.HasPrefix(line, "//") {
			// ignore, comments
		} else if len(line) > 0 {
			if strings.HasPrefix(line, "namespace-passonekeys:") {
				namespace_validator.PassOneOutputs = append(namespace_validator.PassOneOutputs, line)
			} else if strings.HasPrefix(line, "namespace-passtwokeys:") {
				namespace_validator.PassTwoOutputs = append(namespace_validator.PassTwoOutputs, line)
			} else if strings.HasPrefix(line, "watchers.AerospikeStat{Context:\"namespace\",") {
				namespace_validator.Metrics = append(namespace_validator.Metrics, line)
			} else if strings.HasPrefix(line, "node-passonekeys:") {
				node_stats_validator.PassOneOutputs = append(node_stats_validator.PassOneOutputs, line)
			} else if strings.HasPrefix(line, "node-passtwokeys:") {
				node_stats_validator.PassTwoOutputs = append(node_stats_validator.PassTwoOutputs, line)
			} else if strings.HasPrefix(line, "watchers.AerospikeStat{Context:\"node_stats\",") {
				node_stats_validator.Metrics = append(node_stats_validator.Metrics, line)
			} else if strings.HasPrefix(line, "xdr-passonekeys:") {
				xdr_validator.PassOneOutputs = append(xdr_validator.PassOneOutputs, line)
			} else if strings.HasPrefix(line, "xdr-passtwokeys:") {
				xdr_validator.PassTwoOutputs = append(xdr_validator.PassTwoOutputs, line)
			} else if strings.HasPrefix(line, "watchers.AerospikeStat{Context:\"xdr\",") {
				xdr_validator.Metrics = append(xdr_validator.Metrics, line)
			} else if strings.HasPrefix(line, "sets-passonekeys:") {
				sets_validator.PassOneOutputs = append(sets_validator.PassOneOutputs, line)
			} else if strings.HasPrefix(line, "sets-passtwokeys:") {
				sets_validator.PassTwoOutputs = append(sets_validator.PassTwoOutputs, line)
			} else if strings.HasPrefix(line, "watchers.AerospikeStat{Context:\"sets\",") {
				sets_validator.Metrics = append(sets_validator.Metrics, line)
			} else if strings.HasPrefix(line, "sindex-passonekeys:") {
				sindex_validator.PassOneOutputs = append(sindex_validator.PassOneOutputs, line)
			} else if strings.HasPrefix(line, "sindex-passtwokeys:") {
				sindex_validator.PassTwoOutputs = append(sindex_validator.PassTwoOutputs, line)
			} else if strings.HasPrefix(line, "watchers.AerospikeStat{Context:\"sindex\",") {
				sindex_validator.Metrics = append(sindex_validator.Metrics, line)
			} else if strings.HasPrefix(line, "latency-passonekeys:") {
				latency_validator.PassOneOutputs = append(latency_validator.PassOneOutputs, line)
			} else if strings.HasPrefix(line, "latency-passtwokeys:") {
				latency_validator.PassTwoOutputs = append(latency_validator.PassTwoOutputs, line)
			} else if strings.HasPrefix(line, "watchers.AerospikeStat{Context:\"latencies\",") {
				latency_validator.Metrics = append(latency_validator.Metrics, line)
			} else if strings.HasPrefix(line, "users-passonekeys:") {
				users_validator.PassOneOutputs = append(users_validator.PassOneOutputs, line)
			} else if strings.HasPrefix(line, "users-passtwokeys:") {
				users_validator.PassTwoOutputs = append(users_validator.PassTwoOutputs, line)
			} else if strings.HasPrefix(line, "watchers.AerospikeStat{Context:\"users\",") {
				users_validator.Metrics = append(users_validator.Metrics, line)
			}
		}
	}
}

func (unp NodeUnittestValidator) GetPassOneKeys() map[string]string {

	return nil
}

func (unp NodeUnittestValidator) GetPassTwoKeys() map[string]string {
	var outputs = make(map[string]string)

	// fmt.Println("GetPassTwoKeys: ", udp.Namespace_PassTwo)

	out_values := unp.PassTwoOutputs[0]
	out_values = strings.Replace(out_values, "node-passtwokeys:", "", 1)
	out_values = strings.Replace(out_values, "[", "", 1)
	out_values = strings.Replace(out_values, "]", "", 1)
	elements := strings.Split(out_values, " ")
	for i := 0; i < len(elements); i++ {
		// fmt.Println(" adding namespace: ", elements[i], " - as key to ", i)
		outputs["node_stat_"+strconv.Itoa(i)] = elements[i]
	}

	return outputs
}

func (unp NodeUnittestValidator) GetMetricLabelsWithValues() map[string]string {
	var outputs = make(map[string]string)
	for k := range unp.Metrics {
		outputs[unp.Metrics[k]] = unp.Metrics[k]
	}

	return outputs
}

// End Node

// Start Xdr
type XdrUnittestValidator struct {
	PassOneOutputs []string
	PassTwoOutputs []string
	Metrics        []string
}

func (unp XdrUnittestValidator) Initialize(data []string) {
	for _, line := range data {
		if strings.HasPrefix(line, "#") && strings.HasPrefix(line, "//") {
			// ignore, comments
		} else if len(line) > 0 {
			if strings.HasPrefix(line, "xdr-passonekeys:") {
				xdr_validator.PassOneOutputs = append(xdr_validator.PassOneOutputs, line)
			} else if strings.HasPrefix(line, "xdr-passtwokeys:") {
				xdr_validator.PassTwoOutputs = append(xdr_validator.PassTwoOutputs, line)
			} else if strings.HasPrefix(line, "watchers.AerospikeStat{Context:\"xdr\",") {
				xdr_validator.Metrics = append(xdr_validator.Metrics, line)
			}
		}
	}
}

func (unp XdrUnittestValidator) GetPassOneKeys() map[string]string {
	var outputs = make(map[string]string)
	elements := unp.PassOneOutputs[0]
	elements = strings.Replace(elements, "xdr-passonekeys:", "", 1)
	elements = strings.Replace(elements, "]", "", 1)
	elements = strings.Replace(elements, "[", "", 1)

	outputs["xdr"] = elements

	return outputs
}

func (unp XdrUnittestValidator) GetPassTwoKeys() map[string]string {
	var outputs = make(map[string]string)

	out_values := unp.PassTwoOutputs[0]
	out_values = strings.Replace(out_values, "xdr-passtwokeys:", "", 1)
	out_values = strings.Replace(out_values, "[", "", 1)
	out_values = strings.Replace(out_values, "]", "", 1)
	elements := strings.Split(out_values, " ")
	for i := 0; i < len(elements); i++ {
		outputs["xdr_"+strconv.Itoa(i)] = elements[i]
	}

	return outputs
}

func (unp XdrUnittestValidator) GetMetricLabelsWithValues() map[string]string {
	var outputs = make(map[string]string)
	for k := range unp.Metrics {
		outputs[unp.Metrics[k]] = unp.Metrics[k]
	}

	return outputs
}

// End Xdr

// Start Sets
type SetsUnittestValidator struct {
	PassOneOutputs []string
	PassTwoOutputs []string
	Metrics        []string
}

func (unp SetsUnittestValidator) Initialize(data []string) {
	for _, line := range data {
		if strings.HasPrefix(line, "#") && strings.HasPrefix(line, "//") {
			// ignore, comments
		} else if len(line) > 0 {
			if strings.HasPrefix(line, "sets-passonekeys:") {
				sets_validator.PassOneOutputs = append(sets_validator.PassOneOutputs, line)
			} else if strings.HasPrefix(line, "sets-passtwokeys:") {
				sets_validator.PassTwoOutputs = append(sets_validator.PassTwoOutputs, line)
			} else if strings.HasPrefix(line, "watchers.AerospikeStat{Context:\"sets\",") {
				sets_validator.Metrics = append(sets_validator.Metrics, line)
			}
		}
	}
}

func (unp SetsUnittestValidator) GetPassOneKeys() map[string]string {
	return nil
}

func (unp SetsUnittestValidator) GetPassTwoKeys() map[string]string {
	var outputs = make(map[string]string)

	out_values := unp.PassTwoOutputs[0]
	out_values = strings.Replace(out_values, "sets-passtwokeys:", "", 1)
	out_values = strings.Replace(out_values, "[", "", 1)
	out_values = strings.Replace(out_values, "]", "", 1)
	elements := strings.Split(out_values, " ")
	for i := 0; i < len(elements); i++ {
		outputs["sets_"+strconv.Itoa(i)] = elements[i]
	}

	return outputs
}

func (unp SetsUnittestValidator) GetMetricLabelsWithValues() map[string]string {
	var outputs = make(map[string]string)
	for k := range unp.Metrics {
		outputs[unp.Metrics[k]] = unp.Metrics[k]
	}

	return outputs
}

// End Sets

// Start Sindex
type SindexUnittestValidator struct {
	PassOneOutputs []string
	PassTwoOutputs []string
	Metrics        []string
}

func (unp SindexUnittestValidator) Initialize(data []string) {
	for _, line := range data {
		if strings.HasPrefix(line, "#") && strings.HasPrefix(line, "//") {
			// ignore, comments
		} else if len(line) > 0 {
			if strings.HasPrefix(line, "sindex-passonekeys:") {
				sindex_validator.PassOneOutputs = append(sindex_validator.PassOneOutputs, line)
			} else if strings.HasPrefix(line, "sindex-passtwokeys:") {
				sindex_validator.PassTwoOutputs = append(sindex_validator.PassTwoOutputs, line)
			} else if strings.HasPrefix(line, "watchers.AerospikeStat{Context:\"sindex\",") {
				sindex_validator.Metrics = append(sindex_validator.Metrics, line)
			}
		}
	}
}

func (unp SindexUnittestValidator) GetPassOneKeys() map[string]string {
	var outputs = make(map[string]string)
	elements := unp.PassOneOutputs[0]
	elements = strings.Replace(elements, "sindex-passonekeys:", "", 1)
	elements = strings.Replace(elements, "]", "", 1)
	elements = strings.Replace(elements, "[", "", 1)

	outputs["sindex"] = elements

	return outputs
}

func (unp SindexUnittestValidator) GetPassTwoKeys() map[string]string {
	var outputs = make(map[string]string)

	out_values := unp.PassTwoOutputs[0]
	out_values = strings.Replace(out_values, "sindex-passtwokeys:", "", 1)
	out_values = strings.Replace(out_values, "[", "", 1)
	out_values = strings.Replace(out_values, "]", "", 1)
	elements := strings.Split(out_values, " ")
	for i := 0; i < len(elements); i++ {
		outputs["sindex_"+strconv.Itoa(i)] = elements[i]
	}

	return outputs
}

func (unp SindexUnittestValidator) GetMetricLabelsWithValues() map[string]string {
	var outputs = make(map[string]string)
	for k := range unp.Metrics {
		outputs[unp.Metrics[k]] = unp.Metrics[k]
	}

	return outputs
}

// End Sindex

// Start Latency
type LatencyUnittestValidator struct {
	PassOneOutputs []string
	PassTwoOutputs []string
	Metrics        []string
}

func (unp LatencyUnittestValidator) Initialize(data []string) {
	for _, line := range data {
		if strings.HasPrefix(line, "#") && strings.HasPrefix(line, "//") {
			// ignore, comments
		} else if len(line) > 0 {
			if strings.HasPrefix(line, "latency-passonekeys:") {
				latency_validator.PassOneOutputs = append(latency_validator.PassOneOutputs, line)
			} else if strings.HasPrefix(line, "latency-passtwokeys:") {
				latency_validator.PassTwoOutputs = append(latency_validator.PassTwoOutputs, line)
			} else if strings.HasPrefix(line, "watchers.AerospikeStat{Context:\"latencies\",") {
				latency_validator.Metrics = append(latency_validator.Metrics, line)
			}
		}
	}
}

func (unp LatencyUnittestValidator) GetPassOneKeys() map[string]string {
	return nil
}

func (unp LatencyUnittestValidator) GetPassTwoKeys() map[string]string {
	var outputs = make(map[string]string)

	out_values := unp.PassTwoOutputs[0]
	out_values = strings.Replace(out_values, "latency-passtwokeys:", "", 1)
	out_values = strings.Replace(out_values, "[", "", 1)
	out_values = strings.Replace(out_values, "]", "", 1)
	elements := strings.Split(out_values, " ")
	for i := 0; i < len(elements); i++ {
		outputs["latency_"+strconv.Itoa(i)] = elements[i]
	}

	return outputs
}

func (unp LatencyUnittestValidator) GetMetricLabelsWithValues() map[string]string {
	var outputs = make(map[string]string)
	for k := range unp.Metrics {
		outputs[unp.Metrics[k]] = unp.Metrics[k]
	}

	return outputs
}

// End Latency

// Start Latency
type UsersUnittestValidator struct {
	PassOneOutputs []string
	PassTwoOutputs []string
	Metrics        []string
}

func (unp UsersUnittestValidator) Initialize(data []string) {
	for _, line := range data {
		if strings.HasPrefix(line, "#") && strings.HasPrefix(line, "//") {
			// ignore, comments
		} else if len(line) > 0 {
			if strings.HasPrefix(line, "users-passonekeys:") {
				users_validator.PassOneOutputs = append(users_validator.PassOneOutputs, line)
			} else if strings.HasPrefix(line, "users-passtwokeys:") {
				users_validator.PassTwoOutputs = append(users_validator.PassTwoOutputs, line)
			} else if strings.HasPrefix(line, "watchers.AerospikeStat{Context:\"users\",") {
				users_validator.Metrics = append(users_validator.Metrics, line)
			}
		}
	}
}

func (unp UsersUnittestValidator) GetPassOneKeys() map[string]string {
	return nil
}

func (unp UsersUnittestValidator) GetPassTwoKeys() map[string]string {

	return nil
}

func (unp UsersUnittestValidator) GetMetricLabelsWithValues() map[string]string {
	var outputs = make(map[string]string)
	for k := range unp.Metrics {
		outputs[unp.Metrics[k]] = unp.Metrics[k]
	}

	return outputs
}

// End Latency

// Start Prometheus
type PrometheusUnittestValidator struct {
	PassOneOutputs []string
	PassTwoOutputs []string
	Metrics        []string
}

func (unp PrometheusUnittestValidator) Initialize(data []string) {
	for _, line := range data {
		// fmt.Println("Prometheus_Label_and_Values: ", line)
		if len(line) > 0 && strings.HasPrefix(line, "aerospike_") {
			prom_validator.Metrics = append(prom_validator.Metrics, strings.TrimSpace(line))
		}
	}

}

func (unp PrometheusUnittestValidator) GetPassOneKeys() map[string]string {
	return nil
}

func (unp PrometheusUnittestValidator) GetPassTwoKeys() map[string]string {

	return nil
}

func (unp PrometheusUnittestValidator) GetMetricLabelsWithValues() map[string]string {
	var outputs = make(map[string]string)
	for k := range unp.Metrics {
		outputs[unp.Metrics[k]] = unp.Metrics[k]
	}

	return outputs
}

// End Prometheus
