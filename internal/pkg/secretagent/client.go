package secretagent

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"time"
)

// Client fetches one secret at a time from Secret Agent.
type Client interface {
	Fetch(ctx context.Context, resource, key string) ([]byte, error)
	Close() error
}

type agentClient struct {
	mode      Mode
	endpoint  string
	timeout   time.Duration
	tlsConfig *tls.Config
}

// New returns a client for a parsed Secret Agent configuration.
// tlsConfig is required for TLS and mTLS and must be nil for TCP and UDS.
// Certificate verification is always left enabled.
func New(cfg Config, tlsConfig *tls.Config) (Client, error) {
	if cfg.Mode == ModeDisabled {
		return nil, errors.New("secret agent is not configured")
	}

	needsTLS := cfg.Mode == ModeTLS || cfg.Mode == ModeMTLS
	if needsTLS && tlsConfig == nil {
		return nil, errors.New("secret agent tls configuration is required")
	}

	if !needsTLS && tlsConfig != nil {
		return nil, errors.New("secret agent tls configuration is not used for this connection mode")
	}

	if cfg.Timeout <= 0 {
		cfg.Timeout = defaultTimeout
	}

	if tlsConfig != nil {
		tlsConfig = tlsConfig.Clone()
		tlsConfig.InsecureSkipVerify = false
	}

	return &agentClient{
		mode:      cfg.Mode,
		endpoint:  cfg.Endpoint(),
		timeout:   cfg.Timeout,
		tlsConfig: tlsConfig,
	}, nil
}

func (c *agentClient) Close() error {
	return nil
}

func (c *agentClient) Fetch(ctx context.Context, resource, key string) ([]byte, error) {
	if resource == "" || key == "" {
		return nil, errors.New("secret agent resource and key are required")
	}

	if ctx == nil {
		ctx = context.Background()
	}

	conn, err := c.dial(ctx)
	if err != nil {
		return nil, c.wrap(resource, key, "dial", err)
	}

	defer func() { _ = conn.Close() }()

	payload, err := EncodeRequest(resource, key)
	if err != nil {
		return nil, c.wrap(resource, key, "encode", err)
	}

	if err = conn.SetWriteDeadline(time.Now().Add(c.timeout)); err != nil {
		return nil, c.wrap(resource, key, "write", err)
	}

	if err = writeFull(conn, payload); err != nil {
		return nil, c.wrap(resource, key, "write", err)
	}

	if err = conn.SetReadDeadline(time.Now().Add(c.timeout)); err != nil {
		return nil, c.wrap(resource, key, "read", err)
	}

	secret, err := DecodeResponse(conn)
	if err != nil {
		return nil, c.wrap(resource, key, "read", err)
	}

	return secret, nil
}

func (c *agentClient) dial(ctx context.Context) (net.Conn, error) {
	dialCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	dialer := &net.Dialer{Timeout: c.timeout}
	if c.tlsConfig != nil {
		tlsDialer := &tls.Dialer{NetDialer: dialer, Config: c.tlsConfig}
		return tlsDialer.DialContext(dialCtx, "tcp", c.endpoint)
	}

	network := "tcp"
	if c.mode == ModeUDS {
		network = "unix"
	}

	return dialer.DialContext(dialCtx, network, c.endpoint)
}

func (c *agentClient) wrap(resource, key, op string, err error) error {
	return fmt.Errorf("secret agent %s %s: failed to %s %s:%s: %w", c.mode, c.endpoint, op, resource, key, err)
}

func writeFull(conn net.Conn, payload []byte) error {
	for len(payload) > 0 {
		n, err := conn.Write(payload)
		if err != nil {
			return err
		}

		payload = payload[n:]
	}

	return nil
}
