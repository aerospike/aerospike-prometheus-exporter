package config

import (
	"os"

	"github.com/BurntSushi/toml"

	log "github.com/sirupsen/logrus"
)

var GaugeStatHandler GaugeStats

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
func InitGaugeStats(pGaugeStatsFile string) {

	log.Infof("Loading Gauge Stats file %s", pGaugeStatsFile)

	blob, err := os.ReadFile(pGaugeStatsFile)
	if err != nil {
		log.Fatalln(err)
	}

	md, err := toml.Decode(string(blob), &GaugeStatHandler)
	if err != nil {
		log.Fatalln(err)
	}

	// create maps from read stats, this is done as we check each stat if it is a gauge or not
	GaugeStatHandler.NamespaceStats = GaugeStatHandler.createMapFromArray(GaugeStatHandler.Namespace)
	GaugeStatHandler.NodeStats = GaugeStatHandler.createMapFromArray(GaugeStatHandler.Node)
	GaugeStatHandler.SindexStats = GaugeStatHandler.createMapFromArray(GaugeStatHandler.Sindex)
	GaugeStatHandler.SetsStats = GaugeStatHandler.createMapFromArray(GaugeStatHandler.Sets)
	GaugeStatHandler.XdrStats = GaugeStatHandler.createMapFromArray(GaugeStatHandler.Xdr)

	// Nullify/empty the Arrays to avoid duplicate stats-copy
	GaugeStatHandler.Namespace = nil
	GaugeStatHandler.Node = nil
	GaugeStatHandler.Sindex = nil
	GaugeStatHandler.Sets = nil
	GaugeStatHandler.Xdr = nil

	log.Debugln("# of Gauge Keys defined at Gauge Stat level are: ", len(md.Keys()))
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
