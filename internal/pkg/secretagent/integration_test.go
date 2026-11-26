//go:build integration

package secretagent_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/secretagent"
	"github.com/stretchr/testify/require"
)

func TestIntegrationTCP(t *testing.T) {
	client := clearClient(t, os.Getenv("APE_SA_TCP"))
	requireSecret(t, client, "DbPassword", os.Getenv("APE_DB_PASSWORD"))
}

func TestIntegrationUnixSocket(t *testing.T) {
	requireIntegration(t)

	cfg, err := secretagent.Parse(secretagent.RawConfig{
		SocketPath:     os.Getenv("APE_SA_UDS"),
		TimeoutSeconds: 5,
	})
	require.NoError(t, err)

	client, err := secretagent.New(cfg, nil)
	require.NoError(t, err)
	requireSecret(t, client, "DbPassword", os.Getenv("APE_DB_PASSWORD"))
}

func TestIntegrationTLS(t *testing.T) {
	client := tlsClient(t, os.Getenv("APE_SA_TLS"), os.Getenv("APE_SA_CA_FILE"), false)
	requireSecret(t, client, "DbPassword", os.Getenv("APE_DB_PASSWORD"))
}

func TestIntegrationMTLS(t *testing.T) {
	client := tlsClient(t, os.Getenv("APE_SA_MTLS"), os.Getenv("APE_SA_CA_FILE"), true)
	requireSecret(t, client, "DbPassword", os.Getenv("APE_DB_PASSWORD"))
}

func TestIntegrationAlreadyBase64(t *testing.T) {
	client := clearClient(t, os.Getenv("APE_SA_TCP_B64"))
	requireSecret(t, client, "DbPassword", os.Getenv("APE_DB_PASSWORD"))
}

func TestIntegrationCertificateMaterial(t *testing.T) {
	textClient := clearClient(t, os.Getenv("APE_SA_TCP_B64"))
	textCA, textCert, textKey := fetchCertificateMaterial(t, textClient, "secrets:Certs:")
	requirePEMFile(t, "/certs/ca.crt", textCA)
	requirePEMFile(t, "/certs/client.crt", textCert)
	requirePEMFile(t, "/certs/client.key", textKey)
	requireMutualTLSHandshake(t, textCA, textCert, textKey)

	b64Client := clearClient(t, os.Getenv("APE_SA_TCP"))
	b64CA, b64Cert, b64Key := fetchCertificateMaterial(t, b64Client, "secrets:b64:Certs:")
	require.Equal(t, normalizePEM(textCA), normalizePEM(b64CA))
	require.Equal(t, normalizePEM(textCert), normalizePEM(b64Cert))
	require.Equal(t, normalizePEM(textKey), normalizePEM(b64Key))
	requireMutualTLSHandshake(t, b64CA, b64Cert, b64Key)
}

func TestIntegrationUnknownResource(t *testing.T) {
	client := clearClient(t, os.Getenv("APE_SA_TCP"))
	_, err := client.Fetch(context.Background(), "missing", "DbPassword")
	require.Error(t, err)
	require.NotContains(t, err.Error(), os.Getenv("APE_DB_PASSWORD"))
}

func TestIntegrationTLSVerificationFailure(t *testing.T) {
	client := tlsClient(t, os.Getenv("APE_SA_TLS"), os.Getenv("APE_SA_OTHER_CA_FILE"), false)
	_, err := client.Fetch(context.Background(), "APE", "DbPassword")
	require.Error(t, err)
	require.Contains(t, err.Error(), "tls")
	require.NotContains(t, err.Error(), os.Getenv("APE_DB_PASSWORD"))
}

func TestIntegrationMTLSWithoutClientCertificate(t *testing.T) {
	client := tlsClient(t, os.Getenv("APE_SA_MTLS"), os.Getenv("APE_SA_CA_FILE"), false)
	_, err := client.Fetch(context.Background(), "APE", "DbPassword")
	require.Error(t, err)
	require.NotContains(t, err.Error(), os.Getenv("APE_DB_PASSWORD"))
}

func TestIntegrationExporterTCP(t *testing.T) {
	requireExporterMetrics(t, os.Getenv("APE_METRICS_URL"), os.Getenv("APE_LOG_FILE"))
}

func TestIntegrationExporterUDS(t *testing.T) {
	requireExporterMetrics(t, os.Getenv("APE_UDS_METRICS_URL"), os.Getenv("APE_UDS_LOG_FILE"))
}

