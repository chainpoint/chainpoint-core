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

	// Simple retry logic for obtaining self and peer info
	for i := 0; i < 5; i++ {
		status, err = GetStatus(tendermintRPC)
		netInfo, err2 = GetNetInfo(tendermintRPC)
		if err != nil || err2 != nil {
			time.Sleep(5 * time.Second)
			continue
		} else {
			break
		}
	}
	if err != nil || err2 != nil {
		fmt.Println(err)
		fmt.Println(err2)
		return false, ""
	}

	currentNodeID := status.NodeInfo.ID()
	if len(netInfo.Peers) > 0 {
		nodeArray := make([]p2p.DefaultNodeInfo, len(netInfo.Peers)+1)
		for i := 0; i < len(netInfo.Peers); i++ {
			nodeArray[i] = netInfo.Peers[i].NodeInfo
		}
		nodeArray[len(netInfo.Peers)] = status.NodeInfo
		sort.Slice(nodeArray[:], func(i, j int) bool {
			return nodeArray[i].ID() > nodeArray[j].ID()
		})
		if !status.SyncInfo.CatchingUp {
			blockHash := status.SyncInfo.LatestBlockHash
			index := util.GetSeededRandInt([]byte(blockHash), len(nodeArray))
			leader := nodeArray[index]
			return leader.ID() == currentNodeID, string(leader.ID())
		}
		fmt.Println("No leader (not caught up)")
		return false, ""
	}
	fmt.Println(currentNodeID)
	return true, string(currentNodeID)
}

// GetHTTPClient creates an Tendermint RPC client from connection URI/Port details
func GetHTTPClient(tendermintRPC types.TendermintURI) *client.HTTP {
	return client.NewHTTP(fmt.Sprintf("http://%s:%s", tendermintRPC.TMServer, tendermintRPC.TMPort), "/websocket")
}
