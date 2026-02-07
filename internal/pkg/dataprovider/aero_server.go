package dataprovider

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	aero "github.com/aerospike/aerospike-client-go/v8"
	"github.com/aerospike/aerospike-client-go/v8/types"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"
	log "github.com/sirupsen/logrus"
)

// Inherits DataProvider interface
type AerospikeServer struct {
	aeroConnection *aero.Connection
	clientPolicy   *aero.ClientPolicy
	serverHost     *aero.Host
}

func (as *AerospikeServer) RequestInfo(infoKeys []string) (map[string]string, error) {

	return as.fetchRequestInfoFromAerospike(infoKeys)
}

func (as *AerospikeServer) FetchUsersDetails() (bool, []*aero.UserRoles, error) {
	return as.fetchUsersRoles()
}

// Aerospike server interaction related code

const (
	GO_CLIENT_LIBRARY_PATH         = "github.com/aerospike/aerospike-client-go/v8"
	AERO_EXPORTER_LIBRARY_PATH     = "github.com/aerospike/aerospike-prometheus-exporter"
	RETRY_COUNT                int = 3
)

var (
	fullHost string
	user     string
	pass     string
)

func (as *AerospikeServer) initializeAndConnectAerospikeServer() (*aero.Connection, error) {

	fmt.Println("initializing and connecting to aerospike server ")

	fullHost = commons.GetFullHost()

	log.Debugf("Connecting to host %s ", fullHost)

	as.serverHost = aero.NewHost(config.Cfg.Aerospike.Host, int(config.Cfg.Aerospike.Port))

	as.serverHost.TLSName = config.Cfg.Aerospike.NodeTLSName
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

	as.clientPolicy = aero.NewClientPolicy()
	as.clientPolicy.User = string(username)
	as.clientPolicy.Password = string(password)

	switch config.Cfg.Aerospike.AuthMode {
	case "internal", "":
		as.clientPolicy.AuthMode = aero.AuthModeInternal
	case "external":
		as.clientPolicy.AuthMode = aero.AuthModeExternal
	case "pki":
		if len(config.Cfg.Aerospike.CertFile) == 0 || len(config.Cfg.Aerospike.KeyFile) == 0 {
			log.Fatalln("Invalid certificate configuration when using auth mode PKI: cert_file and key_file must be set")
		}
		as.clientPolicy.AuthMode = aero.AuthModePKI
	default:
		log.Fatalln("Invalid auth mode: only `internal`, `external`, `pki` values are accepted.")
	}

	// allow only ONE connection
	as.clientPolicy.ConnectionQueueSize = 1
	as.clientPolicy.Timeout = time.Duration(config.Cfg.Aerospike.Timeout) * time.Second

	as.clientPolicy.TlsConfig = as.initAerospikeTLS()

	return as.createNewConnection()
}

