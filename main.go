package main

import (
	"crypto/tls"
	"flag"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	log "github.com/Sirupsen/logrus"
	aero "github.com/aerospike/aerospike-client-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	configFile = flag.String("config", "/etc/aerospike-prometheus-exporter/ape.toml", "Config File")
	showUsage  = flag.Bool("u", false, "Show usage information")

	fullHost string
	config   *Config
)

func main() {
	flag.Parse()
	if *showUsage {
		flag.Usage()
		os.Exit(0)
	}

	// log.SetFlags(log.LstdFlags | log.Lshortfile)
	config = new(Config)
	InitConfig(*configFile, config)
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

	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		if config.AeroProm.BasicAuthUsername != "" {
			if validateBasicAuth(w, r, config.AeroProm.BasicAuthUsername, config.AeroProm.BasicAuthPassword) {
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

	cfg := &tls.Config{
		MinVersion:               tls.VersionTLS12,
		CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		PreferServerCipherSuites: true,
		// CipherSuites: []uint16{
		// 	tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		// 	tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
		// 	tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
		// 	tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		// },
	}

	srv := &http.Server{
		ReadTimeout:  time.Duration(config.AeroProm.Timeout) * time.Second,
		WriteTimeout: time.Duration(config.AeroProm.Timeout) * time.Second,
		Addr:         config.AeroProm.Bind,
		Handler:      mux,
		TLSConfig:    cfg,
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0),
	}

	log.Infoln("Welcome Aerospike's Prometheus exporter!")
	log.Infof("Listening for Prometheus on: %s\n", config.AeroProm.Bind)

	if config.AeroProm.CertFile != "" && config.AeroProm.KeyFile != "" {
		log.Infoln("Using Cert and Key files to enable TLS...")
		log.Fatalln(srv.ListenAndServeTLS(config.AeroProm.CertFile, config.AeroProm.KeyFile))
		// } else if config.AeroProm.UseLetsEncrypt {
		// 	autocert.
		// 		log.Infoln("Using Let's Encrypt to enable TLS...")
		// 	m := autocert.Manager{
		// 		Prompt: autocert.AcceptTOS,
		// 		// HostPolicy: autocert.HostWhitelist("", ""),
		// 	}

		// 	cacheDir := os.TempDir() + "aeroprom"
		// 	log.Info(cacheDir)
		// 	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		// 		log.Warn("autocert.NewListener not using a cache: %v", err)
		// 	} else {
		// 		m.Cache = autocert.DirCache(cacheDir)
		// 	}

		// 	s := &http.Server{
		// 		Addr:      ":https",
		// 		TLSConfig: m.TLSConfig(),
		// 		Handler:   mux,
		// 	}

		// 	go http.ListenAndServe(":http", m.HTTPHandler(nil))
		// 	log.Fatalln(s.ListenAndServeTLS("", ""))
	}

	log.Fatalln(srv.ListenAndServe())
}
