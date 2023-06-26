package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/BurntSushi/toml"

	log "github.com/sirupsen/logrus"
)

/**
 * Defines the structure which holds various Guage stats for each contexts from a toml file
 */
type GaugeStats struct {
	Namespace []string `toml:"namespace_gauge_stats"`
	Node      []string `toml:"node_gauge_stats"`
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
	// log.Debugln("# of Guage Keys defined at Gauge Stat level are: ", len(md.Keys()))
	fmt.Println("# of Guage Keys defined at Gauge Stat level are: ", len(md.Keys()))
}

/**
 * Check if given stat is a Guage in a given context like Node, Namespace etc.,
 */
func (gm *GaugeStats) isGauge(pContextType ContextType, pStat string) bool {

	if CTX_NAMESPACE == pContextType {
		return gm.isExistsInArray(pStat, gm.Namespace)
	} else if CTX_NODE_STATS == pContextType {
		return gm.isExistsInArray(pStat, gm.Node)
	}

	return false
}

/**
 * getter returns all the Guage stats in requested context
 */
func (gm *GaugeStats) getGaugeStats(pContextType ContextType) []string {

	if CTX_NAMESPACE == pContextType {
		return gm.Namespace
	} else if CTX_NODE_STATS == pContextType {
		return gm.Node
	}

	return nil
}

/**
 * Utility, common logic used to loop through contextual-array like Nodes, Sets etc.,
 */
func (gm *GaugeStats) isExistsInArray(pStat string, pArrStats []string) bool {
	if len(pArrStats) > 0 {
		for _, stat := range pArrStats {
			if strings.EqualFold(stat, pStat) {
				return true
			}
		}
	}
	return false
}
