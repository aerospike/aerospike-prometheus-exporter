package executors

import (
	"bytes"
	"context"
	"fmt"
	"io"
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

	log.Infof("*** Starting REST Metrics Push thread... Service Name: %s, Endpoint: %s ", serviceName, restEndpoint)

	// Start metric collection loop in a goroutine
	go func() {
		ticker := time.NewTicker(time.Duration(config.Cfg.Agent.Rest.ServerStatFetchInterval) * time.Second)
		defer ticker.Stop()

		for {
			// Wait for next tick or shutdown signal
			select {
			case <-ticker.C:
				asRefreshStats, err := statprocessors.Refresh()

				commonLabels := re.getCommonLabels()

				if err != nil {
					log.Errorln("Error while refreshing Aerospike Metrics, error: ", err)
					re.dtSendNodeUp(commonLabels, 0.0)
					return
				}

				re.dtProcessMetrics(commonLabels, asRefreshStats)
				//TODO: for Dynatrace do we need to send systeminfo metrics? -- no, it will be handled by Dynatrace
			case <-commons.ProcessExit:
				// Exit immediately if shutdown signal received
				log.Infof("REST Executor received shutdown signal, shutting down...")
				_, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				return
			}
		}
	}()

	return nil
}

func (re *RestExecutor) createRequest(url string, reqBody *bytes.Buffer) (*http.Request, error) {
	// Create HTTP request
	req, err := http.NewRequest("POST", url, reqBody)
	if err != nil {
		log.Errorf("Error creating HTTP request: url=%s, error=%v", url, err)
		return nil, err
	}

	// Set headers from config
	for key, val := range config.Cfg.Agent.Rest.Headers {
		req.Header.Set(key, val)
	}

	req.Header.Set("Content-Type", "text/plain")

	return req, nil
}

func (re *RestExecutor) sendMetrics(metrics []string) {
	if len(metrics) == 0 {
		return
	}

	// Join all metrics with newlines (Dynatrace accepts multiple metrics separated by newlines)
	metricsBody := strings.Join(metrics, "\n")

	log.Debugf("Sending batch of %d metrics to Dynatrace", len(metrics))

	// Create request body with raw data (equivalent to curl --data-raw)
	req, err := re.createRequest(config.Cfg.Agent.Rest.Endpoint, bytes.NewBufferString(metricsBody))
	if err != nil {
		log.Errorf("Error creating HTTP request: %v", err)
		return
	}

	// Get or create HTTP client
	client := re.getHttpClient()

	// Send request
	resp, err := client.Do(req)

	if err != nil {
		log.Errorf("Error sending HTTP request: %v", err)
		return
	}
	body, err := io.ReadAll(resp.Body)

	if err != nil {
		log.Errorf("Error reading HTTP response body: %v", err)
		return
	}

	defer resp.Body.Close() // nolint: errcheck

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Warnf("Batch HTTP Request Failed, returned non-success status: %d, response: %s", resp.StatusCode, string(body))
		fmt.Println("Batch HTTP Request Failed, returned non-success status: ", resp.StatusCode, string(body))
	} else {
		log.Debugf("Successfully sent batch of %d metrics", len(metrics))
		fmt.Println("Successfully sent batch of metrics to Dynatrace", len(metrics))
	}
}

// Utility functions

func (re *RestExecutor) getHttpClient() *http.Client {
	// Check if client is nil or invalid, then recreate it
	timeout := time.Duration(config.Cfg.Agent.Rest.Timeout) * time.Second

	if re.httpClient == nil {
		log.Infof("HTTP client is nil, creating new client with timeout: %v", timeout)

		// Create HTTP client with proper timeout
		// Default transport handles HTTPS automatically
		re.httpClient = &http.Client{
			Timeout: timeout,
		}
	}
	return re.httpClient
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
