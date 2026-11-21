package commons

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"io"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/secretagent"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestGetSecretExistingSources(t *testing.T) {
	t.Cleanup(func() { RegisterSecretAgent(nil) })

	require.Equal(t, "plain", string(mustSecret(t, "plain")))

	filePath := filepath.Join(t.TempDir(), "secret.txt")
	require.NoError(t, os.WriteFile(filePath, []byte("from-file\n"), 0o600))
	require.Equal(t, "from-file", string(mustSecret(t, "file:"+filePath)))

	t.Setenv("APE_TEST_SECRET", "from-env")
	require.Equal(t, "from-env", string(mustSecret(t, "env:APE_TEST_SECRET")))

	encoded := base64.StdEncoding.EncodeToString([]byte("from-b64"))
	require.Equal(t, "from-b64", string(mustSecret(t, "b64:"+encoded)))

	t.Setenv("APE_TEST_SECRET_B64", encoded)
	require.Equal(t, "from-b64", string(mustSecret(t, "env-b64:APE_TEST_SECRET_B64")))

	_, err := GetSecret("secrets:APE:DbPassword")
	require.ErrorContains(t, err, "secret agent is not configured")
}

func TestGetSecretAndCertificateFromAgent(t *testing.T) {
	t.Cleanup(func() { RegisterSecretAgent(nil) })

	fetcher := &fakeFetcher{values: map[string]string{
		"APE\x00DbPassword": "super-secret-value",
		"APE\x00Cert":       "-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----\n",
	}}
	RegisterSecretAgent(fetcher)

	got, err := GetSecret(" secrets:APE:DbPassword ")
	require.NoError(t, err)
	require.Equal(t, "super-secret-value", string(got))

	got, err = GetSecret("secrets:APE:DbPassword")
	require.NoError(t, err)
	require.Equal(t, "super-secret-value", string(got))
	require.Equal(t, 1, fetcher.calls)

	cert, err := getCertificate("secrets:APE:Cert")
	require.NoError(t, err)
	require.Contains(t, string(cert), "BEGIN CERTIFICATE")

	_, err = GetSecret("secrets:onlyone")
	require.ErrorContains(t, err, "invalid secret reference")

	_, err = GetSecret("secrets:APE:Missing")
	require.ErrorContains(t, err, "not found")
	require.NotContains(t, err.Error(), "super-secret-value")
}

func TestSecretReferenceBase64(t *testing.T) {
	t.Cleanup(func() { RegisterSecretAgent(nil) })

	encoded := base64.StdEncoding.EncodeToString([]byte("pem-bytes"))
	fetcher := &fakeFetcher{values: map[string]string{
		"APE\x00Cert":     encoded,
		"APE\x00my:key":   base64.StdEncoding.EncodeToString([]byte("colon-key")),
		"b64\x00password": "plain-pass",
		"APE\x00Bad":      "****",
		"APE\x00Empty":    "",
	}}
	RegisterSecretAgent(fetcher)

	got, err := GetSecret("secrets:APE:Cert")
	require.NoError(t, err)
	require.Equal(t, encoded, string(got))

	got, err = GetSecret("secrets:b64:APE:Cert")
	require.NoError(t, err)
	require.Equal(t, "pem-bytes", string(got))

	cert, err := getCertificate("secrets:b64:APE:Cert")
	require.NoError(t, err)
	require.Equal(t, "pem-bytes", string(cert))
	require.Equal(t, 2, fetcher.calls)

	got, err = GetSecret("secrets:b64:APE:my:key")
	require.NoError(t, err)
	require.Equal(t, "colon-key", string(got))

	got, err = GetSecret("secrets:b64:password")
	require.NoError(t, err)
	require.Equal(t, "plain-pass", string(got))

	_, err = GetSecret("secrets:b64:APE:Bad")
	require.ErrorContains(t, err, "not valid base64")
	require.NotContains(t, err.Error(), "****")

	_, err = GetSecret("secrets:b64:APE:Empty")
	require.ErrorContains(t, err, "empty secret")

	_, err = GetSecret("secrets:b64:")
	require.ErrorContains(t, err, "invalid secret reference")

	_, err = GetSecret("secrets:b64:APE:")
	require.ErrorContains(t, err, "invalid secret reference")
}

