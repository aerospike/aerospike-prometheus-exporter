package watchers

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	log "github.com/sirupsen/logrus"
)

// LatencyInfoParser provides a reader for Aerospike cluster's response for any of the metric
type LatencyInfoParser struct {
	*bufio.Reader
}

// NewInfoParser provides an instance of the InfoParser
func NewInfoParser(s string) *LatencyInfoParser {
	return &LatencyInfoParser{bufio.NewReader(strings.NewReader(s))}
}

// PeekAndExpect checks if the expected value is present without advancing the reader
func (ip *LatencyInfoParser) PeekAndExpect(s string) error {
	bytes, err := ip.Peek(len(s))
	if err != nil {
		return err
	}

	v := string(bytes)
	if v != s {
		return fmt.Errorf("InfoParser: Wrong value. Peek expected %s, but found %s", s, v)
	}

	return nil
}

// Expect validates the expected value against the one returned by the InfoParser
// This advances the reader by length of the input string.
func (ip *LatencyInfoParser) Expect(s string) error {
	bytes := make([]byte, len(s))

	v, err := ip.Read(bytes)
	if err != nil {
		return err
	}

	if string(bytes) != s {
		return fmt.Errorf("InfoParser: Wrong value. Expected %s, found %d", s, v)
	}

	return nil
}

// ReadUntil reads bytes from the InfoParser by handeling some edge-cases
func (ip *LatencyInfoParser) ReadUntil(delim byte) (string, error) {
	v, err := ip.ReadBytes(delim)

	switch len(v) {
	case 0:
		return string(v), err
	case 1:
		if v[0] == delim {
			return "", err
		}
		return string(v), err
	}

	return string(v[:len(v)-1]), err
}

// Legacy latency stats handlig
// Parse "latency:" info output.
//
// Format (with and without latency data)
// {test}-read:10:17:37-GMT,ops/sec,>1ms,>8ms,>64ms;10:17:47,29648.2,3.44,0.08,0.00;
// error-no-data-yet-or-back-too-small;
// or,
// {test}-write:;
func parseLatencyInfoLegacy(s string, latencyBucketsCount int) map[string]LatencyStatsMap {
	ip := NewInfoParser(s)
	res := map[string]LatencyStatsMap{}

	for {
		namespaceName, operation, err := readNamespaceAndOperation(ip)

		if err != nil {
			break
		}

		if namespaceName == "" && operation == "" {
			continue
		}

		// Might be an empty output if there's no latency data (in 5.1), so continue to next section
		if err := ip.PeekAndExpect(";"); err == nil {
			if err := ip.Expect(";"); err != nil {
				break
			}
			continue
		}

		// Ignore timestamp
		_, err = ip.ReadUntil(',')
		if err != nil {
			break
		}

		// Read bucket labels including ops/sec
		bucketLabelsStr, err := ip.ReadUntil(';')
		if err != nil {
			break
		}
		bLabels := strings.Split(bucketLabelsStr, ",")

		// Ignore timestamp
		_, err = ip.ReadUntil(',')
		if err != nil {
			break
		}

		// Read bucket values
		bucketValuesStr, err := ip.ReadUntil(';')
		if err != nil && err != io.EOF {
			break
		}
		bucketValues := strings.Split(bucketValuesStr, ",")

		// Set bucket labels and bucket values.
		// Convert percentage to exact ops count and compute 'less than or equal to' bucket values for Prometheus histograms.
		// Consider only non-zero buckets and the first zero bucket (since we are converting to 'less than or equal to' buckets)
		bucketValuesFloat := make([]float64, 1)
		bucketLabels := make([]string, 1)

		bucketLabels[0] = "+Inf"
		bucketValuesFloat[0], err = strconv.ParseFloat(bucketValues[0], 64)
		if err != nil {
			log.Error(err)
			break
		}

		for i := 1; i < len(bucketValues); i++ {
			val, err := strconv.ParseFloat(bucketValues[i], 64)
			if err != nil {
				log.Error(err)
				break
			}

			v := bucketValuesFloat[0] - ((val * bucketValuesFloat[0]) / 100)

			bucketLabels = append(bucketLabels, strings.Trim(bLabels[i], "><=ms"))
			bucketValuesFloat = append(bucketValuesFloat, v)

			if latencyBucketsCount > 0 && i >= latencyBucketsCount {
				// latency buckets count limit reached
				break
			}
		}

		// Sanity check
		if len(bucketLabels) != len(bucketValuesFloat) {
			log.Errorf("Error parsing latency values for node: `%s`. Bucket mismatch: buckets: `%v`, values: `%v`", commons.GetFullHost(), bucketLabels, bucketValuesFloat)
			break
		}

		stats := LatencyStatsMap{
			"bucketLabels": bucketLabels,
			"bucketValues": bucketValuesFloat,
			"timeUnit":     "ms",
		}

		if res[namespaceName] == nil {
			res[namespaceName] = LatencyStatsMap{
				operation: stats,
			}
		} else {
			res[namespaceName][operation] = stats
		}
	}

	return res
}

