package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/executors"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/secretagent"

	log "github.com/sirupsen/logrus"
)

var (
	configFile  = flag.String("config", "/etc/aerospike-prometheus-exporter/ape.toml", "Config File")
	showUsage   = flag.Bool("u", false, "Show usage information")
	showVersion = flag.Bool("version", false, "Print version")

	version = "v1.9.0"

	// Gauge related
	//
	gaugeStatsFile = flag.String("gauge-list", "/etc/aerospike-prometheus-exporter/gauge_stats_list.toml", "Gauge stats File")
	// gaugeStatHandler *GaugeStats
)

func main() {
	log.Infof("Welcome to Aerospike Prometheus Exporter %s", version)
	parseCommandlineArgs()

	// initialize configs and gauge-stats
	config.InitConfig(*configFile)
	if err := initSecretAgent(); err != nil {
		log.Fatalf("Failed to initialize Secret Agent: %s", err)
	}

	if err := commons.ResolveStartupSecrets(); err != nil {
		log.Fatalf("Failed to resolve secrets: %s", err)
	}

	config.InitGaugeStats(*gaugeStatsFile)

	// Initialize signal handling once for all executors
	commons.HandleSignals()

	if config.Cfg.Agent.PrometheusEnabled {
		startExecutor(commons.EXECUTOR_MODE_PROM)
	}

	if config.Cfg.Agent.OtelEnabled {
		startExecutor(commons.EXECUTOR_MODE_OTEL)
	}

	select {}
}

func initSecretAgent() error {
	settings, err := secretagent.Parse(config.Cfg.SecretAgentConfigs())
	if err != nil {
		return err
	}

	if settings.Mode == secretagent.ModeDisabled {
		return nil
	}

	tlsConfig, err := commons.BuildSecretAgentTLS(settings)
	if err != nil {
		return err
	}

	client, err := secretagent.New(settings, tlsConfig)
	if err != nil {
		return err
	}

	commons.RegisterSecretAgent(client)
	log.Infof("Secret Agent configured using %s %s", settings.Mode, settings.Endpoint())

	return nil
}

func startExecutor(mode string) {
	metricHandlers := executors.GetExecutors()

	processor := metricHandlers[mode]
	log.Infof("Starting metrics serving mode with '%s'", mode)

	if processor != nil {
		// start processor in a separate thread
		go func() {

			err := processor.Initialize()

			if err != nil {
				fmt.Println("Error while Initializing Processor ", err)
			}
		}()
	}
}

func parseCommandlineArgs() {
	flag.Parse()

	if *showUsage {
		flag.Usage()
		os.Exit(0)
	}

	if *showVersion {
		fmt.Println(version)
		os.Exit(0)
	}
}
