package processors

import (
	"bufio"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/processors"
	tests_utils "github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/tests_utils"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/watchers"
	"github.com/stretchr/testify/assert"
)

func Test_RefreshDefault(t *testing.T) {

	fmt.Println("initializing config ... Test_RefreshDefault")

	// initialize config and gauge-lists
	config.InitConfig(tests_utils.GetConfigfileLocation(tests_utils.TESTS_DEFAULT_CONFIG_FILE))

	// initialize prom
	initialize_prom()
	// generate and validate labels
	all_runTestcase(t, nil)
}

/**
* complete logic to call watcher, generate-mock data and asset is part of this function
 */
func all_runTestcase(t *testing.T, asMetrics []watchers.AerospikeStat) {
	// prometheus http server is initialized
	httpClient := http.Client{Timeout: time.Duration(1) * time.Second}
	resp, err := httpClient.Get("http://localhost:9145/metrics")

	if err != nil {
		fmt.Println("Error while reading Http Response: ", err)
	}
	defer resp.Body.Close()

	metrics_from_prom := []string{}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		text := scanner.Text()
		if len(text) > 0 && strings.HasPrefix(text, "aerospike_") {
			metrics_from_prom = append(metrics_from_prom, strings.TrimSpace(text))
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error while reading Http Response: ", err)
	}

	assert.NotEmpty(t, metrics_from_prom, " received metrics from prom running locally")

	udh := &tests_utils.UnittestDataHandler{}
	pdv := udh.GetUnittestValidator("prometheus")
	expectedOutputs := pdv.GetMetricLabelsWithValues()

	assert.Equal(t, len(expectedOutputs), len(metrics_from_prom))

	// assert values from httpclient with expectedOutputs
	for idx_metrics := range metrics_from_prom {
		entry := metrics_from_prom[idx_metrics]
		assert.Contains(t, expectedOutputs, entry)
	}
}

// Data fetch helpers functions

func initialize_prom() {
	metric_processors := processors.GetMetricProcessors()
	processor := metric_processors[processors.PROM]

	// run Prom as a separate process
	go func() {
		err := processor.Initialize()
		if err != nil {
			fmt.Println("Error while Initializing Processor ", err)
		}
	}()

	fmt.Println("*******************\nPrometheus initialized and running on localhost:9145")
}
