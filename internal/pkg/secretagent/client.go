package secretagent

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"os"
	"syscall"
	"time"
)

// SecretClient fetches one secret at a time from Secret Agent.
type SecretClient interface {
	Fetch(ctx context.Context, resource, key string) ([]byte, error)
	Close() error
}

type secretAgentClient struct {
	mode      Mode
	endpoint  string
	timeout   time.Duration
	tlsConfig *tls.Config
}

// New returns a client for a parsed Secret Agent configuration.
// tlsConfig is required for TLS and mTLS and must be nil for TCP and UDS.
// Certificate verification is always left enabled.
func New(cfg Config, tlsConfig *tls.Config) (SecretClient, error) {
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

	return &secretAgentClient{
		mode:      cfg.Mode,
		endpoint:  cfg.Endpoint(),
		timeout:   cfg.Timeout,
		tlsConfig: tlsConfig,
	}, nil
}

func (c *secretAgentClient) Close() error {
	return nil
}

// Fetch makes one attempt. The configured timeout is a single deadline for the whole
// attempt: dial (including the TLS handshake), write, and read share it.
func (c *secretAgentClient) Fetch(ctx context.Context, resource, key string) ([]byte, error) {
	if resource == "" || key == "" {
		return nil, errors.New("secret agent resource and key are required")
	}

	if ctx == nil {
		ctx = context.Background()
	}

	deadline := time.Now().Add(c.timeout)

	ctx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()

	conn, err := c.dial(ctx)
	if err != nil {
		return nil, c.wrap(resource, key, "dial", err)
	}

	defer func() { _ = conn.Close() }()

	payload, err := EncodeRequest(resource, key)
	if err != nil {
		return nil, c.wrap(resource, key, "encode", err)
	}

	if err = conn.SetDeadline(deadline); err != nil {
		return nil, c.wrap(resource, key, "write", err)
	}

	if err = writeFull(conn, payload); err != nil {
		return nil, c.wrap(resource, key, "write", err)
	}

	secret, err := DecodeResponse(conn)
	if err != nil {
		return nil, c.wrap(resource, key, "read", err)
	}

	return secret, nil
}

func (c *secretAgentClient) dial(ctx context.Context) (net.Conn, error) {
	dialer := &net.Dialer{}
	if c.tlsConfig != nil {
		tlsDialer := &tls.Dialer{NetDialer: dialer, Config: c.tlsConfig}
		return tlsDialer.DialContext(ctx, "tcp", c.endpoint)
	}

	network := "tcp"
	if c.mode == ModeUDS {
		network = "unix"
	}

	return dialer.DialContext(ctx, network, c.endpoint)
}

func (c *secretAgentClient) wrap(resource, key, op string, err error) error {
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

// IsRetryable reports whether a Fetch error means Secret Agent could not be reached in time:
// the connection could not be opened, or the attempt timed out. These can clear up on their own,
// for example when a sidecar is still starting. Everything else will not change on a retry:
// an Error string from the agent, an unreadable response, a TLS verification failure,
// a permission error on the socket, or a bad reference.
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, syscall.EACCES) || errors.Is(err, syscall.EPERM) {
		return false
	}

	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, os.ErrDeadlineExceeded) {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	var opErr *net.OpError
	if errors.As(err, &opErr) && opErr.Op == "dial" {
		return true
	}

	return false
}

// Fetcher is the part of SecretClient that FetchWithRetry needs.
type Fetcher interface {
	Fetch(ctx context.Context, resource, key string) ([]byte, error)
}

// RetryPolicy bounds how long FetchWithRetry keeps trying.
// The zero value means a single attempt.
type RetryPolicy struct {
	// Until is the last moment a new attempt may start. The zero time disables retries.
	Until time.Time
	// Interval is the wait between attempts.
	Interval time.Duration
	// OnRetry, when set, is called before each wait with the error that caused it.
	OnRetry func(err error)
}

// FetchWithRetry fetches one secret and retries only errors that IsRetryable accepts,
// until the policy's deadline. Any other error is returned at once.
func FetchWithRetry(ctx context.Context, fetcher Fetcher, resource, key string, policy RetryPolicy) ([]byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	for {
		secret, err := fetcher.Fetch(ctx, resource, key)
		if err == nil || !IsRetryable(err) {
			return secret, err
		}

		if policy.Until.IsZero() || time.Now().Add(policy.Interval).After(policy.Until) {
			return nil, err
		}

		if policy.OnRetry != nil {
			policy.OnRetry(err)
		}

		select {
		case <-ctx.Done():
			return nil, err
		case <-time.After(policy.Interval):
		}
	}
}
