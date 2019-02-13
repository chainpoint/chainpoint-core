package abci

import (
	"encoding/json"
	"fmt"
	"github.com/tendermint/tendermint/rpc/client"
	"io/ioutil"
	"net/http"
	"sort"
	"time"
)

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

func GetNetInfo(tmServer string, tmPort string) (NetInfo, error){
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

func ElectLeader(tmServer string, tmPort string) (isLeader bool, leader string){
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
		if !status.Result.SyncInfo.CatchingUp{
			blockHash := status.Result.SyncInfo.LatestBlockHash
			index := getSeededRandInt([]byte(blockHash), len(nodeArray))
			leader := nodeArray[index]
			return leader.ID == currentNodeID, leader.ID
		}
		return false, ""
	}else{
		return true, currentNodeID
	}
}

func GetHTTPClient(tmServer string, tmPort string) *client.HTTP {
	return client.NewHTTP(fmt.Sprintf("http://%s:%s", tmServer, tmPort), "/websocket")
}
