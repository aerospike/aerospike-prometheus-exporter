//go:build awse2e

// Package e2e runs the exporter against Secret Agent and checks HTTPS and mutual TLS
// end to end. Run it through ci/secret-agent-aws/run.sh.
package e2e

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	apeBin    = "/tmp/ape"
	gaugeList = "/src/configs/gauge_stats_list.toml"
	certDir   = "/work/certs"
	resource  = "APEAgentSecurity"
	saSocket  = "/run/sa/secret-agent.sock"
	saAddress = "secret-agent:3005"
)

// Probe values stored by scripts/secret.sh. Keep them in sync with that file.
const (
	probeUser         = "ape"
	pwTrailingSpace   = "ape-pass-9f3c "
	pwTrailingNewline = "ape-pass-9f3c\n"
	pwWithColon       = "ape:pass:9f3c"
)

var (
	transport = envOr("APE_IT_SA_TRANSPORT", "tcp")
	backend   = envOr("APE_IT_SA_BACKEND", "aws")
	encoding  = envOr("APE_IT_PEM_ENCODING", "base64")
)

func TestMain(m *testing.M) {
	if os.Getenv("APE_IT_E2E") != "1" {
		fmt.Println("set APE_IT_E2E=1 (run.sh does this) to run the Secret Agent e2e suite")
		os.Exit(0)
	}

	if err := waitForSecretAgent(90 * time.Second); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func TestHTTPSWithServerTLS(t *testing.T) {
	ape := startAPE(t, agentTLS(pemRef("cert_file"), pemRef("key_file_plain"), "", ""))
	ape.waitListening(t)

	t.Run("trusted CA returns 200", func(t *testing.T) {
		requireMetricsOK(t, ape.addr, caPool(t, "ca.crt"), nil)
	})

	t.Run("untrusted CA fails verification", func(t *testing.T) {
		_, err := getMetrics(ape.addr, caPool(t, "other-ca.crt"), nil)
		var unknownAuthority x509.UnknownAuthorityError
		require.ErrorAs(t, err, &unknownAuthority)
	})
}

func TestHTTPSWithMutualTLS(t *testing.T) {
	ape := startAPE(t, agentTLS(pemRef("cert_file"), pemRef("key_file_plain"), pemRef("root_ca"), ""))
	ape.waitListening(t)
	trusted := caPool(t, "ca.crt")

	t.Run("valid client certificate returns 200", func(t *testing.T) {
		requireMetricsOK(t, ape.addr, trusted, clientCert(t, "client"))
	})

	t.Run("no client certificate is rejected", func(t *testing.T) {
		_, err := getMetrics(ape.addr, trusted, nil)
		require.Error(t, err)
	})

	t.Run("client certificate from another CA is rejected", func(t *testing.T) {
		_, err := getMetrics(ape.addr, trusted, clientCert(t, "rogue-client"))
		require.Error(t, err)
	})
}

func TestEncryptedServerKeyUsesResolvedPassphrase(t *testing.T) {
	ape := startAPE(t, agentTLS(pemRef("cert_file"), pemRef("key_file"), pemRef("root_ca"), textRef("key_file_passphrase")))
	ape.waitListening(t)

	requireMetricsOK(t, ape.addr, caPool(t, "ca.crt"), clientCert(t, "client"))
}

func TestPlainValueConvertedToBase64(t *testing.T) {
	if backend == "aws" && os.Getenv("APE_IT_AWS_PLAIN_SECRET_ARN") == "" {
		t.Skip("APE_IT_AWS_PLAIN_SECRET_ARN is unset")
	}

	user, pass := plainCredentials(t)
	require.NoError(t, waitForPlainAgent(90*time.Second))

	ape := startAPEAgainst(t, fmt.Sprintf("basic_auth_username = %q\nbasic_auth_password = %q\n",
		"secrets:b64:APEPlain:username",
		"secrets:b64:APEPlain:password"), plainSecretAgentBlock())
	ape.waitListening(t)

	ok, err := getHTTPMetrics(ape.addr, user, pass)
	require.NoError(t, err)
	defer func() { _ = ok.Body.Close() }()
	require.Equal(t, http.StatusOK, ok.StatusCode)

	// The encoded text must not be accepted as the password. That would mean the exporter skipped the decode.
	encoded := base64.StdEncoding.EncodeToString([]byte(pass))
	rejected, err := getHTTPMetrics(ape.addr, user, encoded)
	require.NoError(t, err)
	defer func() { _ = rejected.Body.Close() }()
	require.Equal(t, http.StatusUnauthorized, rejected.StatusCode)

	logged := ape.out.String()
	require.NotContains(t, logged, pass)
	require.NotContains(t, logged, encoded)
}

type startupCase struct {
	name   string
	agent  string
	want   []string // must appear in the exporter output
	reject []string // must not appear
}

func TestStartupFailures(t *testing.T) {
	cases := []startupCase{
		{
			name:  "invalid key passphrase",
			agent: agentTLS(pemRef("cert_file"), pemRef("key_file"), pemRef("root_ca"), textRef("key_file_passphrase_invalid")),
			want:  []string{"Failed to load server certificate and key"},
		},
		{
			name:  "unknown Secret Agent alias",
			agent: agentTLS(pemPrefix()+"NoSuchAlias:cert_file", pemRef("key_file_plain"), "", ""),
			want:  []string{"failed to resolve Agent.cert_file", "resource not found"},
		},
		{
			name:  "missing key in the secret",
			agent: agentTLS(pemRef("no_such_key"), pemRef("key_file_plain"), "", ""),
			want:  []string{"failed to resolve Agent.cert_file"},
		},
		{
			// The CA is base64 in the secret, and secrets: reads it as text.
			// It used to be skipped silently and the system roots used instead.
			name:  "base64 CA read as text",
			agent: agentTLS(pemRef("cert_file"), pemRef("key_file_plain"), textRef("root_ca_b64"), ""),
			want:  []string{"Failed to load CA certificate", "no PEM certificate found"},
		},
		{
			name:   "mistyped source prefix is not a password",
			agent:  fmt.Sprintf("basic_auth_username = %q\nbasic_auth_password = %q\n", probeUser, "evn:DB_PASSWORD"),
			want:   []string{"Failed to get http basic auth password", "unknown source"},
			reject: []string{"DB_PASSWORD"},
		},
	}

	if backend == "aws" {
		cases = append(cases, startupCase{
			name:  "missing AWS secret",
			agent: agentTLS(pemPrefix()+"APEMissingSecret:cert_file", pemRef("key_file_plain"), "", ""),
			want:  []string{"failed to resolve Agent.cert_file"},
		})
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ape := startAPE(t, tc.agent)
			out := ape.waitExit(t, 60*time.Second)
			for _, want := range tc.want {
				require.Contains(t, out, want)
			}
			for _, reject := range tc.reject {
				require.NotContains(t, out, reject)
			}
		})
	}
}

