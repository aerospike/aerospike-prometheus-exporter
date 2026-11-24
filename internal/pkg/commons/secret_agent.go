package commons

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/secretagent"
	log "github.com/sirupsen/logrus"
)

// Secret Agent can start after APE (a sidecar, for example), so startup waits for it.
// Only dial and timeout errors are retried, and the window is shared by every secret,
// so an agent that never appears fails startup after startupRetryWindow in total.
// They are variables so tests can shorten them.
var (
	startupRetryWindow   = 30 * time.Second
	startupRetryInterval = time.Second
)

// SecretFetcher retrieves one secret from Secret Agent.
type SecretFetcher interface {
	Fetch(ctx context.Context, resource, key string) ([]byte, error)
}

var (
	secretFetcher SecretFetcher
	secretCache   = map[string][]byte{}
	secretStateMu sync.Mutex

	// retryUntil is set only while ResolveStartupSecrets runs. It is zero afterwards,
	// so a fetch outside startup makes a single attempt.
	retryUntil time.Time
)

// RegisterSecretAgent installs the process-wide Secret Agent client.
// Passing nil clears the client and the resolved-secret cache.
func RegisterSecretAgent(fetcher SecretFetcher) {
	secretStateMu.Lock()
	defer secretStateMu.Unlock()

	secretFetcher = fetcher
	secretCache = map[string][]byte{}
}

// BuildSecretAgentTLS loads CA and client certificate material for a TLS Secret Agent connection.
// TCP and UDS return a nil config. Verification is always enabled.
// secretagent.Parse has already rejected secrets: references in these files.
func BuildSecretAgentTLS(cfg secretagent.Config) (*tls.Config, error) {
	if cfg.Mode != secretagent.ModeTLS && cfg.Mode != secretagent.ModeMTLS {
		return nil, nil
	}

	rootCAs, err := LoadCACert(cfg.CAFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load secret agent ca_file: %w", err)
	}

	tlsConfig := &tls.Config{
		RootCAs:            rootCAs,
		ServerName:         cfg.TLSName,
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: false,
	}

	if cfg.Mode == secretagent.ModeMTLS {
		certificates, certErr := LoadServerCertAndKey(cfg.CertFile, cfg.KeyFile, cfg.KeyFilePassphrase)
		if certErr != nil {
			return nil, fmt.Errorf("failed to load secret agent client certificate: %w", certErr)
		}

		tlsConfig.Certificates = certificates
	}

	return tlsConfig, nil
}

// ResolveStartupSecrets fetches every secrets: reference before listeners start.
// Direct text, file, environment, and base64 values stay unresolved until they are used.
// Fields of an executor that is not enabled are skipped, so a reference there neither
// needs Secret Agent nor blocks startup.
// While it runs, unreachable-agent errors are retried for startupRetryWindow in total.
func ResolveStartupSecrets() error {
	secretStateMu.Lock()
	retryUntil = time.Now().Add(startupRetryWindow)
	secretStateMu.Unlock()

	defer func() {
		secretStateMu.Lock()
		retryUntil = time.Time{}
		secretStateMu.Unlock()
	}()

	type field struct {
		name   string
		value  string
		isCert bool
	}

	fields := []field{
		{"Aerospike.user", config.Cfg.Aerospike.User, false},
		{"Aerospike.password", config.Cfg.Aerospike.Password, false},
		{"Aerospike.cert_file", config.Cfg.Aerospike.CertFile, true},
		{"Aerospike.key_file", config.Cfg.Aerospike.KeyFile, true},
		{"Aerospike.root_ca", config.Cfg.Aerospike.RootCA, true},
		{"Aerospike.key_file_passphrase", config.Cfg.Aerospike.KeyFilePassphrase, false},
	}

	if config.Cfg.Agent.PrometheusEnabled {
		fields = append(fields,
			field{"Agent.basic_auth_username", config.Cfg.Agent.BasicAuthUsername, false},
			field{"Agent.basic_auth_password", config.Cfg.Agent.BasicAuthPassword, false},
			field{"Agent.cert_file", config.Cfg.Agent.CertFile, true},
			field{"Agent.key_file", config.Cfg.Agent.KeyFile, true},
			field{"Agent.root_ca", config.Cfg.Agent.RootCA, true},
			field{"Agent.key_file_passphrase", config.Cfg.Agent.KeyFilePassphrase, false},
		)
	}

	if config.Cfg.Agent.OtelEnabled {
		for name, value := range config.Cfg.Agent.Otel.Headers {
			fields = append(fields, field{"Agent.OpenTelemetry.headers." + name, value, false})
		}
	}

	for _, f := range fields {
		if err := resolveSecretField(f.name, f.value, f.isCert); err != nil {
			return err
		}
	}

	return nil
}

