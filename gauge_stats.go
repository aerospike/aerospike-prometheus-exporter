package main

import (
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
func initGaugeStats(pGaugeStatsFile string, pGaugeStats *GaugeStats) {

	log.Infof("Loading Gauge Stats file %s", pGaugeStatsFile)

	// fmt.Println("Loading Gauge Stats file ", configFile)
	blob, err := os.ReadFile(pGaugeStatsFile)
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

	switch pContextType {
	case CTX_NAMESPACE:
		return gm.NamespaceStats[pStat]
	case CTX_NODE_STATS:
		return gm.NodeStats[pStat]
	case CTX_SETS:
		return gm.SetsStats[pStat]
	case CTX_SINDEX:
		return gm.SindexStats[pStat]
	case CTX_XDR:
		return gm.XdrStats[pStat]
	}

	return false
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