// A secret is used exactly as Secret Agent returns it. A trailing space or newline is part of the
// value, so the stored value is accepted and the same value with its tail removed is not.
func TestSecretValueIsUsedAsStored(t *testing.T) {
	cases := []struct {
		name    string
		key     string
		value   string
		trimmed string
	}{
		{"trailing space", "pw_trailing_space", pwTrailingSpace, strings.TrimRight(pwTrailingSpace, " \n")},
		{"trailing newline", "pw_trailing_newline", pwTrailingNewline, strings.TrimRight(pwTrailingNewline, " \n")},
		{"contains colons", "pw_with_colon", pwWithColon, "ape"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ape := startAPE(t, basicAuth(probeUser, textRef(tc.key)))
			ape.waitListening(t)

			requireStatus(t, ape.addr, probeUser, tc.value, http.StatusOK)
			requireStatus(t, ape.addr, probeUser, tc.trimmed, http.StatusUnauthorized)
		})
	}
}

// A value with ':' that does not start with a source name is literal text, with no Secret Agent involved.
func TestLiteralValueWithColonIsKept(t *testing.T) {
	const literal = "Bearer ape:9f3c"

	ape := startAPE(t, basicAuth(probeUser, literal))
	ape.waitListening(t)

	requireStatus(t, ape.addr, probeUser, literal, http.StatusOK)
	requireStatus(t, ape.addr, probeUser, "Bearer ape", http.StatusUnauthorized)
}

// secrets:b64: accepts base64 wrapped at 76 columns, the way GNU base64 writes it.
func TestWrappedBase64Certificate(t *testing.T) {
	ape := startAPE(t, agentTLS("secrets:b64:"+resource+":cert_file_b64_wrapped", pemRef("key_file_plain"), "", ""))
	ape.waitListening(t)

	requireMetricsOK(t, ape.addr, caPool(t, "ca.crt"), nil)
}