func resolveSecretField(name, value string, isCert bool) error {
	value = strings.TrimSpace(value)
	if !secretagent.IsSecretReference(value) {
		return nil
	}

	var err error
	if isCert {
		_, err = getCertificate(value)
	} else {
		_, err = GetSecret(value)
	}

	if err != nil {
		return fmt.Errorf("failed to resolve %s: %w", name, err)
	}

	return nil
}

func fetchConfiguredSecret(reference string) ([]byte, error) {
	reference = strings.TrimSpace(reference)

	secretStateMu.Lock()
	if cached, ok := secretCache[reference]; ok {
		secretStateMu.Unlock()
		return append([]byte(nil), cached...), nil
	}

	fetcher := secretFetcher
	until := retryUntil
	secretStateMu.Unlock()

	if fetcher == nil {
		return nil, errors.New("secret agent is not configured")
	}

	resource, key, isBase64, err := parseSecretReference(reference)
	if err != nil {
		return nil, err
	}

	policy := secretagent.RetryPolicy{
		Until:    until,
		Interval: startupRetryInterval,
		OnRetry: func(err error) {
			log.Warnf("Secret Agent is not reachable yet, retrying: %s", err)
		},
	}

	secret, err := secretagent.FetchWithRetry(context.Background(), fetcher, resource, key, policy)
	if err != nil {
		return nil, err
	}

	if isBase64 {
		secret, err = decodeBase64Secret(secret)
		if err != nil {
			return nil, err
		}
	}

	secretStateMu.Lock()
	secretCache[reference] = append([]byte(nil), secret...)
	secretStateMu.Unlock()

	return secret, nil
}

func parseSecretReference(reference string) (string, string, bool, error) {
	const invalidReference = "invalid secret reference: must be secrets:<resource>:<key> or secrets:b64:<resource>:<key>"

	// secrets:b64:<resource>:<key> is recognized only when a resource and key follow b64.
	// secrets:b64:<key> stays a text reference whose resource is named b64.
	parts := strings.SplitN(reference, ":", 4)
	if len(parts) >= 2 && parts[0] == "secrets" && parts[1] == "b64" {
		if len(parts) == 4 && parts[2] != "" && parts[3] != "" {
			return parts[2], parts[3], true, nil
		}

		if len(parts) == 3 && parts[2] != "" {
			return "b64", parts[2], false, nil
		}

		return "", "", false, errors.New(invalidReference)
	}

	parts = strings.SplitN(reference, ":", 3)
	if len(parts) != 3 || parts[0] != "secrets" || parts[1] == "" || parts[2] == "" {
		return "", "", false, errors.New(invalidReference)
	}

	return parts[1], parts[2], false, nil
}

// decodeBase64Secret decodes one standard-base64 SecretValue.
// ASCII whitespace is not part of the base64 alphabet, so it is removed before decoding:
// wrapped output such as GNU base64 produces, or a stray space or tab, still decodes,
// and the decoded bytes are unchanged.
// The error never includes the secret bytes.
func decodeBase64Secret(secret []byte) ([]byte, error) {
	compact := strings.Map(func(r rune) rune {
		switch r {
		case ' ', '\t', '\n', '\r', '\f', '\v':
			return -1
		}

		return r
	}, string(secret))

	decoded, err := base64.StdEncoding.DecodeString(compact)
	if err != nil {
		return nil, errors.New("secret agent returned a secret that is not valid base64")
	}

	if len(decoded) == 0 {
		return nil, errors.New("secret agent returned an empty secret")
	}

	return decoded, nil
}
