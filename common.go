package main

import (
	"fmt"
	"strconv"
	"strings"

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
