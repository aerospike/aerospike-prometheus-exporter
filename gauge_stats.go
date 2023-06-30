package main

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"

	log "github.com/sirupsen/logrus"
)

/**
 * Defines the structure which holds various Gauge stats for each contexts from a toml file
 */
type GaugeStats struct {

	// used to read from toml config file, these array will be emptied once read and stats are mapped to a file
	Namespace []string `toml:"namespace_gauge_stats"`
	Node      []string `toml:"node_gauge_stats"`
	Sets      []string `toml:"sets_gauge_stats"`
	Sindex    []string `toml:"sindex_gauge_stats"`
	Xdr       []string `toml:"xdr_gauge_stats"`

	// why below maps?
	// all gauge stats are added and mapped as true after reading from toml file
	NamespaceStats map[string]bool
	NodeStats      map[string]bool
	SetsStats      map[string]bool
	SindexStats    map[string]bool
	XdrStats       map[string]bool
}

// Initialize exporter configuration
func initGaugeStats(pConfigFile string, pGaugeStats *GaugeStats) {

	fmt.Println("Loading Gauge Stats file ", configFile)
	blob, err := os.ReadFile(pConfigFile)
	if err != nil {
		log.Fatalln(err)
	}

	md, err := toml.Decode(string(blob), &pGaugeStats)
	if err != nil {
		log.Fatalln(err)
	}

	// create maps from read stats, this is done as we check each stat if it is a gauge or not
	pGaugeStats.NamespaceStats = pGaugeStats.createMapFromArray(pGaugeStats.Namespace)
	pGaugeStats.NodeStats = pGaugeStats.createMapFromArray(pGaugeStats.Node)
	pGaugeStats.SindexStats = pGaugeStats.createMapFromArray(pGaugeStats.Sindex)
	pGaugeStats.SetsStats = pGaugeStats.createMapFromArray(pGaugeStats.Sets)
	pGaugeStats.XdrStats = pGaugeStats.createMapFromArray(pGaugeStats.Xdr)

	// Nullify/empty the Arrays to avoid duplicate stats-copy
	pGaugeStats.Namespace = nil
	pGaugeStats.Node = nil
	pGaugeStats.Sindex = nil
	pGaugeStats.Sets = nil
	pGaugeStats.Xdr = nil

	log.Debugln("# of Gauge Keys defined at Gauge Stat level are: ", len(md.Keys()))
}

/**
 * Check if given stat is a Gauge in a given context like Node, Namespace etc.,
 */
func (gm *GaugeStats) isGauge(pContextType ContextType, pStat string) bool {

	if CTX_NAMESPACE == pContextType {
		return gm.NamespaceStats[pStat]
	} else if CTX_NODE_STATS == pContextType {
		return gm.NodeStats[pStat]
	} else if CTX_SETS == pContextType {
		return gm.SetsStats[pStat]
	} else if CTX_SINDEX == pContextType {
		return gm.SindexStats[pStat]
	} else if CTX_XDR == pContextType {
		return gm.XdrStats[pStat]
	}

	return false
}

/**
 * getter returns all the Gauge stats in requested context
 */
func (gm *GaugeStats) getGaugeStats(pContextType ContextType) []string {

	if CTX_NAMESPACE == pContextType {
		return gm.fetAllGaugeStats(gm.NamespaceStats)
	} else if CTX_NODE_STATS == pContextType {
		return gm.fetAllGaugeStats(gm.NodeStats)
	} else if CTX_SETS == pContextType {
		return gm.fetAllGaugeStats(gm.SetsStats)
	} else if CTX_SINDEX == pContextType {
		return gm.fetAllGaugeStats(gm.SindexStats)
	} else if CTX_XDR == pContextType {
		return gm.fetAllGaugeStats(gm.XdrStats)
	}

	return nil
}

/**
 * Utility, common logic used to loop through contextual-array like Nodes, Sets etc.,
 */
func (gm *GaugeStats) fetAllGaugeStats(statsMap map[string]bool) []string {
	keys := []string{}
	for k := range statsMap {
		keys = append(keys, k)
	}
	return keys
}

/**
 * Utility, common logic used to loop through contextual-array like Nodes, Sets etc.,
 */
func (gm *GaugeStats) createMapFromArray(pArrStats []string) map[string]bool {
	statsMap := make(map[string]bool)
	for _, stat := range pArrStats {
		statsMap[stat] = true
	}
	return statsMap
}
