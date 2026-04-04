package executors

import (
	"context"
	"fmt"
	"sync/atomic"

	"time"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/dataprovider"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/statprocessors"
	log "github.com/sirupsen/logrus"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

type GaugeMetrics struct {
	instrument metric.Int64ObservableGauge
	labels     []attribute.KeyValue
	value      atomic.Int64 // int64, avoid read/write race condition
}

type CounterMetrics struct {
	instrument metric.Int64ObservableCounter
	labels     []attribute.KeyValue
	value      atomic.Int64 // int64, avoid read/write race condition
}

// OTelExecutor is class on its own and also implements and embeds sdkmetric.Exporter interface
//
//	this will give control on various operations like Export, Temporality, Aggregation, ForceFlush, Shutdown
type OtelExecutor struct {
	sdkmetric.Exporter
	// KEY =  one metric + labels
	// Each measurement/instrument = one metric + labels + latest value
	gauges           map[string]*GaugeMetrics
	counters         map[string]*CounterMetrics
	nodeUpMetricName string

	meterProvider *sdkmetric.MeterProvider
	// metricExporter sdkmetric.Exporter

	refreshCounter atomic.Int64

	dataProvider            dataprovider.DataProvider
	sharedState             *statprocessors.StatProcessorSharedState
	statsRefresher          *statprocessors.StatsRefresher
	hostSystemInfoProcessor *statprocessors.HostSystemInfoProcessor
}

// sdkmetric.Exporter interface implementation
func (oe *OtelExecutor) Export(ctx context.Context, rm *metricdata.ResourceMetrics) error {

	// do not send metrics to end point to avoid large spikes in delta when exporter restarts
	if oe.dataProvider.IsServerConnected() && oe.refreshCounter.Load() <= 2 {
		// Return nil to simulate a successful export without actually sending data
		log.Debugf("%s OtelExecutor, Ignoring refresh of metrics export %d", time.Now().Format(time.RFC3339), oe.refreshCounter.Load())
		return nil
	}

	return oe.Exporter.Export(ctx, rm)
}

// Aerospike Otel metrics serving implementation
//
// Initializes an OTLP exporter, and configures the corresponding metric providers
func (oe *OtelExecutor) Initialize() error {
	log.Info("*** Initializing Otel Exporter... ")
	log.Debugf("** OTel endpoint %s", config.Cfg.Agent.Otel.Endpoint)
	log.Debugf("** OTel service name %s", config.Cfg.Agent.Otel.ServiceName)

	// Initialize the stats refresher
	oe.dataProvider = dataprovider.GetProvider(commons.EXECUTOR_MODE_OTEL)
	oe.sharedState = statprocessors.NewStatProcessorSharedState()

	oe.statsRefresher = statprocessors.NewStatsRefresher(oe.dataProvider, oe.sharedState)
	oe.hostSystemInfoProcessor = statprocessors.NewHostSystemInfoProcessor(oe.sharedState)

	// initialize the metric caches
	oe.gauges = make(map[string]*GaugeMetrics)
	oe.counters = make(map[string]*CounterMetrics)

	defaultContext := context.Background()

	resource, err := resource.New(defaultContext,
		resource.WithFromEnv(),
		resource.WithProcess(),
		// resource.WithTelemetrySDK(),
		resource.WithHost(),
		resource.WithContainer(),
		resource.WithAttributes(
			// the service name used to display traces/metrics in backends
			semconv.ServiceNameKey.String(config.Cfg.Agent.Otel.ServiceName),
		),
	)

	if err != nil {
		log.Fatalf("Failed to create OTel Resource %v", err)
	}

	// var exporter sdkmetric.Exporter
	oe.refreshCounter.Store(0)

	if config.Cfg.Agent.Otel.HttpEndpoint != "" {
		oe.Exporter, err = oe.BuildHttpExporter(defaultContext)
	} else {
		// either grpc_endpoint or endpoint is configured
		oe.Exporter, err = oe.BuildGrpcExporter(defaultContext)
	}

	if err != nil {
		log.Fatalf("Failed to create the collector metric exporter %v", err)
	}

	oe.meterProvider = sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(resource),
		sdkmetric.WithReader(
			sdkmetric.NewPeriodicReader(
				oe,
				sdkmetric.WithInterval(time.Duration(config.Cfg.Agent.Otel.PushInterval)*time.Second),
			),
		),
	)

	log.Info("*** Starting Otel Metrics Push thread... ")

	// Start metric collection loop in a goroutine
	go func() {
		ticker := time.NewTicker(time.Duration(config.Cfg.Agent.Otel.ServerStatFetchInterval) * time.Second)
		defer ticker.Stop()

		meter := oe.meterProvider.Meter(config.Cfg.Agent.Otel.ServiceName + "_Meter")

		// defaultCtx := context.Background()
		commonLabels := oe.getCommonLabels()

		oe.nodeUpMetricName = fmt.Sprintf("%s%s%s",
			config.Cfg.Agent.Otel.MetricNamePrefix,
			METRIC_CONTEXT_SEPARATOR[config.Cfg.Agent.Otel.MetricContextSeparator],
			"node_up")

		for {
			// Wait for next tick or shutdown signal
			select {
			case t := <-ticker.C:
				// Ticker drops events from channel buffer if previous event is still in-progress
				log.Debugf("\t *** ticker.C: %s", t.Format(time.RFC3339))

				// Aerospike Refresh stats
				oe.handleAerospikeMetrics(meter, commonLabels)

				// System metrics
				oe.handleSystemInfoMetrics(meter, commonLabels)

			case <-commons.ProcessExit:
				// Exit immediately if shutdown signal received
				log.Info("OTel executor received shutdown signal, shutting down...")
				cxt, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				// pushes any last exports to the receiver
				if err := oe.meterProvider.Shutdown(cxt); err != nil {
					otel.Handle(err)
				}
				return
			}
		}
	}()

	return nil
}

