package dataprovider

import (
	"testing"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/dataprovider"
)

func TestIsAerospikeDaemonName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{name: "comm", input: "asd", expected: true},
		{name: "host path", input: "/usr/bin/asd", expected: true},
		{name: "container path", input: "/opt/aerospike/bin/asd", expected: true},
		{name: "deleted binary", input: "/usr/bin/asd (deleted)", expected: true},
		{name: "other process", input: "/usr/bin/bash", expected: false},
		{name: "empty", input: "", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := dataprovider.IsAerospikeDaemonName(tt.input); got != tt.expected {
				t.Fatalf("IsAerospikeDaemonName(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestIsAsdShmSegment(t *testing.T) {
	asdPIDs := map[int]struct{}{152: {}, 288: {}}

	if !dataprovider.IsAsdShmSegment(152, 999, asdPIDs) {
		t.Fatal("expected cpid owned by asd")
	}

	if !dataprovider.IsAsdShmSegment(999, 288, asdPIDs) {
		t.Fatal("expected lpid owned by asd")
	}

	if dataprovider.IsAsdShmSegment(100, 200, asdPIDs) {
		t.Fatal("expected non-asd segment to be skipped")
	}

	if dataprovider.IsAsdShmSegment(152, 288, map[int]struct{}{}) {
		t.Fatal("expected empty asd pid set to reject segment")
	}
}
