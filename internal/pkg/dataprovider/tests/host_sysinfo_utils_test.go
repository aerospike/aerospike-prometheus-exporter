package dataprovider

import (
	"testing"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/dataprovider"
)

func TestIsAerospikeShmSegment(t *testing.T) {
	tests := []struct {
		name     string
		key      int32
		expected bool
	}{
		{name: "pi base", key: -1375727360, expected: true},
		{name: "data stripe 0", key: -1392504832, expected: true},
		{name: "data stripe 8", key: -1392504825, expected: true},
		{name: "non-aerospike key", key: 12345, expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := dataprovider.IsAerospikeShmSegment(tt.key); got != tt.expected {
				t.Fatalf("IsAerospikeShmSegment(%d) = %v, want %v", tt.key, got, tt.expected)
			}
		})
	}
}
