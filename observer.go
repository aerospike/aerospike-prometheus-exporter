package main

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	aero "github.com/aerospike/aerospike-client-go/v6"
	commons "github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	watchers "github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/watchers"
	log "github.com/sirupsen/logrus"
)

// Observer communicates with Aerospike and helps collecting metrices
type Observer struct {
	conn          *aero.Connection
	newConnection func() (*aero.Connection, error)
	ticks         prometheus.Counter
	watchers      []watchers.Watcher
	mutex         sync.Mutex
}

var (
	// aerospike_node_up metric descriptor
	nodeActiveDesc *prometheus.Desc

	// Node service endpoint, cluster name and build version
	gService, gClusterName, gBuild string

	// Number of retries on info request
	retryCount = 3
)

func initAerospikeTLS() *tls.Config {
	if len(commons.Cfg.Aerospike.RootCA) == 0 && len(commons.Cfg.Aerospike.CertFile) == 0 && len(commons.Cfg.Aerospike.KeyFile) == 0 {
		return nil
	}

	var clientPool []tls.Certificate
	var serverPool *x509.CertPool
	var err error

	serverPool, err = commons.LoadCACert(commons.Cfg.Aerospike.RootCA)
	if err != nil {
		log.Fatal(err)
	}

	if len(commons.Cfg.Aerospike.CertFile) > 0 || len(commons.Cfg.Aerospike.KeyFile) > 0 {
		clientPool, err = commons.LoadServerCertAndKey(commons.Cfg.Aerospike.CertFile, commons.Cfg.Aerospike.KeyFile, commons.Cfg.Aerospike.KeyFilePassphrase)
		if err != nil {
			log.Fatal(err)
		}
	}

	tlsConfig := &tls.Config{
		Certificates:             clientPool,
		RootCAs:                  serverPool,
		InsecureSkipVerify:       false,
		PreferServerCipherSuites: true,
		NameToCertificate:        nil,
	}

	return tlsConfig
}

func newObserver(server *aero.Host, user, pass string) (o *Observer, err error) {
	// initialize aerospike_node_up metric descriptor
	nodeActiveDesc = prometheus.NewDesc(
		"aerospike_node_up",
		"Aerospike node active status",
		[]string{"cluster_name", "service", "build"},
		commons.Cfg.AeroProm.MetricLabels,
	)

	// Get aerospike auth username
	username, err := commons.GetSecret(user)
	if err != nil {
		log.Fatal(err)
	}

	// Get aerospike auth password
	password, err := commons.GetSecret(pass)
	if err != nil {
		log.Fatal(err)
	}

	clientPolicy := aero.NewClientPolicy()
	clientPolicy.User = string(username)
	clientPolicy.Password = string(password)

	authMode := strings.ToLower(strings.TrimSpace(commons.Cfg.Aerospike.AuthMode))

	switch authMode {
	case "internal", "":
		clientPolicy.AuthMode = aero.AuthModeInternal
	case "external":
		clientPolicy.AuthMode = aero.AuthModeExternal
	case "pki":
		if len(commons.Cfg.Aerospike.CertFile) == 0 || len(commons.Cfg.Aerospike.KeyFile) == 0 {
			log.Fatalln("Invalid certificate configuration when using auth mode PKI: cert_file and key_file must be set")
		}
		clientPolicy.AuthMode = aero.AuthModePKI
	default:
		log.Fatalln("Invalid auth mode: only `internal`, `external`, `pki` values are accepted.")
	}

	// allow only ONE connection
	clientPolicy.ConnectionQueueSize = 1
	clientPolicy.Timeout = time.Duration(commons.Cfg.Aerospike.Timeout) * time.Second

	clientPolicy.TlsConfig = initAerospikeTLS()

	if clientPolicy.TlsConfig != nil {
		commons.Infokey_Service = "service-tls-std"
	}

	createNewConnection := func() (*aero.Connection, error) {
		conn, err := aero.NewConnection(clientPolicy, server)
		if err != nil {
			return nil, err
		}

		if clientPolicy.RequiresAuthentication() {
			if err := conn.Login(clientPolicy); err != nil {
				return nil, err
			}
		}

		// Set no connection deadline to re-use connection, but socketTimeout will be in effect
		var deadline time.Time
		err = conn.SetTimeout(deadline, clientPolicy.Timeout)
		if err != nil {
			return nil, err
		}

		return conn, nil
	}

	o = &Observer{
		newConnection: createNewConnection,
		ticks: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: "aerospike",
				Subsystem: "node",
				Name:      "ticks",
				Help:      "Counter that detemines how many times the Aerospike node was scraped for metrics.",
			}),
		watchers: []watchers.Watcher{
			&watchers.NamespaceWatcher{},
		}, // the order is important here
	}

	return o, nil
}

