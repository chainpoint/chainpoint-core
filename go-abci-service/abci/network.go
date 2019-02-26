package abci

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"time"

	"github.com/chainpoint/chainpoint-core/go-abci-service/types"

	"github.com/chainpoint/chainpoint-core/go-abci-service/util"
	"github.com/tendermint/tendermint/rpc/client"
)

// GetStatus retrieves status of our node from RPC
func GetStatus(tendermintRPC types.TendermintURI) (types.NodeStatus, error) {
	resp, err := http.Get(fmt.Sprintf("http://%s:%s/net_info", tendermintRPC.TMServer, tendermintRPC.TMPort))
	if err != nil {
		return types.NodeStatus{}, err
	}
	var status types.NodeStatus
	body, _ := ioutil.ReadAll(resp.Body)
	json.Unmarshal(body, &status)
	resp.Body.Close()
	return status, nil
}

// GetNetInfo retrieves known peer information via rpc
func GetNetInfo(tendermintRPC types.TendermintURI) (types.NetInfo, error) {
	resp, err := http.Get(fmt.Sprintf("http://%s:%s/status", tendermintRPC.TMServer, tendermintRPC.TMPort))
	if err != nil {
		return types.NetInfo{}, err
	}
	var info types.NetInfo
	body, _ := ioutil.ReadAll(resp.Body)
	json.Unmarshal(body, &info)
	resp.Body.Close()
	return info, nil
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
	err = json.Unmarshal([]byte(resp.Response.Data), &anchorState)
	if err != nil {
		fmt.Println(err)
		return types.State{}, err
	}
	return anchorState, nil
}

// ElectLeader deterministically elects a network leader by creating an array of peers and using a blockhash-seeded random int as an index
func ElectLeader(tendermintRPC types.TendermintURI) (isLeader bool, leader string) {
	var status types.NodeStatus
	var netInfo types.NetInfo
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

	currentNodeID := status.Result.NodeInfo.ID
	if len(netInfo.Result.Peers) > 0 {
		nodeArray := make([]types.NodeInfo, len(netInfo.Result.Peers)+1)
		for i := 0; i < len(netInfo.Result.Peers); i++ {
			nodeArray[i] = netInfo.Result.Peers[i].NodeInfo
		}
		nodeArray[len(netInfo.Result.Peers)] = status.Result.NodeInfo
		sort.Slice(nodeArray[:], func(i, j int) bool {
			return nodeArray[i].ID > nodeArray[j].ID
		})
		if !status.Result.SyncInfo.CatchingUp {
			blockHash := status.Result.SyncInfo.LatestBlockHash
			index := util.GetSeededRandInt([]byte(blockHash), len(nodeArray))
			leader := nodeArray[index]
			return leader.ID == currentNodeID, leader.ID
		}
		return false, ""
	}
	return true, currentNodeID
}

// GetHTTPClient creates an Tendermint RPC client from connection URI/Port details
func GetHTTPClient(tendermintRPC types.TendermintURI) *client.HTTP {
	return client.NewHTTP(fmt.Sprintf("http://%s:%s", tendermintRPC.TMServer, tendermintRPC.TMPort), "/websocket")
}
