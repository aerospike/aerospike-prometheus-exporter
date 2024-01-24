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
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	// "go.opentelemetry.io/otel/label"
)

type OtelExecutor struct {
}

// Variables

var (
	currentRefreshStats  []statprocessors.AerospikeStat
	previousRefreshStats map[string]statprocessors.AerospikeStat
	// mapCounterMetricObjects map[string]metric.Float64Counter
	// mapGaugeMetricObjects   map[string]metric.Float64ObservableGauge

)

// Exporter interface implementation
func (oe OtelExecutor) Initialize() error {

	// Observe OS Signals
	commons.HandleSignals()

	// Initialize storage maps
	currentRefreshStats = []statprocessors.AerospikeStat{}
	previousRefreshStats = make(map[string]statprocessors.AerospikeStat)
	// mapCounterMetricObjects = make(map[string]metric.Float64Counter)
	// mapGaugeMetricObjects = make(map[string]metric.Float64ObservableGauge)

	log.Infof("*** Initializing Otel Exporter.. START ")

	shutdown := initProvider()
	defer shutdown()
	log.Infof("*** Starting Otel Metrics Push thread... ")

	// Start a goroutine to handle exit signals
	go func() {
		<-commons.ProcessExit
		log.Debugf("OTel Executor got EXIT signal from OS")
		shutdown()
	}()

	// start push executor
	startMetricExecutor()

	return nil
}

// Aerospike Otel metrics serving implementation
//
// Initializes an OTLP exporter, and configures the corresponding metric providers
func initProvider() func() {

	ctx := context.Background()
	serviceName := config.Cfg.AeroProm.OtelServiceName

	res, err := resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithProcess(),
		// resource.WithTelemetrySDK(),
		resource.WithHost(),
		resource.WithContainer(),
		resource.WithAttributes(
			// the service name used to display traces/metrics in backends
			semconv.ServiceNameKey.String(serviceName),
		),
	)

	handleErr(err, "Failed to create OTel Resource")

	otelAgentAddr := config.Cfg.AeroProm.OtelEndpoint
	headers := readHeaders()

	var metricExp *otlpmetricgrpc.Exporter
	log.Infof("Creating MetricsExporter with TLS %s", strconv.FormatBool(config.Cfg.AeroProm.OtelTlsEnabled))

	if config.Cfg.AeroProm.OtelTlsEnabled {
		metricExp, err = otlpmetricgrpc.New(
			ctx,
			otlpmetricgrpc.WithHeaders(headers),
			otlpmetricgrpc.WithEndpoint(otelAgentAddr),
			otlpmetricgrpc.WithTemporalitySelector(temporalityDeltaSelector),
		)
	} else {
		metricExp, err = otlpmetricgrpc.New(
			ctx,
			otlpmetricgrpc.WithInsecure(),
			otlpmetricgrpc.WithHeaders(headers),
			otlpmetricgrpc.WithEndpoint(otelAgentAddr),
			otlpmetricgrpc.WithTemporalitySelector(temporalityDeltaSelector),
		)
	}

	handleErr(err, "Failed to create the collector metric exporter")

	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(
			sdkmetric.NewPeriodicReader(
				metricExp,
				sdkmetric.WithInterval(time.Duration(config.Cfg.AeroProm.OtelPushInterval)*time.Second),
			),
		),
	)
	otel.SetMeterProvider(meterProvider)

	return func() {
		cxt, cancel := context.WithTimeout(ctx, time.Duration(config.Cfg.AeroProm.Timeout)*time.Second)
		defer cancel()
		log.Infof("shuttting down..., flushing metrics to endpoint")
		// pushes any last exports to the receiver
		if err := meterProvider.Shutdown(cxt); err != nil {
			otel.Handle(err)
		}
	}
}

func temporalityDeltaSelector(instrumentKind sdkmetric.InstrumentKind) metricdata.Temporality {
	if instrumentKind == sdkmetric.InstrumentKindCounter {
		// fmt.Println("*** Input kind is ", instrumentKind, " .. so returning metricdata.CumulativeTemporality==> ", metricdata.CumulativeTemporality)
		return metricdata.CumulativeTemporality
	}
	return metricdata.DeltaTemporality
}

func startMetricExecutor() {

	meter := otel.Meter(config.Cfg.AeroProm.OtelServiceName + "_Meter")

	// defaultCtx := baggage.ContextWithBaggage(context.Background())
	defaultCtx := context.Background()

	commonLabels := getCommonLabels()

	for {
		var err error

		currentRefreshStats, err = statprocessors.Refresh()
		if err != nil {
			log.Errorln(err)
			sendNodeUp(meter, defaultCtx, commonLabels, 0.0)
		} else {
			// aerospike server is up and we are able to fetch data
			sendNodeUp(meter, defaultCtx, commonLabels, 1.0)
			// process metrics
			processAerospikeStats(meter, defaultCtx, commonLabels, currentRefreshStats)
		}

		// sleep for config.N seconds
		time.Sleep(time.Duration(config.Cfg.AeroProm.OtelServerStatFetchInterval) * time.Second)
	}
}
