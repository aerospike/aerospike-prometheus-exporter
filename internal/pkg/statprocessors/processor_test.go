package statprocessors

import (
	"io"
	"testing"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"
	log "github.com/sirupsen/logrus"
)

func useProcessorConfig(t *testing.T) *StatProcessorSharedState {
	t.Helper()

	originalCfg := config.Cfg
	originalGauges := config.GaugeStatHandler
	logger := log.StandardLogger()
	prevOut := logger.Out
	prevLevel := logger.GetLevel()
	logger.SetOutput(io.Discard)
	t.Cleanup(func() {
		config.Cfg = originalCfg
		config.GaugeStatHandler = originalGauges
		logger.SetOutput(prevOut)
		logger.SetLevel(prevLevel)
	})

	config.Cfg = config.Config{}
	config.GaugeStatHandler = config.GaugeStats{
		NamespaceStats: map[string]bool{},
		NodeStats:      map[string]bool{},
		SetsStats:      map[string]bool{},
		SindexStats:    map[string]bool{},
		XdrStats:       map[string]bool{},
		SysInfoStats:   map[string]bool{},
	}

	state := NewStatProcessorSharedState()
	state.ClusterName = "cluster-a"
	state.Service = "127.0.0.1:3000"
	state.Build = "7.2.0.11"
	return state
}

func findStat(stats []AerospikeStat, name string) (AerospikeStat, bool) {
	for _, stat := range stats {
		if stat.Name == name {
			return stat, true
		}
	}
	return AerospikeStat{}, false
}

func findStatWithLabel(stats []AerospikeStat, name string, label string, value string) (AerospikeStat, bool) {
	for _, stat := range stats {
		if stat.Name != name {
			continue
		}
		for i, labelName := range stat.Labels {
			if labelName == label && stat.LabelValues[i] == value {
				return stat, true
			}
		}
	}
	return AerospikeStat{}, false
}

func assertLabel(t *testing.T, stat AerospikeStat, label string, value string) {
	t.Helper()
	for i, labelName := range stat.Labels {
		if labelName == label {
			if stat.LabelValues[i] != value {
				t.Fatalf("label %s = %q, want %q", label, stat.LabelValues[i], value)
			}
			return
		}
	}
	t.Fatalf("metric %s has no label %s", stat.Name, label)
}

func countNamed(stats []AerospikeStat, name string) int {
	count := 0
	for _, stat := range stats {
		if stat.Name == name {
			count++
		}
	}
	return count
}

func metricTypeOf(stat AerospikeStat) commons.MetricType {
	return stat.MType
}