// func (oe *OtelExecutor) BuildGrpcExporter(resource *resource.Resource) (*sdkmetric.MeterProvider, error) {
func (oe *OtelExecutor) BuildGrpcExporter(ctx context.Context) (sdkmetric.Exporter, error) {
	headers := oe.readHeaders()

	var exporter *otlpmetricgrpc.Exporter
	var err error

	// Build options conditionally
	exporterOptions := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithHeaders(headers),
		otlpmetricgrpc.WithEndpoint(config.Cfg.Agent.Otel.GrpcEndpoint),
		otlpmetricgrpc.WithTemporalitySelector(oe.getTemporalitySelector),
	}

	// //NOTE: Testing purposes only. Only add WithInsecure() when TLS is disabled
	// if !config.Cfg.Agent.Otel.OtelTlsEnabled {
	// 	exporterOptions = append(exporterOptions, otlpmetricgrpc.WithInsecure())
	// }

	log.Infof("Creating Otel MetricsExporter with GrpcEndpoint: %s", config.Cfg.Agent.Otel.GrpcEndpoint)
	exporter, err = otlpmetricgrpc.New(ctx, exporterOptions...)

	return exporter, err

}

func (oe *OtelExecutor) BuildHttpExporter(ctx context.Context) (sdkmetric.Exporter, error) {
	headers := oe.readHeaders()

	var err error
	var exporter *otlpmetrichttp.Exporter

	// tlsConfig := &tls.Config{
	// 	InsecureSkipVerify: false,
	// }

	// Build options conditionally
	exporterOptions := []otlpmetrichttp.Option{
		otlpmetrichttp.WithHeaders(headers),
		otlpmetrichttp.WithEndpointURL(config.Cfg.Agent.Otel.HttpEndpoint),
		otlpmetrichttp.WithTemporalitySelector(oe.getTemporalitySelector),
		// otlpmetrichttp.WithTLSClientConfig(tlsConfig),
	}

	// NOTE: Testing purposes only. Only add WithInsecure() when TLS is disabled
	// if !config.Cfg.Agent.Otel.OtelTlsEnabled {
	// 	exporterOptions = append(exporterOptions, otlpmetrichttp.WithInsecure())
	// }

	log.Infof("Creating Otel MetricsExporter with HttpEndpoint: %s", config.Cfg.Agent.Otel.HttpEndpoint)
	exporter, err = otlpmetrichttp.New(ctx, exporterOptions...)

	return exporter, err
}

