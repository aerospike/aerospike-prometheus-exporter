package secretagent

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncodeAndDecodeRoundTrip(t *testing.T) {
	payload, err := EncodeRequest(`res"ource`, "key:with:colon")
	require.NoError(t, err)
	require.Equal(t, magic, binary.BigEndian.Uint32(payload[:4]))
	require.Contains(t, string(payload[8:]), `res\"ource`)
	require.Contains(t, string(payload[8:]), "key:with:colon")

	body := []byte(`{"SecretValue":"db-password\n"}`)
	frame := make([]byte, 8+len(body))
	binary.BigEndian.PutUint32(frame[:4], magic)
	binary.BigEndian.PutUint32(frame[4:8], uint32(len(body)))
	copy(frame[8:], body)

	decoded, err := DecodeResponse(bytes.NewReader(frame))
	require.NoError(t, err)
	require.Equal(t, "db-password", string(decoded))
}

func TestDecodeResponsePlaintext(t *testing.T) {
	decoded, err := DecodeResponse(bytes.NewReader(responseFrame(t, `{"SecretValue":"test"}`)))
	require.NoError(t, err)
	require.Equal(t, "test", string(decoded))
}

func TestDecodeResponseFailures(t *testing.T) {
	t.Run("agent error", func(t *testing.T) {
		frame := responseFrame(t, `{"Error":"resource not found","SecretValue":"c2VjcmV0"}`)
		_, err := DecodeResponse(bytes.NewReader(frame))
		require.EqualError(t, err, "resource not found")
		require.NotContains(t, err.Error(), "c2VjcmV0")
	})

	t.Run("invalid magic", func(t *testing.T) {
		frame := responseFrame(t, `{"SecretValue":"YQ=="}`)
		binary.BigEndian.PutUint32(frame[:4], 1)
		_, err := DecodeResponse(bytes.NewReader(frame))
		require.ErrorContains(t, err, "invalid header")
	})

	t.Run("too large", func(t *testing.T) {
		frame := make([]byte, 8)
		binary.BigEndian.PutUint32(frame[:4], magic)
		binary.BigEndian.PutUint32(frame[4:8], maxResponseBody+1)
		_, err := DecodeResponse(bytes.NewReader(frame))
		require.ErrorContains(t, err, "exceeds")
	})

	t.Run("empty secret", func(t *testing.T) {
		_, err := DecodeResponse(bytes.NewReader(responseFrame(t, `{"SecretValue":"   "}`)))
		require.ErrorContains(t, err, "empty secret")

		decoded, err := DecodeResponse(bytes.NewReader(responseFrame(t, `{"SecretValue":"****"}`)))
		require.NoError(t, err)
		require.Equal(t, "****", string(decoded))
	})

	t.Run("bad json", func(t *testing.T) {
		_, err := DecodeResponse(bytes.NewReader(responseFrame(t, `not-json`)))
		require.EqualError(t, err, "failed to decode secret agent response")
		require.NotContains(t, err.Error(), "not-json")
	})
}

func responseFrame(t *testing.T, body string) []byte {
	t.Helper()

	frame := make([]byte, 8+len(body))
	binary.BigEndian.PutUint32(frame[:4], magic)
	binary.BigEndian.PutUint32(frame[4:8], uint32(len(body)))
	copy(frame[8:], body)

	return frame
}
