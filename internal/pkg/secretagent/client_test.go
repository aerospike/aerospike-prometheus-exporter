package secretagent

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/binary"
	"encoding/json"
	"encoding/pem"
	"io"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFetchOverTCPAndUDS(t *testing.T) {
	secret := "tcp-secret"
	tcp := startClearListener(t, "tcp", "127.0.0.1:0", func(resource, key string) (string, string) {
		require.Equal(t, "APE", resource)
		require.Equal(t, "DbPassword", key)
		return secret, ""
	})

	client := newTestClient(t, RawConfig{Address: "127.0.0.1", Port: portOf(t, tcp)}, nil)
	got, err := client.Fetch(context.Background(), "APE", "DbPassword")
	require.NoError(t, err)
	require.Equal(t, secret, string(got))
	require.NoError(t, client.Close())

	socketPath := shortSocketPath(t)
	startClearListener(t, "unix", socketPath, func(resource, key string) (string, string) {
		require.Equal(t, "APE", resource)
		require.Equal(t, "Token", key)
		return "uds-secret", ""
	})

	udsClient := newTestClient(t, RawConfig{SocketPath: socketPath}, nil)
	got, err = udsClient.Fetch(context.Background(), "APE", "Token")
	require.NoError(t, err)
	require.Equal(t, "uds-secret", string(got))
}

func TestFetchPlaintextSecret(t *testing.T) {
	ln := startClearListener(t, "tcp", "127.0.0.1:0", func(string, string) (string, string) {
		return "test", ""
	})

	client := newTestClient(t, RawConfig{Address: "127.0.0.1", Port: portOf(t, ln)}, nil)
	got, err := client.Fetch(context.Background(), "APE", "DbUser")
	require.NoError(t, err)
	require.Equal(t, "test", string(got))
}

func TestFetchTLSAndMTLS(t *testing.T) {
	material := newCertMaterial(t)

	startTLSListener(t, material.serverTLS(tls.NoClientCert), func(string, string) (string, string) {
		return "tls-secret", ""
	}, func(port int) {
		client := newTestClient(t, RawConfig{
			Address: "127.0.0.1",
			Port:    uint16(port),
			TLSName: "secretagent",
			CAFile:  material.caPath,
		}, testTLSConfig(t, material, false))
		got, err := client.Fetch(context.Background(), "APE", "Password")
		require.NoError(t, err)
		require.Equal(t, "tls-secret", string(got))
	})

	startTLSListener(t, material.serverTLS(tls.RequireAndVerifyClientCert), func(string, string) (string, string) {
		return "mtls-secret", ""
	}, func(port int) {
		client := newTestClient(t, RawConfig{
			Address:  "127.0.0.1",
			Port:     uint16(port),
			TLSName:  "secretagent",
			CAFile:   material.caPath,
			CertFile: material.clientCertPath,
			KeyFile:  material.clientKeyPath,
		}, testTLSConfig(t, material, true))
		got, err := client.Fetch(context.Background(), "APE", "Password")
		require.NoError(t, err)
		require.Equal(t, "mtls-secret", string(got))
	})
}

func TestFetchFailures(t *testing.T) {
	t.Run("agent error", func(t *testing.T) {
		ln := startClearListener(t, "tcp", "127.0.0.1:0", func(string, string) (string, string) {
			return "", "resource not found"
		})
		client := newTestClient(t, RawConfig{Address: "127.0.0.1", Port: portOf(t, ln)}, nil)
		_, err := client.Fetch(context.Background(), "APE", "Missing")
		require.ErrorContains(t, err, "resource not found")
		require.NotContains(t, err.Error(), "super-secret-value")
	})

	t.Run("connection refused", func(t *testing.T) {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		addr := ln.Addr().String()
		require.NoError(t, ln.Close())

		host, portText, err := net.SplitHostPort(addr)
		require.NoError(t, err)
		parsed, err := strconv.Atoi(portText)
		require.NoError(t, err)
		port := uint16(parsed)

		client := newTestClient(t, RawConfig{Address: host, Port: port, TimeoutSeconds: 1}, nil)
		_, err = client.Fetch(context.Background(), "APE", "DbPassword")
		require.Error(t, err)
		require.NotContains(t, err.Error(), "DbPasswordValue")
	})

	t.Run("missing socket", func(t *testing.T) {
		client := newTestClient(t, RawConfig{SocketPath: filepath.Join(shortDir(t), "missing.sock"), TimeoutSeconds: 1}, nil)
		_, err := client.Fetch(context.Background(), "APE", "DbPassword")
		require.Error(t, err)
	})

	t.Run("socket permissions", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("root bypasses socket permissions")
		}

		socketPath := shortSocketPath(t)
		ln := startClearListener(t, "unix", socketPath, func(string, string) (string, string) {
			return "hidden", ""
		})
		require.NoError(t, os.Chmod(socketPath, 0o000))
		t.Cleanup(func() { _ = os.Chmod(socketPath, 0o600) })

		client := newTestClient(t, RawConfig{SocketPath: socketPath, TimeoutSeconds: 1}, nil)
		_, err := client.Fetch(context.Background(), "APE", "DbPassword")
		require.Error(t, err)
		_ = ln
	})
}

