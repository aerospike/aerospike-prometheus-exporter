package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/handlers"

	log "github.com/sirupsen/logrus"
)

var (
	configFile  = flag.String("config", "/etc/aerospike-prometheus-exporter/ape.toml", "Config File")
	showUsage   = flag.Bool("u", false, "Show usage information")
	showVersion = flag.Bool("version", false, "Print version")
	handle_mode = flag.String("handle_mode", "prometheus", "Exporter metrics handling mode")

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
	commons.InitConfig(*configFile)
	commons.InitGaugeStats(*gaugeStatsFile)

	handles := GetSupportedHandlers()
	fmt.Println("Metrics handling mode is ", *handle_mode)
	handles[*handle_mode].Initialize()
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

func GetSupportedHandlers() map[string]handlers.MetricHandlers {
	handles := map[string]handlers.MetricHandlers{
		"prometheus": &handlers.PrometheusMetrics{},
	}

	return handles
}
