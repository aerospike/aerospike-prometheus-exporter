package main

import (
	"io"
	"log"
	"strconv"
	"strings"
)

func parseLatencyInfo(s string) map[string]StatsMap {
	ip := NewInfoParser(s)

	//typical format is {test}-read:10:17:37-GMT,ops/sec,>1ms,>8ms,>64ms;10:17:47,29648.2,3.44,0.08,0.00;

	res := map[string]StatsMap{}
	for {
		if err := ip.Expect("{"); err != nil {
			// it's an error string, read to next section
			if _, err := ip.ReadUntil(';'); err != nil {
				break
			}
			continue
		}

		// Get namespace name
		namespaceName, err := ip.ReadUntil('}')
		if err != nil {
			break
		}

		if err := ip.Expect("-"); err != nil {
			break
		}

		// Get operation (read, write etc.)
		operation, err := ip.ReadUntil(':')
		if err != nil {
			break
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
		bucketLabels := strings.Split(bucketLabelsStr, ",")

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

		// Convert bucket values to float64
		bucketValuesFloat := make([]float64, len(bucketValues))
		for i := range bucketValues {
			bucketValuesFloat[i], _ = strconv.ParseFloat(bucketValues[i], 64)
		}

		// Sanity check
		if len(bucketLabels) != len(bucketValuesFloat) {
			log.Printf("Error parsing latency values for node: `%s`. Bucket mismatch: buckets: `%s`, values: `%s`", fullHost, bucketLabelsStr, bucketValuesStr)
			break
		}

		// Replace 'ops/sec' with '+Inf' for Prometheus histograms
		bucketLabels[0] = "+Inf"

		// Set bucket labels and convert bucket values from percentage to exact ops count.
		// Compute 'less than or equal to' bucket values for Prometheus histograms.
		for i := 1; i < len(bucketValuesFloat); i++ {
			bucketLabels[i] = strings.Trim(bucketLabels[i], "><=ms")
			bucketValuesFloat[i] = bucketValuesFloat[0] - ((bucketValuesFloat[i] * bucketValuesFloat[0]) / 100)
		}

		stats := StatsMap{
			"bucketLabels": bucketLabels,
			"bucketValues": bucketValuesFloat,
		}

		if res[namespaceName] == nil {
			res[namespaceName] = StatsMap{
				operation: stats,
			}
		} else {
			res[namespaceName][operation] = stats
		}
	}

	return res
}