func readNamespaceAndOperation(ip *LatencyInfoParser) (string, string, error) {
	if err := ip.PeekAndExpect("batch-index"); err == nil {
		operation, err := ip.ReadUntil(':')
		return "", operation, err
	}

	if err := ip.Expect("{"); err != nil {
		if _, err := ip.ReadUntil(';'); err != nil {
			return "", "", err
		}
		return "", "", nil
	}

	// Get namespace name
	namespaceName, err := ip.ReadUntil('}')
	if err != nil {
		return "", "", err
	}

	if err := ip.Expect("-"); err != nil {
		return "", "", err
	}

	// Get operation (read, write etc.)
	operation, err := ip.ReadUntil(':')
	if err != nil {
		return "", "", err
	}
	return namespaceName, operation, err
}

// Parse "latencies:" info output
//
// Format (with and without latency data)
// {test}-write:msec,4234.9,28.75,7.40,1.63,0.26,0.03,0.00,0.00,0.00,0.00,0.00,0.00,0.00,0.00,0.00,0.00,0.00,0.00;
// {test}-read:;
func parseLatencyInfo(s string, latencyBucketsCount int) map[string]LatencyStatsMap {
	ip := NewInfoParser(s)
	res := map[string]LatencyStatsMap{}

	for {
		namespaceName, operation, err := readNamespaceAndOperation(ip)

		if err != nil {
			break
		}

		if namespaceName == "" && operation == "" {
			continue
		}

		// Might be an empty output due to no latency data available, so continue to next section
		if err := ip.PeekAndExpect(";"); err == nil {
			if err := ip.Expect(";"); err != nil {
				break
			}
			continue
		}

		// Get time unit - msec or usec
		timeUnit, err := ip.ReadUntil(',')
		if err != nil {
			break
		}

		// Simplify time unit for use in metric name
		timeUnit = strings.TrimSuffix(timeUnit, "ec")

		// Read bucket values
		bucketValuesStr, err := ip.ReadUntil(';')
		if err != nil && err != io.EOF {
			break
		}
		bucketValues := strings.Split(bucketValuesStr, ",")

		// Set bucket labels and bucket values.
		// Convert percentage to exact ops count and compute 'less than or equal to' bucket values for Prometheus histograms.
		// Consider only non-zero buckets and the first zero bucket (since we are converting to 'less than or equal to' buckets)
		bucketValuesFloat := make([]float64, 1)
		bucketLabels := make([]string, 1)

		bucketLabels[0] = "+Inf"
		bucketValuesFloat[0], err = strconv.ParseFloat(bucketValues[0], 64)
		if err != nil {
			log.Error(err)
			break
		}

		for i := 1; i < len(bucketValues); i++ {
			val, err := strconv.ParseFloat(bucketValues[i], 64)
			if err != nil {
				log.Error(err)
				break
			}

			v := bucketValuesFloat[0] - ((val * bucketValuesFloat[0]) / 100)

			bucketLabels = append(bucketLabels, strconv.Itoa(1<<(i-1)))
			bucketValuesFloat = append(bucketValuesFloat, v)

			if latencyBucketsCount > 0 && i >= latencyBucketsCount {
				// latency buckets count limit reached
				break
			}
		}

		// Sanity check
		if len(bucketLabels) != len(bucketValuesFloat) {
			log.Errorf("Error parsing latency values for node: `%s`. Bucket mismatch: buckets: `%v`, values: `%v`", commons.GetFullHost(), bucketLabels, bucketValuesFloat)
			break
		}

		stats := LatencyStatsMap{
			"bucketLabels": bucketLabels,
			"bucketValues": bucketValuesFloat,
			"timeUnit":     timeUnit,
		}

		if res[namespaceName] == nil {
			res[namespaceName] = LatencyStatsMap{
				operation: stats,
			}
		} else {
			res[namespaceName][operation] = stats
		}
	}

	return res
}