func TestSecretReferenceLoadsGeneratedPEM(t *testing.T) {
	t.Cleanup(func() { RegisterSecretAgent(nil) })

	certPEM, keyPEM := generatedPEM(t)
	fetcher := &fakeFetcher{values: map[string]string{
		"Certs\x00client_cert": string(certPEM),
		"Certs\x00client_key":  string(keyPEM),
		"Certs\x00b64_cert":    base64.StdEncoding.EncodeToString(certPEM),
		"Certs\x00b64_key":     base64.StdEncoding.EncodeToString(keyPEM),
	}}
	RegisterSecretAgent(fetcher)

	textCert := mustSecret(t, "secrets:Certs:client_cert")
	textKey := mustSecret(t, "secrets:Certs:client_key")
	_, err := tls.X509KeyPair(textCert, textKey)
	require.NoError(t, err)

	b64Cert := mustSecret(t, "secrets:b64:Certs:b64_cert")
	b64Key := mustSecret(t, "secrets:b64:Certs:b64_key")
	_, err = tls.X509KeyPair(b64Cert, b64Key)
	require.NoError(t, err)
	require.Equal(t, textCert, b64Cert)
	require.Equal(t, textKey, b64Key)
}

func generatedPEM(t *testing.T) (certPEM, keyPEM []byte) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "ape-test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)

	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyDER, err := x509.MarshalECPrivateKey(key)
	require.NoError(t, err)
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	return certPEM, keyPEM
}

func TestResolveStartupSecrets(t *testing.T) {
	original := config.Cfg
	t.Cleanup(func() {
		config.Cfg = original
		RegisterSecretAgent(nil)
	})

	config.Cfg = config.Config{}
	config.Cfg.Agent.PrometheusEnabled = true
	config.Cfg.Agent.OtelEnabled = true
	config.Cfg.Agent.BasicAuthUsername = "secrets:APE:BasicUser"
	config.Cfg.Agent.BasicAuthPassword = "secrets:APE:BasicPassword"
	config.Cfg.Agent.CertFile = "secrets:APE:AgentCert"
	config.Cfg.Agent.KeyFile = "secrets:APE:AgentKey"
	config.Cfg.Agent.RootCA = "secrets:APE:AgentCA"
	config.Cfg.Agent.KeyFilePassphrase = "secrets:APE:AgentPass"
	config.Cfg.Aerospike.User = "secrets:APE:DbUser"
	config.Cfg.Aerospike.Password = "secrets:APE:DbPassword"
	config.Cfg.Aerospike.CertFile = "secrets:APE:DbCert"
	config.Cfg.Aerospike.KeyFile = "secrets:APE:DbKey"
	config.Cfg.Aerospike.RootCA = "secrets:APE:DbCA"
	config.Cfg.Aerospike.KeyFilePassphrase = "file-passphrase"
	config.Cfg.Agent.Otel.Headers = map[string]string{
		"api-key":       "secrets:APE:ApiKey",
		"Authorization": "Bearer literal-token",
	}

	fetcher := &fakeFetcher{values: map[string]string{}}
	for _, key := range []string{
		"BasicUser", "BasicPassword", "AgentCert", "AgentKey", "AgentCA", "AgentPass",
		"DbUser", "DbPassword", "DbCert", "DbKey", "DbCA", "ApiKey",
	} {
		fetcher.values["APE\x00"+key] = "value-" + key
	}
	RegisterSecretAgent(fetcher)

	require.NoError(t, ResolveStartupSecrets())
	require.Equal(t, "secrets:APE:DbPassword", config.Cfg.Aerospike.Password)
	require.Equal(t, 12, fetcher.calls)
	require.NotContains(t, fetcher.keys, "APE\x00file-passphrase")

	_, err := GetSecret("secrets:APE:ApiKey")
	require.NoError(t, err)
	require.Equal(t, 12, fetcher.calls)

	fetcher.fail = true
	config.Cfg.Aerospike.Password = "secrets:APE:Other"
	err = ResolveStartupSecrets()
	require.ErrorContains(t, err, "Aerospike.password")
	require.NotContains(t, err.Error(), "value-DbPassword")
}

