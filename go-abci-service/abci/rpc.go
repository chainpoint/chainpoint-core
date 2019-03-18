package abci

import (
	"encoding/json"
	"errors"
	"fmt"

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

//GetTxByInt : Retrieves a tx by its unique integer ID (txInt)
func GetTxByInt(tendermintRPC types.TendermintURI, txInt int64) (core_types.ResultTxSearch, error) {
	rpc := GetHTTPClient(tendermintRPC)
	defer rpc.Stop()
	txResult, err := rpc.TxSearch(fmt.Sprintf("TxInt=%d", txInt), false, 1, 1)
	if util.LogError(err) != nil {
		return core_types.ResultTxSearch{}, err
	}
	return *txResult, err
}

// GetAbciInfo retrieves custom ABCI status struct detailing the state of our application
func GetAbciInfo(tendermintRPC types.TendermintURI) (types.AnchorState, error) {
	rpc := GetHTTPClient(tendermintRPC)
	defer rpc.Stop()
	resp, err := rpc.ABCIInfo()
	if err != nil {
		return types.AnchorState{}, err
	}
	var anchorState types.AnchorState
	util.LogError(json.Unmarshal([]byte(resp.Response.Data), &anchorState))
	return anchorState, nil
}

// GetHTTPClient creates an Tendermint RPC client from connection URI/Port details
func GetHTTPClient(tendermintRPC types.TendermintURI) *client.HTTP {
	return client.NewHTTP(fmt.Sprintf("http://%s:%s", tendermintRPC.TMServer, tendermintRPC.TMPort), "/websocket")
}
