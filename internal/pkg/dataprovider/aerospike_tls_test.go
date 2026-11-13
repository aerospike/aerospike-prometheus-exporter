package dataprovider

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

func TestAerospikeTLSAndMutualTLS(t *testing.T) {
	original := config.Cfg
	t.Cleanup(func() { config.Cfg = original })

	material := newAerospikeMaterial(t)
	config.Cfg = config.Config{}
	config.Cfg.Aerospike.RootCA = material.caCert
	config.Cfg.Aerospike.NodeTLSName = "aerospike"

	clientTLS := (&AerospikeServer{}).initAerospikeTLS()
	require.NotNil(t, clientTLS)
	require.False(t, clientTLS.InsecureSkipVerify)
	require.Empty(t, clientTLS.Certificates)
	require.NotNil(t, clientTLS.RootCAs)

	addr := serveAerospikeTLS(t, material.serverConfig(tls.NoClientCert))
	require.NoError(t, dialAerospike(t, addr, clientTLS))
	require.Error(t, dialAerospikePlain(t, addr))

	config.Cfg.Aerospike.CertFile = material.clientCert
	config.Cfg.Aerospike.KeyFile = material.clientKey
	mutual := (&AerospikeServer{}).initAerospikeTLS()
	require.NotEmpty(t, mutual.Certificates)

	mutualAddr := serveAerospikeTLS(t, material.serverConfig(tls.RequireAndVerifyClientCert))
	require.NoError(t, dialAerospike(t, mutualAddr, mutual))
	require.Error(t, dialAerospike(t, mutualAddr, clientTLS))
	require.Error(t, dialAerospike(t, mutualAddr, material.rogueClientTLS(t)))
	require.Error(t, dialAerospikePlain(t, mutualAddr))

	encodedCA := base64.StdEncoding.EncodeToString(mustReadAerospike(t, material.caCert))
	encodedCert := base64.StdEncoding.EncodeToString(mustReadAerospike(t, material.clientCert))
	encodedKey := base64.StdEncoding.EncodeToString(mustReadAerospike(t, material.clientKey))
	config.Cfg.Aerospike.RootCA = "b64:" + encodedCA
	config.Cfg.Aerospike.CertFile = "b64:" + encodedCert
	config.Cfg.Aerospike.KeyFile = "b64:" + encodedKey

	fromBase64 := (&AerospikeServer{}).initAerospikeTLS()
	require.NotEmpty(t, fromBase64.Certificates)
	base64Addr := serveAerospikeTLS(t, material.serverConfig(tls.RequireAndVerifyClientCert))
	require.NoError(t, dialAerospike(t, base64Addr, fromBase64))
	require.Error(t, dialAerospike(t, base64Addr, clientTLS))
	require.Error(t, dialAerospikePlain(t, base64Addr))
}

func serveAerospikeTLS(t *testing.T, cfg *tls.Config) string {
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

func dialAerospike(t *testing.T, addr string, cfg *tls.Config) error {
	t.Helper()

	client := cfg.Clone()
	client.ServerName = config.Cfg.Aerospike.NodeTLSName
	conn, err := tls.Dial("tcp", addr, client)
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

func dialAerospikePlain(t *testing.T, addr string) error {
	t.Helper()

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()

	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	if _, err = conn.Write([]byte("GET / HTTP/1.1\r\nHost: aerospike\r\n\r\n")); err != nil {
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

type aerospikeMaterial struct {
	caCert     string
	caPEM      []byte
	serverCert []byte
	serverKey  []byte
	clientCert string
	clientKey  string
	rogueCert  string
	rogueKey   string
}

func (m aerospikeMaterial) serverConfig(auth tls.ClientAuthType) *tls.Config {
	cert, err := tls.X509KeyPair(m.serverCert, m.serverKey)
	if err != nil {
		panic(err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(m.caPEM) {
		panic("ca")
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientCAs:    pool,
		ClientAuth:   auth,
		MinVersion:   tls.VersionTLS12,
	}
}

func (m aerospikeMaterial) rogueClientTLS(t *testing.T) *tls.Config {
	t.Helper()

	savedCert := config.Cfg.Aerospike.CertFile
	savedKey := config.Cfg.Aerospike.KeyFile
	config.Cfg.Aerospike.CertFile = m.rogueCert
	config.Cfg.Aerospike.KeyFile = m.rogueKey
	cfg := (&AerospikeServer{}).initAerospikeTLS()
	config.Cfg.Aerospike.CertFile = savedCert
	config.Cfg.Aerospike.KeyFile = savedKey
	require.NotEmpty(t, cfg.Certificates)

	return cfg
}

func newAerospikeMaterial(t *testing.T) aerospikeMaterial {
	t.Helper()

	caKey, caCert, caPEM := newAerospikeCA(t, "aerospike-unit-ca")
	otherKey, otherCert, _ := newAerospikeCA(t, "aerospike-unit-other")
	serverCertPEM, serverKeyPEM := newAerospikeLeaf(t, caCert, caKey, "aerospike", x509.ExtKeyUsageServerAuth, []string{"aerospike"})
	clientCertPEM, clientKeyPEM := newAerospikeLeaf(t, caCert, caKey, "exporter-client", x509.ExtKeyUsageClientAuth, nil)
	rogueCertPEM, rogueKeyPEM := newAerospikeLeaf(t, otherCert, otherKey, "rogue-client", x509.ExtKeyUsageClientAuth, nil)

	dir := t.TempDir()
	caPath := filepath.Join(dir, "ca.crt")
	clientCert := filepath.Join(dir, "client.crt")
	clientKey := filepath.Join(dir, "client.key")
	rogueCert := filepath.Join(dir, "rogue.crt")
	rogueKey := filepath.Join(dir, "rogue.key")
	require.NoError(t, os.WriteFile(caPath, caPEM, 0o600))
	require.NoError(t, os.WriteFile(clientCert, clientCertPEM, 0o600))
	require.NoError(t, os.WriteFile(clientKey, clientKeyPEM, 0o600))
	require.NoError(t, os.WriteFile(rogueCert, rogueCertPEM, 0o600))
	require.NoError(t, os.WriteFile(rogueKey, rogueKeyPEM, 0o600))

	return aerospikeMaterial{
		caCert:     caPath,
		caPEM:      caPEM,
		serverCert: serverCertPEM,
		serverKey:  serverKeyPEM,
		clientCert: clientCert,
		clientKey:  clientKey,
		rogueCert:  rogueCert,
		rogueKey:   rogueKey,
	}
}

func newAerospikeCA(t *testing.T, name string) (*ecdsa.PrivateKey, *x509.Certificate, []byte) {
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

func newAerospikeLeaf(t *testing.T, parent *x509.Certificate, parentKey *ecdsa.PrivateKey, name string, usage x509.ExtKeyUsage, dns []string) ([]byte, []byte) {
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

func mustReadAerospike(t *testing.T, path string) []byte {
	t.Helper()

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	return data
}
