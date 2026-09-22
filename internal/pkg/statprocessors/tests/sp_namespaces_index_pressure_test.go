package statprocessors

import (
	"testing"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/dataprovider"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/statprocessors"
	"github.com/stretchr/testify/assert"
)

func Test_Namespace_IndexPressure_DefaultIntervalFromConfig(t *testing.T) {
	commons.InitConfigurations(commons.GetWatchersConfigFile(commons.TESTS_DEFAULT_CONFIG_FILE))

	assert.Equal(t, uint8(10), config.Cfg.Aerospike.IndexPressureStatsFetchInterval)
}

func Test_Namespace_IndexPressure_NotInPassTwoWhenIntervalZero(t *testing.T) {
	commons.InitConfigurations(commons.GetWatchersConfigFile(commons.TESTS_DEFAULT_CONFIG_FILE))
	config.Cfg.Aerospike.IndexPressureStatsFetchInterval = 0

	sharedState := statprocessors.NewStatProcessorSharedState()
	nsWatcher := statprocessors.NewNamespaceStatsProcessor(sharedState)

	passOneOutput, err := dataprovider.GetProvider("mock").RequestInfo(nsWatcher.PassOneKeys())
	assert.NoError(t, err)

	passTwoKeys := nsWatcher.PassTwoKeys(passOneOutput)
	assert.NotContains(t, passTwoKeys, "index-pressure")
}

func Test_Namespace_IndexPressure_LoadsIntervalFromToml(t *testing.T) {
	commons.InitConfigurations(commons.GetWatchersConfigFile("tests_data/index_pressure_interval_ape.toml"))
	assert.Equal(t, uint8(7), config.Cfg.Aerospike.IndexPressureStatsFetchInterval)
}

func Test_Namespace_IndexPressure_UsesConfiguredIntervalAtProcessorInit(t *testing.T) {
	commons.InitConfigurations(commons.GetWatchersConfigFile(commons.TESTS_DEFAULT_CONFIG_FILE))
	config.Cfg.Aerospike.IndexPressureStatsFetchInterval = 7

	sharedState := statprocessors.NewStatProcessorSharedState()
	nsWatcher := statprocessors.NewNamespaceStatsProcessor(sharedState)

	passOneOutput, err := dataprovider.GetProvider("mock").RequestInfo(nsWatcher.PassOneKeys())
	assert.NoError(t, err)

	// Flash index not observed yet and interval window not elapsed on first pass-two.
	passTwoKeys := nsWatcher.PassTwoKeys(passOneOutput)
	assert.NotContains(t, passTwoKeys, "index-pressure")

	// Disabling via config is honored on subsequent pass-two calls.
	config.Cfg.Aerospike.IndexPressureStatsFetchInterval = 0
	passTwoKeys = nsWatcher.PassTwoKeys(passOneOutput)
	assert.NotContains(t, passTwoKeys, "index-pressure")
}