func TestTLSVerificationFailures(t *testing.T) {
	material := newCertMaterial(t)
	other := newCertMaterial(t)

	startTLSListener(t, material.serverTLS(tls.NoClientCert), func(string, string) (string, string) {
		return "should-not-return", ""
	}, func(port int) {
		wrongCA := testTLSConfig(t, other, false)
		wrongCA.RootCAs = other.pool
		client, err := New(mustParse(t, RawConfig{
			Address: "127.0.0.1",
			Port:    uint16(port),
			TLSName: "secretagent",
			CAFile:  other.caPath,
		}), wrongCA)
		require.NoError(t, err)

		_, err = client.Fetch(context.Background(), "APE", "Password")
		require.Error(t, err)
		require.NotContains(t, err.Error(), "should-not-return")

		wrongName := testTLSConfig(t, material, false)
		wrongName.ServerName = "other-name"
		client, err = New(mustParse(t, RawConfig{
			Address: "127.0.0.1",
			Port:    uint16(port),
			TLSName: "other-name",
			CAFile:  material.caPath,
		}), wrongName)
		require.NoError(t, err)
		_, err = client.Fetch(context.Background(), "APE", "Password")
		require.Error(t, err)
	})

	startTLSListener(t, material.serverTLS(tls.RequireAndVerifyClientCert), func(string, string) (string, string) {
		return "mtls-secret", ""
	}, func(port int) {
		client := newTestClient(t, RawConfig{
			Address: "127.0.0.1",
			Port:    uint16(port),
			TLSName: "secretagent",
			CAFile:  material.caPath,
		}, testTLSConfig(t, material, false))
		_, err := client.Fetch(context.Background(), "APE", "Password")
		require.Error(t, err)
	})
}

func TestNewRejectsInconsistentTLS(t *testing.T) {
	_, err := New(Config{Mode: ModeDisabled}, nil)
	require.ErrorContains(t, err, "not configured")

	_, err = New(Config{Mode: ModeTCP, Address: "127.0.0.1", Port: 1, Timeout: time.Second}, &tls.Config{})
	require.ErrorContains(t, err, "not used")

	_, err = New(Config{Mode: ModeTLS, Address: "127.0.0.1", Port: 1, Timeout: time.Second}, nil)
	require.ErrorContains(t, err, "tls configuration is required")

	insecure := &tls.Config{InsecureSkipVerify: true, MinVersion: tls.VersionTLS12}
	client, err := New(Config{Mode: ModeTLS, Address: "127.0.0.1", Port: 1, TLSName: "secretagent", Timeout: time.Second}, insecure)
	require.NoError(t, err)
	require.False(t, client.(*agentClient).tlsConfig.InsecureSkipVerify)
	require.True(t, insecure.InsecureSkipVerify)
}

type secretHandler func(resource, key string) (value, agentErr string)

func startClearListener(t *testing.T, network, address string, handler secretHandler) net.Listener {
	t.Helper()

	ln, err := net.Listen(network, address)
	require.NoError(t, err)
	t.Cleanup(func() { _ = ln.Close() })
	go acceptSecrets(ln, handler)

	return ln
}

func startTLSListener(t *testing.T, serverConfig *tls.Config, handler secretHandler, exercise func(port int)) {
	t.Helper()

	ln, err := tls.Listen("tcp", "127.0.0.1:0", serverConfig)
	require.NoError(t, err)
	defer func() { _ = ln.Close() }()

	go acceptSecrets(ln, handler)
	exercise(ln.Addr().(*net.TCPAddr).Port)
}

func acceptSecrets(ln net.Listener, handler secretHandler) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return
		}

		go func(conn net.Conn) {
			defer func() { _ = conn.Close() }()
			_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
			serveSecret(conn, handler)
			buf := make([]byte, 1)
			_, _ = conn.Read(buf)
		}(conn)
	}
}

func serveSecret(conn net.Conn, handler secretHandler) {
	header := make([]byte, 8)
	if _, err := io.ReadFull(conn, header); err != nil {
		return
	}

	length := binary.BigEndian.Uint32(header[4:8])
	if length == 0 || length > maxResponseBody {
		return
	}

	body := make([]byte, length)
	if _, err := io.ReadFull(conn, body); err != nil {
		return
	}

	var incoming request
	if err := json.Unmarshal(body, &incoming); err != nil {
		return
	}

	value, agentErr := handler(incoming.Resource, incoming.SecretKey)
	encoded, err := json.Marshal(response{SecretValue: value, Error: agentErr})
	if err != nil {
		return
	}

	frame := make([]byte, 8+len(encoded))
	binary.BigEndian.PutUint32(frame[:4], magic)
	binary.BigEndian.PutUint32(frame[4:8], uint32(len(encoded)))
	copy(frame[8:], encoded)
	_, _ = conn.Write(frame)
}

