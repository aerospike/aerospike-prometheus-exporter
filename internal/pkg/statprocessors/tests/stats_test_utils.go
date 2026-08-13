package statprocessors

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/statprocessors"
	"github.com/stretchr/testify/assert"
)

var aerospikeStatValuePattern = regexp.MustCompile(`, Value:[^,]+`)

func aerospikeStatKeyIgnoringValue(stat statprocessors.AerospikeStat) string {
	return fmt.Sprintf(
		"statprocessors.AerospikeStat{Context:%#v, Name:%#v, MType:%#v, IsAllowed:%v, IsConfig:%v, Labels:%#v, LabelValues:%#v}",
		stat.Context, stat.Name, stat.MType, stat.IsAllowed, stat.IsConfig, stat.Labels, stat.LabelValues,
	)
}

func aerospikeStatKeyFromExpectedLine(line string) string {
	return aerospikeStatValuePattern.ReplaceAllString(line, "")
}

func expectedStatKeysIgnoringValue(expectedResults map[string]string) map[string]struct{} {
	keys := make(map[string]struct{}, len(expectedResults))
	for line := range expectedResults {
		keys[aerospikeStatKeyFromExpectedLine(line)] = struct{}{}
	}
	return keys
}

func assertAerospikeStatInExpectedResults(t *testing.T, stat statprocessors.AerospikeStat, expectedKeys map[string]struct{}) {
	t.Helper()
	key := aerospikeStatKeyIgnoringValue(stat)
	_, exists := expectedKeys[key]
	assert.True(t, exists, "Failed, did not find expected result (value ignored): "+fmt.Sprintf("%#v", stat))
}
