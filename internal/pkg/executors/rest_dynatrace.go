package executors

import (
	"fmt"
	"strings"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/statprocessors"
)

// Dynatrace metrics line format: metricName,labels metricType,value Optional[timestamp_ms]
const DT_METRIC_FORMAT = "%s,%s %s,%.1f"

func (re *RestExecutor) dtProcessMetrics(commonLabels []string, refreshedStats []statprocessors.AerospikeStat) {
	labels := []string{}

	labels = append(labels, commonLabels...)
	labels = append(labels, re.dtLabels()...)

	// aerospike server is up and we are able to fetch data
	re.dtSendNodeUp(labels, 1.0)

	// Batch metrics - collect at least 500 before sending
	var batchSize = 500
	metricBatch := make([]string, 0, batchSize)
	var metricType string

	// loop through the aerospike metrics and collect them in batches
	for _, stat := range refreshedStats {
		// qualifiedName := stat.QualifyMetricContext() + "_" + NormalizeMetric(stat.Name)
		// qualifiedName := "aerospike.server." + string(stat.Context) + "." + NormalizeMetric(stat.Name)
		qualifiedName := "aserver." + string(stat.Context) + "." + NormalizeMetric(stat.Name)

		metricLabels := []string{}
		for idx, label := range stat.Labels {
			metricLabels = append(metricLabels, fmt.Sprintf("%s=%s", label, stat.LabelValues[idx]))
		}

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
			re.sendMetrics(metricBatch)
			metricBatch = metricBatch[:0] // Reset slice but keep capacity
		}
	}

	// Send any remaining metrics (less than batchSize)
	if len(metricBatch) > 0 {
		re.sendMetrics(metricBatch)
	}
}

func (re *RestExecutor) dtSendNodeUp(labels []string, value float64) {
	// metricName := "aerospike.server.node_up"
	metricName := "aserver.node_up"
	metricType := "gauge"
	metricLabels := strings.Join(labels, ",")

	// Format: metricName,labels metricType value
	nodeUpMetric := fmt.Sprintf(DT_METRIC_FORMAT, metricName, metricLabels, metricType, value)

	fmt.Println(nodeUpMetric)

	re.sendMetrics([]string{nodeUpMetric})
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
