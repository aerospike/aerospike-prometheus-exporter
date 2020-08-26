package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	aero "github.com/aerospike/aerospike-client-go"
	log "github.com/sirupsen/logrus"
)

// Observer communicates with Aerospike and helps collecting metrices
type Observer struct {
	conn          *aero.Connection
	newConnection func() (*aero.Connection, error)
	ticks         prometheus.Counter
	watchers      []Watcher
}

var (
	nodeActiveDesc = prometheus.NewDesc(
		"aerospike_node_up",
		"Aerospike node active status",
		[]string{"cluster_name", "service", "build"},
		nil,
	)

	gService, gClusterName, gBuild string

	retryCount = 3
)

func initTLS() *tls.Config {
	if len(config.Aerospike.RootCA) == 0 && len(config.Aerospike.CertFile) == 0 && len(config.Aerospike.KeyFile) == 0 {
		return nil
	}

	// Try to load system CA certs, otherwise just make an empty pool
	serverPool, err := x509.SystemCertPool()
	if serverPool == nil || err != nil {
		log.Debugf("Adding system certificates to the cert pool failed: %s", err)
		serverPool = x509.NewCertPool()
	}

	if len(config.Aerospike.RootCA) > 0 {
		// Try to load system CA certs and add them to the system cert pool
		caCert := readCertFile(config.Aerospike.RootCA)

		log.Debugf("Adding server certificate `%s` to the pool...", config.Aerospike.RootCA)
		serverPool.AppendCertsFromPEM(caCert)
	}

	var clientPool []tls.Certificate
	if len(config.Aerospike.CertFile) > 0 || len(config.Aerospike.KeyFile) > 0 {

		// Read Cert and Key files
		certFileBytes := readCertFile(config.Aerospike.CertFile)
		keyFileBytes := readCertFile(config.Aerospike.KeyFile)

		// Decode PEM data
		keyBlock, _ := pem.Decode([]byte(keyFileBytes))
		certBlock, _ := pem.Decode([]byte(certFileBytes))

		if keyBlock == nil || certBlock == nil {
			log.Fatalf("Unable to decode PEM data for `%s` or `%s`", config.Aerospike.KeyFile, config.Aerospike.CertFile)
		}

		// Check and Decrypt the the Key Block using passphrase
		if x509.IsEncryptedPEMBlock(keyBlock) {
			keyFilePassphraseBytes, err := getKeyFilePassphrase(config.Aerospike.KeyFilePassphrase)
			if err != nil {
				log.Fatalf("Failed to get Key file passphrase for `%s` : `%s`", config.Aerospike.KeyFile, err)
			}

			decryptedDERBytes, err := x509.DecryptPEMBlock(keyBlock, keyFilePassphraseBytes)
			if err != nil {
				log.Fatalf("Failed to decrypt PEM Block: `%s`", err)
			}

			keyBlock.Bytes = decryptedDERBytes
			keyBlock.Headers = nil
		}

		// Encode PEM data
		keyPEM := pem.EncodeToMemory(keyBlock)
		certPEM := pem.EncodeToMemory(certBlock)

		if keyPEM == nil || certPEM == nil {
			log.Fatalf("Unable to encode PEM data for `%s` or `%s`", config.Aerospike.KeyFile, config.Aerospike.CertFile)
		}

		cert, err := tls.X509KeyPair(certPEM, keyPEM)

		if err != nil {
			log.Fatalf("FAILED: Adding client certificate `%s` and key file `%s` to the pool failed: `%s`", config.Aerospike.CertFile, config.Aerospike.KeyFile, err)
		}

		log.Debugf("Adding client certificate `%s` to the pool...\n", config.Aerospike.CertFile)
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
	authMode := strings.ToLower(strings.TrimSpace(config.Aerospike.AuthMode))
	if authMode != "internal" && authMode != "external" {
		log.Fatalln("Invalid auth mode: only `internal` and `external` values are accepted.")
	}

	clientPolicy := aero.NewClientPolicy()
	clientPolicy.User = user
	clientPolicy.Password = pass
	if authMode == "external" {
		clientPolicy.AuthMode = aero.AuthModeExternal
	}

	// allow only ONE connection
	clientPolicy.ConnectionQueueSize = 1
	clientPolicy.Timeout = time.Duration(config.Aerospike.Timeout) * time.Second

	clientPolicy.TlsConfig = initTLS()

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
		conn.SetTimeout(deadline, clientPolicy.Timeout)

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
		watchers: []Watcher{
			&NamespaceWatcher{},
			&SetWatcher{},
			&LatencyWatcher{},
			&StatsWatcher{},
			&XdrWatcher{}}, // the order is important here
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
		log.Errorln(err)
		ch <- prometheus.MustNewConstMetric(nodeActiveDesc, prometheus.GaugeValue, 0.0, gClusterName, gService, gBuild)
		return
	}

	gClusterName, gService, gBuild = stats["cluster-name"], stats["service"], stats["build"]
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
		rawMetrics, err = aero.RequestInfo(o.conn, infoKeys...)
		if err != nil {
			log.Debug(err)
			continue
		}
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

func (o *Observer) refresh(ch chan<- prometheus.Metric) (map[string]string, error) {
	log.Debugf("Refreshing node %s", fullHost)

	// get first keys
	var infoKeys []string
	for _, c := range o.watchers {
		if keys := c.infoKeys(); len(keys) > 0 {
			infoKeys = append(infoKeys, keys...)
		}
	}

	// request first round of keys
	rawMetrics, err := o.requestInfo(retryCount, infoKeys)
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

	// request second round of keys
	nRawMetrics, err := o.requestInfo(retryCount, infoKeys)
	if err != nil {
		return rawMetrics, err
	}

	rawMetrics = nRawMetrics

	// sanitize the utf8 strings before sending them to watchers
	for k, v := range rawMetrics {
		rawMetrics[k] = sanitizeUTF8(v)
	}

	for i, c := range o.watchers {
		if err := c.refresh(watcherInfoKeys[i], rawMetrics, ch); err != nil {
			return rawMetrics, err
		}
	}

	log.Debugf("Refreshing node was successful")

	return rawMetrics, nil
}
