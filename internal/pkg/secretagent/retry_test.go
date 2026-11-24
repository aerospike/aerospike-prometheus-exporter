package secretagent

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"os"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestIsRetryable(t *testing.T) {
	refused := &net.OpError{Op: "dial", Net: "tcp", Err: &os.SyscallError{Syscall: "connect", Err: syscall.ECONNREFUSED}}
	missingSocket := &net.OpError{Op: "dial", Net: "unix", Err: &os.SyscallError{Syscall: "connect", Err: syscall.ENOENT}}
	denied := &net.OpError{Op: "dial", Net: "unix", Err: &os.SyscallError{Syscall: "connect", Err: syscall.EACCES}}

	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"connection refused", fmt.Errorf("wrapped: %w", refused), true},
		{"missing socket", fmt.Errorf("wrapped: %w", missingSocket), true},
		{"socket permission denied", fmt.Errorf("wrapped: %w", denied), false},
		{"context deadline", fmt.Errorf("wrapped: %w", context.DeadlineExceeded), true},
		{"i/o deadline", fmt.Errorf("wrapped: %w", os.ErrDeadlineExceeded), true},
		{"agent error string", errors.New("resource not found"), false},
		{"bad response", errors.New("secret agent response has invalid header"), false},
		{"tls verification", fmt.Errorf("wrapped: %w", &tls.CertificateVerificationError{Err: errors.New("unknown authority")}), false},
		{"canceled", context.Canceled, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, IsRetryable(tc.err))
		})
	}
}

type scriptedFetcher struct {
	mu    sync.Mutex
	errs  []error
	value string
	calls int
}

func (f *scriptedFetcher) Fetch(context.Context, string, string) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.calls++
	if len(f.errs) > 0 {
		err := f.errs[0]
		f.errs = f.errs[1:]

		return nil, err
	}

	return []byte(f.value), nil
}

var errUnreachable = fmt.Errorf("secret agent: %w", &net.OpError{Op: "dial", Net: "tcp", Err: syscall.ECONNREFUSED})

func TestFetchWithRetry(t *testing.T) {
	policy := func() RetryPolicy {
		return RetryPolicy{Until: time.Now().Add(2 * time.Second), Interval: 5 * time.Millisecond}
	}

	t.Run("succeeds once the agent is reachable", func(t *testing.T) {
		fetcher := &scriptedFetcher{errs: []error{errUnreachable, errUnreachable}, value: "pw"}
		retries := 0
		p := policy()
		p.OnRetry = func(error) { retries++ }

		got, err := FetchWithRetry(context.Background(), fetcher, "APE", "k", p)
		require.NoError(t, err)
		require.Equal(t, "pw", string(got))
		require.Equal(t, 3, fetcher.calls)
		require.Equal(t, 2, retries)
	})

	t.Run("agent error is not retried", func(t *testing.T) {
		fetcher := &scriptedFetcher{errs: []error{errors.New("resource not found")}, value: "pw"}

		_, err := FetchWithRetry(context.Background(), fetcher, "APE", "k", policy())
		require.EqualError(t, err, "resource not found")
		require.Equal(t, 1, fetcher.calls)
	})

	t.Run("gives up at the deadline", func(t *testing.T) {
		fetcher := &scriptedFetcher{errs: repeat(errUnreachable, 1000), value: "pw"}
		p := RetryPolicy{Until: time.Now().Add(100 * time.Millisecond), Interval: 10 * time.Millisecond}

		start := time.Now()
		_, err := FetchWithRetry(context.Background(), fetcher, "APE", "k", p)
		require.ErrorIs(t, err, syscall.ECONNREFUSED)
		require.Greater(t, fetcher.calls, 2)
		require.Less(t, time.Since(start), time.Second)
	})

	t.Run("zero policy makes one attempt", func(t *testing.T) {
		fetcher := &scriptedFetcher{errs: []error{errUnreachable}, value: "pw"}

		_, err := FetchWithRetry(context.Background(), fetcher, "APE", "k", RetryPolicy{})
		require.Error(t, err)
		require.Equal(t, 1, fetcher.calls)
	})

	t.Run("canceled context stops the wait", func(t *testing.T) {
		fetcher := &scriptedFetcher{errs: repeat(errUnreachable, 1000), value: "pw"}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		p := RetryPolicy{Until: time.Now().Add(time.Minute), Interval: time.Hour}
		start := time.Now()
		_, err := FetchWithRetry(ctx, fetcher, "APE", "k", p)
		require.Error(t, err)
		require.Less(t, time.Since(start), time.Second)
	})
}

