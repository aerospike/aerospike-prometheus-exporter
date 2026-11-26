package statprocessors

import (
	"testing"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestXdrStatsProcessorRequestsDatacenterAndNamespaceKeys(t *testing.T) {
	state := useProcessorConfig(t)
	processor := NewXdrStatsProcessor(state)

	assert.Equal(t, []string{KEY_XDR_METADATA}, processor.PassOneKeys())

	state.CheckpointSaveInProgress = true
	assert.Nil(t, processor.PassOneKeys())
	assert.Nil(t, processor.PassTwoKeys(nil))
	state.CheckpointSaveInProgress = false

	keys := processor.PassTwoKeys(map[string]string{
		KEY_XDR_METADATA: "dcs=backup,",
		KEY_NS_METADATA:  "test",
	})
	assert.Equal(t, []string{
		"get-stats:context=xdr;dc=backup",
		"get-config:context=xdr;dc=backup",
		"get-config:context=xdr;dc=backup;namespace=test",
		"get-stats:context=xdr;dc=backup;namespace=test",
	}, keys)
}

func TestXdrStatsProcessorRefreshNamesMetricsByScope(t *testing.T) {
	state := useProcessorConfig(t)
	config.Cfg.Aerospike.XdrMetricsAllowlist = []string{"lag"}
	config.GaugeStatHandler.XdrStats["dc_recoveries_pending"] = true

	processor := NewXdrStatsProcessor(state)
	statKey := "get-stats:context=xdr;dc=backup"
	configKey := "get-config:context=xdr;dc=backup"
	nsKey := "get-stats:context=xdr;dc=backup;namespace=test"

	stats, err := processor.Refresh([]string{statKey}, map[string]string{statKey: "lag=4;success=9"})
	require.NoError(t, err)
	lag, ok := findStat(stats, "lag")
	require.True(t, ok)
	assert.Equal(t, 4.0, lag.Value)
	assert.True(t, lag.IsAllowed)
	assert.Equal(t, commons.MetricTypeCounter, metricTypeOf(lag))
	assertLabel(t, lag, commons.METRIC_LABEL_DC_NAME, "backup")
	success, ok := findStat(stats, "success")
	require.True(t, ok)
	assert.False(t, success.IsAllowed)

	processor = NewXdrStatsProcessor(state)
	stats, err = processor.Refresh([]string{configKey}, map[string]string{configKey: "recoveries_pending=2"})
	require.NoError(t, err)
	pending, ok := findStat(stats, "dc_recoveries_pending")
	require.True(t, ok)
	assert.Equal(t, 2.0, pending.Value)
	assert.Equal(t, commons.MetricTypeGauge, metricTypeOf(pending))

	processor = NewXdrStatsProcessor(state)
	stats, err = processor.Refresh([]string{nsKey}, map[string]string{nsKey: "lag=7"})
	require.NoError(t, err)
	nsLag, ok := findStat(stats, "dc_namespace_lag")
	require.True(t, ok)
	assert.Equal(t, 7.0, nsLag.Value)
	assertLabel(t, nsLag, commons.METRIC_LABEL_NS, "test")
	assertLabel(t, nsLag, commons.METRIC_LABEL_DC_NAME, "backup")
}
