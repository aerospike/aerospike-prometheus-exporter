package executors

import (
	"context"
	"testing"

	aero "github.com/aerospike/aerospike-client-go/v8"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/statprocessors"
	"github.com/stretchr/testify/assert"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

type connectedProvider struct{}

func (connectedProvider) RequestInfo([]string) (map[string]string, error) { return nil, nil }
func (connectedProvider) FetchUsersDetails() (bool, []*aero.UserRoles, error) {
	return false, nil, nil
}
func (connectedProvider) IsServerConnected() bool { return true }

func jobStat(context commons.ContextType, name string, mType commons.MetricType, jobID string, state string) statprocessors.AerospikeStat {
	return statprocessors.AerospikeStat{
		Context:     context,
		Name:        name,
		MType:       mType,
		IsAllowed:   true,
		Value:       1,
		Labels:      []string{commons.METRIC_LABEL_JOB_ID, commons.METRIC_LABEL_STATE},
		LabelValues: []string{jobID, state},
	}
}

// collectStates returns the "state" attribute of every data point reported for a metric.
func collectStates(t *testing.T, reader *sdkmetric.ManualReader, metricName string) []string {
	var rm metricdata.ResourceMetrics
	assert.NoError(t, reader.Collect(context.Background(), &rm))

	states := []string{}
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != metricName {
				continue
			}

			switch data := m.Data.(type) {
			case metricdata.Gauge[int64]:
				for _, dp := range data.DataPoints {
					v, _ := dp.Attributes.Value(commons.METRIC_LABEL_STATE)
					states = append(states, v.AsString())
				}
			case metricdata.Sum[int64]:
				for _, dp := range data.DataPoints {
					v, _ := dp.Attributes.Value(commons.METRIC_LABEL_STATE)
					states = append(states, v.AsString())
				}
			}
		}
	}

	return states
}

// Test_Otel_PerJobSeriesAreSwept checks that a backup job's previous state, and a job that is no
// longer reported, stop being exported, while series from other contexts are left alone.
func Test_Otel_PerJobSeriesAreSwept(t *testing.T) {
	config.Cfg.Agent.Otel.MetricNamePrefix = "aerospike.server"
	config.Cfg.Agent.Otel.MetricContextSeparator = "period"
	config.Cfg.Agent.Otel.AllMetricsAsGauge = false

	reader := sdkmetric.NewManualReader()
	meter := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader)).Meter("test")

	oe := &OtelExecutor{
		gauges:       make(map[string]*GaugeMetrics),
		counters:     make(map[string]*CounterMetrics),
		perJobKeys:   make(map[string]struct{}),
		dataProvider: connectedProvider{},
	}

	nsStat := jobStat(commons.CTX_NAMESPACE, "master_objects", commons.MetricTypeGauge, "-", "-")

	// first refresh: the job is in INIT, and a restore counter and a namespace stat are reported
	oe.processAndPushStats(meter, nil, []statprocessors.AerospikeStat{
		jobStat(commons.CTX_BACKUP, "job", commons.MetricTypeGauge, "b1", "INIT"),
		jobStat(commons.CTX_RESTORE, "applied", commons.MetricTypeCounter, "r1", "ACTIVE"),
		nsStat,
	}, true)

	assert.Equal(t, []string{"INIT"}, collectStates(t, reader, "aerospike.server.backup.job"))
	assert.Equal(t, []string{"ACTIVE"}, collectStates(t, reader, "aerospike.server.restore.applied"))

	// second refresh: the job moved on, the restore job aged out, and the namespace stat is absent once
	oe.processAndPushStats(meter, nil, []statprocessors.AerospikeStat{
		jobStat(commons.CTX_BACKUP, "job", commons.MetricTypeGauge, "b1", "BASE_SCAN_ACTIVE"),
	}, true)

	assert.Equal(t, []string{"BASE_SCAN_ACTIVE"}, collectStates(t, reader, "aerospike.server.backup.job"),
		"the INIT series must stop reporting once the job changes state")
	assert.Empty(t, collectStates(t, reader, "aerospike.server.restore.applied"),
		"a job that is no longer reported must stop being exported")
	assert.Len(t, collectStates(t, reader, "aerospike.server.namespace.master_objects"), 1,
		"non-job series are never swept")

	// a refresh that does not sweep (system info) must not remove backup series
	oe.processAndPushStats(meter, nil, nil, false)
	assert.Equal(t, []string{"BASE_SCAN_ACTIVE"}, collectStates(t, reader, "aerospike.server.backup.job"))

	// no backup jobs left
	oe.processAndPushStats(meter, nil, nil, true)
	assert.Empty(t, collectStates(t, reader, "aerospike.server.backup.job"))
	assert.Empty(t, oe.perJobKeys)
}

// Test_Otel_BlockedStatsAreNotExported checks that a stat filtered out by an allowlist or blocklist
// (IsAllowed=false) never becomes an OTel series, matching the Prometheus path.
func Test_Otel_BlockedStatsAreNotExported(t *testing.T) {
	config.Cfg.Agent.Otel.MetricNamePrefix = "aerospike.server"
	config.Cfg.Agent.Otel.MetricContextSeparator = "period"
	config.Cfg.Agent.Otel.AllMetricsAsGauge = false

	reader := sdkmetric.NewManualReader()
	meter := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader)).Meter("test")

	oe := &OtelExecutor{
		gauges:       make(map[string]*GaugeMetrics),
		counters:     make(map[string]*CounterMetrics),
		perJobKeys:   make(map[string]struct{}),
		dataProvider: connectedProvider{},
	}

	allowed := jobStat(commons.CTX_NAMESPACE, "master_objects", commons.MetricTypeGauge, "-", "-")
	blocked := jobStat(commons.CTX_NAMESPACE, "client_read_success", commons.MetricTypeCounter, "-", "-")
	blocked.IsAllowed = false
	blockedJob := jobStat(commons.CTX_BACKUP, "job", commons.MetricTypeGauge, "b1", "INIT")
	blockedJob.IsAllowed = false

	oe.processAndPushStats(meter, nil, []statprocessors.AerospikeStat{allowed, blocked, blockedJob}, true)

	assert.Len(t, collectStates(t, reader, "aerospike.server.namespace.master_objects"), 1)
	assert.Empty(t, collectStates(t, reader, "aerospike.server.namespace.client_read_success"),
		"a blocked stat must not be exported")
	assert.Empty(t, collectStates(t, reader, "aerospike.server.backup.job"),
		"a blocked backup stat must not be exported")
	assert.Len(t, oe.gauges, 1)
	assert.Empty(t, oe.counters)
}
