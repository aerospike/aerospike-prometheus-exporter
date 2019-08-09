package main

import (
	"bytes"
	"flag"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	aero "github.com/aerospike/aerospike-client-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	bind       = flag.String("b", ":9145", "AeroProm bind address")
	host       = flag.String("h", "127.0.0.1", "Aerospike server seed hostnames or IP addresses")
	port       = flag.Int("p", 3000, "Aerospike server seed hostname or IP address port number")
	user       = flag.String("U", "", "User name")
	pass       = flag.String("P", "", "User password")
	authMode   = flag.String("A", "internal", "Authentication mode: internal | external")
	timeout    = flag.Int("T", 5000, "Connection timeout to the server node in milliseconds")
	showUsage  = flag.Bool("u", false, "Show usage information")
	resolution = flag.Int("r", 5, "Database info calls (seconds)")
	tags       = flag.String("tags", "", "Tags to pass to prometheus in labels. Useful for querying for grafana or alerting")

	logger   *log.Logger
	fullHost string
)

func main() {
	flag.Parse()

	fullHost = *host + ":" + strconv.Itoa(*port)

	var buf bytes.Buffer
	logger = log.New(&buf, "", log.LstdFlags|log.Lshortfile)
	logger.SetOutput(os.Stdout)

	// use all cpus in the system for concurrency
	*authMode = strings.ToLower(strings.TrimSpace(*authMode))
	if *authMode != "internal" && *authMode != "external" {
		log.Fatalln("Invalid auth mode: only `internal` and `external` values are accepted.")
	}

	host := aero.NewHost(*host, *port)
	observer, err := newObserver(host, *user, *pass)
	if err != nil {
		log.Fatalln(err)
	}

	promReg := prometheus.NewRegistry()
	promReg.MustRegister(observer)

	http.Handle("/metrics", promhttp.HandlerFor(promReg, promhttp.HandlerOpts{}))

	log.Println("Welcome to AeroProm, Aerospike's Prometheus exporter!")
	log.Printf("Listening for Prometheus on: %s\n", *bind)
	log.Fatalln(http.ListenAndServe(*bind, nil))
}
