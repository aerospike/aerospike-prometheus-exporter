package main

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/gobwas/glob"
	"github.com/jameskeane/bcrypt"
	"github.com/prometheus/client_golang/prometheus"
)

func makeMetric(namespace, name string, t metricType, labels ...string) promMetric {
	promDesc := prometheus.NewDesc(
		namespace+"_"+normalizeMetric(name),
		normalizeDesc(name),
		labels,
		nil,
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

// Regex for indentifying globbing patterns (or standard wildcards) in the whitelist.
var globbingPattern = regexp.MustCompile(`[|]|\*|\?|{|}|\\|!`)

// Filter metrics based on configured whitelist.
func getWhitelistedMetrics(rawMetrics map[string]metricType, whitelist []string, whitelistEnabled bool) map[string]metricType {
	if whitelistEnabled {
		whitelistedMetrics := make(map[string]metricType)

		for _, stat := range whitelist {
			if globbingPattern.MatchString(stat) {
				ge := glob.MustCompile(stat)

				for k, v := range rawMetrics {
					if ge.Match(k) {
						whitelistedMetrics[k] = v
					}
				}
			} else {
				if val, ok := rawMetrics[stat]; ok {
					whitelistedMetrics[stat] = val
				}
			}
		}

		return whitelistedMetrics
	}

	return rawMetrics
}