// Gauges don't have temporality (they're instantaneous values), as SDK still calls this selector.
// For gauges, the SDK will ignore the temporality setting.
// For counters, we are using Delta temporality to ensure that the metrics are compatible with Dynatrace and Datadog
//
// NOTE: Dynatrace and Datadog does not support MONOTONIC_CUMULATIVE_SUM - Aerospike counters are monotonic
//
//	So, we are using Delta temporality for counters,  histograms and gauges
//	This is to ensure that the metrics are compatible with Dynatrace and Datadog
//	* avoid any issues with the metrics collection
//	* ensure that the metrics are compatible with Dynatrace, New Relic and Datadog
func (oe *OtelExecutor) getTemporalitySelector(instrumentKind sdkmetric.InstrumentKind) metricdata.Temporality {

	if instrumentKind == sdkmetric.InstrumentKindObservableCounter &&
		config.Cfg.Agent.Otel.CounterTemporality == commons.TEMPORALITY_DELTA {

		return metricdata.DeltaTemporality
	}

	return metricdata.CumulativeTemporality
}

func (oe *OtelExecutor) handleAerospikeMetrics(meter metric.Meter, commonLabels []attribute.KeyValue) {

	asRefreshStats, err := oe.statsRefresher.Refresh()

	labels := []attribute.KeyValue{
		attribute.String("aerospike_cluster", oe.sharedState.ClusterName),
		attribute.String("aerospike_service", oe.sharedState.Service),
		attribute.String("build", oe.sharedState.Build),
		attribute.String("node_id", oe.sharedState.NodeId),
	}

	if err != nil {
		log.Errorf("Error while refreshing Aerospike Metrics, error: %v", err)

		// Reset counter, so when server comes back we do not send large counter value again
		oe.refreshCounter.Store(0)

		// some providers do not accept empty values for label-values,
		//   if exporter starts and unable to connect server, NodeId will be null/empty
		if oe.sharedState.NodeId != "" && oe.sharedState.ClusterName != "" && oe.sharedState.Service != "" {
			oe.sendNodeUp(meter, append(commonLabels, labels...), 0)
			return
		}

		// aerospike server is down, send common labels
		oe.sendNodeUp(meter, commonLabels, 0)
		return
	}

	// aerospike server is up and we are able to fetch data, send common + server labels
	oe.sendNodeUp(meter, append(commonLabels, labels...), 1)

	// Increment counter if server refresh is successful
	oe.refreshCounter.Add(1)
	log.Debugf("%s Aerospike refreshCounter incremented to %d", time.Now().Format(time.RFC3339), oe.refreshCounter.Load())

	// process metrics
	oe.processAndPushStats(meter, commonLabels, asRefreshStats)

}

func (oe *OtelExecutor) handleSystemInfoMetrics(meter metric.Meter, commonLabels []attribute.KeyValue) {
	sysInfoRefreshStats, err := oe.hostSystemInfoProcessor.RefreshSystemInfo()

	if err != nil {
		log.Errorln("Error while refreshing SystemInfo, error: ", err)
		return
	}

	// process metrics
	oe.processAndPushStats(meter, commonLabels, sysInfoRefreshStats)
}

// Utility functions
func (oe *OtelExecutor) readHeaders() map[string]string {
	headers := make(map[string]string)
	// headers["api-key"] = "abcdefghijklmnopqrstuvwxyz"
	headerPairs := config.Cfg.Agent.Otel.Headers

	if len(headerPairs) > 0 {
		for k, v := range headerPairs {
			headers[k] = v
		}
	}
	log.Debugf("** OTel header count %d", len(headers))

	return headers
}
