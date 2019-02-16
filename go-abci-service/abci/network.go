package abci

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"time"

	"github.com/chainpoint/chainpoint-core/go-abci-service/util"
	"github.com/tendermint/tendermint/rpc/client"
)

/* Retrieves status of our node from RPC*/
func GetStatus(tmServer string, tmPort string) (NodeStatus, error) {
	resp, err := http.Get(fmt.Sprintf("http://%s:%s/status", tmServer, tmPort))
	if err != nil {
		return NodeStatus{}, err
	}
	var status NodeStatus
	body, _ := ioutil.ReadAll(resp.Body)
	json.Unmarshal(body, &status)
	resp.Body.Close()
	return status, nil
}

/*Retrieves known peer information via rpc*/
func GetNetInfo(tmServer string, tmPort string) (NetInfo, error) {
	resp, err := http.Get(fmt.Sprintf("http://%s:%s/net_info", tmServer, tmPort))
	if err != nil {
		return NetInfo{}, err
	}
	var info NetInfo
	body, _ := ioutil.ReadAll(resp.Body)
	json.Unmarshal(body, &info)
	resp.Body.Close()
	return info, nil
}

/* Retrieves custom ABCI status struct detailing the state of our application */
func GetAbciInfo(tmServer string, tmPort string) (State, error) {
	rpc := GetHTTPClient(tmServer, tmPort)
	defer rpc.Stop()
	resp, err := rpc.ABCIInfo()
	if err != nil {
		return State{}, err
	}
	var anchorState State
	err = json.Unmarshal([]byte(resp.Response.Data), &anchorState)
	if err != nil {
		fmt.Println(err)
		return State{}, err
	}
	return anchorState, nil
}

/* Deterministically elects a network leader by creating an array of peers and using a blockhash-seeded random int as an index */
func ElectLeader(tmServer string, tmPort string) (isLeader bool, leader string) {
	var status NodeStatus
	var netInfo NetInfo
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
		fmt.Println(err)
		fmt.Println(err2)
		return false, ""
	}

	currentNodeID := status.Result.NodeInfo.ID
	if len(netInfo.Result.Peers) > 0 {
		nodeArray := make([]NodeInfo, len(netInfo.Result.Peers)+1)
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
	} else {
		return true, currentNodeID
	}
}

/* Creates an Tendermint RPC client from connection URI/Port details */
func GetHTTPClient(tmServer string, tmPort string) *client.HTTP {
	return client.NewHTTP(fmt.Sprintf("http://%s:%s", tmServer, tmPort), "/websocket")
}
