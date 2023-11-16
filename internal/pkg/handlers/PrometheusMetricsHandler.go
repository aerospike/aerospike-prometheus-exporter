package handlers

import (
	"crypto/tls"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/watchers"

	log "github.com/sirupsen/logrus"
)

type PrometheusMetricsHandler struct {
	observer *Observer
}

func (pm PrometheusMetricsHandler) Initialize() error {
	mux := http.NewServeMux()

	pm.observer = NewObserver()

	promReg := prometheus.NewRegistry()
	promReg.MustRegister(pm.observer)

	// Get http basic auth username
	httpBasicAuthUsernameBytes, err := commons.GetSecret(config.Cfg.AeroProm.BasicAuthUsername)
	if err != nil {
		log.Fatal(err)
	}
	httpBasicAuthUsername := string(httpBasicAuthUsernameBytes)

	// Get http basic auth password
	httpBasicAuthPasswordBytes, err := commons.GetSecret(config.Cfg.AeroProm.BasicAuthPassword)
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
		ReadTimeout:  time.Duration(config.Cfg.AeroProm.Timeout) * time.Second,
		WriteTimeout: time.Duration(config.Cfg.AeroProm.Timeout) * time.Second,
		Addr:         config.Cfg.AeroProm.Bind,
		Handler:      mux,
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
	}

	log.Infof("Listening for Prometheus on: %s", config.Cfg.AeroProm.Bind)

	if len(config.Cfg.AeroProm.CertFile) > 0 && len(config.Cfg.AeroProm.KeyFile) > 0 {
		log.Info("Enabling HTTPS ...")
		srv.TLSConfig = initExporterTLS()
		log.Fatalln(srv.ListenAndServeTLS("", ""))
	}

	log.Fatalln(srv.ListenAndServe())

	return nil
}

// initExporterTLS initializes and returns TLS config to be used to serve metrics over HTTPS
func initExporterTLS() *tls.Config {
	serverPool, err := commons.LoadServerCertAndKey(config.Cfg.AeroProm.CertFile, config.Cfg.AeroProm.KeyFile, config.Cfg.AeroProm.KeyFilePassphrase)
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
	if len(config.Cfg.AeroProm.RootCA) > 0 {
		caPool, err := commons.LoadCACert(config.Cfg.AeroProm.RootCA)
		if err != nil {
			log.Fatal(err)
		}

		tlsConfig.ClientCAs = caPool
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
	}

	return tlsConfig
}

/**
 * Constructs Prometheus parameters required which are needed to push metrics to Prometheus
 */

func makePromMetric(as watchers.AerospikeStat, pLabels ...string) (*prometheus.Desc, prometheus.ValueType) {

	qualifiedName := as.QualifyMetricContext() + "_" + NormalizeMetric(as.Name)
	promDesc := prometheus.NewDesc(
		qualifiedName,
		NormalizeDesc(as.Name),
		pLabels,
		config.Cfg.AeroProm.MetricLabels,
	)

	if as.MType == commons.MetricTypeGauge {
		return promDesc, prometheus.GaugeValue
	}

	return promDesc, prometheus.CounterValue
}

// This is a common utility, used by all the watchers to push metric to prometheus
func PushToPrometheus(asMetric watchers.AerospikeStat, ch chan<- prometheus.Metric) {

	if asMetric.IsAllowed {
		// handle any panic from prometheus, this may occur when prom encounters a config/stat with special characters
		defer func() {
			if r := recover(); r != nil {
				// fmt.Println(" panic error : ", string(asMetric.Context), asMetric.Name)
				log.Tracef("%s recovered from panic while handling stat %s", string(asMetric.Context), asMetric.Name)
			}
		}()

		desc, valueType := makePromMetric(asMetric, asMetric.Labels...)
		ch <- prometheus.MustNewConstMetric(desc, valueType, asMetric.Value, asMetric.LabelValues...)

	}
}
