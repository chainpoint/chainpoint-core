package abci

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/tendermint/tendermint/p2p"

	"github.com/chainpoint/chainpoint-core/go-abci-service/types"
	core_types "github.com/tendermint/tendermint/rpc/core/types"

	"github.com/chainpoint/chainpoint-core/go-abci-service/util"
	"github.com/tendermint/tendermint/rpc/client"
)

// GetStatus retrieves status of our node. Can't use RPC because remote_ip has buggy encoding.
func GetStatus(tendermintRPC types.TendermintURI) (core_types.ResultStatus, error) {
	rpc := GetHTTPClient(tendermintRPC)
	defer rpc.Stop()
	if rpc == nil {
		return core_types.ResultStatus{}, errors.New("rpc failure")
	}
	status, err := rpc.Status()
	if util.LogError(err) != nil {
		return core_types.ResultStatus{}, err
	}
	return *status, err
}

// GetNetInfo retrieves known peer information. Can't use RPC because remote_ip has buggy encoding.
func GetNetInfo(tendermintRPC types.TendermintURI) (core_types.ResultNetInfo, error) {
	rpc := GetHTTPClient(tendermintRPC)
	defer rpc.Stop()
	if rpc == nil {
		return core_types.ResultNetInfo{}, errors.New("rpc failure")
	}
	netInfo, err := rpc.NetInfo()
	if util.LogError(err) != nil {
		return core_types.ResultNetInfo{}, err
	}
	return *netInfo, err
}

// GetAbciInfo retrieves custom ABCI status struct detailing the state of our application
func GetAbciInfo(tendermintRPC types.TendermintURI) (types.State, error) {
	rpc := GetHTTPClient(tendermintRPC)
	defer rpc.Stop()
	resp, err := rpc.ABCIInfo()
	if err != nil {
		return types.State{}, err
	}
	var anchorState types.State
	util.LogError(json.Unmarshal([]byte(resp.Response.Data), &anchorState))
	return anchorState, nil
}

// ElectLeader deterministically elects a network leader by creating an array of peers and using a blockhash-seeded random int as an index
func ElectLeader(tendermintRPC types.TendermintURI) (isLeader bool, leader string) {
	var status core_types.ResultStatus
	var netInfo core_types.ResultNetInfo
	var err error
	var err2 error

	status, err = GetStatus(tendermintRPC)
	netInfo, err2 = GetNetInfo(tendermintRPC)

	if err != nil || err2 != nil {
		fmt.Println(err)
		fmt.Println(err2)
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
		for i := 0; i < 5; i++ {
			var index int
			if i == 0 {
				blockHash := status.SyncInfo.LatestBlockHash
				index = util.GetSeededRandInt([]byte(blockHash), len(nodeArray)) //seed the first time
			} else {
				index = util.GetRandInt(len(nodeArray))
			}
			leader := nodeArray[index]
			if leader.NodeInfo.ID() == currentNodeID && !status.SyncInfo.CatchingUp {
				return true, string(leader.NodeInfo.ID())
			} else if status.SyncInfo.CatchingUp {
				continue
			}
			tendermintRPC.TMServer = leader.RemoteIP
			syncStatus, err := GetStatus(tendermintRPC)
			if util.LogError(err) != nil {
				continue
			}
			if !syncStatus.SyncInfo.CatchingUp {
				return leader.NodeInfo.ID() == currentNodeID, string(leader.NodeInfo.ID())
			}
			time.Sleep(5 * time.Second)
		}
		fmt.Println("No leader (not caught up)")
		return false, ""
	}
	return true, string(currentNodeID)
}

// GetHTTPClient creates an Tendermint RPC client from connection URI/Port details
func GetHTTPClient(tendermintRPC types.TendermintURI) *client.HTTP {
	return client.NewHTTP(fmt.Sprintf("http://%s:%s", tendermintRPC.TMServer, tendermintRPC.TMPort), "/websocket")
}
