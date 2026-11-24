package statprocessors

import (
	"testing"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestIsMetricAllowedAppliesAllowBlockAndGlob(t *testing.T) {
	useProcessorConfig(t)

	assert.True(t, isMetricAllowed(commons.CTX_NAMESPACE, "objects"))

	config.Cfg.Aerospike.NamespaceMetricsAllowlist = []string{"objects", "*bytes*"}
	assert.True(t, isMetricAllowed(commons.CTX_NAMESPACE, "objects"))
	assert.True(t, isMetricAllowed(commons.CTX_NAMESPACE, "data_used_bytes"))
	assert.False(t, isMetricAllowed(commons.CTX_NAMESPACE, "dead_partitions"))

	config.Cfg.Aerospike.NamespaceMetricsBlocklist = []string{"data_used_bytes"}
	assert.False(t, isMetricAllowed(commons.CTX_NAMESPACE, "data_used_bytes"), "a stat on both lists is blocked")
	assert.True(t, isMetricAllowed(commons.CTX_NAMESPACE, "objects"))

	config.Cfg.Aerospike.NodeMetricsAllowlist = []string{"client_connections"}
	assert.True(t, isMetricAllowed(commons.CTX_NODE_STATS, "client_connections"))
	assert.False(t, isMetricAllowed(commons.CTX_NODE_STATS, "objects"))

	config.Cfg.Aerospike.SetMetricsBlocklist = []string{"truncate*"}
	assert.False(t, isMetricAllowed(commons.CTX_SETS, "truncate-lut"))
	assert.True(t, isMetricAllowed(commons.CTX_SETS, "objects"))

	config.Cfg.Aerospike.SindexMetricsAllowlist = []string{"entries"}
	assert.True(t, isMetricAllowed(commons.CTX_SINDEX, "entries"))
	assert.False(t, isMetricAllowed(commons.CTX_SINDEX, "entries_per_rec"))

	config.Cfg.Aerospike.XdrMetricsBlocklist = []string{"*success*"}
	assert.False(t, isMetricAllowed(commons.CTX_XDR, "success"))
	assert.True(t, isMetricAllowed(commons.CTX_XDR, "lag"))
}
