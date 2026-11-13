package statprocessors

import (
	"testing"
	"time"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestIndexPressureFollowsConfiguredInterval(t *testing.T) {
	original := config.Cfg
	t.Cleanup(func() { config.Cfg = original })

	config.Cfg = config.Config{}
	config.Cfg.Aerospike.IndexPressureStatsFetchInterval = 3

	nw := NewNamespaceStatsProcessor(NewStatProcessorSharedState())
	assert.Equal(t, 3.0, nw.idxPressureFetchInterval)

	passOne := map[string]string{KEY_NS_METADATA: "test"}
	nw.isFlashStatSentByServer = false
	nw.idxPressurePreviousFetchTime = time.Now().Add(-4 * time.Minute)
	assert.False(t, nw.canSendIndexPressureInfoKey())
	assert.NotContains(t, nw.PassTwoKeys(passOne), KEY_NS_INDEX_PRESSURE)

	nw.isFlashStatSentByServer = true
	nw.idxPressurePreviousFetchTime = time.Now().Add(-4 * time.Minute)
	assert.True(t, nw.canSendIndexPressureInfoKey())
	assert.Contains(t, nw.PassTwoKeys(passOne), KEY_NS_INDEX_PRESSURE)
	assert.NotContains(t, nw.PassTwoKeys(passOne), KEY_NS_INDEX_PRESSURE)

	config.Cfg.Aerospike.IndexPressureStatsFetchInterval = 0
	nw.idxPressurePreviousFetchTime = time.Now().Add(-4 * time.Minute)
	assert.False(t, nw.canSendIndexPressureInfoKey())
	assert.NotContains(t, nw.PassTwoKeys(passOne), KEY_NS_INDEX_PRESSURE)
}