func TestGetSecretKeepsUnprefixedColonValuesAsText(t *testing.T) {
	t.Cleanup(func() { RegisterSecretAgent(nil) })

	// The text before the first ':' is not a source name: a space, a digit first, or a symbol.
	for _, value := range []string{"Bearer abc:def", "12:34", "p@ss:word", "a b:c", ":leading"} {
		got, err := GetSecret(value)
		require.NoError(t, err, value)
		require.Equal(t, value, string(got))
	}
}

func TestGetSecretRejectsUnknownSourcePrefix(t *testing.T) {
	t.Cleanup(func() { RegisterSecretAgent(nil) })

	for _, value := range []string{"evn:DB_PASSWORD", "fil:/etc/pw", "env-b64x:NAME", "user:pass", "host:3000", "FILE:/etc/pw", "secret:APE:key", "password:"} {
		got, err := GetSecret(value)
		require.ErrorContains(t, err, "unknown source", value)
		require.Nil(t, got)
		require.NotContains(t, err.Error(), "DB_PASSWORD", "the error must not echo the value")
		require.NotContains(t, err.Error(), "pass")
	}

	// Known sources are untouched.
	t.Setenv("APE_TEST_SECRET", "from-env")
	require.Equal(t, "from-env", string(mustSecret(t, "env:APE_TEST_SECRET")))
	require.Equal(t, "x", string(mustSecret(t, "b64:"+base64.StdEncoding.EncodeToString([]byte("x")))))
}

func TestGetCertificateRejectsUnknownSourcePrefix(t *testing.T) {
	for _, value := range []string{"evn-b64:CA", "fil:/etc/ca.pem", "ca:old.pem"} {
		got, err := getCertificate(value)
		require.ErrorContains(t, err, "unknown source", value)
		require.Nil(t, got)
		require.NotContains(t, err.Error(), "ca.pem", "the error must not echo the value")
	}
}

func TestGetCertificateReadsPathWithColon(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ca:old.pem")
	require.NoError(t, os.WriteFile(path, []byte("pem-bytes"), 0o600))

	got, err := getCertificate(path)
	require.NoError(t, err)
	require.Equal(t, "pem-bytes", string(got))
}

func TestBundledConfigLeavesSecretAgentDisabled(t *testing.T) {
	path := filepath.Join("..", "..", "..", "configs", "ape.toml")
	blob, err := os.ReadFile(path)
	require.NoError(t, err)

	var cfg config.Config
	_, err = toml.Decode(string(blob), &cfg)
	require.NoError(t, err)

	settings, err := secretagent.Parse(cfg.SecretAgentConfigs())
	require.NoError(t, err)
	require.Equal(t, secretagent.ModeDisabled, settings.Mode)
}

type fakeFetcher struct {
	mu     sync.Mutex
	values map[string]string
	keys   []string
	calls  int
	fail   bool

	// failWith is returned by the first failFor calls, before values are consulted.
	failWith error
	failFor  int
}

func (f *fakeFetcher) Fetch(_ context.Context, resource, key string) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.calls++
	id := resource + "\x00" + key
	f.keys = append(f.keys, id)
	if f.failWith != nil && f.calls <= f.failFor {
		return nil, f.failWith
	}

	if f.fail {
		return nil, errLookup
	}

	value, ok := f.values[id]
	if !ok {
		return nil, errLookup
	}

	return []byte(value), nil
}

var errLookup = lookupError("not found")

type lookupError string

func (e lookupError) Error() string { return string(e) }

func mustSecret(t *testing.T, value string) []byte {
	t.Helper()

	got, err := GetSecret(value)
	require.NoError(t, err)

	return got
}

func TestDecodeBase64SecretIgnoresWhitespace(t *testing.T) {
	raw := []byte(strings.Repeat("pem-bytes-0123456789-", 12))
	encoded := base64.StdEncoding.EncodeToString(raw)

	var wrapped strings.Builder
	for i := 0; i < len(encoded); i += 76 {
		end := min(i+76, len(encoded))
		wrapped.WriteString(encoded[i:end] + "\n")
	}

	for name, input := range map[string]string{
		"plain":        encoded,
		"lf wrapped":   wrapped.String(),
		"crlf wrapped": strings.ReplaceAll(wrapped.String(), "\n", "\r\n"),
		"space":        encoded[:10] + " " + encoded[10:],
		"tab":          encoded[:10] + "\t" + encoded[10:],
		"surrounding":  "  " + encoded + " \t\n",
	} {
		got, err := decodeBase64Secret([]byte(input))
		require.NoError(t, err, name)
		require.Equal(t, raw, got, name)
	}

	for name, input := range map[string]string{
		"bad alphabet": "****",
		"only spaces":  " \n\t ",
		"empty":        "",
	} {
		_, err := decodeBase64Secret([]byte(input))
		require.Error(t, err, name)
		require.NotContains(t, err.Error(), input+"x")
	}
}

