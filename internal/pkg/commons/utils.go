package commons

import (
	"bytes"
	"crypto/subtle"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"

	goversion "github.com/hashicorp/go-version"
)

// Default info commands
var (
	Infokey_ClusterName = "cluster-name"
	Infokey_Service     = "service-clear-std"
	Infokey_Build       = "build"
)

var (
	descReplacerFunc = strings.NewReplacer("_", " ", "-", " ", ".", " ")
	// TODO: re-check why we need below replacer, is it because of the replace char sequences
	metricReplacerFunc = strings.NewReplacer(".", "_", "-", "_", " ", "_")
)

// Utility functions

func NormalizeDesc(s string) string {
	return descReplacerFunc.Replace(s)
}

func NormalizeMetric(s string) string {
	return metricReplacerFunc.Replace(s)
}

func ParseStats(s, sep string) map[string]string {
	stats := make(map[string]string, strings.Count(s, sep)+1)
	s2 := strings.Split(s, sep)
	for _, s := range s2 {
		list := strings.SplitN(s, "=", 2)
		switch len(list) {
		case 0, 1:
		case 2:
			stats[list[0]] = list[1]
		default:
			stats[list[0]] = strings.Join(list[1:], "=")
		}
	}

	return stats
}

func TryConvert(s string) (float64, error) {
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f, nil
	}

	if b, err := strconv.ParseBool(s); err == nil {
		if b {
			return 1, nil
		}
		return 0, nil
	}

	return 0, fmt.Errorf("invalid value `%s`. Only Float or Boolean are accepted", s)
}

// Check HTTP Basic Authentication.
// Validate username, password from the http request against the configured values.
func ValidateBasicAuth(r *http.Request, username string, password string) bool {
	user, pass, ok := r.BasicAuth()

	if !ok || subtle.ConstantTimeCompare([]byte(user), []byte(username)) != 1 || subtle.ConstantTimeCompare([]byte(pass), []byte(password)) != 1 {
		return false
	}

	return true
}

// Get secret
// secretConfig can be one of the following,
// 1. "<secret>" (secret directly)
// 2. "file:<file-that-contains-secret>" (file containing secret)
// 3. "env:<environment-variable-that-contains-secret>" (environment variable containing secret)
// 4. "env-b64:<environment-variable-that-contains-base64-encoded-secret>" (environment variable containing base64 encoded secret)
// 5. "b64:<base64-encoded-secret>" (base64 encoded secret)
func GetSecret(secretConfig string) ([]byte, error) {
	secretSource := strings.SplitN(secretConfig, ":", 2)

	if len(secretSource) == 2 {
		switch secretSource[0] {
		case "file":
			return readFromFile(secretSource[1])

		case "env":
			secret, ok := os.LookupEnv(secretSource[1])
			if !ok {
				return nil, fmt.Errorf("environment variable %s not set", secretSource[1])
			}

			return []byte(secret), nil

		case "env-b64":
			return GetValueFromBase64EnvVar(secretSource[1])

		case "b64":
			return GetValueFromBase64(secretSource[1])

		default:
			return nil, fmt.Errorf("invalid source: %s", secretSource[0])
		}
	}

	return []byte(secretConfig), nil
}

// Get certificate
// certConfig can be one of the following,
// 1. "<file-path>" (certificate file path directly)
// 2. "file:<file-path>" (certificate file path)
// 3. "env-b64:<environment-variable-that-contains-base64-encoded-certificate>" (environment variable containing base64 encoded certificate)
// 4. "b64:<base64-encoded-certificate>" (base64 encoded certificate)
func getCertificate(certConfig string) ([]byte, error) {
	certificateSource := strings.SplitN(certConfig, ":", 2)

	if len(certificateSource) == 2 {
		switch certificateSource[0] {
		case "file":
			return readFromFile(certificateSource[1])

		case "env-b64":
			return GetValueFromBase64EnvVar(certificateSource[1])

		case "b64":
			return GetValueFromBase64(certificateSource[1])

		default:
			return nil, fmt.Errorf("invalid source %s", certificateSource[0])
		}
	}

	// Assume certConfig is a file path (backward compatible)
	return readFromFile(certConfig)
}

