package statprocessors

import (
	"strings"

	commons "github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	config "github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"

	log "github.com/sirupsen/logrus"
)

type SindexStatsProcessor struct {
	sindexMetrics map[string]AerospikeStat
}

const (
	KEY_SINDEX_LIST_COMMAND = "sindex-list"
	KEY_SINDEX_COMMAND      = "sindex"
)

func (siw *SindexStatsProcessor) PassOneKeys() []string {
	if config.Cfg.Aerospike.DisableSindexMetrics {
		// disabled
		log.Tracef("sindex-passonekeys:nil")
		return nil
	}

	ge, err := isBuildVersionGreaterThanOrEqual(Build, "7.0.0.0")

	if err != nil {
		return nil
	}

	if ge {
		log.Tracef("sindex-passonekeys:%s", []string{KEY_SINDEX_LIST_COMMAND})
		return []string{KEY_SINDEX_LIST_COMMAND}
	}

	// older versions
	log.Tracef("sindex-passonekeys:%s", []string{KEY_SINDEX_COMMAND})
	return []string{KEY_SINDEX_COMMAND}
}

func (siw *SindexStatsProcessor) PassTwoKeys(rawMetrics map[string]string) (sindexCommands []string) {
	if config.Cfg.Aerospike.DisableSindexMetrics {
		// disabled
		return nil
	}

	log.Tracef("sindex:%v", rawMetrics["sindex"])

	sindexesMeta := strings.Split(rawMetrics["sindex"], ";")
	sindexCommands = siw.getSindexCommands(sindexesMeta)

	log.Tracef("sindex-passtwokeys:%s", sindexCommands)

	return sindexCommands
}

// getSindexCommands returns list of commands to fetch sindex statistics
func (siw *SindexStatsProcessor) getSindexCommands(sindexesMeta []string) (sindexCommands []string) {
	for _, sindex := range sindexesMeta {
		stats := commons.ParseStats(sindex, ":")
		sindexCommands = append(sindexCommands, "sindex/"+stats["ns"]+"/"+stats["indexname"])
	}

	return sindexCommands
}

func (siw *SindexStatsProcessor) Refresh(infoKeys []string, rawMetrics map[string]string) ([]AerospikeStat, error) {
	if config.Cfg.Aerospike.DisableSindexMetrics {
		// disabled
		return nil, nil
	}

	if siw.sindexMetrics == nil {
		siw.sindexMetrics = make(map[string]AerospikeStat)
	}

	var allMetricsToSend = []AerospikeStat{}

	for _, sindex := range infoKeys {
		if strings.HasPrefix(sindex, "sindex/") {
			log.Tracef("sindex-stats:%v:%v", sindex, rawMetrics[sindex])

			sindexInfoKey := strings.ReplaceAll(sindex, "sindex/", "")
			sindexInfoKeySplit := strings.Split(sindexInfoKey, "/")
			nsName := sindexInfoKeySplit[0]
			sindexName := sindexInfoKeySplit[1]
			log.Tracef("sindex-stats:%s:%s:%s", nsName, sindexName, rawMetrics[sindex])

			stats := commons.ParseStats(rawMetrics[sindex], ";")
			for stat, value := range stats {
				pv, err := commons.TryConvert(value)
				if err != nil {
					continue
				}
				asMetric, exists := siw.sindexMetrics[stat]

				if !exists {
					allowed := isMetricAllowed(commons.CTX_SINDEX, stat)
					asMetric = NewAerospikeStat(commons.CTX_SINDEX, stat, allowed)
					siw.sindexMetrics[stat] = asMetric
				}

				labels := []string{commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE, commons.METRIC_LABEL_NS, commons.METRIC_LABEL_SINDEX}
				labelValues := []string{ClusterName, Service, nsName, sindexName}

				asMetric.updateValues(pv, labels, labelValues)
				allMetricsToSend = append(allMetricsToSend, asMetric)

			}
		}

	}

	return allMetricsToSend, nil
}
