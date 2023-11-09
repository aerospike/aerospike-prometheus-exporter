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

	aero "github.com/aerospike/aerospike-client-go/v6"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"

	log "github.com/sirupsen/logrus"
)

var (
	configFile  = flag.String("config", "/etc/aerospike-prometheus-exporter/ape.toml", "Config File")
	showUsage   = flag.Bool("u", false, "Show usage information")
	showVersion = flag.Bool("version", false, "Print version")

	fullHost string
	// config   *Config

	version = "v1.9.0"

	// Gauge related
	//
	gaugeStatsFile = flag.String("gauge-list", "/etc/aerospike-prometheus-exporter/gauge_stats_list.toml", "Gauge stats File")
	// gaugeStatHandler *GaugeStats
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

	// config = new(Config)
	// initConfig(*configFile, config)
	commons.InitConfig(*configFile)
	commons.Cfg.ValidateAndUpdate()

	// initialize Gauge metric definitions
	// gaugeStatHandler = new(GaugeStats)
	// initGaugeStats(*gaugeStatsFile, gaugeStatHandler)

	fullHost = net.JoinHostPort(commons.Cfg.Aerospike.Host, strconv.Itoa(int(commons.Cfg.Aerospike.Port)))

	host := aero.NewHost(commons.Cfg.Aerospike.Host, int(commons.Cfg.Aerospike.Port))
	host.TLSName = commons.Cfg.Aerospike.NodeTLSName

	observer, err := newObserver(host, commons.Cfg.Aerospike.User, commons.Cfg.Aerospike.Password)
	if err != nil {
		log.Fatalln(err)
	}

	promReg := prometheus.NewRegistry()
	promReg.MustRegister(observer)

	mux := http.NewServeMux()

	// Get http basic auth username
	httpBasicAuthUsernameBytes, err := commons.GetSecret(commons.Cfg.AeroProm.BasicAuthUsername)
	if err != nil {
		log.Fatal(err)
	}
	httpBasicAuthUsername := string(httpBasicAuthUsernameBytes)

	// Get http basic auth password
	httpBasicAuthPasswordBytes, err := commons.GetSecret(commons.Cfg.AeroProm.BasicAuthPassword)
	if err != nil {
		log.Fatal(err)
	}
	httpBasicAuthPassword := string(httpBasicAuthPasswordBytes)

	// Handle "/metrics" url
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		if httpBasicAuthUsername != "" {
			if commons.ValidateBasicAuth(r, httpBasicAuthUsername, httpBasicAuthPassword) {
				promhttp.HandlerFor(promReg, promhttp.HandlerOpts{}).ServeHTTP(w, r)
				return
			}
			log.Warnf("Request from %s - Unauthorized", r.RemoteAddr)

			w.Header().Set("WWW-Authenticate", `Basic realm="AEROSPIKE-PROMETHEUS-EXPORTER-REALM"`)
			w.WriteHeader(401)
			_, err := w.Write([]byte("401 Unauthorized\n"))
			if err != nil {
				log.Warnf("failed to write http response data for /metrics: %s", err.Error())
			}
		} else {
			promhttp.HandlerFor(promReg, promhttp.HandlerOpts{}).ServeHTTP(w, r)
		}
	})

	// Handle "/health" url
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte(`OK`))
		if err != nil {
			log.Warnf("failed to write http response for /health: %s", err.Error())
		}
	})

	// Handle "/" root url
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte(`<html>
			<head><title>Aerospike Prometheus Exporter</title></head>
			<body>
			<h1>Aerospike Prometheus Exporter</h1>
			<p>Go to <a href='` + "/metrics" + `'>Metrics</a></p>
			</body>
			</html>
		`))
		if err != nil {
			log.Warnf("failed to write http response for root /: %s", err.Error())
		}
	})

	srv := &http.Server{
		ReadTimeout:  time.Duration(commons.Cfg.AeroProm.Timeout) * time.Second,
		WriteTimeout: time.Duration(commons.Cfg.AeroProm.Timeout) * time.Second,
		Addr:         commons.Cfg.AeroProm.Bind,
		Handler:      mux,
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
	}

	log.Infof("Listening for Prometheus on: %s", commons.Cfg.AeroProm.Bind)

	if len(commons.Cfg.AeroProm.CertFile) > 0 && len(commons.Cfg.AeroProm.KeyFile) > 0 {
		log.Info("Enabling HTTPS ...")
		srv.TLSConfig = initExporterTLS()
		log.Fatalln(srv.ListenAndServeTLS("", ""))
	}

	log.Fatalln(srv.ListenAndServe())
}

// initExporterTLS initializes and returns TLS config to be used to serve metrics over HTTPS
func initExporterTLS() *tls.Config {
	serverPool, err := commons.LoadServerCertAndKey(commons.Cfg.AeroProm.CertFile, commons.Cfg.AeroProm.KeyFile, commons.Cfg.AeroProm.KeyFilePassphrase)
	if err != nil {
		log.Fatal(err)
	}

	tlsConfig := &tls.Config{
		Certificates:             serverPool,
		MinVersion:               tls.VersionTLS12,
		CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		PreferServerCipherSuites: true,
		InsecureSkipVerify:       false,
	}

	// if root CA provided, client validation is enabled (mutual TLS)
	if len(commons.Cfg.AeroProm.RootCA) > 0 {
		caPool, err := commons.LoadCACert(commons.Cfg.AeroProm.RootCA)
		if err != nil {
			log.Fatal(err)
		}

		tlsConfig.ClientCAs = caPool
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
	}

	return tlsConfig
}
