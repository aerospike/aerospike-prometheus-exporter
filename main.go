package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"strconv"

	aero "github.com/aerospike/aerospike-client-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	bind        = flag.String("b", ":9145", "Bind address to listen for Prometheus")
	host        = flag.String("h", "127.0.0.1", "Aerospike server seed hostname or IP address")
	port        = flag.Int("p", 3000, "Aerospike server seed hostname or IP address port number")
	user        = flag.String("U", "", "User name")
	pass        = flag.String("P", "", "User password")
	authMode    = flag.String("A", "internal", "Authentication mode: internal | external (e.g. LDAP)")
	certFile    = flag.String("certFile", "", "Cert File")
	keyFile     = flag.String("keyFile", "", "Key File")
	nodeTLSName = flag.String("tlsName", "", "Node TLS Name")
	rootCA      = flag.String("rootCA", "", "Server Certificate")
	timeout     = flag.Int("T", 5000, "Connection timeout to the server node in milliseconds")
	showUsage   = flag.Bool("u", false, "Show usage information")
	resolution  = flag.Int("r", 5, "Database info calls (seconds)")
	tags        = flag.String("tags", "", "Tags to pass to prometheus in labels. Useful for querying for grafana or alerting")

	fullHost string
)

func main() {
	flag.Parse()
	if *showUsage {
		flag.Usage()
		os.Exit(0)
	}

	fullHost = *host + ":" + strconv.Itoa(*port)

	log.SetFlags(log.LstdFlags | log.Lshortfile)

	host := aero.NewHost(*host, *port)
	host.TLSName = *nodeTLSName
	observer, err := newObserver(host, *user, *pass)
	if err != nil {
		log.Fatalln(err)
	}

	promReg := prometheus.NewRegistry()
	promReg.MustRegister(observer)

	http.Handle("/metrics", promhttp.HandlerFor(promReg, promhttp.HandlerOpts{}))

	log.Println("Welcome Aerospike's Prometheus exporter!")
	log.Printf("Listening for Prometheus on: %s\n", *bind)
	log.Fatalln(http.ListenAndServe(*bind, nil))
}
