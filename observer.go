package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"strings"
	"sync"
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
	mutex         sync.Mutex
}

var (
	// aerospike_node_up metric descriptor
	nodeActiveDesc *prometheus.Desc

	// Node service endpoint, cluster name and build version
	gService, gClusterName, gBuild string

	// Number of retries on info request
	retryCount = 3

	// Default info commands
	ikClusterName = "cluster-name"
	ikService     = "service-clear-std"
	ikBuild       = "build"
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
		caCert, err := getCertificate(config.Aerospike.RootCA)
		if err != nil {
			log.Fatal(err)
		}

		log.Debugf("Adding CA certificate to the pool...")
		serverPool.AppendCertsFromPEM(caCert)
	}

	var clientPool []tls.Certificate
	if len(config.Aerospike.CertFile) > 0 || len(config.Aerospike.KeyFile) > 0 {

		// Read cert file
		certFileBytes, err := getCertificate(config.Aerospike.CertFile)
		if err != nil {
			log.Fatal(err)
		}

		// Read key file
		keyFileBytes, err := getCertificate(config.Aerospike.KeyFile)
		if err != nil {
			log.Fatal(err)
		}

		// Decode PEM data
		keyBlock, _ := pem.Decode([]byte(keyFileBytes))
		certBlock, _ := pem.Decode([]byte(certFileBytes))

		if keyBlock == nil || certBlock == nil {
			log.Fatalf("Failed to decode PEM data for key or certificate")
		}

		// Check and Decrypt the the Key Block using passphrase
		if x509.IsEncryptedPEMBlock(keyBlock) {
			keyFilePassphraseBytes, err := getSecret(config.Aerospike.KeyFilePassphrase)
			if err != nil {
				log.Fatalf("Failed to get key passphrase: `%s`", err)
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
			log.Fatalf("Failed to encode PEM data for key or certificate")
		}

		cert, err := tls.X509KeyPair(certPEM, keyPEM)

		if err != nil {
			log.Fatalf("Failed to add client certificate and key to the pool: `%s`", err)
		}

		log.Debugf("Adding client certificate and key to the pool...")
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
	// initialize aerospike_node_up metric descriptor
	nodeActiveDesc = prometheus.NewDesc(
		"aerospike_node_up",
		"Aerospike node active status",
		[]string{"cluster_name", "service", "build"},
		config.AeroProm.MetricLabels,
	)

	// use all cpus in the system for concurrency
	authMode := strings.ToLower(strings.TrimSpace(config.Aerospike.AuthMode))
	if authMode != "internal" && authMode != "external" {
		log.Fatalln("Invalid auth mode: only `internal` and `external` values are accepted.")
	}

	// Get aerospike auth username
	username, err := getSecret(user)
	if err != nil {
		log.Fatal(err)
	}

	// Get aerospike auth password
	password, err := getSecret(pass)
	if err != nil {
		log.Fatal(err)
	}

	clientPolicy := aero.NewClientPolicy()
	clientPolicy.User = string(username)
	clientPolicy.Password = string(password)
	if authMode == "external" {
		clientPolicy.AuthMode = aero.AuthModeExternal
	}

	// allow only ONE connection
	clientPolicy.ConnectionQueueSize = 1
	clientPolicy.Timeout = time.Duration(config.Aerospike.Timeout) * time.Second

	clientPolicy.TlsConfig = initTLS()

	if clientPolicy.TlsConfig != nil {
		ikService = "service-tls-std"
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
	// Protect against concurrent scrapes
	o.mutex.Lock()
	defer o.mutex.Unlock()

	o.ticks.Inc()
	ch <- o.ticks

	stats, err := o.refresh(ch)
	if err != nil {
		log.Errorln(err)
		ch <- prometheus.MustNewConstMetric(nodeActiveDesc, prometheus.GaugeValue, 0.0, gClusterName, gService, gBuild)
		return
	}

	gClusterName, gService, gBuild = stats[ikClusterName], stats[ikService], stats[ikBuild]
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

func (o *Observer) refresh(ch chan<- prometheus.Metric) (map[string]string, error) {
	log.Debugf("Refreshing node %s", fullHost)

	// fetch first set of info keys
	var infoKeys []string
	for _, c := range o.watchers {
		if keys := c.passOneKeys(); len(keys) > 0 {
			infoKeys = append(infoKeys, keys...)
		}
	}

	// info request for first set of info keys
	rawMetrics, err := o.requestInfo(retryCount, infoKeys)
	if err != nil {
		return nil, err
	}

	// fetch second second set of info keys
	infoKeys = []string{ikClusterName, ikService, ikBuild}
	watcherInfoKeys := make([][]string, len(o.watchers))
	for i, c := range o.watchers {
		if keys := c.passTwoKeys(rawMetrics); len(keys) > 0 {
			infoKeys = append(infoKeys, keys...)
			watcherInfoKeys[i] = keys
		}
	}

	// info request for second set of info keys
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
