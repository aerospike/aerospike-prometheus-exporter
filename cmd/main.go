package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/executors"

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
	config.InitGaugeStats(*gaugeStatsFile)

	if config.Cfg.Agent.PrometheusEnabled {
		startExecutor("prometheus")
	}
	if config.Cfg.Agent.OtelEnabled {
		startExecutor("otel")
	}

	select {}
}

func startExecutor(mode string) {
	metric_handlers := executors.GetExecutors()

	processor := metric_handlers[mode]
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