// Read content from file
func readFromFile(filePath string) ([]byte, error) {
	dataBytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read from file `%s`: `%v`", filePath, err)
	}

	data := bytes.TrimSuffix(dataBytes, []byte("\n"))

	return data, nil
}

// Get decoded base64 value from environment variable
func GetValueFromBase64EnvVar(envVar string) ([]byte, error) {
	b64Value, ok := os.LookupEnv(envVar)
	if !ok {
		return nil, fmt.Errorf("environment variable %s not set", envVar)
	}

	return GetValueFromBase64(b64Value)
}

// Get decoded base64 value
func GetValueFromBase64(b64Value string) ([]byte, error) {
	value, err := base64.StdEncoding.DecodeString(b64Value)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 value: %v", err)
	}

	return value, nil
}

// loadCACert returns CA set of certificates (cert pool)
// reads CA certificate based on the certConfig and adds it to the pool
func LoadCACert(certConfig string) (*x509.CertPool, error) {
	certificates, err := x509.SystemCertPool()
	if certificates == nil || err != nil {
		certificates = x509.NewCertPool()
	}

	if len(certConfig) > 0 {
		caCert, err := getCertificate(certConfig)
		if err != nil {
			return nil, err
		}

		certificates.AppendCertsFromPEM(caCert)
	}

	return certificates, nil
}

// loadServerCertAndKey reads server certificate and associated key file based on certConfig and keyConfig
// returns parsed server certificate
// if the private key is encrypted, it will be decrypted using key file passphrase
func LoadServerCertAndKey(certConfig, keyConfig, keyPassConfig string) ([]tls.Certificate, error) {
	var certificates []tls.Certificate

	// Read cert file
	certFileBytes, err := getCertificate(certConfig)
	if err != nil {
		return nil, err
	}

	// Read key file
	keyFileBytes, err := getCertificate(keyConfig)
	if err != nil {
		return nil, err
	}

	// Decode PEM data
	keyBlock, _ := pem.Decode(keyFileBytes)

	if keyBlock == nil {
		return nil, fmt.Errorf("failed to decode PEM data for key")
	}

	// Check and Decrypt the the Key Block using passphrase
	if x509.IsEncryptedPEMBlock(keyBlock) { // nolint:staticcheck
		keyFilePassphraseBytes, err := GetSecret(keyPassConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to get key passphrase: `%s`", err)
		}

		decryptedDERBytes, err := x509.DecryptPEMBlock(keyBlock, keyFilePassphraseBytes) // nolint:staticcheck
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt PEM Block: `%s`", err)
		}

		keyBlock.Bytes = decryptedDERBytes
		keyBlock.Headers = nil
	}

	// Encode PEM data
	keyPEM := pem.EncodeToMemory(keyBlock)

	if keyPEM == nil {
		return nil, fmt.Errorf("failed to encode PEM data for key")
	}

	cert, err := tls.X509KeyPair(certFileBytes, keyPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to add certificate and key to the pool: `%s`", err)
	}

	certificates = append(certificates, cert)

	return certificates, nil
}

func SanitizeUTF8(lv string) string {
	if utf8.ValidString(lv) {
		return lv
	}
	fixUtf := func(r rune) rune {
		if r == utf8.RuneError {
			return 65533
		}
		return r
	}

	return strings.Map(fixUtf, lv)
}

func BuildVersionGreaterThanOrEqual(rawMetrics map[string]string, ref string) (bool, error) {
	if len(rawMetrics["build"]) == 0 {
		return false, fmt.Errorf("couldn't get build version")
	}

	ver := rawMetrics["build"]
	version, err := goversion.NewVersion(ver)
	if err != nil {
		return false, fmt.Errorf("error parsing build version %s: %v", ver, err)
	}

	refVersion, err := goversion.NewVersion(ref)
	if err != nil {
		return false, fmt.Errorf("error parsing reference version %s: %v", ref, err)
	}

	if version.GreaterThanOrEqual(refVersion) {
		return true, nil
	}

	return false, nil
}

func GetFullHost() string {
	return net.JoinHostPort(config.Cfg.Aerospike.Host, strconv.Itoa(int(config.Cfg.Aerospike.Port)))
}
