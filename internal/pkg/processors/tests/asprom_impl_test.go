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

const (
	UNIQUE_METRICS_COUNT = 489
)

var DEFAULT_PROM_URL = "http://localhost:9145/metrics"

var metrics_from_prom = []string{}

func Test_Initialize_Prom_Exporter(t *testing.T) {

	fmt.Println("initializing config ... Test_Initialize_Prom_Exporter")
	// initialize config and gauge-lists
	initConfigsAndGauges()

	// initialize prom
	initialize_prom_processor()

	// generate and validate labels, global call, only once else Prom will not serve metrics as there is no change
	metrics_from_prom = make_http_call_to_prom_processor(t, nil)
}

func Test_RefreshDefault(t *testing.T) {

	fmt.Println("initializing config ... Test_RefreshDefault")

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

func Test_A_Unique_Metrics_Count(t *testing.T) {

	fmt.Println("initializing config ... Test_Unique_Metrics_Count")

	var unique_metric_names = make(map[string]string)

	// find unique metric-names (excluding Label and values)
	for idx_metric := range metrics_from_prom {
		metric := metrics_from_prom[idx_metric]
		fmt.Println("* metric: ", metric, "\t", strings.Index(metric, "{"))
		metric_name := metric
		if strings.Index(metric, "{") > 0 {
			metric_name = metric[0:strings.Index(metric, "{")]
		}

		unique_metric_names[metric_name] = metric_name
	}

	assert.Equal(t, len(unique_metric_names), UNIQUE_METRICS_COUNT, "No of Metrics dispatched to Prom CHANGED")
}

/**
* makes a http call to the running prom and returns the output
 */
func make_http_call_to_prom_processor(t *testing.T, asMetrics []watchers.AerospikeStat) []string {
	// prometheus http server is initialized
	httpClient := http.Client{Timeout: time.Duration(1) * time.Second}
	resp, err := httpClient.Get(DEFAULT_PROM_URL)

	if err != nil {
		fmt.Println("Error while reading Http Response: ", err)
	}
	defer resp.Body.Close()

	// reinitialize global array
	metrics_from_prom = []string{}

	scanner := bufio.NewScanner(resp.Body)
	// fmt.Println("*** START ")
	for scanner.Scan() {
		text := scanner.Text()
		// fmt.Println(text)
		if len(text) > 0 && strings.HasPrefix(text, "aerospike_") {
			metrics_from_prom = append(metrics_from_prom, strings.TrimSpace(text))
		}
	}
	// fmt.Println("*** END ")

	if err := scanner.Err(); err != nil {
		fmt.Println("Error while reading Http Response: ", err)
	}

	assert.NotEmpty(t, metrics_from_prom, " NO metrics received from prom running locally")

	return metrics_from_prom
}

// Data fetch helpers functions

func initialize_prom_processor() {
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

// config initialization
var TESTS_DEFAULT_GAUGE_LIST_FILE = "configs/gauge_stats_list.toml"

func initConfigsAndGauges() {
	// Initialize and validate Gauge config
	// initialize config and gauge-lists
	config.InitConfig(tests_utils.GetConfigfileLocation(tests_utils.TESTS_MOCK_CONFIG_FILE))

	// l_cwd, _ := os.Getwd()
	// config.InitGaugeStats(l_cwd + "/../../../../" + TESTS_DEFAULT_GAUGE_LIST_FILE)
	config.InitGaugeStats(tests_utils.GetConfigfileLocation(tests_utils.TESTS_DEFAULT_GAUGE_LIST_FILE))

}