func requireExporterMetrics(t *testing.T, url, logFile string) {
	t.Helper()
	requireIntegration(t)

	user := os.Getenv("APE_BASIC_AUTH_USER")
	password := os.Getenv("APE_BASIC_AUTH_PASSWORD")
	dbPassword := os.Getenv("APE_DB_PASSWORD")

	unauthenticated, err := http.Get(url)
	if err == nil {
		_ = unauthenticated.Body.Close()
	}

	var status int
	require.Eventually(t, func() bool {
		request, reqErr := http.NewRequest(http.MethodGet, url, nil)
		if reqErr != nil {
			return false
		}

		request.SetBasicAuth(user, password)
		response, reqErr := http.DefaultClient.Do(request)
		if reqErr != nil {
			return false
		}

		defer func() { _ = response.Body.Close() }()
		_, _ = io.Copy(io.Discard, response.Body)
		status = response.StatusCode

		return status == http.StatusOK
	}, 3*time.Minute, time.Second)
	require.Equal(t, http.StatusOK, status)

	denied, err := http.Get(url)
	require.NoError(t, err)
	defer func() { _ = denied.Body.Close() }()
	require.Equal(t, http.StatusUnauthorized, denied.StatusCode)

	logBody, err := os.ReadFile(logFile)
	require.NoError(t, err)
	require.NotContains(t, string(logBody), password)
	require.NotContains(t, string(logBody), dbPassword)
}

func requireIntegration(t *testing.T) {
	t.Helper()

	if os.Getenv("APE_SECRET_AGENT_IT") != "1" {
		t.Skip("set APE_SECRET_AGENT_IT=1 to run Secret Agent integration tests")
	}
}

func fetchCertificateMaterial(t *testing.T, client secretagent.SecretClient, prefix string) (ca, cert, key []byte) {
	t.Helper()

	commons.RegisterSecretAgent(client)
	t.Cleanup(func() { commons.RegisterSecretAgent(nil) })

	var err error
	ca, err = commons.GetSecret(prefix + "ca_cert")
	require.NoError(t, err)
	cert, err = commons.GetSecret(prefix + "client_cert")
	require.NoError(t, err)
	key, err = commons.GetSecret(prefix + "client_key")
	require.NoError(t, err)
	require.NoError(t, client.Close())

	return ca, cert, key
}

func requirePEMFile(t *testing.T, path string, got []byte) {
	t.Helper()

	file, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, normalizePEM(file), normalizePEM(got))
}

func normalizePEM(in []byte) string {
	return strings.TrimRight(string(in), " \t\n\r\f\v")
}

func requireMutualTLSHandshake(t *testing.T, caPEM, certPEM, keyPEM []byte) {
	t.Helper()

	pair, err := tls.X509KeyPair(certPEM, keyPEM)
	require.NoError(t, err)

	pool := x509.NewCertPool()
	require.True(t, pool.AppendCertsFromPEM(caPEM))

	conn, err := tls.Dial("tcp", os.Getenv("APE_SA_MTLS"), &tls.Config{
		Certificates: []tls.Certificate{pair},
		RootCAs:      pool,
		ServerName:   os.Getenv("APE_SA_TLS_NAME"),
		MinVersion:   tls.VersionTLS12,
	})
	require.NoError(t, err)
	require.NoError(t, conn.Close())
}

func requireSecret(t *testing.T, client secretagent.SecretClient, key, want string) {
	t.Helper()

	commons.RegisterSecretAgent(client)
	t.Cleanup(func() { commons.RegisterSecretAgent(nil) })

	got, err := commons.GetSecret("secrets:b64:APE:" + key)
	require.NoError(t, err)
	require.Equal(t, want, string(got))
	require.NoError(t, client.Close())
}

func clearClient(t *testing.T, endpoint string) secretagent.SecretClient {
	t.Helper()
	requireIntegration(t)

	host, port := splitEndpoint(t, endpoint)
	cfg, err := secretagent.Parse(secretagent.RawConfig{
		Address:        host,
		Port:           port,
		TimeoutSeconds: 5,
	})
	require.NoError(t, err)

	client, err := secretagent.New(cfg, nil)
	require.NoError(t, err)

	return client
}

func tlsClient(t *testing.T, endpoint, caFile string, mutual bool) secretagent.SecretClient {
	t.Helper()
	requireIntegration(t)

	host, port := splitEndpoint(t, endpoint)
	raw := secretagent.RawConfig{
		Address:        host,
		Port:           port,
		TLSName:        os.Getenv("APE_SA_TLS_NAME"),
		CAFile:         caFile,
		TimeoutSeconds: 5,
	}
	if mutual {
		raw.CertFile = os.Getenv("APE_SA_CLIENT_CERT")
		raw.KeyFile = os.Getenv("APE_SA_CLIENT_KEY")
	}

	cfg, err := secretagent.Parse(raw)
	require.NoError(t, err)

	tlsConfig, err := commons.BuildSecretAgentTLS(cfg)
	require.NoError(t, err)
	require.NotNil(t, tlsConfig)
	require.False(t, tlsConfig.InsecureSkipVerify)
	require.Equal(t, uint16(tls.VersionTLS12), tlsConfig.MinVersion)

	client, err := secretagent.New(cfg, tlsConfig)
	require.NoError(t, err)

	return client
}

func splitEndpoint(t *testing.T, endpoint string) (string, uint16) {
	t.Helper()

	host, portText, err := net.SplitHostPort(endpoint)
	require.NoError(t, err)
	parsed, err := strconv.Atoi(portText)
	require.NoError(t, err)

	return host, uint16(parsed)
}
