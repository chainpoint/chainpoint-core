package abci

import (
	"fmt"
	"sort"

	"github.com/chainpoint/chainpoint-core/go-abci-service/util"
	"github.com/tendermint/tendermint/p2p"
	core_types "github.com/tendermint/tendermint/rpc/core/types"
)

// ElectLeader deterministically elects a network leader by creating an array of peers and using a blockhash-seeded random int as an index
func (app *AnchorApplication) ElectLeader() (isLeader bool, leaderID string) {
	var status core_types.ResultStatus
	var netInfo core_types.ResultNetInfo
	var err error
	var err2 error

	status, err = GetStatus(app.config.TendermintRPC)
	netInfo, err2 = GetNetInfo(app.config.TendermintRPC)

	if util.LogError(err) != nil || util.LogError(err2) != nil {
		return false, ""
	}
	app.logger.Debug(fmt.Sprintf("Blockhash Seed: %s", status.SyncInfo.LatestBlockHash.String()))
	return determineLeader(status, netInfo.Peers, status.SyncInfo.LatestBlockHash.String())
}

// determineLeader accepts current node status and a peer array, then finds a leader based on the latest blockhash
func determineLeader(status core_types.ResultStatus, peers []core_types.Peer, seed string) (isLeader bool, leaderID string) {
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
		index := util.GetSeededRandInt([]byte(seed), len(nodeArray)) //seed the first time
		fmt.Printf("Elected index %d of core array (len %d\n)", index, len(nodeArray))
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