// The same CA, stored as base64 on one line, is accepted when it is read with secrets:b64:.
func TestBase64CAReadAsBase64(t *testing.T) {
	ape := startAPE(t, agentTLS(pemRef("cert_file"), pemRef("key_file_plain"), "secrets:b64:"+resource+":root_ca_b64", ""))
	ape.waitListening(t)

	trusted := caPool(t, "ca.crt")
	requireMetricsOK(t, ape.addr, trusted, clientCert(t, "client"))

	_, err := getMetrics(ape.addr, trusted, nil)
	require.Error(t, err, "the CA was loaded, so a client without a certificate is rejected")
}

// Startup waits for a Secret Agent that is not listening yet, and gives up after a bounded time.
func TestStartupWaitsForSecretAgent(t *testing.T) {
	if transport == "uds" {
		t.Skip("the delayed agent is a TCP forwarder")
	}

	agent := agentTLS(pemRef("cert_file"), pemRef("key_file_plain"), "", "")

	t.Run("agent appears after a few seconds", func(t *testing.T) {
		port := delayedForwarder(t, saAddress, 4*time.Second)
		ape := startAPEAgainst(t, agent, secretAgentEndpoint("127.0.0.1", port, ""))
		ape.waitListening(t)

		requireMetricsOK(t, ape.addr, caPool(t, "ca.crt"), nil)
		require.Contains(t, ape.out.String(), "not reachable yet, retrying")
	})

	t.Run("agent never appears", func(t *testing.T) {
		port := delayedForwarder(t, saAddress, time.Hour)
		start := time.Now()
		ape := startAPEAgainst(t, agent, secretAgentEndpoint("127.0.0.1", port, ""))
		out := ape.waitExit(t, 90*time.Second)
		elapsed := time.Since(start)

		require.Contains(t, out, "failed to resolve Agent.cert_file")
		require.Contains(t, out, "connection refused")
		require.GreaterOrEqual(t, elapsed, 25*time.Second, "the window is about 30s")
		require.Less(t, elapsed, 60*time.Second)
	})
}

func pemPrefix() string {
	if encoding == "base64" {
		return "secrets:b64:"
	}

	return "secrets:"
}

func pemRef(key string) string {
	return pemPrefix() + resource + ":" + key
}

func textRef(key string) string {
	return "secrets:" + resource + ":" + key
}

func agentTLS(certFile, keyFile, rootCA, passphrase string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "cert_file = %q\nkey_file = %q\n", certFile, keyFile)
	if rootCA != "" {
		fmt.Fprintf(&b, "root_ca = %q\n", rootCA)
	}
	if passphrase != "" {
		fmt.Fprintf(&b, "key_file_passphrase = %q\n", passphrase)
	}

	return b.String()
}

func basicAuth(user, password string) string {
	return fmt.Sprintf("basic_auth_username = %q\nbasic_auth_password = %q\n", user, password)
}

func requireStatus(t *testing.T, addr, user, pass string, want int) {
	t.Helper()

	resp, err := getHTTPMetrics(addr, user, pass)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, resp.Body)
	require.Equal(t, want, resp.StatusCode)
}

// delayedForwarder reserves a local port, starts forwarding it to target after delay, and returns
// the port. Until then a connection to it is refused, like a sidecar that has not started yet.
// TLS passes through untouched, so the exporter still verifies the real agent's certificate.
func delayedForwarder(t *testing.T, target string, delay time.Duration) int {
	t.Helper()

	reserve, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := reserve.Addr().(*net.TCPAddr).Port
	require.NoError(t, reserve.Close())

	stop := make(chan struct{})
	var ln net.Listener
	var lnMu sync.Mutex

	t.Cleanup(func() {
		close(stop)
		lnMu.Lock()
		defer lnMu.Unlock()
		if ln != nil {
			_ = ln.Close()
		}
	})

	go func() {
		select {
		case <-time.After(delay):
		case <-stop:
			return
		}

		listener, listenErr := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if listenErr != nil {
			return
		}

		lnMu.Lock()
		ln = listener
		lnMu.Unlock()

		for {
			client, acceptErr := listener.Accept()
			if acceptErr != nil {
				return
			}

			go forward(client, target)
		}
	}()

	return port
}

func forward(client net.Conn, target string) {
	defer func() { _ = client.Close() }()

	upstream, err := net.DialTimeout("tcp", target, 5*time.Second)
	if err != nil {
		return
	}
	defer func() { _ = upstream.Close() }()

	done := make(chan struct{}, 2)
	go func() { _, _ = io.Copy(upstream, client); done <- struct{}{} }()
	go func() { _, _ = io.Copy(client, upstream); done <- struct{}{} }()
	<-done
}

