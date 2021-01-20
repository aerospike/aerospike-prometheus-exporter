package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	aero "github.com/aerospike/aerospike-client-go"
	log "github.com/sirupsen/logrus"
)

var (
	configFile  = flag.String("config", "/etc/aerospike-prometheus-exporter/ape.toml", "Config File")
	showUsage   = flag.Bool("u", false, "Show usage information")
	showVersion = flag.Bool("version", false, "Print version")

	fullHost string
	config   *Config

	version = "v1.1.4"
)

func main() {
	flag.Parse()
	if *showUsage {
		flag.Usage()
		os.Exit(0)
	}

	if *showVersion {
		fmt.Println(version)
		os.Exit(0)
	}

	log.Infof("Welcome to Aerospike Prometheus Exporter %s", version)

	config = new(Config)
	initConfig(*configFile, config)
	config.validateAndUpdate()

	fullHost = net.JoinHostPort(config.Aerospike.Host, strconv.Itoa(int(config.Aerospike.Port)))

	host := aero.NewHost(config.Aerospike.Host, int(config.Aerospike.Port))
	host.TLSName = config.Aerospike.NodeTLSName

	observer, err := newObserver(host, config.Aerospike.User, config.Aerospike.Password)
	if err != nil {
		log.Fatalln(err)
	}

	promReg := prometheus.NewRegistry()
	promReg.MustRegister(observer)

	mux := http.NewServeMux()

	// Get http basic auth username
	httpBasicAuthUsernameBytes, err := getSecret(config.AeroProm.BasicAuthUsername)
	if err != nil {
		log.Fatal(err)
	}
	httpBasicAuthUsername := string(httpBasicAuthUsernameBytes)

	// Get http basic auth password
	httpBasicAuthPasswordBytes, err := getSecret(config.AeroProm.BasicAuthPassword)
	if err != nil {
		log.Fatal(err)
	}
	httpBasicAuthPassword := string(httpBasicAuthPasswordBytes)

	// Handle "/metrics" url
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		if httpBasicAuthUsername != "" {
			if validateBasicAuth(w, r, httpBasicAuthUsername, httpBasicAuthPassword) {
				promhttp.HandlerFor(promReg, promhttp.HandlerOpts{}).ServeHTTP(w, r)
				return
			}
			log.Warnf("Request from %s - Unauthorized", r.RemoteAddr)

			w.Header().Set("WWW-Authenticate", `Basic realm="AEROSPIKE-PROMETHEUS-EXPORTER-REALM"`)
			w.WriteHeader(401)
			w.Write([]byte("401 Unauthorized\n"))
		} else {
			promhttp.HandlerFor(promReg, promhttp.HandlerOpts{}).ServeHTTP(w, r)
		}
	})

	// Handle "/health" url
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`OK`))
	})

	// Handle "/" root url
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
			<head><title>Aerospike Prometheus Exporter</title></head>
			<body>
			<h1>Aerospike Prometheus Exporter</h1>
			<p>Go to <a href='` + "/metrics" + `'>Metrics</a></p>
			</body>
			</html>
		`))
	})

	cfg := &tls.Config{
		MinVersion:               tls.VersionTLS12,
		CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		PreferServerCipherSuites: true,
	}

	srv := &http.Server{
		ReadTimeout:  time.Duration(config.AeroProm.Timeout) * time.Second,
		WriteTimeout: time.Duration(config.AeroProm.Timeout) * time.Second,
		Addr:         config.AeroProm.Bind,
		Handler:      mux,
		TLSConfig:    cfg,
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0),
	}

	log.Infof("Listening for Prometheus on: %s", config.AeroProm.Bind)

	if config.AeroProm.CertFile != "" && config.AeroProm.KeyFile != "" {
		log.Infoln("Enabling HTTPS ...")
		log.Debugf("Using cert file %s and key file %s", config.AeroProm.CertFile, config.AeroProm.KeyFile)
		log.Fatalln(srv.ListenAndServeTLS(config.AeroProm.CertFile, config.AeroProm.KeyFile))
	}

	log.Fatalln(srv.ListenAndServe())
}