func newTestClient(t *testing.T, raw RawConfig, tlsConfig *tls.Config) Client {
	t.Helper()

	if raw.TimeoutSeconds == 0 {
		raw.TimeoutSeconds = 2
	}

	client, err := New(mustParse(t, raw), tlsConfig)
	require.NoError(t, err)

	return client
}

func mustParse(t *testing.T, raw RawConfig) Config {
	t.Helper()

	cfg, err := Parse(raw)
	require.NoError(t, err)

	return cfg
}

func shortDir(t *testing.T) string {
	t.Helper()

	dir, err := os.MkdirTemp("/tmp", "ape")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	return dir
}

func shortSocketPath(t *testing.T) string {
	t.Helper()

	return filepath.Join(shortDir(t), "a.sock")
}

func portOf(t *testing.T, ln net.Listener) uint16 {
	t.Helper()

	return uint16(ln.Addr().(*net.TCPAddr).Port)
}

type certMaterial struct {
	caPath         string
	clientCertPath string
	clientKeyPath  string
	serverCert     tls.Certificate
	pool           *x509.CertPool
	clientCert     tls.Certificate
}

func (m certMaterial) serverTLS(auth tls.ClientAuthType) *tls.Config {
	cfg := &tls.Config{
		Certificates: []tls.Certificate{m.serverCert},
		MinVersion:   tls.VersionTLS12,
		ClientAuth:   auth,
	}

	if auth != tls.NoClientCert {
		cfg.ClientCAs = m.pool
	}

	return cfg
}

func testTLSConfig(t *testing.T, material certMaterial, withClientCert bool) *tls.Config {
	t.Helper()

	cfg := &tls.Config{
		RootCAs:    material.pool,
		ServerName: "secretagent",
		MinVersion: tls.VersionTLS12,
	}

	if withClientCert {
		cfg.Certificates = []tls.Certificate{material.clientCert}
	}

	return cfg
}

func newCertMaterial(t *testing.T) certMaterial {
	t.Helper()

	caKey := newKey(t)
	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "ape-test-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}
	caDER := createCert(t, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	caCert := parseCert(t, caDER)

	serverKey := newKey(t)
	serverTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "secretagent"},
		DNSNames:     []string{"secretagent"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	serverDER := createCert(t, serverTemplate, caCert, &serverKey.PublicKey, caKey)

	clientKey := newKey(t)
	clientTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(3),
		Subject:      pkix.Name{CommonName: "ape-client"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	clientDER := createCert(t, clientTemplate, caCert, &clientKey.PublicKey, caKey)

	dir := t.TempDir()
	caPath := filepath.Join(dir, "ca.pem")
	clientCertPath := filepath.Join(dir, "client.pem")
	clientKeyPath := filepath.Join(dir, "client.key")
	require.NoError(t, os.WriteFile(caPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER}), 0o600))
	require.NoError(t, os.WriteFile(clientCertPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: clientDER}), 0o600))
	keyDER, err := x509.MarshalECPrivateKey(clientKey)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(clientKeyPath, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}), 0o600))

	serverPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: serverDER})
	serverKeyDER, err := x509.MarshalECPrivateKey(serverKey)
	require.NoError(t, err)
	serverKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: serverKeyDER})
	serverCert, err := tls.X509KeyPair(serverPEM, serverKeyPEM)
	require.NoError(t, err)

	clientPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: clientDER})
	clientKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	clientCert, err := tls.X509KeyPair(clientPEM, clientKeyPEM)
	require.NoError(t, err)

	pool := x509.NewCertPool()
	pool.AddCert(caCert)

	return certMaterial{
		caPath:         caPath,
		clientCertPath: clientCertPath,
		clientKeyPath:  clientKeyPath,
		serverCert:     serverCert,
		pool:           pool,
		clientCert:     clientCert,
	}
}

func newKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	return key
}

func createCert(t *testing.T, template, parent *x509.Certificate, pub *ecdsa.PublicKey, signer *ecdsa.PrivateKey) []byte {
	t.Helper()

	der, err := x509.CreateCertificate(rand.Reader, template, parent, pub, signer)
	require.NoError(t, err)

	return der
}

func parseCert(t *testing.T, der []byte) *x509.Certificate {
	t.Helper()

	cert, err := x509.ParseCertificate(der)
	require.NoError(t, err)

	return cert
}
