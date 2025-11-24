package statprocessors

import (
	"fmt"

	log "github.com/sirupsen/logrus"
)

const (
	USER_AGENTS = "user-agents"
)

type UserAgentsStatsProcessor struct {
	xdrMetrics map[string]AerospikeStat
}

func (xw *UserAgentsStatsProcessor) PassOneKeys() []string {
	// this is used to fetch the dcs metadata, we send same get-config command to fetch the dc-names required in next steps
	return nil
}

func (xw *UserAgentsStatsProcessor) PassTwoKeys(rawMetrics map[string]string) []string {
	log.Tracef("user-agent-passonekeys:%s", []string{USER_AGENTS})
	return []string{KEY_XDR_METADATA}
}

// refresh prom metrics - parse the given rawMetrics (both config and stats ) and push to given channel
func (xw *UserAgentsStatsProcessor) Refresh(infoKeys []string, rawMetrics map[string]string) ([]AerospikeStat, error) {

	if xw.xdrMetrics == nil {
		xw.xdrMetrics = make(map[string]AerospikeStat)
	}

	var allMetricsToSend = []AerospikeStat{}

	for _, key := range infoKeys {

		xdrRawMetrics := rawMetrics[key]
		fmt.Println(xdrRawMetrics)
	}

	return allMetricsToSend, nil
}
