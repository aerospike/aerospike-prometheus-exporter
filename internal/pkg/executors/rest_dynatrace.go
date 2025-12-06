package executors

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/statprocessors"
	log "github.com/sirupsen/logrus"
)

// Dynatrace metrics line format: metricName,labels metricType,value Optional[timestamp_ms]
const DT_METRIC_FORMAT = "%s,%s %s,%.1f"

func (re *RestExecutor) dtProcessMetrics(commonLabels []string, refreshedStats []statprocessors.AerospikeStat) {
	labels := []string{}

	labels = append(labels, commonLabels...)
	// labels = append(labels, re.dtLabels()...)

	// aerospike server is up and we are able to fetch data
	re.dtSendNodeUp(labels, 1.0)

	// Batch metrics - collect at least 500 before sending
	var batchSize = 500
	metricBatch := make([]string, 0, batchSize)
	var metricType string
	noErrorWhileSendingMetrics := true

	// loop through the aerospike metrics and collect them in batches
	for _, stat := range refreshedStats {
		// qualifiedName := stat.QualifyMetricContext() + "_" + NormalizeMetric(stat.Name)
		// qualifiedName := "aerospike.server." + string(stat.Context) + "." + NormalizeMetric(stat.Name)
		qualifiedName := "aserver." + string(stat.Context) + "." + NormalizeMetric(stat.Name)

		metricLabels := []string{}
		for idx, label := range stat.Labels {
			metricLabels = append(metricLabels, fmt.Sprintf("%s=%s", label, stat.LabelValues[idx]))
		}

		metricLabels = append(metricLabels, commonLabels...)

		// Dynatrace append .count for counters, we are sending all metrics as gauges
		metricType = "gauge"

		// if stat.MType == commons.MetricTypeGauge {
		// 	metricType = "gauge"
		// } else {
		// 	metricType = "count"
		// }

		formattedMetric := fmt.Sprintf(DT_METRIC_FORMAT, qualifiedName, strings.Join(metricLabels, ","), metricType, stat.Value)
		metricBatch = append(metricBatch, formattedMetric)

		// Send batch when it reaches the minimum size
		// Dynatrace max batch size is upto 1MB
		if len(metricBatch) >= batchSize {
			statusCode, responseBody := re.sendMetrics(metricBatch)

			if !re.dtProcessResponseBody(statusCode, responseBody) {
				noErrorWhileSendingMetrics = false
				break
			}
			metricBatch = metricBatch[:0] // Reset slice but keep capacity
		}
	}

	// Send any remaining metrics (less than batchSize)
	if noErrorWhileSendingMetrics && len(metricBatch) > 0 {
		statusCode, responseBody := re.sendMetrics(metricBatch)

		// Ignore the response body and status code, as we are sending the last batch of metrics
		re.dtProcessResponseBody(statusCode, responseBody)
	}
}

func (re *RestExecutor) dtSendNodeUp(labels []string, value float64) {
	// metricName := "aerospike.server.node_up"
	metricName := "aserver.node_up"
	metricType := "gauge"
	metricLabels := strings.Join(labels, ",")

	// Format: metricName,labels metricType value
	nodeUpMetric := fmt.Sprintf(DT_METRIC_FORMAT, metricName, metricLabels, metricType, value)

	statusCode, responseBody := re.sendMetrics([]string{nodeUpMetric})
	re.dtProcessResponseBody(statusCode, responseBody)
}

func (re *RestExecutor) dtLabels() []string {
	labels := []string{}

	if statprocessors.Service != "" {
		labels = append(labels, "dt.entity.host=HOST-"+statprocessors.Service)
	}

	if config.Cfg.Agent.Rest.ServiceName != "" {
		labels = append(labels, "dt.entity.service=SERVICE-"+config.Cfg.Agent.Rest.ServiceName)
	}

	return labels
}

func (re *RestExecutor) dtProcessResponseBody(statusCode int, responseBody []byte) bool {
	if statusCode < 200 || statusCode >= 300 {
		log.Errorf("Error while sending metrics to Dynatrace, statusCode: %d, responseBody: %s", statusCode, string(responseBody))
		return false
	}

	// Parse JSON response from Dynatrace
	// Success response -- {"linesOk":1,"linesInvalid":0,"error":null,"warnings":null}
	// Failure response -- {"linesOk":0,"linesInvalid":1,"error":
	// {"code":400,"message":"1 invalid lines","invalidLines":
	// [{"line":1,"error":"unexpected end of input","identifier":"aerospike_xdr_dc_namespace_ship_versions_interval"}]},"warnings":null}

	var response map[string]interface{}
	if err := json.Unmarshal(responseBody, &response); err != nil {
		log.Warnf("Failed to parse Dynatrace response JSON: %v, response body: %s", err, string(responseBody))
		return false
	}

	// Extract values from response
	linesOk, ok := response["linesOk"].(float64)
	if !ok {
		log.Warnf("Invalid or missing 'linesOk' in Dynatrace response: %s", string(responseBody))
		return false
	}

	linesInvalid, ok := response["linesInvalid"].(float64)
	if !ok {
		log.Warnf("Invalid or missing 'linesInvalid' in Dynatrace response: %s", string(responseBody))
		return false
	}

	// Check for error
	if errorVal, exists := response["error"]; exists && errorVal != nil {
		log.Errorf("Dynatrace ingestion error: %v", errorVal)
		return false
	}

	// Check if ingestion was successful
	success := int(linesOk) > 0 && int(linesInvalid) == 0

	if !success {
		log.Warnf("Metrics ingestion partially failed - linesOk: %d, linesInvalid: %d",
			int(linesOk), int(linesInvalid))
		return false
	}

	// Log warnings if present
	if warnings, exists := response["warnings"]; exists && warnings != nil {
		if warningsList, ok := warnings.([]interface{}); ok && len(warningsList) > 0 {
			log.Warnf("Dynatrace ingestion warnings: %v", warningsList)
		}
	}

	log.Debugf("Successfully ingested metrics to Dynatrace - linesOk: %d", int(linesOk))
	return true
}
