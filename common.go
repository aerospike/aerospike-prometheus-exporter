package main

import (
	"bytes"
	"crypto/subtle"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/gobwas/glob"
	"github.com/jameskeane/bcrypt"
	"github.com/prometheus/client_golang/prometheus"
)

func makeMetric(namespace, name string, t metricType, constLabels map[string]string, labels ...string) promMetric {
	promDesc := prometheus.NewDesc(
		namespace+"_"+normalizeMetric(name),
		normalizeDesc(name),
		labels,
		constLabels,
	)

	switch t {
	case mtCounter:
		return promMetric{
			origDesc:  name,
			desc:      promDesc,
			valueType: prometheus.CounterValue,
		}
	case mtGauge:
		return promMetric{
			origDesc:  name,
			desc:      promDesc,
			valueType: prometheus.GaugeValue,
		}
	}

	panic("Should not reach here...")
}

var descReplacerFunc = strings.NewReplacer("_", " ", "-", " ", ".", " ")

func normalizeDesc(s string) string {
	return descReplacerFunc.Replace(s)
}

var metricReplacerFunc = strings.NewReplacer(".", "_", "-", "_", " ", "_")

func normalizeMetric(s string) string {
	return metricReplacerFunc.Replace(s)
}

func parseStats(s, sep string) map[string]string {
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

func tryConvert(s string) (float64, error) {
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f, nil
	}

	if b, err := strconv.ParseBool(s); err == nil {
		if b {
			return 1, nil
		}
		return 0, nil
	}

	return 0, fmt.Errorf("Invalid value `%s`. Only Float or Boolean are accepted", s)
}

// take from github.com/aerospike/aerospike-client-go/admin_command.go
func hashPassword(password string) ([]byte, error) {
	// Hashing the password with the cost of 10, with a static salt
	const salt = "$2a$10$7EqJtq98hPqEX7fNZaFWoO"
	hashedPassword, err := bcrypt.Hash(password, salt)
	if err != nil {
		return nil, err
	}
	return []byte(hashedPassword), nil
}

// Check HTTP Basic Authentication.
// Validate username, password from the http request against the configured values.
func validateBasicAuth(r *http.Request, username string, password string) bool {
	user, pass, ok := r.BasicAuth()

	if !ok || subtle.ConstantTimeCompare([]byte(user), []byte(username)) != 1 || subtle.ConstantTimeCompare([]byte(pass), []byte(password)) != 1 {
		return false
	}

	return true
}

// Regex for indentifying globbing patterns (or standard wildcards) in the metrics allowlist and blocklist.
var globbingPattern = regexp.MustCompile(`\[|\]|\*|\?|\{|\}|\\|!`)

// Filter metrics
// Runs the raw metrics through allowlist first and the resulting metrics through blocklist
func getFilteredMetrics(rawMetrics map[string]metricType, allowlist []string, allowlistEnabled bool, blocklist []string, blocklistEnabled bool) map[string]metricType {
	filteredMetrics := filterAllowedMetrics(rawMetrics, allowlist, allowlistEnabled)
	filterBlockedMetrics(filteredMetrics, blocklist, blocklistEnabled)

	return filteredMetrics
}

// Filter metrics based on configured allowlist.
func filterAllowedMetrics(rawMetrics map[string]metricType, allowlist []string, allowlistEnabled bool) map[string]metricType {
	if !allowlistEnabled {
		return rawMetrics
	}

	filteredMetrics := make(map[string]metricType)

	for _, stat := range allowlist {
		if globbingPattern.MatchString(stat) {
			ge := glob.MustCompile(stat)

			for k, v := range rawMetrics {
				if ge.Match(k) {
					filteredMetrics[k] = v
				}
			}
		} else {
			if val, ok := rawMetrics[stat]; ok {
				filteredMetrics[stat] = val
			}
		}
	}

	return filteredMetrics
}

// Filter metrics based on configured blocklist.
func filterBlockedMetrics(filteredMetrics map[string]metricType, blocklist []string, blocklistEnabled bool) {
	if !blocklistEnabled {
		return
	}

	for _, stat := range blocklist {
		if globbingPattern.MatchString(stat) {
			ge := glob.MustCompile(stat)

			for k := range filteredMetrics {
				if ge.Match(k) {
					delete(filteredMetrics, k)
				}
			}
		} else {
			delete(filteredMetrics, stat)
		}
	}
}

