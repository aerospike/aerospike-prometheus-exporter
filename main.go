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

	aero "github.com/aerospike/aerospike-client-go/v5"
	log "github.com/sirupsen/logrus"
)

var (
	configFile  = flag.String("config", "/etc/aerospike-prometheus-exporter/ape.toml", "Config File")
	showUsage   = flag.Bool("u", false, "Show usage information")
	showVersion = flag.Bool("version", false, "Print version")

	fullHost string
	config   *Config

	version = "v1.5.2"
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
			if validateBasicAuth(r, httpBasicAuthUsername, httpBasicAuthPassword) {
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

	srv := &http.Server{
		ReadTimeout:  time.Duration(config.AeroProm.Timeout) * time.Second,
		WriteTimeout: time.Duration(config.AeroProm.Timeout) * time.Second,
		Addr:         config.AeroProm.Bind,
		Handler:      mux,
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
	}

	log.Infof("Listening for Prometheus on: %s", config.AeroProm.Bind)

	if len(config.AeroProm.CertFile) > 0 && len(config.AeroProm.KeyFile) > 0 {
		log.Info("Enabling HTTPS ...")
		srv.TLSConfig = initExporterTLS()
		log.Fatalln(srv.ListenAndServeTLS("", ""))
	}

	log.Fatalln(srv.ListenAndServe())
}

// initExporterTLS initializes and returns TLS config to be used to serve metrics over HTTPS
func initExporterTLS() *tls.Config {
	serverPool, err := loadServerCertAndKey(config.AeroProm.CertFile, config.AeroProm.KeyFile, config.AeroProm.KeyFilePassphrase)
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
	if len(config.AeroProm.RootCA) > 0 {
		caPool, err := loadCACert(config.AeroProm.RootCA)
		if err != nil {
			log.Fatal(err)
		}

		tlsConfig.ClientCAs = caPool
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
	}

	return tlsConfig
}
