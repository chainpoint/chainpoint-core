package abci

import (
	"sort"

	"github.com/chainpoint/chainpoint-core/go-abci-service/types"
	"github.com/chainpoint/chainpoint-core/go-abci-service/util"
	"github.com/tendermint/tendermint/p2p"
	core_types "github.com/tendermint/tendermint/rpc/core/types"
)

// ElectLeader deterministically elects a network leader by creating an array of peers and using a blockhash-seeded random int as an index
func ElectLeader(tendermintRPC types.TendermintURI) (isLeader bool, leaderID string) {
	var status core_types.ResultStatus
	var netInfo core_types.ResultNetInfo
	var err error
	var err2 error

	status, err = GetStatus(tendermintRPC)
	netInfo, err2 = GetNetInfo(tendermintRPC)

	if util.LogError(err) != nil || util.LogError(err2) != nil {
		return false, ""
	}

	return determineLeaderIndex(status, netInfo.Peers)
}

func determineLeaderIndex(status core_types.ResultStatus, peers []core_types.Peer) (isLeader bool, leaderID string) {
	currentNodeID := status.NodeInfo.ID()
	if len(peers) > 0 {
		nodeArray := make([]core_types.Peer, 0)
		for i := 0; i < len(peers); i++ {
			peers[i].RemoteIP = util.DecodeIP(peers[i].RemoteIP)
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
		blockHash := status.SyncInfo.LatestBlockHash
		index := util.GetSeededRandInt([]byte(blockHash), len(nodeArray)) //seed the first time
		leader := nodeArray[index]
		if leader.NodeInfo.ID() == currentNodeID {
			if !status.SyncInfo.CatchingUp {
				return true, string(leader.NodeInfo.ID())
			}
		}
		return false, string(leader.NodeInfo.ID())
	}
	return true, string(currentNodeID)
}