func TestSecretAgentValueIsNotTrimmed(t *testing.T) {
	t.Cleanup(func() { RegisterSecretAgent(nil) })

	fetcher := &fakeFetcher{values: map[string]string{
		"APE\x00Spaced":  "secret ",
		"APE\x00Newline": "secret\n",
	}}
	RegisterSecretAgent(fetcher)

	require.Equal(t, "secret ", string(mustSecret(t, "secrets:APE:Spaced")))
	require.Equal(t, "secret\n", string(mustSecret(t, "secrets:APE:Newline")))
}

func TestLoadCACertRejectsNonPEM(t *testing.T) {
	t.Cleanup(func() { RegisterSecretAgent(nil) })

	certPEM, _ := generatedPEM(t)
	encoded := base64.StdEncoding.EncodeToString(certPEM)
	fetcher := &fakeFetcher{values: map[string]string{
		"APE\x00CA": encoded, // stored as base64
	}}
	RegisterSecretAgent(fetcher)

	// Read as text, the base64 is not PEM. It used to be ignored and the system roots used instead.
	pool, err := LoadCACert("secrets:APE:CA")
	require.ErrorContains(t, err, "no PEM certificate found")
	require.Nil(t, pool)
	require.NotContains(t, err.Error(), encoded)

	pool, err = LoadCACert("secrets:b64:APE:CA")
	require.NoError(t, err)
	require.NotNil(t, pool)

	path := filepath.Join(t.TempDir(), "ca.pem")
	require.NoError(t, os.WriteFile(path, []byte("not a certificate"), 0o600))
	_, err = LoadCACert(path)
	require.ErrorContains(t, err, "no PEM certificate found")

	// No CA configured still returns the system pool.
	pool, err = LoadCACert("")
	require.NoError(t, err)
	require.NotNil(t, pool)
}

func TestResolveStartupSecretsSkipsDisabledExecutors(t *testing.T) {
	original := config.Cfg
	t.Cleanup(func() {
		config.Cfg = original
		RegisterSecretAgent(nil)
	})

	fetcher := &fakeFetcher{values: map[string]string{}}
	for _, key := range []string{"DbUser", "BasicUser", "AgentCert", "ApiKey"} {
		fetcher.values["APE\x00"+key] = "value-" + key
	}

	set := func(prom, otel bool) {
		config.Cfg = config.Config{}
		config.Cfg.Agent.PrometheusEnabled = prom
		config.Cfg.Agent.OtelEnabled = otel
		config.Cfg.Aerospike.User = "secrets:APE:DbUser"
		config.Cfg.Agent.BasicAuthUsername = "secrets:APE:BasicUser"
		config.Cfg.Agent.CertFile = "secrets:APE:AgentCert"
		config.Cfg.Agent.Otel.Headers = map[string]string{"api-key": "secrets:APE:ApiKey"}
		fetcher.calls, fetcher.keys = 0, nil
		RegisterSecretAgent(fetcher)
	}

	set(true, false)
	require.NoError(t, ResolveStartupSecrets())
	require.ElementsMatch(t, []string{"APE\x00DbUser", "APE\x00BasicUser", "APE\x00AgentCert"}, fetcher.keys)

	set(false, true)
	require.NoError(t, ResolveStartupSecrets())
	require.ElementsMatch(t, []string{"APE\x00DbUser", "APE\x00ApiKey"}, fetcher.keys)

	// A reference in a disabled section does not need Secret Agent at all.
	set(false, false)
	RegisterSecretAgent(nil)
	config.Cfg.Aerospike.User = "plain-user"
	require.NoError(t, ResolveStartupSecrets())
}

