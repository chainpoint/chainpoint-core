package abci

import (
	"fmt"
	"sort"

	"github.com/chainpoint/chainpoint-core/go-abci-service/util"
	"github.com/tendermint/tendermint/p2p"
	core_types "github.com/tendermint/tendermint/rpc/core/types"
)

// ElectLeader deterministically elects a network leader by creating an array of peers and using a blockhash-seeded random int as an index
func (app *AnchorApplication) ElectLeader(numLeaders int) (isLeader bool, leaderID []string) {
	var status core_types.ResultStatus
	var netInfo core_types.ResultNetInfo
	var err error
	var err2 error

	status, err = app.rpc.GetStatus()
	netInfo, err2 = app.rpc.GetNetInfo()

	if util.LogError(err) != nil || util.LogError(err2) != nil {
		return false, []string{}
	}
	blockHash := status.SyncInfo.LatestBlockHash.String()
	app.logger.Info(fmt.Sprintf("Blockhash Seed: %s", blockHash))
	return determineLeader(numLeaders, status, netInfo, blockHash)
}

//GetSortedPeerList : returns sorted list of peers including self
func GetSortedPeerList(status core_types.ResultStatus, netInfo core_types.ResultNetInfo) []core_types.Peer {
	peers := netInfo.Peers
	nodeArray := make([]core_types.Peer, 0)
	for i := 0; i < len(peers); i++ {
		peers[i].RemoteIP = util.DetermineIP(peers[i])
		nodeArray = append(nodeArray, peers[i])
	}
	selfPeer := core_types.Peer{
		NodeInfo:         status.NodeInfo,
		IsOutbound:       false,
		ConnectionStatus: p2p.ConnectionStatus{},
		RemoteIP:         "127.0.0.1",
	}
	nodeArray = append(nodeArray, selfPeer)
	sort.Slice(nodeArray[:], func(i, j int) bool {
		return nodeArray[i].NodeInfo.ID() > nodeArray[j].NodeInfo.ID()
	})
	return nodeArray
}

// determineLeader accepts current node status and a peer array, then finds a leader based on the latest blockhash
func determineLeader(numLeaders int, status core_types.ResultStatus, netInfo core_types.ResultNetInfo, seed string) (isLeader bool, leaderIDs []string) {
	currentNodeID := status.NodeInfo.ID()
	if len(netInfo.Peers) > 0 {
		nodeArray := GetSortedPeerList(status, netInfo)
		index := util.GetSeededRandInt([]byte(seed), len(nodeArray)) //seed the first time
		if err := util.RotateLeft(nodeArray[:], index); err != nil { //get a wrapped-around slice of numLeader leaders
			util.LogError(err)
			return false, []string{}
		}
		leaders := make([]core_types.Peer, 0)
		if numLeaders <= len(nodeArray) {
			leaders = nodeArray[0:numLeaders]
		} else {
			leaders = nodeArray[0:1]
		}
		leaderStrings := make([]string, 0)
		iAmLeader := false
		for _, leader := range leaders {
			leaderStrings = append(leaderStrings, string(leader.NodeInfo.ID()))
			if leader.NodeInfo.ID() == currentNodeID && !status.SyncInfo.CatchingUp {
				iAmLeader = true
			}
		}
		return iAmLeader, leaderStrings
	}
	return true, []string{string(currentNodeID)}
}
