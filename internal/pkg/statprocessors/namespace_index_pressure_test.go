package statprocessors

import (
	"testing"
	"time"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestCanSendIndexPressureHonorsConfiguredInterval(t *testing.T) {
	commons.InitConfigurations(commons.GetWatchersConfigFile(commons.TESTS_DEFAULT_CONFIG_FILE))
	config.Cfg.Aerospike.IndexPressureStatsFetchInterval = 3

	nw := NewNamespaceStatsProcessor(NewStatProcessorSharedState())
	assert.Equal(t, 3.0, nw.idxPressureFetchInterval)

	nw.isFlashStatSentByServer = true
	nw.idxPressurePreviousFetchTime = time.Now().Add(-4 * time.Minute)
	assert.True(t, nw.canSendIndexPressureInfoKey())

	nw.idxPressurePreviousFetchTime = time.Now()
	assert.False(t, nw.canSendIndexPressureInfoKey())

	config.Cfg.Aerospike.IndexPressureStatsFetchInterval = 0
	assert.False(t, nw.canSendIndexPressureInfoKey())
}
