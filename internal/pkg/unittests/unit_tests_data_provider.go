package unittests

import (
	"bufio"
	"fmt"
	"os"
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

var TEST_DATA_FILE = "tests/mock_test_data.txt"

// read mock test data from a file
var Is_Unittests_Initialized = 0

type UnittestDataValidator interface {
	GetPassOneKeys(udp UnittestDataProvider) map[string]string
	GetPassTwoKeys(udp UnittestDataProvider) map[string]string
	CompareMetricLabelsWithValues(udp UnittestDataProvider, metrics map[string]string) bool
}

type UnittestDataProvider struct {
	Namespace_PassOne           []string
	Namespace_PassTwo           []string
	Namespaces_Label_and_Values []string
}

func (md *UnittestDataProvider) Initialize() {

	// avoid multiple initializations
	if Is_Unittests_Initialized == 1 {
		// fmt.Println("Mock data provider already Initialized: ")
		return
	}
	// Mark as initialized
	Is_Unittests_Initialized = 1

	filePath := TEST_DATA_FILE
	readFile, err := os.Open(filePath)

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
			} else if strings.HasPrefix(line, "namespace-passonekeys:") {
				md.Namespace_PassOne = append(md.Namespace_PassOne, line)
			} else if strings.HasPrefix(line, "namespace-passtwokeys:") {
				md.Namespace_PassTwo = append(md.Namespace_PassTwo, line)
			}
		}
	}
}

func (md *UnittestDataProvider) GetUnittestValidator(key string) UnittestDataValidator {

	md.Initialize()

	switch key {
	case "namespace":
		return &UnittestNamespaceValidator{}
	}
	return nil
}

type UnittestNamespaceValidator struct {
	PassOneOutputs []string
	PassTwoOutputs []string
	Metrics        []string
}

func (unp UnittestNamespaceValidator) GetPassOneKeys(udp UnittestDataProvider) map[string]string {
	var outputs = make(map[string]string)
	outputs["namespaces"] = udp.Namespace_PassOne[0]

	return outputs
}

func (unp UnittestNamespaceValidator) GetPassTwoKeys(udp UnittestDataProvider) map[string]string {
	var outputs = make(map[string]string)

	out_values := udp.Namespace_PassTwo[0]
	out_values = strings.Replace(out_values, "[", "", 1)
	out_values = strings.Replace(out_values, "]", "", 1)
	elements := strings.Split(out_values, " ")
	for i := 0; i < len(elements); i++ {
		outputs[elements[i]] = elements[i]
	}

	return outputs
}

func (unp UnittestNamespaceValidator) CompareMetricLabelsWithValues(udp UnittestDataProvider, metrics map[string]string) bool {
	return false
}
