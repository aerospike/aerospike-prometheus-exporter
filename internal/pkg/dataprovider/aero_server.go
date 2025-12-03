package dataprovider

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	aero "github.com/aerospike/aerospike-client-go/v8"
	"github.com/aerospike/aerospike-client-go/v8/types"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"
	"github.com/sirupsen/logrus"
)

// Inherits DataProvider interface
type AerospikeServer struct {
}

func (asm AerospikeServer) RequestInfo(infoKeys []string) (map[string]string, error) {
	return fetchRequestInfoFromAerospike(infoKeys)
}

func (asm AerospikeServer) FetchUsersDetails() (bool, []*aero.UserRoles, error) {
	return fetchUsersRoles()
}

// Aerospike server interaction related code

const (
	GO_CLIENT_LIBRARY_PATH     = "github.com/aerospike/aerospike-client-go/v8"
	AERO_EXPORTER_LIBRARY_PATH = "github.com/aerospike/aerospike-prometheus-exporter"
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

func initializeAndConnectAerospikeServer() (*aero.Connection, error) {

	fullHost = commons.GetFullHost()

	logrus.Debugf("Connecting to host %s ", fullHost)

	asServerHost = aero.NewHost(config.Cfg.Aerospike.Host, int(config.Cfg.Aerospike.Port))

	asServerHost.TLSName = config.Cfg.Aerospike.NodeTLSName
	user = config.Cfg.Aerospike.User
	pass = config.Cfg.Aerospike.Password

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

	switch config.Cfg.Aerospike.AuthMode {
	case "internal", "":
		clientPolicy.AuthMode = aero.AuthModeInternal
	case "external":
		clientPolicy.AuthMode = aero.AuthModeExternal
	case "pki":
		if len(config.Cfg.Aerospike.CertFile) == 0 || len(config.Cfg.Aerospike.KeyFile) == 0 {
			log.Fatalln("Invalid certificate configuration when using auth mode PKI: cert_file and key_file must be set")
		}
		clientPolicy.AuthMode = aero.AuthModePKI
	default:
		log.Fatalln("Invalid auth mode: only `internal`, `external`, `pki` values are accepted.")
	}

	// allow only ONE connection
	clientPolicy.ConnectionQueueSize = 1
	clientPolicy.Timeout = time.Duration(config.Cfg.Aerospike.Timeout) * time.Second

	clientPolicy.TlsConfig = initAerospikeTLS()

	return createNewConnection()
}

func initAerospikeTLS() *tls.Config {
	var clientPool []tls.Certificate
	var serverPool *x509.CertPool

	// load the server / client certificates
	serverPool, clientPool = commons.LoadServerOrClientCertificates()

	if serverPool != nil || clientPool != nil {
		// we either have server pool only (oneway-tls) or both serverPool and clientPoll (mTLS)
		// only clientPool without serverPool is invalid config.
		tlsConfig := &tls.Config{
			Certificates:             clientPool,
			RootCAs:                  serverPool,
			InsecureSkipVerify:       false,
			PreferServerCipherSuites: true,
			NameToCertificate:        nil,
		}
		return tlsConfig
	}

	return nil
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

func fetchRequestInfoFromAerospike(infoKeys []string) (map[string]string, error) {
	var err error
	rawMetrics := make(map[string]string)

	// Retry for connection, timeout, network errors
	// including errors from RequestInfo()
	for i := 0; i < retryCount; i++ {
		// Validate existing connection
		if asConnection == nil || !asConnection.IsConnected() {
			// Create new connection
			asConnection, err = initializeAndConnectAerospikeServer()
			if err != nil {
				logrus.Debug("Error while connecting to aerospike server: ", err)
				continue
			}

			// Set user-agent
			err = setUserAgent()
			if err != nil {
				logrus.Debug("Error while setting user-agent: ", err)
				continue
			}
		}

		// Info request
		rawMetrics, err = asConnection.RequestInfo(infoKeys...)
		if err != nil {
			logrus.Debug("Error while requestInfo ( infoKeys...), closing connection : Error is: ", err, " and infoKeys: ", infoKeys)
			asConnection.Close()
			//TODO: do we need to assign nil to asConnection? i.e. asConnection = nil
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

func fetchUsersRoles() (bool, []*aero.UserRoles, error) {

	shouldFetchUserStatistics := true

	admPlcy := aero.NewAdminPolicy()
	admPlcy.Timeout = time.Duration(config.Cfg.Aerospike.Timeout) * time.Second
	admCmd := aero.NewAdminCommand(nil)

	var users []*aero.UserRoles
	var aeroErr aero.Error
	var err error

	for i := 0; i < retryCount; i++ {
		// Validate existing connection
		if asConnection == nil || !asConnection.IsConnected() {
			// Create new connection
			asConnection, err = initializeAndConnectAerospikeServer()
			if err != nil {
				logrus.Debug(err)
				continue
			}
		}

		// query users
		users, aeroErr = admCmd.QueryUsers(asConnection, admPlcy)

		if aeroErr != nil {
			// Do not retry if there's role violation.
			// This could be a permanent error leading to unnecessary errors on server end.
			if aeroErr.Matches(types.ROLE_VIOLATION) {
				shouldFetchUserStatistics = false
				logrus.Debugf("Unable to fetch user statistics: %s", aeroErr.Error())
				break
			}

			if len(aeroErr.Error()) > 0 {
				logrus.Warnf("Error while querying users: %s", aeroErr.Error())
				continue
			}
		}

		break
	}

	return shouldFetchUserStatistics, users, nil
}

func setUserAgent() error {
	// Server expected format "user-agent-version","client-library-version","exporter-version/app-id-info"

	// Exporter version
	appId := commons.GetModuleVersion(AERO_EXPORTER_LIBRARY_PATH)
	// Aerospike GO client library version
	clientLibraryVersion := commons.GetModuleVersion(GO_CLIENT_LIBRARY_PATH)

	// set user-agent
	userAgentId := fmt.Sprintf("1,go-%s,ape-%s", clientLibraryVersion, appId)
	userAgentCommand := fmt.Sprintf("user-agent-set:value=%s", base64.StdEncoding.EncodeToString([]byte(userAgentId)))

	command := []string{userAgentCommand}

	logrus.Debug("Setting User-Agent in Server: infoKeys: ", command)
	_, err := asConnection.RequestInfo(command...)
	if err != nil {
		return err
	}

	return nil
}
