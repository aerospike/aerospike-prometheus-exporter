package commons

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/secretagent"
)

// SecretFetcher retrieves one secret from Secret Agent.
type SecretFetcher interface {
	Fetch(ctx context.Context, resource, key string) ([]byte, error)
}

var (
	secretFetcher SecretFetcher
	secretCache   = map[string][]byte{}
	secretStateMu sync.Mutex
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
func BuildSecretAgentTLS(cfg secretagent.Config) (*tls.Config, error) {
	if cfg.Mode != secretagent.ModeTLS && cfg.Mode != secretagent.ModeMTLS {
		return nil, nil
	}

	fields := []struct {
		name  string
		value string
	}{
		{"ca_file", cfg.CAFile},
		{"cert_file", cfg.CertFile},
		{"key_file", cfg.KeyFile},
		{"key_file_passphrase", cfg.KeyFilePassphrase},
	}

	for _, field := range fields {
		if secretagent.IsSecretReference(field.value) {
			return nil, fmt.Errorf("secret agent %s cannot use a secrets: reference", field.name)
		}
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
func ResolveStartupSecrets() error {
	fields := []struct {
		name   string
		value  string
		isCert bool
	}{
		{"Agent.basic_auth_username", config.Cfg.Agent.BasicAuthUsername, false},
		{"Agent.basic_auth_password", config.Cfg.Agent.BasicAuthPassword, false},
		{"Agent.cert_file", config.Cfg.Agent.CertFile, true},
		{"Agent.key_file", config.Cfg.Agent.KeyFile, true},
		{"Agent.root_ca", config.Cfg.Agent.RootCA, true},
		{"Agent.key_file_passphrase", config.Cfg.Agent.KeyFilePassphrase, false},
		{"Aerospike.user", config.Cfg.Aerospike.User, false},
		{"Aerospike.password", config.Cfg.Aerospike.Password, false},
		{"Aerospike.cert_file", config.Cfg.Aerospike.CertFile, true},
		{"Aerospike.key_file", config.Cfg.Aerospike.KeyFile, true},
		{"Aerospike.root_ca", config.Cfg.Aerospike.RootCA, true},
		{"Aerospike.key_file_passphrase", config.Cfg.Aerospike.KeyFilePassphrase, false},
	}

	for _, field := range fields {
		if err := resolveSecretField(field.name, field.value, field.isCert); err != nil {
			return err
		}
	}

	for name, value := range config.Cfg.Agent.Otel.Headers {
		if err := resolveSecretField("Agent.OpenTelemetry.headers."+name, value, false); err != nil {
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
	secretStateMu.Unlock()

	if fetcher == nil {
		return nil, errors.New("secret agent is not configured")
	}

	resource, key, err := parseSecretReference(reference)
	if err != nil {
		return nil, err
	}

	secret, err := fetcher.Fetch(context.Background(), resource, key)
	if err != nil {
		return nil, err
	}

	secretStateMu.Lock()
	secretCache[reference] = append([]byte(nil), secret...)
	secretStateMu.Unlock()

	return secret, nil
}

func parseSecretReference(reference string) (string, string, error) {
	parts := strings.SplitN(reference, ":", 3)
	if len(parts) != 3 || parts[0] != "secrets" || parts[1] == "" || parts[2] == "" {
		return "", "", errors.New("invalid secret reference: must be secrets:<resource>:<key>")
	}

	return parts[1], parts[2], nil
}