func secretAgentBlock() string {
	return secretAgentEndpoint("secret-agent", 3005, saSocket)
}

func plainSecretAgentBlock() string {
	return secretAgentEndpoint("secret-agent-plain", 3005, "/run/sa/secret-agent-plain.sock")
}

func secretAgentEndpoint(host string, port int, socketPath string) string {
	switch transport {
	case "uds":
		return fmt.Sprintf("socket_path = %q\ntimeout = 10\n", socketPath)
	case "tls":
		return fmt.Sprintf("address = %q\nport = %d\ntls_name = \"secret-agent\"\nca_file = %q\ntimeout = 10\n",
			host, port, filepath.Join(certDir, "ca.crt"))
	case "mtls":
		return fmt.Sprintf("address = %q\nport = %d\ntls_name = \"secret-agent\"\nca_file = %q\ncert_file = %q\nkey_file = %q\ntimeout = 10\n",
			host, port, filepath.Join(certDir, "ca.crt"), filepath.Join(certDir, "client.crt"), filepath.Join(certDir, "client.key"))
	default:
		return fmt.Sprintf("address = %q\nport = %d\ntimeout = 10\n", host, port)
	}
}

func plainCredentials(t *testing.T) (string, string) {
	t.Helper()

	if backend == "file" {
		var doc struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		blob, err := os.ReadFile("/work/sa/plain.json")
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(blob, &doc))
		require.NotEmpty(t, doc.Username)
		require.NotEmpty(t, doc.Password)

		return doc.Username, doc.Password
	}

	user := os.Getenv("APE_IT_PLAIN_USERNAME")
	pass := os.Getenv("APE_IT_PLAIN_PASSWORD")
	require.NotEmpty(t, user, "APE_IT_PLAIN_USERNAME is required with APE_IT_AWS_PLAIN_SECRET_ARN")
	require.NotEmpty(t, pass, "APE_IT_PLAIN_PASSWORD is required with APE_IT_AWS_PLAIN_SECRET_ARN")

	return user, pass
}

type apeProcess struct {
	addr string
	cmd  *exec.Cmd
	out  *syncBuffer
	done chan struct{}
}

func startAPE(t *testing.T, agent string) *apeProcess {
	t.Helper()

	return startAPEAgainst(t, agent, secretAgentBlock())
}

func startAPEAgainst(t *testing.T, agent, secretAgent string) *apeProcess {
	t.Helper()

	addr := freeAddr(t)
	config := fmt.Sprintf(`[Agent]
enable_prometheus = true
enable_open_telemetry = false
bind = %q
timeout = 10
log_level = "info"
%s
[Agent.SecretAgent]
%s
[Aerospike]
db_host = "127.0.0.1"
db_port = 3000
timeout = 1
`, addr, agent, secretAgent)

	path := filepath.Join(t.TempDir(), "ape.toml")
	require.NoError(t, os.WriteFile(path, []byte(config), 0o600))

	ape := &apeProcess{addr: addr, out: &syncBuffer{}, done: make(chan struct{})}
	ape.cmd = exec.Command(apeBin, "--config", path, "--gauge-list", gaugeList)
	ape.cmd.Stdout = ape.out
	ape.cmd.Stderr = ape.out
	require.NoError(t, ape.cmd.Start())

	go func() {
		_ = ape.cmd.Wait()
		close(ape.done)
	}()

	t.Cleanup(func() {
		ape.stop()
		out := ape.out.String()
		if leaked := leakedSecrets(t, out); len(leaked) > 0 {
			t.Errorf("exporter output contains secret material: %s", strings.Join(leaked, ", "))
		}
		if t.Failed() {
			t.Logf("exporter output (redacted):\n%s", redact(t, out))
		}
	})

	return ape
}

func (a *apeProcess) waitListening(t *testing.T) {
	t.Helper()

	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-a.done:
			t.Fatalf("exporter exited before listening (exit %d)", a.cmd.ProcessState.ExitCode())
		default:
		}

		conn, err := net.DialTimeout("tcp", a.addr, time.Second)
		if err == nil {
			_ = conn.Close()
			return
		}

		time.Sleep(250 * time.Millisecond)
	}

	t.Fatalf("exporter did not listen on %s", a.addr)
}

