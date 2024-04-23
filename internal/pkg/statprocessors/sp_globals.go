package statprocessors

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aerospike/aerospike-prometheus-exporter/internal/pkg/commons"
	log "github.com/sirupsen/logrus"
)

type GlobalStatsProcessor struct {
	globalMetrics map[string]AerospikeStat
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

	// fmt.Println("\t *** GlobalStatsProcessor ", cmds)

	return cmds
}

func (sw *GlobalStatsProcessor) Refresh(infoKeys []string, rawMetrics map[string]string) ([]AerospikeStat, error) {
	// parse peers-command output
	allMetricsToSend, err := parseServerPeersMetrics(rawMetrics)

	return allMetricsToSend, err
}

// utility will check if we can fetch the node-peers
func canFetchServerPeers() bool {

	//TODO: write logic

	// difference between current-time and last-fetch, if its > defined-value, then true
	timeDiff := time.Since(serverPeersPreviousFetchTime)

	// if index-type=false or sindex-type=flash is returned by server
	//    and every N seconds - where N is mentioned "indexPressureFetchIntervalInSeconds"
	isTimeOk := timeDiff.Minutes() >= serverPeersFetchInterval

	return isTimeOk

}

func parseServerPeersMetrics(rawMetrics map[string]string) ([]AerospikeStat, error) {
	peerNodesData := rawMetrics[Infokey_PeersCommand]

	fmt.Println("peerNodesData ==> ", peerNodesData)
	var allMetricsToSend = []AerospikeStat{}

	// 4,3000,[[BB9060011AC4202,,[172.17.0.6]],[BB90F0011AC4202,,[172.17.0.15]]]
	if len(strings.Trim(peerNodesData, " ")) > 0 {
		peerNodesData = strings.Trim(peerNodesData, " ")

		peersVersionAndPort := peerNodesData[0 : strings.Index(peerNodesData, "[")-1]
		peerNodesData = peerNodesData[strings.Index(peerNodesData, "["):]
		peerNodesData = peerNodesData[1 : len(peerNodesData)-2]

		peersGenVersion := strings.Split(peersVersionAndPort, ",")[0]
		peersPort := strings.Split(peersVersionAndPort, ",")[1]

		peerNodeInfos := strings.Split(peerNodesData, "],")

		labels := []string{commons.METRIC_LABEL_CLUSTER_NAME, commons.METRIC_LABEL_SERVICE,
			commons.METRIC_LABEL_GEN, commons.METRIC_LABEL_PORT,
			commons.METRIC_LABEL_NODE_ID, commons.METRIC_LABEL_TLS_NAME}

		// append the labels array with 10 ENDPOINTS labels, this is to accommodate the IP address of all the peers/neighbours
		// each IP address consits 15 chars max, hence 10x15= 150 + 9 (commas) i.e. 159 chars

		for i := 1; i <= 5; i++ {
			labels = append(labels, commons.METRIC_LABEL_ENDPOINT_LIST_PREFIX+strconv.Itoa(i))
		}

		var nodeId, peerTcp, peerIps string
		lablelValuesCounter := 1
		endpointLabelIndex := 0
		peerIpsLabelValues := []string{"na", "na", "na", "na", "na"}

		for _, ele := range peerNodeInfos {
			peerInfo := strings.Split(ele[1:len(ele)], ",")

			nodeId = peerInfo[0]
			peerTcp = peerInfo[1]
			if len(strings.Trim(peerInfo[1], " ")) == 0 {
				peerTcp = "std"
			}

			localPeerIps := peerInfo[2][1 : len(peerInfo[2])-1]

			// get only 1st IP address
			localPeerIps = strings.Split(localPeerIps, ",")[0]

			peerIps = peerIps + localPeerIps + ","

			lablelValuesCounter++
			if lablelValuesCounter > 0 {
				lablelValuesCounter = 0
				peerIpsLabelValues[endpointLabelIndex] = peerIps[0 : len(peerIps)-1]
				peerIps = ""
				endpointLabelIndex++
			}
		}

		// remove last comma
		// peerIps = peerIps[0 : len(peerIps)-1]
		labelValues := []string{ClusterName, Service, peersGenVersion, peersPort, nodeId, peerTcp}
		labelValues = append(labelValues, peerIpsLabelValues...)

		asMetric := NewAerospikeStat(commons.CTX_GLOBAL, "peers_details", true)

		asMetric.updateValues(1, labels, labelValues)
		allMetricsToSend = append(allMetricsToSend, asMetric)

		fmt.Println("\t**** ", labels)
		fmt.Println("\t**** ", labelValues)
		fmt.Println(peersGenVersion, peersPort, nodeId, peerTcp, peerIps)

	}

	return allMetricsToSend, nil
}
