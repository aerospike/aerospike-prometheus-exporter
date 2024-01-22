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
	configFile   = flag.String("config", "/etc/aerospike-prometheus-exporter/ape.toml", "Config File")
	showUsage    = flag.Bool("u", false, "Show usage information")
	showVersion  = flag.Bool("version", false, "Print version")
	serving_mode = flag.String("serve_mode", executors.PROMETHEUS, "Exporter metrics serving mode")

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

	// metric_handlers := executors.GetExecutors()

	// servmodes := strings.Split(*serving_mode, ",")

	fmt.Println("\t\t *** config.Cfg.AeroProm.OtelMode: ", config.Cfg.AeroProm.OtelMode)
	fmt.Println("\t\t *** config.Cfg.AeroProm.PrometheusMode: ", config.Cfg.AeroProm.PrometheusMode)
	if config.Cfg.AeroProm.PrometheusMode {
		startExecutor("prometheus")
	}
	if config.Cfg.AeroProm.OtelMode {
		startExecutor("otel")
	}

	// for _, mode := range servmodes {
	// 	processor := metric_handlers[mode]
	// 	log.Infof("Metrics serving mode is %s", mode)
	// 	if processor != nil {
	// 		// start processor in a separate thread
	// 		go func() {
	// 			err := processor.Initialize()
	// 			if err != nil {
	// 				fmt.Println("Error while Initializing Processor ", err)
	// 			}
	// 		}()
	// 	} else {
	// 		fmt.Println("Supported 'serve_mode' options [ prometheus,otel ] - provided config is ", mode)
	// 		log.Infof("Supported 'serve_mode' options [ prometheus,otel ] - provided config is %s", mode)
	// 	}
	// }

	select {}
}

func startExecutor(mode string) {
	metric_handlers := executors.GetExecutors()

	processor := metric_handlers[mode]
	log.Infof("Starting metrics serving mode with %s", mode)
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
