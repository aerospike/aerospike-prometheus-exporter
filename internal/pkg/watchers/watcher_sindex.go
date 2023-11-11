package watchers

import (
	"strings"

	commons "github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	config "github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/config"

	log "github.com/sirupsen/logrus"
)

type SindexWatcher struct {
	sindexMetrics map[string]commons.AerospikeStat
}

func (siw *SindexWatcher) PassOneKeys() []string {
	if config.Cfg.Aerospike.DisableSindexMetrics {
		// disabled
		return nil
	}

	return []string{"sindex"}
}

func (siw *SindexWatcher) PassTwoKeys(rawMetrics map[string]string) (sindexCommands []string) {
	if config.Cfg.Aerospike.DisableSindexMetrics {
		// disabled
		return nil
	}

	sindexesMeta := strings.Split(rawMetrics["sindex"], ";")
	sindexCommands = siw.getSindexCommands(sindexesMeta)

	return sindexCommands
}

// getSindexCommands returns list of commands to fetch sindex statistics
func (siw *SindexWatcher) getSindexCommands(sindexesMeta []string) (sindexCommands []string) {
	for _, sindex := range sindexesMeta {
		stats := commons.ParseStats(sindex, ":")
		sindexCommands = append(sindexCommands, "sindex/"+stats["ns"]+"/"+stats["indexname"])
	}

	return sindexCommands
}

func (siw *SindexWatcher) Refresh(infoKeys []string, rawMetrics map[string]string) ([]WatcherMetric, error) {
	if config.Cfg.Aerospike.DisableSindexMetrics {
		// disabled
		return nil, nil
	}

	if siw.sindexMetrics == nil {
		siw.sindexMetrics = make(map[string]commons.AerospikeStat)
	}

	var metrics_to_send = []WatcherMetric{}

	for _, sindex := range infoKeys {
		sindexInfoKey := strings.ReplaceAll(sindex, "sindex/", "")
		sindexInfoKeySplit := strings.Split(sindexInfoKey, "/")
		nsName := sindexInfoKeySplit[0]
		sindexName := sindexInfoKeySplit[1]
		log.Tracef("sindex-stats:%s:%s:%s", nsName, sindexName, rawMetrics[sindex])

		clusterName := rawMetrics[commons.Infokey_ClusterName]
		service := rawMetrics[commons.Infokey_Service]

		stats := commons.ParseStats(rawMetrics[sindex], ";")
		for stat, value := range stats {
			pv, err := commons.TryConvert(value)
			if err != nil {
				continue
			}
			asMetric, exists := siw.sindexMetrics[stat]

			if !exists {
				asMetric = commons.NewAerospikeStat(commons.CTX_SINDEX, stat)
				siw.sindexMetrics[stat] = asMetric
			}

			labels := []string{commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE, commons.METRIC_LABEL_NS, commons.METRIC_LABEL_SINDEX}
			labelValues := []string{clusterName, service, nsName, sindexName}

			// pushToPrometheus(asMetric, pv, labels, labelsValues, ch)
			metrics_to_send = append(metrics_to_send, WatcherMetric{asMetric, pv, labels, labelValues})

		}

	}

	return metrics_to_send, nil
}
