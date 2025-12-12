package executors

import (
	"context"
	"crypto/tls"
	"sync"
	"sync/atomic"

	"time"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"
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
	instrument metric.Float64ObservableGauge
	labels     []attribute.KeyValue
	value      atomic.Value // float64, avoid read/write race condition
}

type CounterMetrics struct {
	instrument metric.Float64ObservableCounter
	labels     []attribute.KeyValue
	value      atomic.Value // float64, avoid read/write race condition
}

type OtelExecutor struct {
	// KEY =  one metric + labels
	// Each measurement/instrument = one metric + labels + latest value
	gauges   map[string]*GaugeMetrics
	counters map[string]*CounterMetrics

	mutex sync.Mutex
}

// Exporter interface implementation
// Aerospike Otel metrics serving implementation
//
// Initializes an OTLP exporter, and configures the corresponding metric providers
func (oe *OtelExecutor) Initialize() error {

	log.Infof("Otel sending thread started, sending data to : %s", config.Cfg.Agent.Otel.Endpoint)

	log.Infof("*** Initializing Otel Exporter... ")
	log.Debug("** OTel endpoint ", config.Cfg.Agent.Otel.Endpoint)
	log.Debug("** OTel service name ", config.Cfg.Agent.Otel.ServiceName)

	// initialize the gauges map
	oe.gauges = make(map[string]*GaugeMetrics)
	oe.counters = make(map[string]*CounterMetrics)

	defaultContext := context.Background()
	var meterProvider *sdkmetric.MeterProvider
	var metricExporter sdkmetric.Exporter

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

	if config.Cfg.Agent.Otel.HttpEndpoint != "" {
		metricExporter, err = oe.createHttpExporter(defaultContext)
	} else {
		// either grpc_endpoint or endpoint is configured
		metricExporter, err = oe.createGrpcExporter(defaultContext)
	}

	if err != nil {
		log.Fatalf("Failed to create the collector metric exporter %v", err)
	}

	meterProvider = sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(resource),
		sdkmetric.WithReader(
			sdkmetric.NewPeriodicReader(
				metricExporter,
				sdkmetric.WithInterval(time.Duration(config.Cfg.Agent.Otel.PushInterval)*time.Second),
			),
		),
	)

	otel.SetMeterProvider(meterProvider)

	log.Infof("*** Starting Otel Metrics Push thread... ")

	// Start metric collection loop in a goroutine
	go func() {
		ticker := time.NewTicker(time.Duration(config.Cfg.Agent.Otel.ServerStatFetchInterval) * time.Second)
		defer ticker.Stop()

		meter := otel.Meter(config.Cfg.Agent.Otel.ServiceName + "_Meter")
		defaultCtx := context.Background()
		commonLabels := oe.getCommonLabels()

		for {
			// Wait for next tick or shutdown signal
			select {
			case <-ticker.C:
				// Try to acquire lock non-blocking - skip if already locked
				// TODO: discuss with Sunil, if we can use a different approach to avoid the mutex
				if !oe.mutex.TryLock() {
					log.Debug("Skipping metrics collection - mutex already locked (previous collection still in progress)")
					continue
				}

				// Aerospike Refresh stats
				oe.handleAerospikeMetrics(meter, defaultCtx, commonLabels)

				// System metrics
				oe.handleSystemInfoMetrics(meter, defaultCtx, commonLabels)

				oe.mutex.Unlock()

			case <-commons.ProcessExit:
				// Exit immediately if shutdown signal received
				log.Infof("OTel executor received shutdown signal, shutting down...")
				cxt, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				// pushes any last exports to the receiver
				if err := meterProvider.Shutdown(cxt); err != nil {
					otel.Handle(err)
				}
				return
			}
		}
	}()

	return nil
}

// func (oe *OtelExecutor) createGrpcExporter(resource *resource.Resource) (*sdkmetric.MeterProvider, error) {
func (oe *OtelExecutor) createGrpcExporter(ctx context.Context) (sdkmetric.Exporter, error) {
	headers := oe.readHeaders()

	var metricExp *otlpmetricgrpc.Exporter
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
	metricExp, err = otlpmetricgrpc.New(ctx, exporterOptions...)

	return metricExp, err

}

// func (oe *OtelExecutor) createHttpExporter(resource *resource.Resource) (*sdkmetric.MeterProvider, error) {
func (oe *OtelExecutor) createHttpExporter(ctx context.Context) (sdkmetric.Exporter, error) {

	headers := oe.readHeaders()
	var err error
	var metricExp *otlpmetrichttp.Exporter

	// NOTE: Below is only for testing purposes
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}

	// Build options conditionally
	exporterOptions := []otlpmetrichttp.Option{
		otlpmetrichttp.WithHeaders(headers),
		otlpmetrichttp.WithEndpointURL(config.Cfg.Agent.Otel.HttpEndpoint),
		otlpmetrichttp.WithTemporalitySelector(oe.getTemporalitySelector),
		otlpmetrichttp.WithTLSClientConfig(tlsConfig),
	}

	// NOTE: Testing purposes only. Only add WithInsecure() when TLS is disabled
	// if !config.Cfg.Agent.Otel.OtelTlsEnabled {
	// 	exporterOptions = append(exporterOptions, otlpmetrichttp.WithInsecure())
	// }

	log.Infof("Creating Otel MetricsExporter with HttpEndpoint: %s", config.Cfg.Agent.Otel.HttpEndpoint)
	metricExp, err = otlpmetrichttp.New(ctx, exporterOptions...)

	return metricExp, err
}

//	Gauges don't have temporality (they're instantaneous values), as SDK still calls this selector.
//	For gauges, the SDK will ignore the temporality setting.
//	Dynatrace supports both Delta and Cumulative temporality for metrics that support it.
//
// NOTE: Dynatrace and Datadog does not support MONOTONIC_CUMULATIVE_SUM - Aerospike counters are monotonic
//
//	So, we are using Delta temporality for counters,  histograms and gauges
//	This is to ensure that the metrics are compatible with Dynatrace and Datadog
//	* avoid any issues with the metrics collection
//	* ensure that the metrics are compatible with Dynatrace, New Relic and Datadog
func (oe *OtelExecutor) getTemporalitySelector(instrumentKind sdkmetric.InstrumentKind) metricdata.Temporality {
	return metricdata.DeltaTemporality
}

func (oe *OtelExecutor) handleAerospikeMetrics(meter metric.Meter, ctx context.Context, commonLabels []attribute.KeyValue) {
	asRefreshStats, err := statprocessors.Refresh()

	if err != nil {
		log.Errorln("Error while refreshing Aerospike Metrics, error: ", err)
		oe.sendNodeUp(meter, commonLabels, 0.0)
		return
	}

	// aerospike server is up and we are able to fetch data
	oe.sendNodeUp(meter, commonLabels, 1.0)

	// process metrics
	oe.processAndPushStats(meter, ctx, commonLabels, asRefreshStats)

}

func (oe *OtelExecutor) handleSystemInfoMetrics(meter metric.Meter, ctx context.Context, commonLabels []attribute.KeyValue) {
	sysInfoRefreshStats, err := statprocessors.RefreshSystemInfo()

	if err != nil {
		log.Errorln("Error while refreshing SystemInfo, error: ", err)
		return
	}

	// process metrics
	oe.processAndPushStats(meter, ctx, commonLabels, sysInfoRefreshStats)
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
	log.Debug("** OTel header count ", len(headers))

	return headers
}