var errRefused = &net.OpError{Op: "dial", Net: "tcp", Err: &os.SyscallError{Syscall: "connect", Err: syscall.ECONNREFUSED}}

func fastRetry(t *testing.T, window time.Duration) {
	t.Helper()

	oldWindow, oldInterval := startupRetryWindow, startupRetryInterval
	startupRetryWindow, startupRetryInterval = window, 5*time.Millisecond

	// The retry warnings are expected here and would flood the test output.
	oldOut := log.StandardLogger().Out
	log.SetOutput(io.Discard)

	t.Cleanup(func() {
		startupRetryWindow, startupRetryInterval = oldWindow, oldInterval
		log.SetOutput(oldOut)
	})
}

func startupConfig(t *testing.T, fields map[string]string) {
	t.Helper()

	original := config.Cfg
	t.Cleanup(func() {
		config.Cfg = original
		RegisterSecretAgent(nil)
	})

	config.Cfg = config.Config{}
	config.Cfg.Aerospike.User = fields["user"]
	config.Cfg.Aerospike.Password = fields["password"]
}

func TestResolveStartupSecretsWaitsForAgent(t *testing.T) {
	fastRetry(t, 2*time.Second)
	startupConfig(t, map[string]string{"user": "secrets:APE:DbUser", "password": "secrets:APE:DbPassword"})

	fetcher := &fakeFetcher{
		values:   map[string]string{"APE\x00DbUser": "u", "APE\x00DbPassword": "p"},
		failWith: errRefused,
		failFor:  4,
	}
	RegisterSecretAgent(fetcher)

	require.NoError(t, ResolveStartupSecrets())
	require.Equal(t, "u", string(mustSecret(t, "secrets:APE:DbUser")))
	require.Equal(t, 6, fetcher.calls, "4 refused, then user and password")
}

func TestResolveStartupSecretsRetryWindowIsShared(t *testing.T) {
	fastRetry(t, 150*time.Millisecond)
	startupConfig(t, map[string]string{"user": "secrets:APE:DbUser", "password": "secrets:APE:DbPassword"})

	fetcher := &fakeFetcher{values: map[string]string{}, failWith: errRefused, failFor: 1 << 30}
	RegisterSecretAgent(fetcher)

	start := time.Now()
	err := ResolveStartupSecrets()
	require.ErrorContains(t, err, "Aerospike.user")
	require.ErrorIs(t, err, syscall.ECONNREFUSED)
	require.Less(t, time.Since(start), time.Second, "the first secret uses the window and the run stops there")

	callsAfterFirst := fetcher.calls
	require.Greater(t, callsAfterFirst, 2)
}

func TestResolveStartupSecretsDoesNotRetryOtherErrors(t *testing.T) {
	fastRetry(t, 5*time.Second)
	startupConfig(t, map[string]string{"user": "secrets:APE:Missing"})

	fetcher := &fakeFetcher{values: map[string]string{}} // answers with "not found"
	RegisterSecretAgent(fetcher)

	start := time.Now()
	err := ResolveStartupSecrets()
	require.ErrorContains(t, err, "not found")
	require.Equal(t, 1, fetcher.calls)
	require.Less(t, time.Since(start), time.Second)

	// Bad base64 is not a connection problem either.
	fetcher = &fakeFetcher{values: map[string]string{"APE\x00Bad": "****"}}
	RegisterSecretAgent(fetcher)
	config.Cfg.Aerospike.User = "secrets:b64:APE:Bad"
	err = ResolveStartupSecrets()
	require.ErrorContains(t, err, "not valid base64")
	require.Equal(t, 1, fetcher.calls)
}

func TestFetchOutsideStartupMakesOneAttempt(t *testing.T) {
	fastRetry(t, 5*time.Second)
	t.Cleanup(func() { RegisterSecretAgent(nil) })

	fetcher := &fakeFetcher{values: map[string]string{"APE\x00Late": "v"}, failWith: errRefused, failFor: 1 << 30}
	RegisterSecretAgent(fetcher)

	_, err := GetSecret("secrets:APE:Late")
	require.ErrorIs(t, err, syscall.ECONNREFUSED)
	require.Equal(t, 1, fetcher.calls)
}
