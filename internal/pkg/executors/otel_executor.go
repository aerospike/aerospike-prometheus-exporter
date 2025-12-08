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

	log.Infof("Otel sending thread started, sending data to : %s", config.Cfg.Agent.Otel.OtelEndpoint)

	log.Infof("*** Initializing Otel Exporter... ")

	ctx := context.Background()
	var meterProvider *sdkmetric.MeterProvider

	resource, err := resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithProcess(),
		// resource.WithTelemetrySDK(),
		resource.WithHost(),
		resource.WithContainer(),
		resource.WithAttributes(
			// the service name used to display traces/metrics in backends
			semconv.ServiceNameKey.String(config.Cfg.Agent.Otel.OtelServiceName),
		),
	)

	handleErr(err, "Failed to create OTel Resource")

	if config.Cfg.Agent.Otel.OtelEndpointType == "grpc" {
		meterProvider, err = oe.GetOtelGrpcMetricProvider(resource)
	} else {
		meterProvider, err = oe.GetOtelHttMetricProvider(resource)
	}

	handleErr(err, "Failed to create the collector metric exporter")

	otel.SetMeterProvider(meterProvider)

	log.Infof("*** Starting Otel Metrics Push thread... ")

	// Start metric collection loop in a goroutine
	go func() {
		ticker := time.NewTicker(time.Duration(config.Cfg.Agent.Otel.OtelServerStatFetchInterval) * time.Second)
		defer ticker.Stop()

		meter := otel.Meter(config.Cfg.Agent.Otel.OtelServiceName + "_Meter")
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

func (oe OtelExecutor) GetOtelGrpcMetricProvider(resource *resource.Resource) (*sdkmetric.MeterProvider, error) {
	headers := oe.readHeaders()

	ctx := context.Background()

	var metricExp *otlpmetricgrpc.Exporter
	var err error

	log.Debug("** OTel endpoint ", config.Cfg.Agent.Otel.OtelEndpoint)
	log.Debug("** OTel header count ", len(headers))
	log.Debug("** OTel service name ", config.Cfg.Agent.Otel.OtelServiceName)
	log.Debug("** OTel TLS flag enabled? ", config.Cfg.Agent.Otel.OtelTlsEnabled)

	log.Infof("Creating MetricsExporter with TLS %s", strconv.FormatBool(config.Cfg.Agent.Otel.OtelTlsEnabled))

	if config.Cfg.Agent.Otel.OtelTlsEnabled {

		metricExp, err = otlpmetricgrpc.New(
			ctx,
			otlpmetricgrpc.WithHeaders(headers),
			otlpmetricgrpc.WithEndpoint(config.Cfg.Agent.Otel.OtelEndpoint),
			otlpmetricgrpc.WithTemporalitySelector(oe.getTemporalitySelector),
			// otlpmetricgrpc.WithAggregationSelector(getAggregationSelector),
		)
	} else {
		metricExp, err = otlpmetricgrpc.New(
			ctx,
			otlpmetricgrpc.WithInsecure(),
			otlpmetricgrpc.WithHeaders(headers),
			otlpmetricgrpc.WithEndpoint(config.Cfg.Agent.Otel.OtelEndpoint),
			otlpmetricgrpc.WithTemporalitySelector(oe.getTemporalitySelector),
			// otlpmetricgrpc.WithAggregationSelector(getAggregationSelector),
		)

	}

	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(resource),
		sdkmetric.WithReader(
			sdkmetric.NewPeriodicReader(
				metricExp,
				sdkmetric.WithInterval(time.Duration(config.Cfg.Agent.Otel.OtelPushInterval)*time.Second),
			),
		),
	)

	return meterProvider, err
}

func (oe OtelExecutor) GetOtelHttMetricProvider(resource *resource.Resource) (*sdkmetric.MeterProvider, error) {
	headers := oe.readHeaders()

	ctx := context.Background()

	var err error

	log.Debug("** OTel endpoint ", config.Cfg.Agent.Otel.OtelEndpoint)
	log.Debug("** OTel service name ", config.Cfg.Agent.Otel.OtelServiceName)
	log.Debug("** OTel TLS flag enabled? ", config.Cfg.Agent.Otel.OtelTlsEnabled)

	log.Infof("Creating MetricsExporter with TLS %s", strconv.FormatBool(config.Cfg.Agent.Otel.OtelTlsEnabled))

	var metricExp *otlpmetrichttp.Exporter

	log.Infof("Creating MetricsExporter with TLS %s", strconv.FormatBool(config.Cfg.Agent.Otel.OtelTlsEnabled))

	if config.Cfg.Agent.Otel.OtelTlsEnabled {

		metricExp, err = otlpmetrichttp.New(
			ctx,
			otlpmetrichttp.WithHeaders(headers),
			otlpmetrichttp.WithEndpoint(config.Cfg.Agent.Otel.OtelEndpoint),
			// otlpmetrichttp.WithURLPath("api/v2/otlp/v1/metrics"),
			otlpmetrichttp.WithURLPath(config.Cfg.Agent.Otel.OtelEndpointURL),
			otlpmetrichttp.WithTemporalitySelector(oe.getTemporalitySelector),
			// otlpmetrichttp.WithAggregationSelector(getAggregationSelector),
		)
	} else {
		metricExp, err = otlpmetrichttp.New(
			ctx,
			otlpmetrichttp.WithInsecure(),
			otlpmetrichttp.WithHeaders(headers),
			otlpmetrichttp.WithEndpoint(config.Cfg.Agent.Otel.OtelEndpoint),
			// otlpmetrichttp.WithURLPath("api/v2/otlp/v1/metrics"),
			otlpmetrichttp.WithURLPath(config.Cfg.Agent.Otel.OtelEndpointURL),
			otlpmetrichttp.WithTemporalitySelector(oe.getTemporalitySelector),
			// otlpmetrichttp.WithAggregationSelector(getAggregationSelector),
		)

	}

	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(resource),
		sdkmetric.WithReader(
			sdkmetric.NewPeriodicReader(
				metricExp,
				sdkmetric.WithInterval(time.Duration(config.Cfg.Agent.Otel.OtelPushInterval)*time.Second),
			),
		),
	)

	return meterProvider, err
}

func (oe OtelExecutor) getTemporalitySelector(instrumentKind sdkmetric.InstrumentKind) metricdata.Temporality {
	// if instrumentKind == sdkmetric.InstrumentKindCounter {
	// 	return metricdata.CumulativeTemporality
	// }
	// return metricdata.DeltaTemporality

	switch instrumentKind {
	case sdkmetric.InstrumentKindCounter,
		sdkmetric.InstrumentKindObservableCounter,
		sdkmetric.InstrumentKindHistogram,
		sdkmetric.InstrumentKindUpDownCounter,
		sdkmetric.InstrumentKindObservableUpDownCounter:
		return metricdata.CumulativeTemporality
	default:
		return metricdata.DeltaTemporality // or Cumulative if you *really* want
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
	headerPairs := config.Cfg.Agent.Otel.OtelHeaders
	if len(headerPairs) > 0 {
		for k, v := range headerPairs {
			headers[k] = v
		}
	}
	log.Debug("** OTel header count ", len(headers))

	return headers
}
