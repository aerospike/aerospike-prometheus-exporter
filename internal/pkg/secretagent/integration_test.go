//go:build integration

package secretagent_test

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
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
		IsBase64:       true,
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

func TestIntegrationExporterMetricsAndLogs(t *testing.T) {
	requireIntegration(t)

	url := os.Getenv("APE_METRICS_URL")
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

	logBody, err := os.ReadFile(os.Getenv("APE_LOG_FILE"))
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

func requireSecret(t *testing.T, client secretagent.Client, key, want string) {
	t.Helper()

	got, err := client.Fetch(context.Background(), "APE", key)
	require.NoError(t, err)
	require.Equal(t, want, string(got))
	require.NoError(t, client.Close())
}

func clearClient(t *testing.T, endpoint string) secretagent.Client {
	t.Helper()
	requireIntegration(t)

	host, port := splitEndpoint(t, endpoint)
	cfg, err := secretagent.Parse(secretagent.RawConfig{
		Address:        host,
		Port:           port,
		TimeoutSeconds: 5,
		IsBase64:       true,
	})
	require.NoError(t, err)

	client, err := secretagent.New(cfg, nil)
	require.NoError(t, err)

	return client
}

func tlsClient(t *testing.T, endpoint, caFile string, mutual bool) secretagent.Client {
	t.Helper()
	requireIntegration(t)

	host, port := splitEndpoint(t, endpoint)
	raw := secretagent.RawConfig{
		Address:        host,
		Port:           port,
		TLSName:        os.Getenv("APE_SA_TLS_NAME"),
		CAFile:         caFile,
		TimeoutSeconds: 5,
		IsBase64:       true,
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
