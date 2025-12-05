package executors

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/statprocessors"
	log "github.com/sirupsen/logrus"
)

type RestExecutor struct {
	httpClient *http.Client
}

// Exporter interface implementation
//
// Initializes an REST exporter, and configures the corresponding metric providers
func (re RestExecutor) Initialize() error {

	log.Infof("*** Initializing REST Exporter... ")

	serviceName := config.Cfg.Agent.Rest.ServiceName
	restEndpoint := config.Cfg.Agent.Rest.Endpoint
	headers := config.Cfg.Agent.Rest.Headers

	log.Infof("*** Starting REST Metrics Push thread... Service Name: %s, Endpoint: %s, Headers: %v", serviceName, restEndpoint, headers)

	log.Infof("*** Starting Otel Metrics Push thread... ")

	// Start metric collection loop in a goroutine
	go func() {
		ticker := time.NewTicker(time.Duration(config.Cfg.Agent.Rest.ServerStatFetchInterval) * time.Second)
		defer ticker.Stop()
		commonLabels := re.getCommonLabels()

		for {
			// Wait for next tick or shutdown signal
			select {
			case <-ticker.C:
				re.refreshMetrics(commonLabels)
			case <-commons.ProcessExit:
				// Exit immediately if shutdown signal received
				log.Infof("OTel executor received shutdown signal, shutting down...")
				_, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				return
			}
		}
	}()

	return nil
}

func (re RestExecutor) refreshMetrics(commonLabels []string) {
	asRefreshStats, err := statprocessors.Refresh()

	if err != nil {
		log.Errorln("Error while refreshing Aerospike Metrics, error: ", err)
		re.sendNodeUp(commonLabels, 0.0)
		return
	}

	// aerospike server is up and we are able to fetch data
	// sendNodeUp(meter, commonLabels, 1.0)

	log.Debugf("Aerospike Metrics refreshed: %v", asRefreshStats)

}

func (re *RestExecutor) getHttpClient() *http.Client {
	// Check if client is nil or invalid, then recreate it
	timeout := time.Duration(config.Cfg.Agent.Rest.Timeout) * time.Second

	if re.httpClient == nil {

		log.Debugf("HTTP client is nil or invalid or Timeout mismatch, creating new client")

		re.httpClient = &http.Client{
			Timeout: timeout,
		}
	}
	return re.httpClient
}

func (re *RestExecutor) sendNodeUp(labels []string, value float64) {
	// check and use existing connection or create a new one
	metricName := "aerospike.server.node_up"
	metricType := "gauge"
	metricLabels := strings.Join(labels, ",")

	// Format: metricName,labels metricType value
	nodeUpMetric := fmt.Sprintf("%s,%s %s %.1f", metricName, metricLabels, metricType, value)

	// Get or create HTTP client
	client := re.getHttpClient()

	// Create request body with raw data (equivalent to curl --data-raw)
	// This sends the data as-is without any encoding
	req, err := re.sendRequest(config.Cfg.Agent.Rest.Endpoint, bytes.NewBufferString(nodeUpMetric))
	if err != nil {
		log.Errorf("Error creating HTTP request: %v", err)
		return
	}

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("Error sending HTTP request: %v", err)
		return
	}
	defer resp.Body.Close() // nolint: errcheck

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Warnf("HTTP request returned non-success status: %d", resp.StatusCode)
	} else {
		log.Debugf("Successfully sent node_up metric: %s", nodeUpMetric)
	}
}

func (re RestExecutor) sendRequest(url string, reqBody *bytes.Buffer) (*http.Request, error) {
	// Create HTTP request
	req, err := http.NewRequest("POST", config.Cfg.Agent.Rest.Endpoint, reqBody)
	if err != nil {
		log.Errorf("Error creating HTTP request: %v", err)
		return nil, err
	}

	// Set headers from config
	headers := config.Cfg.Agent.Rest.Headers
	for key, val := range headers {
		req.Header.Set(key, val)
	}

	req.Header.Set("Content-Type", "text/plain")

	return req, nil
}

func (re RestExecutor) getCommonLabels() []string {
	// aerospike.node.up,cluster=prod,ns=cplane,node=1 gauge,0.0/1.0

	mlabels := config.Cfg.Agent.MetricLabels
	labels := []string{}

	if len(mlabels) > 0 {
		for k, v := range mlabels {
			labels = append(labels, fmt.Sprintf("%s=%s", k, v))
		}
	}

	if statprocessors.ClusterName != "" {
		labels = append(labels, fmt.Sprintf("cluster_name=%s", statprocessors.ClusterName))
	}
	if statprocessors.Service != "" {
		labels = append(labels, fmt.Sprintf("service=%s", statprocessors.Service))
	}
	if statprocessors.Build != "" {
		labels = append(labels, fmt.Sprintf("build=%s", statprocessors.Build))
	}

	return labels
}