func repeat(err error, n int) []error {
	out := make([]error, n)
	for i := range out {
		out[i] = err
	}

	return out
}

// The timeout is one deadline for dial, write and read together. A slow TLS handshake and a
// slow reply that are each shorter than the timeout must still fail when their sum is not.
func TestFetchUsesOneDeadlineForTheWholeAttempt(t *testing.T) {
	material := newCertMaterial(t)

	ln, err := tls.Listen("tcp", "127.0.0.1:0", material.serverTLS(tls.NoClientCert))
	require.NoError(t, err)
	t.Cleanup(func() { _ = ln.Close() })

	go func() {
		for {
			conn, acceptErr := ln.Accept()
			if acceptErr != nil {
				return
			}

			go func(conn net.Conn) {
				defer func() { _ = conn.Close() }()
				_ = conn.SetDeadline(time.Now().Add(5 * time.Second))

				time.Sleep(700 * time.Millisecond) // the client handshake waits here

				serveSecret(conn, func(string, string) (string, string) {
					time.Sleep(700 * time.Millisecond) // and the client read waits here
					return "late", ""
				})
			}(conn)
		}
	}()

	client := newTestClient(t, RawConfig{
		Address:        "127.0.0.1",
		Port:           portOf(t, ln),
		TLSName:        "secretagent",
		CAFile:         material.caPath,
		TimeoutSeconds: 1,
	}, testTLSConfig(t, material, false))

	start := time.Now()
	_, err = client.Fetch(context.Background(), "APE", "Password")
	elapsed := time.Since(start)

	require.Error(t, err)
	require.True(t, IsRetryable(err), "a timeout is retryable: %v", err)
	require.Less(t, elapsed, 1300*time.Millisecond, "one 1s deadline, not 1s per step")
	require.NotContains(t, err.Error(), "late")
}

// A server that accepts the connection and never answers must not hold an attempt past the retry
// deadline, even when the client timeout is much longer.
func TestFetchWithRetryCapsAnAttemptAtTheRetryDeadline(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = ln.Close() })

	go func() {
		for {
			conn, acceptErr := ln.Accept()
			if acceptErr != nil {
				return
			}

			// Hold the connection open and never reply.
			t.Cleanup(func() { _ = conn.Close() })
		}
	}()

	client := newTestClient(t, RawConfig{
		Address:        "127.0.0.1",
		Port:           portOf(t, ln),
		TimeoutSeconds: 10,
	}, nil)

	policy := RetryPolicy{Until: time.Now().Add(50 * time.Millisecond), Interval: 10 * time.Millisecond}

	start := time.Now()
	_, err = FetchWithRetry(context.Background(), client, "APE", "Password", policy)
	elapsed := time.Since(start)

	require.Error(t, err)
	require.True(t, IsRetryable(err), "a timeout is retryable: %v", err)
	require.GreaterOrEqual(t, elapsed, 40*time.Millisecond)
	require.Less(t, elapsed, 500*time.Millisecond, "the attempt must end near 50ms, not at the 10s client timeout")
}

// The caller's context deadline also caps the attempt when it is the earliest.
func TestFetchHonorsAnEarlierCallerDeadline(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = ln.Close() })

	go func() {
		for {
			conn, acceptErr := ln.Accept()
			if acceptErr != nil {
				return
			}

			t.Cleanup(func() { _ = conn.Close() })
		}
	}()

	client := newTestClient(t, RawConfig{Address: "127.0.0.1", Port: portOf(t, ln), TimeoutSeconds: 10}, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err = client.Fetch(ctx, "APE", "Password")

	require.Error(t, err)
	require.Less(t, time.Since(start), 500*time.Millisecond)
}
