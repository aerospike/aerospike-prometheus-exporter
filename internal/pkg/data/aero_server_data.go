package data

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"log"
	"strings"
	"time"

	aero "github.com/aerospike/aerospike-client-go/v6"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/sirupsen/logrus"
)

var (
	fullHost   string
	user       string
	pass       string
	retryCount int = 3

	asConnection *aero.Connection
	clientPolicy *aero.ClientPolicy
	asServerHost *aero.Host
)

func NewAerospikeConnection() {

	fullHost = commons.GetFullHost()

	logrus.Debugf("Connecting to host %s ", fullHost)

	asServerHost = aero.NewHost(commons.Cfg.Aerospike.Host, int(commons.Cfg.Aerospike.Port))
	asServerHost.TLSName = commons.Cfg.Aerospike.NodeTLSName
	user = commons.Cfg.Aerospike.User
	pass = commons.Cfg.Aerospike.Password

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

	clientPolicy = aero.NewClientPolicy()
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

	clientPolicy.TlsConfig = InitAerospikeTLS()

	if clientPolicy.TlsConfig != nil {
		commons.Infokey_Service = "service-tls-std"
	}

}

func InitAerospikeTLS() *tls.Config {
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

func createNewConnection() (*aero.Connection, error) {
	asConnection, err := aero.NewConnection(clientPolicy, asServerHost)
	if err != nil {
		return nil, err
	}

	if clientPolicy.RequiresAuthentication() {
		if err := asConnection.Login(clientPolicy); err != nil {
			return nil, err
		}
	}

	// Set no connection deadline to re-use connection, but socketTimeout will be in effect
	var deadline time.Time
	err = asConnection.SetTimeout(deadline, clientPolicy.Timeout)
	if err != nil {
		return nil, err
	}

	return asConnection, nil
}

func RequestInfo(infoKeys []string) (map[string]string, error) {
	var err error
	rawMetrics := make(map[string]string)

	// Retry for connection, timeout, network errors
	// including errors from RequestInfo()
	for i := 0; i < retryCount; i++ {
		// Validate existing connection
		if asConnection == nil || !asConnection.IsConnected() {
			// Create new connection
			asConnection, err = createNewConnection()
			if err != nil {
				logrus.Debug("Error while connecting to aerospike server: ", err)
				continue
			}
		}

		// Info request
		rawMetrics, err = asConnection.RequestInfo(infoKeys...)
		if err != nil {
			logrus.Debug("Error while requestInfo ( infoKeys...) : ", err)
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