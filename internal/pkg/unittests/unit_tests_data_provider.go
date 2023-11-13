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

type UnittestDataProvider struct {
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
			}
		}
	}
}

func (md *UnittestDataProvider) GetExpectedPassTwoKeys(key string) []string {
	var results []string

	return results
}