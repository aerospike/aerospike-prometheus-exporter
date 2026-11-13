package secretagent

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

// magic is the Secret Agent header marker. Aerospike Server uses the same value.
const magic uint32 = 0x51dec1cc

// maxResponseBody is large enough for PEM material and small enough to reject a hostile length.
const maxResponseBody = 256 * 1024

const trailingWhitespace = " \t\n\r\f\v"

type request struct {
	Resource  string `json:"Resource"`
	SecretKey string `json:"SecretKey"`
}

type response struct {
	SecretValue string `json:"SecretValue,omitempty"`
	Error       string `json:"Error,omitempty"`
}

// EncodeRequest builds the 8-byte header and JSON body used by Secret Agent.
func EncodeRequest(resource, key string) ([]byte, error) {
	body, err := json.Marshal(request{Resource: resource, SecretKey: key})
	if err != nil {
		return nil, fmt.Errorf("failed to encode secret agent request: %w", err)
	}

	payload := make([]byte, 8+len(body))
	binary.BigEndian.PutUint32(payload[:4], magic)
	binary.BigEndian.PutUint32(payload[4:8], uint32(len(body)))
	copy(payload[8:], body)

	return payload, nil
}

// DecodeResponse reads one Secret Agent response and returns SecretValue as text.
// Base64 decoding is selected by the secrets:b64: reference, not by this response.
// The returned error never includes the secret bytes.
func DecodeResponse(r io.Reader) ([]byte, error) {
	header := make([]byte, 8)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, fmt.Errorf("failed to read secret agent response header: %w", err)
	}

	receivedMagic := binary.BigEndian.Uint32(header[:4])
	if receivedMagic != magic {
		return nil, fmt.Errorf("secret agent response has invalid header")
	}

	length := binary.BigEndian.Uint32(header[4:8])
	if length == 0 {
		return nil, errors.New("secret agent returned an empty response")
	}

	if length > maxResponseBody {
		return nil, fmt.Errorf("secret agent response exceeds %d bytes", maxResponseBody)
	}

	body := make([]byte, length)
	if _, err := io.ReadFull(r, body); err != nil {
		return nil, fmt.Errorf("failed to read secret agent response body: %w", err)
	}

	var decoded response
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, errors.New("failed to decode secret agent response")
	}

	if decoded.Error != "" {
		return nil, errors.New(decoded.Error)
	}

	payload := strings.TrimRight(decoded.SecretValue, trailingWhitespace)
	if payload == "" {
		return nil, errors.New("secret agent returned an empty secret")
	}

	return []byte(payload), nil
}
