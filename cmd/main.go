package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/metrichandlers"

	log "github.com/sirupsen/logrus"
)

var (
	configFile   = flag.String("config", "/etc/aerospike-prometheus-exporter/ape.toml", "Config File")
	showUsage    = flag.Bool("u", false, "Show usage information")
	showVersion  = flag.Bool("version", false, "Print version")
	serving_mode = flag.String("serve_mode", metrichandlers.PROM, "Exporter metrics serving mode")

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

	metric_handlers := metrichandlers.GetMetricHandlers()
	processor := metric_handlers[*serving_mode]

	if processor != nil {
		log.Infof("Metrics serving mode is %s", *serving_mode)
		err := processor.Initialize()
		if err != nil {
			log.Errorln(err)
		}
	} else {
		fmt.Println("Supported 'serve_mode' options [ prometheus ] ")
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
