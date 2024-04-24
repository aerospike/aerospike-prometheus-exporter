package statprocessors

import (
	"strconv"
	"strings"
	"time"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	log "github.com/sirupsen/logrus"
)

var (
	MAX_PEERS_PER_LABEL = 32

	// time interval to fetch index-pressure
	serverPeersFetchInterval = 0.1

	// Time when  Server Peers were last-fetched
	serverPeersPreviousFetchTime = time.Now()
)

type GlobalStatsProcessor struct {
}

func (sw *GlobalStatsProcessor) PassOneKeys() []string {
	log.Tracef("globals-passtwokeys:nil")

	return nil

}

func (sw *GlobalStatsProcessor) PassTwoKeys(rawMetrics map[string]string) []string {
	var cmds []string
	// cmds = append(cmds, "build")
	if canFetchServerPeers() {
		cmds = append(cmds, Infokey_PeersCommand)
	}
	log.Tracef("globals-passonekeys:%s", cmds)

	return cmds
}

func (sw *GlobalStatsProcessor) Refresh(infoKeys []string, rawMetrics map[string]string) ([]AerospikeStat, error) {
	// parse peers-command output
	allMetricsToSend, err := parseServerPeersMetrics(rawMetrics)

	return allMetricsToSend, err
}

// utility will check if we can fetch the node-peers
func canFetchServerPeers() bool {

	// difference between current-time and last-fetch, if its > defined-value, then true
	timeDiff := time.Since(serverPeersPreviousFetchTime)

	// if index-type=false or sindex-type=flash is returned by server
	//    and every N seconds - where N is mentioned "indexPressureFetchIntervalInSeconds"
	isTimeOk := timeDiff.Minutes() >= serverPeersFetchInterval

	return isTimeOk

}

func parseServerPeersMetrics(rawMetrics map[string]string) ([]AerospikeStat, error) {
	peerNodesData := rawMetrics[Infokey_PeersCommand]

	var allMetricsToSend = []AerospikeStat{}

	// 4,3000,[[BB9060011AC4202,,[172.17.0.6,172.17.0.A]],[BB90F0011AC4202,,[172.17.0.15]]]
	if len(strings.Trim(peerNodesData, " ")) > 0 {
		peerNodesData = strings.Trim(peerNodesData, " ")

		peersVersionAndPort := peerNodesData[0 : strings.Index(peerNodesData, "[")-1]

		peerNodesData = peerNodesData[strings.Index(peerNodesData, "["):]
		// If no nodes, skip
		if peerNodesData == "[]" || len(strings.Trim(peerNodesData, " ")) == 0 {
			return allMetricsToSend, nil
		}
		peerNodesData = peerNodesData[1 : len(peerNodesData)-2]

		peersGenVersion := strings.Split(peersVersionAndPort, ",")[0]
		peersPort := strings.Split(peersVersionAndPort, ",")[1]

		peerNodeInfos := strings.Split(peerNodesData, "],")

		labels := []string{commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE, commons.METRIC_LABEL_NODE_ID,
			commons.METRIC_LABEL_GEN, commons.METRIC_LABEL_PORT}
		labelValues := []string{ClusterName, Service, NodeId, peersGenVersion, peersPort}

		var peerNodeId, batchPeerNodeIds string
		peersCounter := 0
		endpointLabelIndex := 1

		// parse each peer-info and append to the labels, each label will have max of 32 peer-node-id's
		//
		for _, ele := range peerNodeInfos {
			peerInfo := strings.Split(ele[1:], ",")

			peerNodeId = peerInfo[0]
			batchPeerNodeIds = batchPeerNodeIds + peerNodeId + ","

			peersCounter++
			if peersCounter == MAX_PEERS_PER_LABEL {
				labels = append(labels, commons.METRIC_LABEL_ENDPOINT_LIST_PREFIX+strconv.Itoa(endpointLabelIndex))
				labelValues = append(labelValues, batchPeerNodeIds[0:len(batchPeerNodeIds)-1])

				// reset/increment all local variable/counters
				peersCounter = 0
				batchPeerNodeIds = ""
				endpointLabelIndex++
			}
		}

		if len(batchPeerNodeIds) > 0 {
			labels = append(labels, commons.METRIC_LABEL_ENDPOINT_LIST_PREFIX+strconv.Itoa(endpointLabelIndex))
			labelValues = append(labelValues, batchPeerNodeIds[0:len(batchPeerNodeIds)-1])
		}

		asMetric := NewAerospikeStat(commons.CTX_GLOBAL, "peers_details", true)

		asMetric.updateValues(1, labels, labelValues)
		allMetricsToSend = append(allMetricsToSend, asMetric)

	}

	return allMetricsToSend, nil
}
