package abci

import (
	"sort"
	"time"

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

	currentNodeID := status.NodeInfo.ID()
	if len(netInfo.Peers) > 0 {
		nodeArray := make([]core_types.Peer, 0)
		for i := 0; i < len(netInfo.Peers); i++ {
			netInfo.Peers[i].RemoteIP = util.DecodeIP(netInfo.Peers[i].RemoteIP)
			nodeArray = append(nodeArray, netInfo.Peers[i])
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
		// This loop determines a leader and checks if it's still syncing. If so, it finds another leader
		for i := 0; i < 5; i++ { // Only retry five times so we don't zombify anchor process
			var index int
			if i == 0 { // Only seed during the first loop. Calls thereafter should still be using the seed
				blockHash := status.SyncInfo.LatestBlockHash
				index = util.GetSeededRandInt([]byte(blockHash), len(nodeArray)) //seed the first time
			} else {
				index = util.GetRandInt(len(nodeArray))
			}
			leader := nodeArray[index]
			if leader.NodeInfo.ID() == currentNodeID && !status.SyncInfo.CatchingUp { //If the current node is leader and is synced
				return true, string(leader.NodeInfo.ID())
			} else if status.SyncInfo.CatchingUp {
				continue //if not, choose another leader
			}
			tendermintRPC.TMServer = leader.RemoteIP
			syncStatus, err := GetStatus(tendermintRPC) //check sync status of chosen leader
			if util.LogError(err) != nil {
				continue //If we can't talk to this core, choose another leader
			}
			if !syncStatus.SyncInfo.CatchingUp {
				return leader.NodeInfo.ID() == currentNodeID, string(leader.NodeInfo.ID()) // if synced, return this leader
			}
			time.Sleep(5 * time.Second)
		}
		return false, ""
	}
	return true, string(currentNodeID)
}
