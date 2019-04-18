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

// RPC : hold abstract http client for mocking purposes
type RPC struct {
	client *client.HTTP
}

// NewRPCClient : Creates a new client connected to a tendermint instance at web socket "tendermintRPC"
func NewRPCClient(tendermintRPC types.TendermintURI) (rpc *RPC) {
	return &RPC{
		client: client.NewHTTP(fmt.Sprintf("http://%s:%s", tendermintRPC.TMServer, tendermintRPC.TMPort), "/websocket"),
	}
}

// BroadcastTx : Synchronously broadcasts a transaction to the local Tendermint node
func (rpc *RPC) BroadcastTx(txType string, data string, version int64, time int64, stackID string) (core_types.ResultBroadcastTx, error) {
	tx := types.Tx{TxType: txType, Data: data, Version: version, Time: time, CoreID: stackID}
	result, err := rpc.client.BroadcastTxSync([]byte(util.EncodeTx(tx)))
	if util.LogError(err) != nil {
		return core_types.ResultBroadcastTx{}, err
	}
	return *result, nil
}

// GetStatus retrieves status of our node. Can't use RPC because remote_ip has buggy encoding.
func (rpc *RPC) GetStatus() (core_types.ResultStatus, error) {
	if rpc == nil {
		return core_types.ResultStatus{}, errors.New("rpc failure")
	}
	status, err := rpc.client.Status()
	if util.LogError(err) != nil {
		return core_types.ResultStatus{}, err
	}
	return *status, err
}

// GetNetInfo retrieves known peer information. Can't use RPC because remote_ip has buggy encoding.
func (rpc *RPC) GetNetInfo() (core_types.ResultNetInfo, error) {
	if rpc == nil {
		return core_types.ResultNetInfo{}, errors.New("rpc failure")
	}
	netInfo, err := rpc.client.NetInfo()
	if util.LogError(err) != nil {
		return core_types.ResultNetInfo{}, err
	}
	return *netInfo, err
}

//GetTxByInt : Retrieves a tx by its unique integer ID (txInt)
func (rpc *RPC) GetTxByInt(txInt int64) (core_types.ResultTxSearch, error) {
	txResult, err := rpc.client.TxSearch(fmt.Sprintf("TxInt=%d", txInt), false, 1, 1)
	if util.LogError(err) != nil {
		return core_types.ResultTxSearch{}, err
	}
	return *txResult, err
}

// GetAbciInfo retrieves custom ABCI status struct detailing the state of our application
func (rpc *RPC) GetAbciInfo() (types.AnchorState, error) {
	resp, err := rpc.client.ABCIInfo()
	if err != nil {
		return types.AnchorState{}, err
	}
	var anchorState types.AnchorState
	util.LogError(json.Unmarshal([]byte(resp.Response.Data), &anchorState))
	return anchorState, nil
}
