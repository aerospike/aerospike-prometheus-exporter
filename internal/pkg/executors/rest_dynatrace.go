package executors

import (
	"fmt"
	"strings"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/statprocessors"
)

const DT_METRIC_FORMAT = "%s,%s %s,%.1f"

func (re *RestExecutor) dtProcessMetrics(commonLabels []string, asRefreshStats []statprocessors.AerospikeStat) {
	// aerospike server is up and we are able to fetch data
	re.dtSendNodeUp(commonLabels, 1.0)

	// Batch metrics - collect at least 100 before sending
	var batchSize = 500
	metricBatch := make([]string, 0, batchSize)
	var metricType string

	// loop through the aerospike metrics and collect them in batches
	for _, stat := range asRefreshStats {
		// qualifiedName := stat.QualifyMetricContext() + "_" + NormalizeMetric(stat.Name)
		qualifiedName := stat.QualifyMetricContext() + ".server." + NormalizeMetric(stat.Name)

		metricLabels := []string{}
		for idx, label := range stat.Labels {
			metricLabels = append(metricLabels, fmt.Sprintf("%s=%s", label, stat.LabelValues[idx]))
		}

		if stat.MType == commons.MetricTypeGauge {
			metricType = "gauge"
		} else {
			metricType = "count"
		}

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
	metricName := "aerospike.server.node_up"
	metricType := "gauge"
	metricLabels := strings.Join(labels, ",")

	// Format: metricName,labels metricType value
	nodeUpMetric := fmt.Sprintf(DT_METRIC_FORMAT, metricName, metricLabels, metricType, value)

	re.sendMetrics([]string{nodeUpMetric})
}
