package processors

import (
	"bufio"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"
	tests_utils "github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/tests_utils"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/watchers"
)

func Test_RefreshDefault(t *testing.T) {

	fmt.Println("initializing config ... Test_RefreshDefault")

	// initialize config and gauge-lists
	config.InitConfig(tests_utils.GetConfigfileLocation(tests_utils.TESTS_DEFAULT_CONFIG_FILE))

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
		metrics_from_prom = append(metrics_from_prom, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error while reading Http Response: ", err)
	}

	if len(metrics_from_prom) > 0 {
		for idx_metrics := range metrics_from_prom {
			fmt.Println(metrics_from_prom[idx_metrics])
		}
	}
}

// Data fetch helpers functions

func initialize_prom() {
	metric_processors := GetMetricProcessors()
	processor := metric_processors[PROM]

	// run Prom as a separate process
	go processor.Initialize()
	fmt.Println("*******************\nPrometheus initialized and running on localhost:9145")
}