// Describe function of Prometheus' Collector interface
func (o *Observer) Describe(ch chan<- *prometheus.Desc) {}

// Collect function of Prometheus' Collector interface
func (o *Observer) Collect(ch chan<- prometheus.Metric) {
	// Protect against concurrent scrapes
	o.mutex.Lock()
	defer o.mutex.Unlock()

	o.ticks.Inc()
	ch <- o.ticks

	// refresh metrics from various watchers,
	watcher_metrics, err := o.refresh(ch)
	if err != nil {
		log.Errorln(err)
		ch <- prometheus.MustNewConstMetric(nodeActiveDesc, prometheus.GaugeValue, 0.0, gClusterName, gService, gBuild)
		return
	}

	// push the fetched metrics to prometheus
	for _, wm := range watcher_metrics {
		commons.PushToPrometheus(wm.Metric, wm.Value, wm.Labels, wm.LabelValues, ch)
	}

	ch <- prometheus.MustNewConstMetric(nodeActiveDesc, prometheus.GaugeValue, 1.0, gClusterName, gService, gBuild)
}

func (o *Observer) requestInfo(retryCount int, infoKeys []string) (map[string]string, error) {
	var err error
	rawMetrics := make(map[string]string)

	// Retry for connection, timeout, network errors
	// including errors from RequestInfo()
	for i := 0; i < retryCount; i++ {
		// Validate existing connection
		if o.conn == nil || !o.conn.IsConnected() {
			// Create new connection
			o.conn, err = o.newConnection()
			if err != nil {
				log.Debug(err)
				continue
			}
		}

		// Info request
		rawMetrics, err = o.conn.RequestInfo(infoKeys...)
		if err != nil {
			log.Debug(err)
			continue
		}

		break
	}

	if len(rawMetrics) == 1 {
		for k := range rawMetrics {
			if strings.HasPrefix(strings.ToUpper(k), "ERROR:") {
				return nil, errors.New(k)
			}
		}
	}

	return rawMetrics, err
}

func (o *Observer) refresh(ch chan<- prometheus.Metric) ([]watchers.WatcherMetric, error) {
	log.Debugf("Refreshing node %s", fullHost)

	// array to accumulate all metrics, which later will be dispatched by various observers
	var all_metrics_to_send = []watchers.WatcherMetric{}

	// fetch first set of info keys
	var infoKeys []string
	for _, c := range o.watchers {
		if keys := c.PassOneKeys(); len(keys) > 0 {
			infoKeys = append(infoKeys, keys...)
		}
	}
	// append infoKey "build" - this is removed from WatcherLatencies to avoid forced watcher sequence during refresh
	infoKeys = append(infoKeys, "build")

	// info request for first set of info keys, this retrives configs from server
	//   from namespaces,server/node-stats, xdr
	//   if for any context (like jobs, latencies etc.,) no configs, they are not sent to server
	passOneOutput, err := o.requestInfo(retryCount, infoKeys)
	if err != nil {
		return nil, err
	}

	// fetch second second set of info keys
	infoKeys = []string{commons.Infokey_ClusterName, commons.Infokey_Service, commons.Infokey_Build}
	watcherInfoKeys := make([][]string, len(o.watchers))
	for i, c := range o.watchers {
		if keys := c.PassTwoKeys(passOneOutput); len(keys) > 0 {
			infoKeys = append(infoKeys, keys...)
			watcherInfoKeys[i] = keys
		}
	}

	// info request for second set of info keys, this retrieves all the stats from server
	rawMetrics, err := o.requestInfo(retryCount, infoKeys)
	if err != nil {
		return all_metrics_to_send, err
	}

	// set global values
	gClusterName, gService, gBuild = rawMetrics[commons.Infokey_ClusterName], rawMetrics[commons.Infokey_Service], rawMetrics[commons.Infokey_Build]

	// sanitize the utf8 strings before sending them to watchers
	for k, v := range rawMetrics {
		rawMetrics[k] = commons.SanitizeUTF8(v)
	}

	// sanitize the utf8 strings before sending them to watchers
	for i, c := range o.watchers {
		l_watcher_metrics, err := c.Refresh(watcherInfoKeys[i], rawMetrics)
		if err != nil {
			return all_metrics_to_send, err
		}
		all_metrics_to_send = append(all_metrics_to_send, l_watcher_metrics...)
	}

	log.Debugf("Refreshing node was successful")

	return all_metrics_to_send, nil
}
