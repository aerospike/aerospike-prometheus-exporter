package main

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"log"
	"strings"
	"time"

	aero "github.com/aerospike/aerospike-client-go"
	"github.com/prometheus/client_golang/prometheus"
)

// Observer communicates with Aerospike and helps collecting metrices
type Observer struct {
	conn     *aero.Connection
	connGen  func() (*aero.Connection, error)
	ticks    prometheus.Counter
	watchers []Watcher
}

var (
	nodeActiveDesc = prometheus.NewDesc(
		"aerospike_node_up",
		"Wether node is active",
		[]string{"cluster_name", "service", "build"},
		nil,
	)

	gService, gClusterName, gBuild string
)

func initTLS() *tls.Config {
	if len(*rootCA) == 0 && len(*certFile) == 0 && len(*keyFile) == 0 {
		return nil
	}

	// Try to load system CA certs, otherwise just make an empty pool
	serverPool, err := x509.SystemCertPool()
	if serverPool == nil || err != nil {
		log.Printf("Adding system certificates to the cert pool failed: %s", err)
		serverPool = x509.NewCertPool()
	}

	if len(*rootCA) > 0 {
		// Try to load system CA certs and add them to the system cert pool
		caCert, err := ioutil.ReadFile(*rootCA)
		if err != nil {
			log.Fatalf("FAILED: Adding server certificate `%s` to the pool failed: %s", *rootCA, err)
		}

		log.Printf("Adding server certificate `%s` to the pool...", *rootCA)
		serverPool.AppendCertsFromPEM(caCert)
	}

	var clientPool []tls.Certificate
	if len(*certFile) > 0 || len(*keyFile) > 0 {
		// client Cert
		cert, err := tls.LoadX509KeyPair(*certFile, *keyFile)
		if err != nil {
			log.Fatalf("FAILED: Adding client certificate `%s` to the pool failed: `%s`", *certFile, err)
		}

		log.Printf("Adding client certificate `%s` to the pool...\n", *certFile)
		clientPool = append(clientPool, cert)
	}

	tlsConfig := &tls.Config{
		Certificates:             clientPool,
		RootCAs:                  serverPool,
		InsecureSkipVerify:       false,
		PreferServerCipherSuites: true,
	}
	tlsConfig.BuildNameToCertificate()

	return tlsConfig
}

func newObserver(server *aero.Host, user, pass string) (o *Observer, err error) {
	// use all cpus in the system for concurrency
	*authMode = strings.ToLower(strings.TrimSpace(*authMode))
	if *authMode != "internal" && *authMode != "external" {
		log.Fatalln("Invalid auth mode: only `internal` and `external` values are accepted.")
	}

	clientPolicy := aero.NewClientPolicy()
	clientPolicy.User = user
	clientPolicy.Password = pass
	if *authMode == "external" {
		clientPolicy.AuthMode = aero.AuthModeExternal
	}

	// allow only ONE connection
	clientPolicy.ConnectionQueueSize = 1
	clientPolicy.Timeout = time.Duration(*timeout) * time.Millisecond

	clientPolicy.TlsConfig = initTLS()

	newConnGen := func() (*aero.Connection, error) {
		conn, err := aero.NewConnection(clientPolicy, server)
		if err != nil {
			return nil, err
		}

		if clientPolicy.RequiresAuthentication() {
			if err := conn.Login(clientPolicy); err != nil {
				return nil, err
			}
		}

		return conn, nil
	}

	o = &Observer{
		connGen: newConnGen,
		ticks: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: "aerospike",
				Subsystem: "node",
				Name:      "ticks",
				Help:      "Counter that detemines how many times the Aerospike node was scraped for metrics.",
			}),
		watchers: []Watcher{
			&NamespaceWatcher{},
			&SetWatcher{},
			&LatencyWatcher{},
			&StatsWatcher{}}, // the order is important here
	}

	return o, nil
}

// Describe function of Prometheus' Collector interface
func (o *Observer) Describe(ch chan<- *prometheus.Desc) {
	return
}

// Collect function of Prometheus' Collector interface
func (o *Observer) Collect(ch chan<- prometheus.Metric) {
	o.ticks.Inc()
	ch <- o.ticks

	stats, err := o.refresh(ch)
	if err != nil {
		log.Println(err)
		ch <- prometheus.MustNewConstMetric(nodeActiveDesc, prometheus.GaugeValue, 0.0, gClusterName, gService, gBuild)
		return
	}

	gClusterName, gService, gBuild = stats["cluster-name"], stats["service"], stats["build"]
	ch <- prometheus.MustNewConstMetric(nodeActiveDesc, prometheus.GaugeValue, 1.0, gClusterName, gService, gBuild)
}

func (o *Observer) refresh(ch chan<- prometheus.Metric) (map[string]string, error) {
	log.Println("Trying to refresh the node...")

	var err error

	// prepare a connection
	conn := o.conn
	if conn == nil || !conn.IsConnected() {
		o.conn = nil
		conn, err = o.connGen()
		if err != nil {
			return nil, err
		}
	}

	// get first keys
	var infoKeys []string
	for _, c := range o.watchers {
		if keys := c.infoKeys(); len(keys) > 0 {
			infoKeys = append(infoKeys, keys...)
		}
	}

	// request first round of keys
	rawMetrics, err := aero.RequestInfo(conn, infoKeys...)
	if err != nil {
		return nil, err
	}

	// get first keys
	infoKeys = []string{"cluster-name", "service", "build"}
	watcherInfoKeys := make([][]string, len(o.watchers))
	for i, c := range o.watchers {
		if keys := c.detailKeys(rawMetrics); len(keys) > 0 {
			infoKeys = append(infoKeys, keys...)
			watcherInfoKeys[i] = keys
		}
	}

	// log.Println(infoKeys)

	// request second round of keys
	nRawMetrics, err := aero.RequestInfo(conn, infoKeys...)
	if err != nil {
		return rawMetrics, err
	}

	rawMetrics = nRawMetrics
	// log.Println(rawMetrics)

	accu := make(map[string]interface{}, 16)
	for i, c := range o.watchers {
		if err := c.refresh(watcherInfoKeys[i], rawMetrics, accu, ch); err != nil {
			return rawMetrics, err
		}
	}

	log.Println("Refreshing node was successful.")

	return rawMetrics, nil
}
