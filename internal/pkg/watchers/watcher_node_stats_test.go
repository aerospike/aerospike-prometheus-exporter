package watchers

import (
	"fmt"
	"testing"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"
	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/data"
	tests_utils "github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/tests_utils"
	"github.com/stretchr/testify/assert"
)

func Test_Node_PassOneKeys(t *testing.T) {

	fmt.Println("initializing config ... Test_Node_PassOneKeys")

	// Check passoneKeys
	nodeWatcher := &NodeStatsWatcher{}
	nwPassOneKeys := nodeWatcher.PassOneKeys()

	udh := &tests_utils.UnittestDataHandler{}
	ndv := udh.GetUnittestValidator("node")
	passOneOutputs := ndv.GetPassOneKeys(*udh)

	assert.Nil(t, nwPassOneKeys, passOneOutputs)

}

func Test_Node_PassTwoKeys(t *testing.T) {

	fmt.Println("initializing config ... Test_Node_PassTwoKeys")

	// initialize config and gauge-lists
	config.InitConfig(tests_utils.GetConfigfileLocation(tests_utils.TESTS_DEFAULT_CONFIG_FILE))

	// Check passoneKeys
	nodeWatcher := &NodeStatsWatcher{}
	nwPassOneKeys := nodeWatcher.PassOneKeys()
	passOneOutput, _ := data.GetProvider().RequestInfo(nwPassOneKeys)
	fmt.Println("TestPassTwoKeys: passOneOutput: ", passOneOutput)
	passTwokeyOutputs := nodeWatcher.PassTwoKeys(passOneOutput)

	udh := &tests_utils.UnittestDataHandler{}
	ndv := udh.GetUnittestValidator("node")
	passTwoOutputs := ndv.GetPassTwoKeys(*udh)

	fmt.Println("Node Watcher - passTwokeyOutputs: ", passTwokeyOutputs)
	fmt.Println("Node Watcher - passOneOutputs: ", passTwoOutputs)

}

func Test_Node_RefreshDefault(t *testing.T) {

	fmt.Println("initializing config ... Test_Node_RefreshDefault")
	// Initialize and validate config

	// initialize config and gauge-lists
	config.InitConfig(tests_utils.GetConfigfileLocation(tests_utils.TESTS_DEFAULT_CONFIG_FILE))

	node_runTestcase(t)
}

/**
* complete logic to call watcher, generate-mock data and asset is part of this function
 */
func node_runTestcase(t *testing.T) {

}