func (a *apeProcess) waitExit(t *testing.T, timeout time.Duration) string {
	t.Helper()

	select {
	case <-a.done:
	case <-time.After(timeout):
		t.Fatalf("exporter kept running; expected a startup failure")
	}

	require.NotZero(t, a.cmd.ProcessState.ExitCode(), "exporter should exit with a failure status")

	return a.out.String()
}

func (a *apeProcess) stop() {
	select {
	case <-a.done:
		return
	default:
	}

	_ = a.cmd.Process.Signal(syscall.SIGTERM)
	select {
	case <-a.done:
	case <-time.After(15 * time.Second):
		_ = a.cmd.Process.Kill()
		<-a.done
	}
}

func requireMetricsOK(t *testing.T, addr string, roots *x509.CertPool, certs []tls.Certificate) {
	t.Helper()

	resp, err := getMetrics(addr, roots, certs)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, resp.Body)
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func getHTTPMetrics(addr, user, pass string) (*http.Response, error) {
	request, err := http.NewRequest(http.MethodGet, "http://"+addr+"/metrics", nil)
	if err != nil {
		return nil, err
	}

	request.SetBasicAuth(user, pass)
	client := &http.Client{Timeout: 30 * time.Second}

	return client.Do(request)
}

func getMetrics(addr string, roots *x509.CertPool, certs []tls.Certificate) (*http.Response, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:      roots,
				Certificates: certs,
				ServerName:   "localhost",
				MinVersion:   tls.VersionTLS12,
			},
		},
	}

	resp, err := client.Get("https://" + addr + "/metrics")
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func caPool(t *testing.T, name string) *x509.CertPool {
	t.Helper()

	pool := x509.NewCertPool()
	require.True(t, pool.AppendCertsFromPEM(readCert(t, name)))

	return pool
}

func clientCert(t *testing.T, name string) []tls.Certificate {
	t.Helper()

	pair, err := tls.X509KeyPair(readCert(t, name+".crt"), readCert(t, name+".key"))
	require.NoError(t, err)

	return []tls.Certificate{pair}
}

func readCert(t *testing.T, name string) []byte {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(certDir, name))
	require.NoError(t, err)

	return data
}

// sensitiveValues returns fragments that must never appear in exporter output:
// both passphrases, and pieces of every private key in PEM and base64 form.
func sensitiveValues(t *testing.T) map[string]string {
	t.Helper()

	values := map[string]string{}
	for _, name := range []string{"server.pass", "server.wrong-pass"} {
		values[name] = strings.TrimSpace(string(readCert(t, name)))
	}

	values["probe password"] = strings.TrimSpace(pwTrailingSpace)
	values["probe password with colons"] = pwWithColon

	for _, name := range []string{"server.key", "server-enc.key", "client.key", "rogue-client.key"} {
		raw := readCert(t, name)
		block, _ := pem.Decode(raw)
		require.NotNil(t, block, name)
		body := base64.StdEncoding.EncodeToString(block.Bytes)
		values[name+" body"] = body[64:128]

		encoded := base64.StdEncoding.EncodeToString(raw)
		values[name+" base64"] = encoded[len(encoded)/2 : len(encoded)/2+48]
	}

	return values
}

func leakedSecrets(t *testing.T, out string) []string {
	t.Helper()

	var leaked []string
	for name, value := range sensitiveValues(t) {
		if value != "" && strings.Contains(out, value) {
			leaked = append(leaked, name)
		}
	}

	return leaked
}

func redact(t *testing.T, out string) string {
	t.Helper()

	for _, value := range sensitiveValues(t) {
		if value != "" {
			out = strings.ReplaceAll(out, value, "[REDACTED]")
		}
	}

	return out
}

func waitForPlainAgent(timeout time.Duration) error {
	if transport == "uds" {
		return waitForAgent("unix", "/run/sa/secret-agent-plain.sock", timeout)
	}

	return waitForAgent("tcp", "secret-agent-plain:3005", timeout)
}

func waitForSecretAgent(timeout time.Duration) error {
	network, address := "tcp", saAddress
	if transport == "uds" {
		network, address = "unix", saSocket
	}

	return waitForAgent(network, address, timeout)
}

func waitForAgent(network, address string, timeout time.Duration) error {

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout(network, address, time.Second)
		if err == nil {
			_ = conn.Close()
			return nil
		}

		time.Sleep(500 * time.Millisecond)
	}

	return errors.New("Secret Agent did not become reachable at " + address)
}

func freeAddr(t *testing.T) string {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := ln.Addr().String()
	require.NoError(t, ln.Close())

	return addr
}

func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}

	return fallback
}

type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buf.String()
}
