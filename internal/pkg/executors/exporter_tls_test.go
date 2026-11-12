package executors

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestExporterTLSAndMutualTLS(t *testing.T) {
	original := config.Cfg
	t.Cleanup(func() { config.Cfg = original })

	material := newExporterMaterial(t)
	config.Cfg = config.Config{}
	config.Cfg.Agent.CertFile = material.serverCert
	config.Cfg.Agent.KeyFile = material.serverKey

	serverTLS := initExporterTLS()
	require.Equal(t, tls.NoClientCert, serverTLS.ClientAuth)
	require.Equal(t, uint16(tls.VersionTLS12), serverTLS.MinVersion)
	require.False(t, serverTLS.InsecureSkipVerify)

	addr := serveTLS(t, serverTLS)
	require.NoError(t, dialExporter(t, addr, material.pool, nil))
	require.Error(t, dialPlain(t, addr))

	config.Cfg.Agent.RootCA = material.caCert
	mutual := initExporterTLS()
	require.Equal(t, tls.RequireAndVerifyClientCert, mutual.ClientAuth)
	require.NotNil(t, mutual.ClientCAs)

	mutualAddr := serveTLS(t, mutual)
	require.Error(t, dialExporter(t, mutualAddr, material.pool, nil))
	require.NoError(t, dialExporter(t, mutualAddr, material.pool, material.client))
	require.Error(t, dialExporter(t, mutualAddr, material.pool, material.rogue))

	encodedCert := base64.StdEncoding.EncodeToString(mustRead(t, material.serverCert))
	encodedKey := base64.StdEncoding.EncodeToString(mustRead(t, material.serverKey))
	encodedCA := base64.StdEncoding.EncodeToString(mustRead(t, material.caCert))
	config.Cfg.Agent.CertFile = "b64:" + encodedCert
	config.Cfg.Agent.KeyFile = "b64:" + encodedKey
	config.Cfg.Agent.RootCA = "b64:" + encodedCA

	fromBase64 := initExporterTLS()
	require.Equal(t, tls.RequireAndVerifyClientCert, fromBase64.ClientAuth)
	base64Addr := serveTLS(t, fromBase64)
	require.NoError(t, dialExporter(t, base64Addr, material.pool, material.client))
	require.Error(t, dialExporter(t, base64Addr, material.pool, nil))
}

func serveTLS(t *testing.T, cfg *tls.Config) string {
	t.Helper()

	ln, err := tls.Listen("tcp", "127.0.0.1:0", cfg)
	require.NoError(t, err)
	t.Cleanup(func() { _ = ln.Close() })

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				defer func() { _ = conn.Close() }()
				tlsConn, ok := conn.(*tls.Conn)
				if !ok {
					return
				}
				_ = tlsConn.SetDeadline(time.Now().Add(5 * time.Second))
				if err := tlsConn.Handshake(); err != nil {
					return
				}
				_, _ = tlsConn.Write([]byte{1})
			}()
		}
	}()

	return ln.Addr().String()
}

func dialExporter(t *testing.T, addr string, roots *x509.CertPool, certs []tls.Certificate) error {
	t.Helper()

	conn, err := tls.Dial("tcp", addr, &tls.Config{
		RootCAs:      roots,
		Certificates: certs,
		ServerName:   "localhost",
		MinVersion:   tls.VersionTLS12,
	})
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()

	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	if _, err = conn.Write([]byte{0}); err != nil {
		return err
	}
	buf := make([]byte, 1)
	_, err = conn.Read(buf)

	return err
}

func dialPlain(t *testing.T, addr string) error {
	t.Helper()

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()

	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	if _, err = conn.Write([]byte("GET /metrics HTTP/1.1\r\nHost: localhost\r\n\r\n")); err != nil {
		return err
	}

	buf := make([]byte, 64)
	n, err := conn.Read(buf)
	if n >= 5 && string(buf[:5]) == "HTTP/" {
		return nil
	}
	if err != nil {
		return err
	}

	return errors.New("plaintext request was not served over TLS")
}

type exporterMaterial struct {
	serverCert string
	serverKey  string
	caCert     string
	pool       *x509.CertPool
	client     []tls.Certificate
	rogue      []tls.Certificate
}

func newExporterMaterial(t *testing.T) exporterMaterial {
	t.Helper()

	caKey, caCert, caPEM := newCA(t, "exporter-unit-ca")
	otherKey, otherCert, _ := newCA(t, "exporter-unit-other")
	serverCertPEM, serverKeyPEM := newLeaf(t, caCert, caKey, "localhost", x509.ExtKeyUsageServerAuth, []string{"localhost"})
	clientCertPEM, clientKeyPEM := newLeaf(t, caCert, caKey, "exporter-client", x509.ExtKeyUsageClientAuth, nil)
	rogueCertPEM, rogueKeyPEM := newLeaf(t, otherCert, otherKey, "rogue-client", x509.ExtKeyUsageClientAuth, nil)

	dir := t.TempDir()
	serverCert := filepath.Join(dir, "server.crt")
	serverKey := filepath.Join(dir, "server.key")
	caPath := filepath.Join(dir, "ca.crt")
	require.NoError(t, os.WriteFile(serverCert, serverCertPEM, 0o600))
	require.NoError(t, os.WriteFile(serverKey, serverKeyPEM, 0o600))
	require.NoError(t, os.WriteFile(caPath, caPEM, 0o600))

	client, err := tls.X509KeyPair(clientCertPEM, clientKeyPEM)
	require.NoError(t, err)
	rogue, err := tls.X509KeyPair(rogueCertPEM, rogueKeyPEM)
	require.NoError(t, err)

	pool := x509.NewCertPool()
	require.True(t, pool.AppendCertsFromPEM(caPEM))

	return exporterMaterial{
		serverCert: serverCert,
		serverKey:  serverKey,
		caCert:     caPath,
		pool:       pool,
		client:     []tls.Certificate{client},
		rogue:      []tls.Certificate{rogue},
	}
}

func newCA(t *testing.T, name string) (*ecdsa.PrivateKey, *x509.Certificate, []byte) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: name},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)
	cert, err := x509.ParseCertificate(der)
	require.NoError(t, err)

	return key, cert, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

func newLeaf(t *testing.T, parent *x509.Certificate, parentKey *ecdsa.PrivateKey, name string, usage x509.ExtKeyUsage, dns []string) ([]byte, []byte) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	template := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: name},
		DNSNames:     dns,
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{usage},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, parent, &key.PublicKey, parentKey)
	require.NoError(t, err)
	keyDER, err := x509.MarshalECPrivateKey(key)
	require.NoError(t, err)

	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
		pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	return data
}
