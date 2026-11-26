package commons

import (
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadServerCertAndKeyWithoutSecretAgent(t *testing.T) {
	certPEM, keyPEM := generatedPEM(t)
	dir := t.TempDir()
	certPath := filepath.Join(dir, "cert.pem")
	keyPath := filepath.Join(dir, "key.pem")
	require.NoError(t, os.WriteFile(certPath, certPEM, 0o600))
	require.NoError(t, os.WriteFile(keyPath, keyPEM, 0o600))

	encodedCert := base64.StdEncoding.EncodeToString(certPEM)
	encodedKey := base64.StdEncoding.EncodeToString(keyPEM)
	t.Setenv("APE_TEST_CERT", encodedCert)
	t.Setenv("APE_TEST_KEY", encodedKey)

	cases := []struct {
		name string
		cert string
		key  string
	}{
		{name: "path", cert: certPath, key: keyPath},
		{name: "file", cert: "file:" + certPath, key: "file:" + keyPath},
		{name: "env-b64", cert: "env-b64:APE_TEST_CERT", key: "env-b64:APE_TEST_KEY"},
		{name: "b64", cert: "b64:" + encodedCert, key: "b64:" + encodedKey},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			certs, err := LoadServerCertAndKey(tc.cert, tc.key, "")
			require.NoError(t, err)
			require.Len(t, certs, 1)
			require.NotEmpty(t, certs[0].Certificate)
		})
	}
}

func TestLoadCACertWithoutSecretAgent(t *testing.T) {
	certPEM, _ := generatedPEM(t)
	path := filepath.Join(t.TempDir(), "ca.pem")
	require.NoError(t, os.WriteFile(path, certPEM, 0o600))
	encoded := base64.StdEncoding.EncodeToString(certPEM)
	t.Setenv("APE_TEST_CA", encoded)

	cases := []struct {
		name   string
		source string
	}{
		{name: "path", source: path},
		{name: "file", source: "file:" + path},
		{name: "env-b64", source: "env-b64:APE_TEST_CA"},
		{name: "b64", source: "b64:" + encoded},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pool, err := LoadCACert(tc.source)
			require.NoError(t, err)
			require.NotNil(t, pool)
		})
	}
}

func TestLoadEncryptedKeyWithoutSecretAgent(t *testing.T) {
	certPEM, keyPEM := generatedPEM(t)
	const passphrase = "unit-pass-9f3c"
	encrypted := encryptPEM(t, keyPEM, passphrase)

	dir := t.TempDir()
	certPath := filepath.Join(dir, "cert.pem")
	keyPath := filepath.Join(dir, "key.pem")
	passPath := filepath.Join(dir, "pass")
	require.NoError(t, os.WriteFile(certPath, certPEM, 0o600))
	require.NoError(t, os.WriteFile(keyPath, encrypted, 0o600))
	require.NoError(t, os.WriteFile(passPath, []byte(passphrase), 0o600))

	encodedPass := base64.StdEncoding.EncodeToString([]byte(passphrase))
	t.Setenv("APE_TEST_PASS", passphrase)
	t.Setenv("APE_TEST_PASS_B64", encodedPass)

	cases := []struct {
		name string
		pass string
	}{
		{name: "direct", pass: passphrase},
		{name: "file", pass: "file:" + passPath},
		{name: "env", pass: "env:APE_TEST_PASS"},
		{name: "env-b64", pass: "env-b64:APE_TEST_PASS_B64"},
		{name: "b64", pass: "b64:" + encodedPass},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			certs, err := LoadServerCertAndKey(certPath, keyPath, tc.pass)
			require.NoError(t, err)
			require.Len(t, certs, 1)
		})
	}

	_, err := LoadServerCertAndKey(certPath, keyPath, "wrong-pass")
	require.ErrorContains(t, err, "failed to decrypt PEM Block")
}

func encryptPEM(t *testing.T, keyPEM []byte, passphrase string) []byte {
	t.Helper()

	block, _ := pem.Decode(keyPEM)
	require.NotNil(t, block)
	encrypted, err := x509.EncryptPEMBlock(rand.Reader, block.Type, block.Bytes, []byte(passphrase), x509.PEMCipherAES256) // nolint:staticcheck
	require.NoError(t, err)

	return pem.EncodeToMemory(encrypted)
}
