package main

import (
	"bytes"
	"crypto/subtle"
	"errors"
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
	log "github.com/sirupsen/logrus"
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
func validateBasicAuth(w http.ResponseWriter, r *http.Request, username string, password string) bool {
	user, pass, ok := r.BasicAuth()

	if !ok || subtle.ConstantTimeCompare([]byte(user), []byte(username)) != 1 || subtle.ConstantTimeCompare([]byte(pass), []byte(password)) != 1 {
		return false
	}

	return true
}

// Regex for indentifying globbing patterns (or standard wildcards) in the metrics allowlist and blocklist.
var globbingPattern = regexp.MustCompile(`[|]|\*|\?|{|}|\\|!`)

// Filter metrics
// Runs the raw metrics through allowlist first and the resulting metrics through blocklist
func getFilteredMetrics(rawMetrics map[string]metricType, allowlist []string, allowlistEnabled bool, blocklist []string, blocklistEnabled bool) map[string]metricType {
	filteredMetrics := filterAllowedMetrics(rawMetrics, allowlist, allowlistEnabled)
	filterBlockedMetrics(filteredMetrics, blocklist, blocklistEnabled)

	return filteredMetrics
}

// Filter metrics based on configured allowlist.
func filterAllowedMetrics(rawMetrics map[string]metricType, allowlist []string, allowlistEnabled bool) map[string]metricType {
	if allowlistEnabled {
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

	return rawMetrics
}

// Filter metrics based on configured blocklist.
func filterBlockedMetrics(filteredMetrics map[string]metricType, blocklist []string, blocklistEnabled bool) {
	if blocklistEnabled {
		for _, stat := range blocklist {
			if globbingPattern.MatchString(stat) {
				ge := glob.MustCompile(stat)

				for k := range filteredMetrics {
					if ge.Match(k) {
						delete(filteredMetrics, k)
					}
				}
			} else {
				if _, ok := filteredMetrics[stat]; ok {
					delete(filteredMetrics, stat)
				}
			}
		}
	}
}

// Get key file passphrase from environment variable or from a file or directly from the config variable
// keyFilePassphraseConfig can be one of the following,
// 1. "<passphrase>"
// 2. "file:<file-that-contains-passphrase>"
// 3. "env:<environment-variable-that-contains-passphrase>"
func getKeyFilePassphrase(keyFilePassphraseConfig string) ([]byte, error) {
	keyFilePassphraseSource := strings.SplitN(keyFilePassphraseConfig, ":", 2)

	if len(keyFilePassphraseSource) == 2 {
		if keyFilePassphraseSource[0] == "file" {
			dataBytes, err := ioutil.ReadFile(keyFilePassphraseSource[1])
			if err != nil {
				return nil, err
			}

			keyPassphrase := bytes.TrimSuffix(dataBytes, []byte("\n"))

			return keyPassphrase, nil
		}

		if keyFilePassphraseSource[0] == "env" {
			keyPassphrase, ok := os.LookupEnv(keyFilePassphraseSource[1])
			if !ok {
				return nil, errors.New("Environment variable " + keyFilePassphraseSource[1] + " not set")
			}

			return []byte(keyPassphrase), nil
		}
	}

	return []byte(keyFilePassphraseConfig), nil
}

// Read certificate file and abort if any errors
// Returns file content as byte array
func readCertFile(filename string) []byte {
	dataBytes, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatalf("Failed to read certificate or key file `%s` : `%s`", filename, err)
	}

	return dataBytes
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
