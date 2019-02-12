package abci

import (
	"fmt"
	"github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tendermint/rpc/client"
	"github.com/tendermint/tendermint/rpc/core/types"
	"sort"
	"time"
)

func GetStatus(tmServer string, tmPort string) (core_types.ResultStatus, error){
	rpc := GetHTTPClient(tmServer, tmPort)
	defer rpc.Stop()
	result, err := rpc.Status()
	return *result, err
}

func GetNetInfo(tmServer string, tmPort string) (core_types.ResultNetInfo, error){
	rpc := GetHTTPClient(tmServer, tmPort)
	defer rpc.Stop()
	result, err := rpc.NetInfo()
	return *result, err
}

func ElectLeader(tmServer string, tmPort string) (isLeader bool, leader p2p.ID){
	var status core_types.ResultStatus
	var netInfo core_types.ResultNetInfo
	var err error
	var err2 error

	// Simple retry logic for obtaining self and peer info
	for i := 0; i < 5; i++ {
		status, err = GetStatus(tmServer, tmPort)
		netInfo, err2 = GetNetInfo(tmServer, tmPort)
		if err != nil || err2 != nil {
			time.Sleep(5 * time.Second)
			continue
		} else {
			break
		}
	}
	if err != nil || err2 != nil {
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
		if !status.SyncInfo.CatchingUp{
			blockHash := status.SyncInfo.LatestBlockHash
			index := getSeededRandInt(blockHash, len(nodeArray))
			leader := nodeArray[index]
			return leader.ID() == currentNodeID, leader.ID()
		}
		return false, ""
	}else{
		return true, currentNodeID
	}
}

func GetHTTPClient(tmServer string, tmPort string) *client.HTTP {
	return client.NewHTTP(fmt.Sprintf("http://%s:%s", tmServer, tmPort), "/websocket")
}
