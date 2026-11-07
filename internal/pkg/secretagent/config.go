package secretagent

import (
	"errors"
	"net"
	"strconv"
	"strings"
	"time"
)

const defaultTimeout = 5 * time.Second

// Mode is how APE connects to Secret Agent.
type Mode int

const (
	// ModeDisabled means [Agent.SecretAgent] is not configured.
	ModeDisabled Mode = iota
	// ModeTCP is plaintext TCP.
	ModeTCP
	// ModeTLS is TCP with server certificate verification.
	ModeTLS
	// ModeMTLS is TCP with server verification and a client certificate.
	ModeMTLS
	// ModeUDS is a Unix domain socket.
	ModeUDS
)

func (m Mode) String() string {
	switch m {
	case ModeTCP:
		return "tcp"
	case ModeTLS:
		return "tls"
	case ModeMTLS:
		return "mtls"
	case ModeUDS:
		return "uds"
	default:
		return "disabled"
	}
}

// RawConfig is the [Agent.SecretAgent] block as written in ape.toml.
type RawConfig struct {
	Address           string
	Port              uint16
	SocketPath        string
	TLSName           string
	CAFile            string
	CertFile          string
	KeyFile           string
	KeyFilePassphrase string
	TimeoutSeconds    uint16
	// IsBase64 is ape.toml is_sa_base64. When true, SecretValue is base64.
	// When false, SecretValue is the secret text.
	IsBase64 bool
}

// Config is a validated Secret Agent connection.
type Config struct {
	Mode              Mode
	Address           string
	Port              uint16
	SocketPath        string
	TLSName           string
	CAFile            string
	CertFile          string
	KeyFile           string
	KeyFilePassphrase string
	Timeout           time.Duration
	IsBase64          bool
}

// Endpoint is the dial address. TCP modes use host:port. UDS uses the socket path.
func (c Config) Endpoint() string {
	if c.Mode == ModeUDS {
		return c.SocketPath
	}

	return net.JoinHostPort(c.Address, strconv.Itoa(int(c.Port)))
}

// IsSecretReference reports whether value uses the secrets:<resource>:<key> scheme.
func IsSecretReference(value string) bool {
	return strings.HasPrefix(strings.TrimSpace(value), "secrets:")
}

// Parse validates Secret Agent connection settings and infers the connection mode.
// An empty block is disabled. A timeout with no endpoint is ignored.
func Parse(raw RawConfig) (Config, error) {
	address := strings.TrimSpace(raw.Address)
	socketPath := strings.TrimSpace(raw.SocketPath)
	tlsName := strings.TrimSpace(raw.TLSName)
	caFile := strings.TrimSpace(raw.CAFile)
	certFile := strings.TrimSpace(raw.CertFile)
	keyFile := strings.TrimSpace(raw.KeyFile)
	keyPass := strings.TrimSpace(raw.KeyFilePassphrase)

	if err := rejectSecretReferences(caFile, certFile, keyFile, keyPass); err != nil {
		return Config{}, err
	}

	tlsConfigured := tlsName != "" || caFile != "" || certFile != "" || keyFile != "" || keyPass != ""
	tcpConfigured := address != "" || raw.Port != 0
	udsConfigured := socketPath != ""

	if !tcpConfigured && !udsConfigured && !tlsConfigured {
		return Config{Mode: ModeDisabled}, nil
	}

	if udsConfigured && (tcpConfigured || tlsConfigured) {
		return Config{}, errors.New("secret agent configuration is ambiguous: socket_path cannot be combined with address, port, or TLS settings")
	}

	timeout := defaultTimeout
	if raw.TimeoutSeconds > 0 {
		timeout = time.Duration(raw.TimeoutSeconds) * time.Second
	}

	if udsConfigured {
		return Config{
			Mode:       ModeUDS,
			SocketPath: socketPath,
			Timeout:    timeout,
			IsBase64:   raw.IsBase64,
		}, nil
	}

	if address == "" || raw.Port == 0 {
		return Config{}, errors.New("secret agent tcp configuration requires both address and port")
	}

	if !tlsConfigured {
		return Config{
			Mode:     ModeTCP,
			Address:  address,
			Port:     raw.Port,
			Timeout:  timeout,
			IsBase64: raw.IsBase64,
		}, nil
	}

	if tlsName == "" || caFile == "" {
		return Config{}, errors.New("secret agent tls requires tls_name and ca_file")
	}

	if (certFile == "") != (keyFile == "") {
		return Config{}, errors.New("secret agent cert_file and key_file must both be set for mutual TLS")
	}

	if keyPass != "" && keyFile == "" {
		return Config{}, errors.New("secret agent key_file_passphrase requires key_file")
	}

	mode := ModeTLS
	if certFile != "" {
		mode = ModeMTLS
	}

	return Config{
		Mode:              mode,
		Address:           address,
		Port:              raw.Port,
		TLSName:           tlsName,
		CAFile:            caFile,
		CertFile:          certFile,
		KeyFile:           keyFile,
		KeyFilePassphrase: keyPass,
		Timeout:           timeout,
		IsBase64:          raw.IsBase64,
	}, nil
}

func rejectSecretReferences(caFile, certFile, keyFile, keyPass string) error {
	fields := []struct {
		name  string
		value string
	}{
		{"ca_file", caFile},
		{"cert_file", certFile},
		{"key_file", keyFile},
		{"key_file_passphrase", keyPass},
	}

	for _, field := range fields {
		if IsSecretReference(field.value) {
			return errors.New("secret agent " + field.name + " cannot use a secrets: reference")
		}
	}

	return nil
}