func (as *AerospikeServer) initAerospikeTLS() *tls.Config {
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

func (as *AerospikeServer) createNewConnection() (*aero.Connection, error) {
	var err error
	as.aeroConnection, err = aero.NewConnection(as.clientPolicy, as.serverHost)

	if err != nil {
		return nil, err
	}

	if as.clientPolicy.RequiresAuthentication() {
		if err := as.aeroConnection.Login(as.clientPolicy); err != nil {
			return nil, err
		}
	}

	// Set no connection deadline to re-use connection, but socketTimeout will be in effect
	var deadline time.Time
	err = as.aeroConnection.SetTimeout(deadline, as.clientPolicy.Timeout)

	if err != nil {
		return nil, err
	}

	return as.aeroConnection, nil
}

func (as *AerospikeServer) fetchRequestInfoFromAerospike(infoKeys []string) (map[string]string, error) {
	var err error
	requestInfoResponse := make(map[string]string)

	// Retry for connection, timeout, network errors
	// including errors from RequestInfo()
	for i := 0; i < RETRY_COUNT; i++ {
		// Validate existing connection
		var callStartTime = time.Now()
		if as.aeroConnection == nil || !as.aeroConnection.IsConnected() {
			// Create new connection
			as.aeroConnection, err = as.initializeAndConnectAerospikeServer()

			if err != nil {
				log.Debugf("Error while connecting to aerospike server: %v", err)
				continue
			}

			log.Debugf("Connection to Server took %v", time.Since(callStartTime))

			// Set user-agent
			err = as.setUserAgent()

			if err != nil {
				log.Debugf("Error while setting user-agent: %v", err)
				continue
			}
		}

		// Info request
		callStartTime = time.Now()
		requestInfoResponse, err = as.aeroConnection.RequestInfo(infoKeys...)

		log.Debugf("RequestInfo took %v", time.Since(callStartTime))

		if err != nil {
			log.Errorf("Error while requestInfo ( infoKeys...), closing connection : Error is: %v and infoKeys: %v", err, infoKeys)
			as.aeroConnection.Close()
			// making nil, to force a connection, if any n/w disruption happen between my connection call
			//   and requestinfo call, -- it internall will fail because of n/w disruption
			as.aeroConnection = nil
			continue
		}

		break
	}

	if len(requestInfoResponse) == 1 {
		for k := range requestInfoResponse {
			if strings.HasPrefix(strings.ToUpper(k), "ERROR:") {
				return nil, errors.New(k)
			}
		}
	}

	return requestInfoResponse, err
}

func (as *AerospikeServer) fetchUsersRoles() (bool, []*aero.UserRoles, error) {

	shouldFetchUserStatistics := true

	admPlcy := aero.NewAdminPolicy()
	admPlcy.Timeout = time.Duration(config.Cfg.Aerospike.Timeout) * time.Second
	admCmd := aero.NewAdminCommand(nil)

	var users []*aero.UserRoles
	var aeroErr aero.Error
	var err error

	for i := 0; i < RETRY_COUNT; i++ {
		// Validate existing connection
		if as.aeroConnection == nil || !as.aeroConnection.IsConnected() {
			// Create new connection
			as.aeroConnection, err = as.initializeAndConnectAerospikeServer()
			if err != nil {
				log.Debugf("Error while initializing and connecting to aerospike server: %s", err)
				continue
			}
		}

		// query users
		users, aeroErr = admCmd.QueryUsers(as.aeroConnection, admPlcy)

		if aeroErr != nil {
			// Do not retry if there's role violation.
			// This could be a permanent error leading to unnecessary errors on server end.
			if aeroErr.Matches(types.ROLE_VIOLATION) {
				shouldFetchUserStatistics = false
				log.Debugf("Unable to fetch user statistics: %s", aeroErr.Error())
				break
			}

			if len(aeroErr.Error()) > 0 {
				log.Warnf("Error while querying users: %s", aeroErr.Error())
				continue
			}
		}

		break
	}

	return shouldFetchUserStatistics, users, nil
}

func (as *AerospikeServer) setUserAgent() error {
	// Server expected format "user-agent-version","client-library-version","exporter-version/app-id-info"

	// Exporter version
	appId := commons.GetModuleVersion(AERO_EXPORTER_LIBRARY_PATH)
	// Aerospike GO client library version
	clientLibraryVersion := commons.GetModuleVersion(GO_CLIENT_LIBRARY_PATH)

	// set user-agent
	userAgentId := fmt.Sprintf("1,go-%s,ape-%s", clientLibraryVersion, appId)
	userAgentCommand := fmt.Sprintf("user-agent-set:value=%s", base64.StdEncoding.EncodeToString([]byte(userAgentId)))

	command := []string{userAgentCommand}

	log.Debug("Setting User-Agent in Server: infoKeys: ", command)
	_, err := as.aeroConnection.RequestInfo(command...)

	if err != nil {
		return err
	}

	return nil
}
