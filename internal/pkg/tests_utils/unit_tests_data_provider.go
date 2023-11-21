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

var TEST_PROM_DATA_FILE = "tests_data/exporter_mock_results.txt"
var TEST_WATCHER_DATA_FILE = "tests_data/watcher_mock_results.txt"

// read mock test data from a file
var Is_Unittests_Initialized = 0

type UnittestDataValidator interface {
	GetPassOneKeys(udp UnittestDataHandler) map[string]string
	GetPassTwoKeys(udp UnittestDataHandler) map[string]string
	GetMetricLabelsWithValues(udh UnittestDataHandler) map[string]string
}

type UnittestDataHandler struct {
	// Watchers - Namespace
	Namespace_PassOne           []string
	Namespace_PassTwo           []string
	Namespaces_Label_and_Values []string

	// Watchers - Node
	Node_PassOne          []string
	Node_PassTwo          []string
	Node_Label_and_Values []string

	// Watchers - Xdr
	Xdr_PassOne          []string
	Xdr_PassTwo          []string
	Xdr_Label_and_Values []string

	// Watchers - Sets
	Sets_PassOne          []string
	Sets_PassTwo          []string
	Sets_Label_and_Values []string

	// Watchers - Sindex
	Sindex_PassOne          []string
	Sindex_PassTwo          []string
	Sindex_Label_and_Values []string

	// Watchers - Latency
	Latency_PassOne          []string
	Latency_PassTwo          []string
	Latency_Label_and_Values []string

	// Watchers - Users
	Users_PassOne          []string
	Users_PassTwo          []string
	Users_Label_and_Values []string
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

func (md *UnittestDataHandler) GetUnittestValidator(key string) UnittestDataValidator {

	md.Initialize()

	switch key {
	case "namespace":
		return &NamespaceUnittestValidator{}
	case "node":
		return &NodeUnittestValidator{}
	case "xdr":
		return &XdrUnittestValidator{}
	case "sets":
		return &SetsUnittestValidator{}
	case "sindex":
		return &SindexUnittestValidator{}
	case "latency":
		return &LatencyUnittestValidator{}
	case "users":
		return &UsersUnittestValidator{}
	}
	return nil
}

// Internal helper functions
func (md *UnittestDataHandler) loadPrometheusData() {
	filePath := TEST_PROM_DATA_FILE
	cwd, _ := os.Getwd()
	fileLocation := cwd + "/../../../" + filePath
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

	for _, line := range fileLines {
		if strings.HasPrefix(line, "#") && strings.HasPrefix(line, "//") {
			// ignore, comments
		} else if len(line) > 0 {
			if strings.HasPrefix(line, "namespace_expected_output:") {
				md.Namespaces_Label_and_Values = append(md.Namespaces_Label_and_Values, line)
			}
		}
	}
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

	for _, line := range fileLines {
		if strings.HasPrefix(line, "#") && strings.HasPrefix(line, "//") {
			// ignore, comments
		} else if len(line) > 0 {
			if strings.HasPrefix(line, "namespace-passonekeys:") {
				md.Namespace_PassOne = append(md.Namespace_PassOne, line)
			} else if strings.HasPrefix(line, "namespace-passtwokeys:") {
				md.Namespace_PassTwo = append(md.Namespace_PassTwo, line)
			} else if strings.HasPrefix(line, "watchers.AerospikeStat{Context:\"namespace\",") {
				md.Namespaces_Label_and_Values = append(md.Namespaces_Label_and_Values, line)
			} else if strings.HasPrefix(line, "node-passonekeys:") {
				md.Node_PassOne = append(md.Node_PassOne, line)
			} else if strings.HasPrefix(line, "node-passtwokeys:") {
				md.Node_PassTwo = append(md.Node_PassTwo, line)
			} else if strings.HasPrefix(line, "watchers.AerospikeStat{Context:\"node_stats\",") {
				md.Node_Label_and_Values = append(md.Node_Label_and_Values, line)
			} else if strings.HasPrefix(line, "xdr-passonekeys:") {
				md.Xdr_PassOne = append(md.Xdr_PassOne, line)
			} else if strings.HasPrefix(line, "xdr-passtwokeys:") {
				md.Xdr_PassTwo = append(md.Xdr_PassTwo, line)
			} else if strings.HasPrefix(line, "watchers.AerospikeStat{Context:\"xdr\",") {
				md.Xdr_Label_and_Values = append(md.Xdr_Label_and_Values, line)
			} else if strings.HasPrefix(line, "sets-passonekeys:") {
				md.Sets_PassOne = append(md.Sets_PassOne, line)
			} else if strings.HasPrefix(line, "sets-passtwokeys:") {
				md.Sets_PassTwo = append(md.Sets_PassTwo, line)
			} else if strings.HasPrefix(line, "watchers.AerospikeStat{Context:\"sets\",") {
				md.Sets_Label_and_Values = append(md.Sets_Label_and_Values, line)
			} else if strings.HasPrefix(line, "sindex-passonekeys:") {
				md.Sindex_PassOne = append(md.Sindex_PassOne, line)
			} else if strings.HasPrefix(line, "sindex-passtwokeys:") {
				md.Sindex_PassTwo = append(md.Sindex_PassTwo, line)
			} else if strings.HasPrefix(line, "watchers.AerospikeStat{Context:\"sindex\",") {
				md.Sindex_Label_and_Values = append(md.Sindex_Label_and_Values, line)
			} else if strings.HasPrefix(line, "latency-passonekeys:") {
				md.Latency_PassOne = append(md.Latency_PassOne, line)
			} else if strings.HasPrefix(line, "latency-passtwokeys:") {
				md.Latency_PassTwo = append(md.Latency_PassTwo, line)
			} else if strings.HasPrefix(line, "watchers.AerospikeStat{Context:\"latencies\",") {
				md.Latency_Label_and_Values = append(md.Latency_Label_and_Values, line)
			} else if strings.HasPrefix(line, "users-passonekeys:") {
				md.Users_PassOne = append(md.Latency_PassOne, line)
			} else if strings.HasPrefix(line, "users-passtwokeys:") {
				md.Users_PassTwo = append(md.Latency_PassTwo, line)
			} else if strings.HasPrefix(line, "watchers.AerospikeStat{Context:\"users\",") {
				md.Users_Label_and_Values = append(md.Users_Label_and_Values, line)
			}
		}
	}
	// fmt.Println("\n\n****\nmd.Node_PassTwo: ", md.Node_PassTwo, "\n\n***")
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

func (unp NamespaceUnittestValidator) GetPassOneKeys(udh UnittestDataHandler) map[string]string {
	var outputs = make(map[string]string)
	elements := udh.Namespace_PassOne[0]
	elements = strings.Replace(elements, "namespace-passonekeys:", "", 1)
	elements = strings.Replace(elements, "]", "", 1)
	elements = strings.Replace(elements, "[", "", 1)

	outputs["namespaces"] = elements

	return outputs
}

func (unp NamespaceUnittestValidator) GetPassTwoKeys(udh UnittestDataHandler) map[string]string {
	var outputs = make(map[string]string)

	// fmt.Println("GetPassTwoKeys: ", udp.Namespace_PassTwo)

	out_values := udh.Namespace_PassTwo[0]
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

func (unp NamespaceUnittestValidator) GetMetricLabelsWithValues(udh UnittestDataHandler) map[string]string {

	var outputs = make(map[string]string)
	for k := range udh.Namespaces_Label_and_Values {
		outputs[udh.Namespaces_Label_and_Values[k]] = udh.Namespaces_Label_and_Values[k]
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

func (unp NodeUnittestValidator) GetPassOneKeys(udh UnittestDataHandler) map[string]string {

	return nil
}

func (unp NodeUnittestValidator) GetPassTwoKeys(udh UnittestDataHandler) map[string]string {
	var outputs = make(map[string]string)

	// fmt.Println("GetPassTwoKeys: ", udp.Namespace_PassTwo)

	out_values := udh.Node_PassTwo[0]
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

func (unp NodeUnittestValidator) GetMetricLabelsWithValues(udh UnittestDataHandler) map[string]string {
	var outputs = make(map[string]string)
	for k := range udh.Node_Label_and_Values {
		outputs[udh.Node_Label_and_Values[k]] = udh.Node_Label_and_Values[k]
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

func (unp XdrUnittestValidator) GetPassOneKeys(udh UnittestDataHandler) map[string]string {
	var outputs = make(map[string]string)
	elements := udh.Xdr_PassOne[0]
	elements = strings.Replace(elements, "xdr-passonekeys:", "", 1)
	elements = strings.Replace(elements, "]", "", 1)
	elements = strings.Replace(elements, "[", "", 1)

	outputs["xdr"] = elements

	return outputs
}

func (unp XdrUnittestValidator) GetPassTwoKeys(udh UnittestDataHandler) map[string]string {
	var outputs = make(map[string]string)

	out_values := udh.Xdr_PassTwo[0]
	out_values = strings.Replace(out_values, "xdr-passtwokeys:", "", 1)
	out_values = strings.Replace(out_values, "[", "", 1)
	out_values = strings.Replace(out_values, "]", "", 1)
	elements := strings.Split(out_values, " ")
	for i := 0; i < len(elements); i++ {
		outputs["xdr_"+strconv.Itoa(i)] = elements[i]
	}

	return outputs
}

func (unp XdrUnittestValidator) GetMetricLabelsWithValues(udh UnittestDataHandler) map[string]string {
	var outputs = make(map[string]string)
	for k := range udh.Xdr_Label_and_Values {
		outputs[udh.Xdr_Label_and_Values[k]] = udh.Xdr_Label_and_Values[k]
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

func (unp SetsUnittestValidator) GetPassOneKeys(udh UnittestDataHandler) map[string]string {
	return nil
}

func (unp SetsUnittestValidator) GetPassTwoKeys(udh UnittestDataHandler) map[string]string {
	var outputs = make(map[string]string)

	out_values := udh.Sets_PassTwo[0]
	out_values = strings.Replace(out_values, "sets-passtwokeys:", "", 1)
	out_values = strings.Replace(out_values, "[", "", 1)
	out_values = strings.Replace(out_values, "]", "", 1)
	elements := strings.Split(out_values, " ")
	for i := 0; i < len(elements); i++ {
		outputs["sets_"+strconv.Itoa(i)] = elements[i]
	}

	return outputs
}

func (unp SetsUnittestValidator) GetMetricLabelsWithValues(udh UnittestDataHandler) map[string]string {
	var outputs = make(map[string]string)
	for k := range udh.Sets_Label_and_Values {
		outputs[udh.Sets_Label_and_Values[k]] = udh.Sets_Label_and_Values[k]
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

func (unp SindexUnittestValidator) GetPassOneKeys(udh UnittestDataHandler) map[string]string {
	var outputs = make(map[string]string)
	elements := udh.Sindex_PassOne[0]
	elements = strings.Replace(elements, "sindex-passonekeys:", "", 1)
	elements = strings.Replace(elements, "]", "", 1)
	elements = strings.Replace(elements, "[", "", 1)

	outputs["sindex"] = elements

	return outputs
}

func (unp SindexUnittestValidator) GetPassTwoKeys(udh UnittestDataHandler) map[string]string {
	var outputs = make(map[string]string)

	out_values := udh.Sindex_PassTwo[0]
	out_values = strings.Replace(out_values, "sindex-passtwokeys:", "", 1)
	out_values = strings.Replace(out_values, "[", "", 1)
	out_values = strings.Replace(out_values, "]", "", 1)
	elements := strings.Split(out_values, " ")
	for i := 0; i < len(elements); i++ {
		outputs["sindex_"+strconv.Itoa(i)] = elements[i]
	}

	return outputs
}

func (unp SindexUnittestValidator) GetMetricLabelsWithValues(udh UnittestDataHandler) map[string]string {
	var outputs = make(map[string]string)
	for k := range udh.Sindex_Label_and_Values {
		outputs[udh.Sindex_Label_and_Values[k]] = udh.Sindex_Label_and_Values[k]
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

func (unp LatencyUnittestValidator) GetPassOneKeys(udh UnittestDataHandler) map[string]string {
	return nil
}

func (unp LatencyUnittestValidator) GetPassTwoKeys(udh UnittestDataHandler) map[string]string {
	var outputs = make(map[string]string)

	out_values := udh.Latency_PassTwo[0]
	out_values = strings.Replace(out_values, "latency-passtwokeys:", "", 1)
	out_values = strings.Replace(out_values, "[", "", 1)
	out_values = strings.Replace(out_values, "]", "", 1)
	elements := strings.Split(out_values, " ")
	for i := 0; i < len(elements); i++ {
		outputs["latency_"+strconv.Itoa(i)] = elements[i]
	}

	return outputs
}

func (unp LatencyUnittestValidator) GetMetricLabelsWithValues(udh UnittestDataHandler) map[string]string {
	var outputs = make(map[string]string)
	for k := range udh.Latency_Label_and_Values {
		outputs[udh.Latency_Label_and_Values[k]] = udh.Latency_Label_and_Values[k]
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

func (unp UsersUnittestValidator) GetPassOneKeys(udh UnittestDataHandler) map[string]string {
	return nil
}

func (unp UsersUnittestValidator) GetPassTwoKeys(udh UnittestDataHandler) map[string]string {

	return nil
}

func (unp UsersUnittestValidator) GetMetricLabelsWithValues(udh UnittestDataHandler) map[string]string {
	var outputs = make(map[string]string)
	for k := range udh.Users_Label_and_Values {
		outputs[udh.Users_Label_and_Values[k]] = udh.Users_Label_and_Values[k]
	}

	return outputs
}

// End Latency