// Get secret
// secretConfig can be one of the following,
// 1. "<secret>" (secret directly)
// 2. "file:<file-that-contains-secret>" (file containing secret)
// 3. "env:<environment-variable-that-contains-secret>" (environment variable containing secret)
// 4. "env-b64:<environment-variable-that-contains-base64-encoded-secret>" (environment variable containing base64 encoded secret)
// 5. "b64:<base64-encoded-secret>" (base64 encoded secret)
func getSecret(secretConfig string) ([]byte, error) {
	secretSource := strings.SplitN(secretConfig, ":", 2)

	if len(secretSource) == 2 {
		switch secretSource[0] {
		case "file":
			return readFromFile(secretSource[1])

		case "env":
			secret, ok := os.LookupEnv(secretSource[1])
			if !ok {
				return nil, fmt.Errorf("Environment variable %s not set", secretSource[1])
			}

			return []byte(secret), nil

		case "env-b64":
			return getValueFromBase64EnvVar(secretSource[1])

		case "b64":
			return getValueFromBase64(secretSource[1])

		default:
			return nil, fmt.Errorf("Invalid source: %s", secretSource[0])
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
			return getValueFromBase64EnvVar(certificateSource[1])

		case "b64":
			return getValueFromBase64(certificateSource[1])

		default:
			return nil, fmt.Errorf("Invalid source %s", certificateSource[0])
		}
	}

	// Assume certConfig is a file path (backward compatible)
	return readFromFile(certConfig)
}

// Read content from file
func readFromFile(filePath string) ([]byte, error) {
	dataBytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("Failed to read from file `%s`: `%v`", filePath, err)
	}

	data := bytes.TrimSuffix(dataBytes, []byte("\n"))

	return data, nil
}

// Get decoded base64 value from environment variable
func getValueFromBase64EnvVar(envVar string) ([]byte, error) {
	b64Value, ok := os.LookupEnv(envVar)
	if !ok {
		return nil, fmt.Errorf("Environment variable %s not set", envVar)
	}

	return getValueFromBase64(b64Value)
}

// Get decoded base64 value
func getValueFromBase64(b64Value string) ([]byte, error) {
	value, err := base64.StdEncoding.DecodeString(b64Value)
	if err != nil {
		return nil, fmt.Errorf("Failed to decode base64 value: %v", err)
	}

	return value, nil
}

// loadCACert returns CA set of certificates (cert pool)
// reads CA certificate based on the certConfig and adds it to the pool
func loadCACert(certConfig string) (*x509.CertPool, error) {
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
func loadServerCertAndKey(certConfig, keyConfig, keyPassConfig string) ([]tls.Certificate, error) {
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
	certBlock, _ := pem.Decode(certFileBytes)

	if keyBlock == nil || certBlock == nil {
		return nil, fmt.Errorf("Failed to decode PEM data for key or certificate")
	}

	// Check and Decrypt the the Key Block using passphrase
	if x509.IsEncryptedPEMBlock(keyBlock) {
		keyFilePassphraseBytes, err := getSecret(keyPassConfig)
		if err != nil {
			return nil, fmt.Errorf("Failed to get key passphrase: `%s`", err)
		}

		decryptedDERBytes, err := x509.DecryptPEMBlock(keyBlock, keyFilePassphraseBytes)
		if err != nil {
			return nil, fmt.Errorf("Failed to decrypt PEM Block: `%s`", err)
		}

		keyBlock.Bytes = decryptedDERBytes
		keyBlock.Headers = nil
	}

	// Encode PEM data
	keyPEM := pem.EncodeToMemory(keyBlock)
	certPEM := pem.EncodeToMemory(certBlock)

	if keyPEM == nil || certPEM == nil {
		return nil, fmt.Errorf("Failed to encode PEM data for key or certificate")
	}

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, fmt.Errorf("Failed to add certificate and key to the pool: `%s`", err)
	}

	certificates = append(certificates, cert)

	return certificates, nil
}

func sanitizeUTF8(lv string) string {
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
