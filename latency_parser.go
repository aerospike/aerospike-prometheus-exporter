package main

import (
	"fmt"
	"io"
	"log"
	"math"
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

		ns, err := ip.ReadUntil('}')
		if err != nil {
			break
		}

		if err := ip.Expect("-"); err != nil {
			break
		}

		op, err := ip.ReadUntil(':')
		if err != nil {
			break
		}

		timestamp, err := ip.ReadUntil(',')
		if err != nil {
			break
		}

		if _, err := ip.ReadUntil(','); err != nil {
			break
		}

		bucketsStr, err := ip.ReadUntil(';')
		if err != nil {
			break
		}
		buckets := strings.Split(bucketsStr, ",")

		_, err = ip.ReadUntil(',')
		if err != nil {
			break
		}

		opsCount, err := ip.ReadFloat(',')
		if err != nil {
			break
		}

		valBucketsStr, err := ip.ReadUntil(';')
		if err != nil && err != io.EOF {
			break
		}
		valBuckets := strings.Split(valBucketsStr, ",")
		valBucketsFloat := make([]float64, len(valBuckets))
		for i := range valBuckets {
			valBucketsFloat[i], _ = strconv.ParseFloat(valBuckets[i], 64)
		}

		// calc precise in-between percents
		lineAggPct := float64(0)
		for i := len(valBucketsFloat) - 1; i > 0; i-- {
			lineAggPct += valBucketsFloat[i]
			valBucketsFloat[i-1] = math.Max(0, valBucketsFloat[i-1]-lineAggPct)
		}

		if len(buckets) != len(valBuckets) {
			log.Println(fmt.Errorf("error parsing latency values for node: `%s`. Bucket mismatch: buckets: `%s`, values: `%s`", fullHost, bucketsStr, valBucketsStr))
			break
		}

		// log.Println(s)
		for i := range valBucketsFloat {
			// log.Println(">>>>>>>>", ns, op, buckets[i], valBucketsFloat[i], opsCount)
			valBucketsFloat[i] *= (opsCount / 100)
		}

		stats := StatsMap{
			"tps":        opsCount,
			"timestamp":  timestamp,
			"buckets":    buckets,
			"valBuckets": valBucketsFloat,
		}

		if res[ns] == nil {
			res[ns] = StatsMap{
				op: stats,
			}
		} else {
			res[ns][op] = stats
		}
	}

	return res
}
