package executors

import (
	"crypto/tls"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/statprocessors"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/systeminfo"

	log "github.com/sirupsen/logrus"
)

type PrometheusHttpExecutor struct {
	promimpl *PrometheusImpl
}

func (pm PrometheusHttpExecutor) Initialize() error {
	mux := http.NewServeMux()

	pm.promimpl = NewPrometheusImpl()

	promReg := prometheus.NewRegistry()
	promReg.MustRegister(pm.promimpl)

	// Get http basic auth username
	httpBasicAuthUsernameBytes, err := commons.GetSecret(config.Cfg.Agent.Prom.BasicAuthUsername)
	if err != nil {
		log.Fatal(err)
	}
	httpBasicAuthUsername := string(httpBasicAuthUsernameBytes)

	// Get http basic auth password
	httpBasicAuthPasswordBytes, err := commons.GetSecret(config.Cfg.Agent.Prom.BasicAuthPassword)
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
		ReadTimeout:  time.Duration(config.Cfg.Agent.Timeout) * time.Second,
		WriteTimeout: time.Duration(config.Cfg.Agent.Timeout) * time.Second,
		Addr:         config.Cfg.Agent.Prom.Bind,
		Handler:      mux,
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
	}

	log.Infof("Listening for Prometheus on: %s", config.Cfg.Agent.Prom.Bind)

	if len(config.Cfg.Agent.Prom.CertFile) > 0 && len(config.Cfg.Agent.Prom.KeyFile) > 0 {
		log.Info("Enabling HTTPS ...")
		srv.TLSConfig = initExporterTLS()
		log.Fatalln(srv.ListenAndServeTLS("", ""))
	}

	log.Fatalln(srv.ListenAndServe())

	return nil
}

// initExporterTLS initializes and returns TLS config to be used to serve metrics over HTTPS
func initExporterTLS() *tls.Config {
	serverPool, err := commons.LoadServerCertAndKey(config.Cfg.Agent.Prom.CertFile, config.Cfg.Agent.Prom.KeyFile, config.Cfg.Agent.Prom.KeyFilePassphrase)
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
	if len(config.Cfg.Agent.Prom.RootCA) > 0 {
		caPool, err := commons.LoadCACert(config.Cfg.Agent.Prom.RootCA)
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

func makePromMetric(as statprocessors.AerospikeStat, pLabels ...string) (*prometheus.Desc, prometheus.ValueType) {

	qualifiedName := as.QualifyMetricContext() + "_" + NormalizeMetric(as.Name)
	promDesc := prometheus.NewDesc(
		qualifiedName,
		NormalizeDesc(as.Name),
		pLabels,
		config.Cfg.Agent.MetricLabels,
	)

	if as.MType == commons.MetricTypeGauge {
		return promDesc, prometheus.GaugeValue
	}

	return promDesc, prometheus.CounterValue
}

// This is a common utility, used by all the statprocessors to push metric to prometheus
func PushToPrometheus(asMetric statprocessors.AerospikeStat, ch chan<- prometheus.Metric) {

	if asMetric.IsAllowed {
		// handle any panic from prometheus, this may occur when prom encounters a config/stat with special characters
		defer func() {
			if r := recover(); r != nil {
				log.Tracef("%s recovered from panic while handling stat %s", string(asMetric.Context), asMetric.Name)
			}
		}()

		desc, valueType := makePromMetric(asMetric, asMetric.Labels...)
		ch <- prometheus.MustNewConstMetric(desc, valueType, asMetric.Value, asMetric.LabelValues...)

	}
}

// This is a common utility, used by all the statprocessors to push metric to prometheus
func PushSystemInfoMetricToPrometheus(sysMetric systeminfo.SystemInfoStat, ch chan<- prometheus.Metric) {

	if sysMetric.IsAllowed {
		// handle any panic from prometheus, this may occur when prom encounters a config/stat with special characters
		defer func() {
			if r := recover(); r != nil {
				log.Tracef("%s recovered from panic while handling stat %s", string(sysMetric.Context), sysMetric.Name)
			}
		}()

		desc, valueType := makeSystemInfoPromMetric(sysMetric, sysMetric.Labels...)
		ch <- prometheus.MustNewConstMetric(desc, valueType, sysMetric.Value, sysMetric.LabelValues...)

	}
}

func makeSystemInfoPromMetric(sm systeminfo.SystemInfoStat, pLabels ...string) (*prometheus.Desc, prometheus.ValueType) {

	qualifiedName := sm.QualifyMetricContext() + "_" + NormalizeMetric(sm.Name)
	promDesc := prometheus.NewDesc(
		qualifiedName,
		NormalizeDesc(sm.Name),
		pLabels,
		config.Cfg.Agent.MetricLabels,
	)

	if sm.MType == commons.MetricTypeGauge {
		return promDesc, prometheus.GaugeValue
	}

	return promDesc, prometheus.CounterValue
}
