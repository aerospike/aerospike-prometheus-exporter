package secretagent

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParseDisabled(t *testing.T) {
	cfg, err := Parse(RawConfig{})
	require.NoError(t, err)
	require.Equal(t, ModeDisabled, cfg.Mode)

	cfg, err = Parse(RawConfig{TimeoutSeconds: 9})
	require.NoError(t, err)
	require.Equal(t, ModeDisabled, cfg.Mode)
}

func TestParseTCPAndUDS(t *testing.T) {
	tcp, err := Parse(RawConfig{Address: "127.0.0.1", Port: 3005})
	require.NoError(t, err)
	require.Equal(t, ModeTCP, tcp.Mode)
	require.Equal(t, "127.0.0.1:3005", tcp.Endpoint())
	require.Equal(t, defaultTimeout, tcp.Timeout)

	custom, err := Parse(RawConfig{Address: "localhost", Port: 3005, TimeoutSeconds: 2})
	require.NoError(t, err)
	require.Equal(t, 2*time.Second, custom.Timeout)

	uds, err := Parse(RawConfig{SocketPath: " /tmp/secret-agent.sock "})
	require.NoError(t, err)
	require.Equal(t, ModeUDS, uds.Mode)
	require.Equal(t, "/tmp/secret-agent.sock", uds.Endpoint())
}

func TestParseTLSModes(t *testing.T) {
	tlsCfg, err := Parse(RawConfig{
		Address: "secret-agent",
		Port:    3005,
		TLSName: "secretagent",
		CAFile:  "file:/tmp/ca.pem",
	})
	require.NoError(t, err)
	require.Equal(t, ModeTLS, tlsCfg.Mode)

	mtls, err := Parse(RawConfig{
		Address:           "secret-agent",
		Port:              3005,
		TLSName:           "secretagent",
		CAFile:            "/tmp/ca.pem",
		CertFile:          "env-b64:CLIENT_CERT",
		KeyFile:           "b64:a2V5",
		KeyFilePassphrase: "env:KEY_PASS",
	})
	require.NoError(t, err)
	require.Equal(t, ModeMTLS, mtls.Mode)
	require.Equal(t, "env:KEY_PASS", mtls.KeyFilePassphrase)
}

func TestParseRejectsAmbiguousAndIncomplete(t *testing.T) {
	cases := []struct {
		name string
		raw  RawConfig
		want string
	}{
		{
			name: "tcp and uds",
			raw:  RawConfig{Address: "127.0.0.1", Port: 3005, SocketPath: "/tmp/agent.sock"},
			want: "ambiguous",
		},
		{
			name: "uds and tls",
			raw:  RawConfig{SocketPath: "/tmp/agent.sock", TLSName: "secretagent", CAFile: "/tmp/ca.pem"},
			want: "ambiguous",
		},
		{
			name: "address only",
			raw:  RawConfig{Address: "127.0.0.1"},
			want: "both address and port",
		},
		{
			name: "port only",
			raw:  RawConfig{Port: 3005},
			want: "both address and port",
		},
		{
			name: "tls without name",
			raw:  RawConfig{Address: "127.0.0.1", Port: 3005, CAFile: "/tmp/ca.pem"},
			want: "tls_name and ca_file",
		},
		{
			name: "tls without ca",
			raw:  RawConfig{Address: "127.0.0.1", Port: 3005, TLSName: "secretagent"},
			want: "tls_name and ca_file",
		},
		{
			name: "cert without key",
			raw:  RawConfig{Address: "127.0.0.1", Port: 3005, TLSName: "secretagent", CAFile: "/tmp/ca.pem", CertFile: "/tmp/cert.pem"},
			want: "mutual TLS",
		},
		{
			name: "passphrase without key",
			raw:  RawConfig{Address: "127.0.0.1", Port: 3005, TLSName: "secretagent", CAFile: "/tmp/ca.pem", KeyFilePassphrase: "pw"},
			want: "key_file_passphrase requires key_file",
		},
		{
			name: "secrets reference on ca",
			raw:  RawConfig{Address: "127.0.0.1", Port: 3005, TLSName: "secretagent", CAFile: "secrets:APE:CA"},
			want: "cannot use a secrets: reference",
		},
		{
			name: "secrets reference on key passphrase",
			raw:  RawConfig{KeyFilePassphrase: " secrets:APE:Pass "},
			want: "cannot use a secrets: reference",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse(tc.raw)
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.want)
		})
	}
}
