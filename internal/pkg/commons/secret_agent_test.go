package commons

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/secretagent"
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

func TestResolveStartupSecrets(t *testing.T) {
	original := config.Cfg
	t.Cleanup(func() {
		config.Cfg = original
		RegisterSecretAgent(nil)
	})

	config.Cfg = config.Config{}
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
}

func (f *fakeFetcher) Fetch(_ context.Context, resource, key string) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.calls++
	id := resource + "\x00" + key
	f.keys = append(f.keys, id)
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
