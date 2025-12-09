package executors

import (
	"context"
	"strconv"

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

type OtelExecutor struct {
}

// Exporter interface implementation
// Aerospike Otel metrics serving implementation
//
// Initializes an OTLP exporter, and configures the corresponding metric providers
func (oe OtelExecutor) Initialize() error {

	log.Infof("Otel sending thread started, sending data to : %s", config.Cfg.Agent.Otel.Endpoint)

	log.Infof("*** Initializing Otel Exporter... ")
	log.Debug("** OTel endpoint ", config.Cfg.Agent.Otel.Endpoint)
	log.Debug("** OTel service name ", config.Cfg.Agent.Otel.ServiceName)
	log.Debug("** OTel TLS flag enabled? ", config.Cfg.Agent.Otel.TlsEnabled)

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
				// Aerospike Refresh stats
				oe.handleAerospikeMetrics(meter, defaultCtx, commonLabels)

				// System metrics
				oe.handleSystemInfoMetrics(meter, defaultCtx, commonLabels)
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

// func (oe OtelExecutor) createGrpcExporter(resource *resource.Resource) (*sdkmetric.MeterProvider, error) {
func (oe OtelExecutor) createGrpcExporter(ctx context.Context) (sdkmetric.Exporter, error) {
	headers := oe.readHeaders()

	var metricExp *otlpmetricgrpc.Exporter
	var err error

	log.Infof("Creating Otel GRPC MetricsExporter with TLS %s", strconv.FormatBool(config.Cfg.Agent.Otel.TlsEnabled))

	// Build options conditionally
	exporterOptions := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithHeaders(headers),
		otlpmetricgrpc.WithEndpoint(oe.getGrpcEndpointToUse()),
		otlpmetricgrpc.WithTemporalitySelector(oe.getTemporalitySelector),
	}

	// // Only add WithInsecure() when TLS is disabled
	// if !config.Cfg.Agent.Otel.OtelTlsEnabled {
	// 	exporterOptions = append(exporterOptions, otlpmetricgrpc.WithInsecure())
	// }

	metricExp, err = otlpmetricgrpc.New(ctx, exporterOptions...)

	return metricExp, err

}

// func (oe OtelExecutor) createHttpExporter(resource *resource.Resource) (*sdkmetric.MeterProvider, error) {
func (oe OtelExecutor) createHttpExporter(ctx context.Context) (sdkmetric.Exporter, error) {

	headers := oe.readHeaders()
	var err error
	var metricExp *otlpmetrichttp.Exporter

	log.Infof("Creating Otel HTTP MetricsExporter with TLS %s", strconv.FormatBool(config.Cfg.Agent.Otel.TlsEnabled))

	// NOTE: Below is only for testing purposes
	// tlsConfig := &tls.Config{
	// 	InsecureSkipVerify: true,
	// }

	// Build options conditionally
	exporterOptions := []otlpmetrichttp.Option{
		otlpmetrichttp.WithHeaders(headers),
		otlpmetrichttp.WithEndpointURL(config.Cfg.Agent.Otel.HttpEndpoint),
		otlpmetrichttp.WithTemporalitySelector(oe.getTemporalitySelector),
		// otlpmetrichttp.WithTLSClientConfig(tlsConfig),
	}

	// // Only add WithInsecure() when TLS is disabled
	// if !config.Cfg.Agent.Otel.OtelTlsEnabled {
	// 	exporterOptions = append(exporterOptions, otlpmetrichttp.WithInsecure())
	// }

	metricExp, err = otlpmetrichttp.New(ctx, exporterOptions...)

	return metricExp, err
}

func (oe OtelExecutor) getGrpcEndpointToUse() string {

	if len(config.Cfg.Agent.Otel.GrpcEndpoint) != 0 {
		return config.Cfg.Agent.Otel.GrpcEndpoint
	}

	return config.Cfg.Agent.Otel.Endpoint
}

// getTemporalitySelector returns the appropriate temporality for each instrument kind
// NOTE:
//  Gauges don't have temporality (they're instantaneous values),
//  but the SDK still calls this selector. For gauges, the SDK will ignore the temporality setting.
//  Dynatrace supports both Delta and Cumulative temporality for metrics that support it.

// NOTE: Dynatrace and Datadog does not support MONOTONIC_CUMULATIVE_SUM - Aerospike counters are monotonic
//
//	So, we are using Delta temporality for counters and histograms
//	and Delta temporality for gauges
//	This is to ensure that the metrics are compatible with Dynatrace and Datadog
//	* avoid any issues with the metrics collection
//	* ensure that the metrics are compatible with Dynatrace and Datadog
//	* avoid any issues with the metrics collection
func (oe OtelExecutor) getTemporalitySelector(instrumentKind sdkmetric.InstrumentKind) metricdata.Temporality {
	switch instrumentKind {
	case sdkmetric.InstrumentKindCounter,
		sdkmetric.InstrumentKindObservableCounter,
		sdkmetric.InstrumentKindHistogram,
		sdkmetric.InstrumentKindUpDownCounter,
		sdkmetric.InstrumentKindObservableUpDownCounter:
		// Use Delta temporality for counters and histograms
		// This sends only the change since the last export, which is efficient
		return metricdata.DeltaTemporality
	case sdkmetric.InstrumentKindObservableGauge,
		sdkmetric.InstrumentKindGauge:
		// Gauges are instantaneous values and don't technically have temporality
		// However, if the SDK requires a value, we return DeltaTemporality
		// The SDK will handle gauges as instantaneous snapshots regardless
		return metricdata.DeltaTemporality
	default:
		// Default to Delta temporality for any unknown types
		return metricdata.DeltaTemporality
	}
}

func (oe OtelExecutor) handleAerospikeMetrics(meter metric.Meter, ctx context.Context, commonLabels []attribute.KeyValue) {
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

func (oe OtelExecutor) handleSystemInfoMetrics(meter metric.Meter, ctx context.Context, commonLabels []attribute.KeyValue) {
	sysInfoRefreshStats, err := statprocessors.RefreshSystemInfo()

	if err != nil {
		log.Errorln("Error while refreshing SystemInfo, error: ", err)
		return
	}

	// process metrics
	oe.processAndPushStats(meter, ctx, commonLabels, sysInfoRefreshStats)
}

// Utility functions
func (oe OtelExecutor) readHeaders() map[string]string {
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
