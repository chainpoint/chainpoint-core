package tendermintrpc

import (
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/chainpoint/chainpoint-core/types"
	"github.com/tendermint/tendermint/libs/log"
	rpchttp "github.com/tendermint/tendermint/rpc/client/http"
	core_types "github.com/tendermint/tendermint/rpc/core/types"

	"github.com/chainpoint/chainpoint-core/util"
)

// RPC : hold abstract http client for mocking purposes
type RPC struct {
	client *rpchttp.HTTP
	logger log.Logger
}

// NewRPCClient : Creates a new client connected to a tendermint instance at web socket "tendermintRPC"
func NewRPCClient(tendermintRPC types.TendermintConfig, logger log.Logger) (rpc *RPC) {
	c, _ := rpchttp.NewWithTimeout(fmt.Sprintf("http://%s:%s", tendermintRPC.TMServer, tendermintRPC.TMPort), "/websocket", 2)
	return &RPC{
		client: c,
		logger: logger,
	}
}

//LogError : log tendermintRpc errors
func (rpc *RPC) LogError(err error) error {
	if err != nil {
		rpc.logger.Error(fmt.Sprintf("Error in %s: %s", util.GetCurrentFuncName(2), err.Error()))
	}
	return err
}

// BroadcastTx : Synchronously broadcasts a transaction to the local Tendermint node
func (rpc *RPC) BroadcastTx(txType string, data string, version int64, time int64, stackID string, privateKey *ecdsa.PrivateKey) (core_types.ResultBroadcastTx, error) {
	tx := types.Tx{TxType: txType, Data: data, Version: version, Time: time, CoreID: stackID}
	result, err := rpc.client.BroadcastTxSync([]byte(util.EncodeTxWithKey(tx, privateKey)))
	if rpc.LogError(err) != nil {
		return core_types.ResultBroadcastTx{}, err
	}
	return *result, nil
}

// BroadcastTx : Synchronously broadcasts a transaction to the local Tendermint node
func (rpc *RPC) BroadcastTxWithMeta(txType string, data string, version int64, time int64, stackID string, meta string, privateKey *ecdsa.PrivateKey) (core_types.ResultBroadcastTx, error) {
	tx := types.Tx{TxType: txType, Data: data, Version: version, Time: time, CoreID: stackID, Meta: meta}
	result, err := rpc.client.BroadcastTxSync([]byte(util.EncodeTxWithKey(tx, privateKey)))
	if rpc.LogError(err) != nil {
		return core_types.ResultBroadcastTx{}, err
	}
	return *result, nil
}

// BroadcastTxCommit : Synchronously broadcasts a transaction to the local Tendermint node THIS IS BLOCKING
func (rpc *RPC) BroadcastTxCommit(txType string, data string, version int64, time int64, stackID string, privateKey *ecdsa.PrivateKey) (core_types.ResultBroadcastTxCommit, error) {
	tx := types.Tx{TxType: txType, Data: data, Version: version, Time: time, CoreID: stackID}
	result, err := rpc.client.BroadcastTxCommit([]byte(util.EncodeTxWithKey(tx, privateKey)))
	if rpc.LogError(err) != nil {
		return core_types.ResultBroadcastTxCommit{}, err
	}
	return *result, nil
}

// GetStatus retrieves status of our node.
func (rpc *RPC) GetStatus() (core_types.ResultStatus, error) {
	if rpc == nil {
		return core_types.ResultStatus{}, errors.New("tendermintRpc failure")
	}
	status, err := rpc.client.Status()
	if rpc.LogError(err) != nil {
		return core_types.ResultStatus{}, err
	}
	return *status, err
}

// GetNetInfo retrieves known peer information.
func (rpc *RPC) GetNetInfo() (core_types.ResultNetInfo, error) {
	if rpc == nil {
		return core_types.ResultNetInfo{}, errors.New("tendermintRpc failure")
	}
	netInfo, err := rpc.client.NetInfo()
	if rpc.LogError(err) != nil {
		return core_types.ResultNetInfo{}, err
	}
	return *netInfo, err
}

//GetTxByInt : Retrieves a tx by its unique integer ID (txInt)
func (rpc *RPC) GetTxByInt(txInt int64) (core_types.ResultTxSearch, error) {
	txResult, err := rpc.client.TxSearch(fmt.Sprintf("CAL.TxInt=%d", txInt), false, 1, 1, "")
	if rpc.LogError(err) != nil {
		return core_types.ResultTxSearch{}, err
	}
	return *txResult, err
}

//GetTxByHash : Retrieves a tx by its unique string ID (txid)
func (rpc *RPC) GetTxByHash(txid string) (core_types.ResultTx, error) {
	hash, err := hex.DecodeString(txid)
	if rpc.LogError(err) != nil {
		return core_types.ResultTx{}, err
	}
	txResult, err := rpc.client.Tx(hash, false)
	if rpc.LogError(err) != nil {
		return core_types.ResultTx{}, err
	}
	return *txResult, err
}

// GetAbciInfo retrieves custom ABCI status struct detailing the state of our application
func (rpc *RPC) GetAbciInfo() (types.AnchorState, error) {
	resp, err := rpc.client.ABCIInfo()
	if rpc.LogError(err) != nil {
		return types.AnchorState{}, err
	}
	var anchorState types.AnchorState
	util.LogError(json.Unmarshal([]byte(resp.Response.Data), &anchorState))
	return anchorState, nil
}

//GetValidators : retrieves list of validators at a particular block height
func (rpc *RPC) GetValidators(height int64) (core_types.ResultValidators, error) {
	resp, err := rpc.client.Validators(&height, 1, 300)
	if rpc.LogError(err) != nil {
		return core_types.ResultValidators{}, err
	}
	return *resp, nil
}

//GetGenesis : retrieves genesis file for initialization
func (rpc *RPC) GetGenesis() (core_types.ResultGenesis, error) {
	resp, err := rpc.client.Genesis()
	if rpc.LogError(err) != nil {
		return core_types.ResultGenesis{}, err
	}
	return *resp, nil
}

// GetTxRange gets all CAL TXs within a particular range
func (rpc *RPC) GetCalTxRange(minTxInt int64, maxTxInt int64) ([]core_types.ResultTx, error) {
	if maxTxInt <= minTxInt {
		return nil, errors.New("max of tx range is less than or equal to min")
	}
	Txs := []core_types.ResultTx{}
	for i := minTxInt; i < maxTxInt; i++ {
		txResult, err := rpc.client.TxSearch(fmt.Sprintf("CAL.TxInt=%d", i), false, 1, 1, "")
		if err != nil {
			return nil, err
		} else if txResult.TotalCount > 0 {
			for _, tx := range txResult.Txs {
				Txs = append(Txs, *tx)
			}
		}
	}
	return Txs, nil
}

// GetIndexForCalTx : get transactional index tag for a given tendermint calendar hash
func (rpc *RPC) GetIndexForCalTx(txid string) (int64, error) {
	calResult, err := rpc.GetTxByHash(txid)
	if err != nil {
		return 0, err
	}
	var index int64
	for _, event := range calResult.TxResult.Events {
		for _, tag := range event.Attributes {
			if string(tag.Key) == "TxInt" {
				index = util.ByteToInt64(string(tag.Value))
				return index, nil
			}
		}
	}
	return 0, errors.New(fmt.Sprintf("no txInt index found for %s", txid))
}

// GetBtcaForCalTx : retrieve the corresponding btca tx for a given calendar tx
func (rpc *RPC) GetBtcaForCalTx(txid string) (types.BtcTxMsg, error) {
	index, err := rpc.GetIndexForCalTx(txid)
	if err != nil {
		return types.BtcTxMsg{}, err
	}
	queryLine := fmt.Sprintf("BTC-A.TxInt>%d", index)
	txResult, err := rpc.client.TxSearch(queryLine, false, 1, 5, "asc")
	if rpc.LogError(err) != nil {
		return types.BtcTxMsg{}, err
	}
	for _, res := range txResult.Txs {
		tx, err := util.DecodeTx(res.Tx)
		if err == nil {
			btcMsg := types.BtcTxMsg{}
			if err := json.Unmarshal([]byte(tx.Data), &btcMsg); rpc.LogError(err) == nil && index >= btcMsg.BeginCalTxInt && index < btcMsg.EndCalTxInt {
				return btcMsg, nil
			}
		}
	}
	return types.BtcTxMsg{}, errors.New(fmt.Sprintf("No matches from %d results for cal index %d", txResult.TotalCount, index))
}

//GetAnchoringCore : gets core to whom last anchor is attributed
func (rpc *RPC) GetAnchoringCore(queryLine string) (string, error) {
	txResult, err := rpc.client.TxSearch(queryLine, false, 1, 1, "")
	if rpc.LogError(err) == nil {
		for _, tx := range txResult.Txs {
			decoded, err := util.DecodeTx(tx.Tx)
			if rpc.LogError(err) != nil {
				continue
			}
			return decoded.CoreID, nil
		}
	}
	return "", err
}

// GetBTCCForBtcRoot: retrieves and verifies existence of btcc tx
func (rpc *RPC) GetBTCCForBtcRoot(btcMonObj types.BtcMonMsg) (hash []byte) {
	btccQueryLine := fmt.Sprintf("BTC-C.BTCC='%s'", btcMonObj.BtcHeadRoot)
	txResult, err := rpc.client.TxSearch(btccQueryLine, false, 1, 1, "")
	if rpc.LogError(err) == nil {
		for _, tx := range txResult.Txs {
			hash = tx.Hash
			rpc.logger.Info(fmt.Sprint("Found BTC-C Hash from confirmation leader: %v", hash))
		}
	}
	return hash
}

// GetBTCCForBtcTx: retrieves and verifies existence of btcc tx
func (rpc *RPC) GetAnchorHeight(btcTxObj types.BtcTxMsg) ([]byte, int64) {
	btccQueryLine := fmt.Sprintf("BTC-C.BTCCTX='%s'", btcTxObj.BtcTxID)
	txResult, err := rpc.client.TxSearch(btccQueryLine, false, 1, 1, "")
	if rpc.LogError(err) == nil {
		for _, tx := range txResult.Txs {
			for _, tags := range tx.TxResult.Events {
				for _, pairs := range tags.Attributes {
					if string(pairs.Key) == "BTCCBH" {
						blockHeight := util.ByteToInt64(string(pairs.Value))
						rpc.logger.Info(fmt.Sprint("Found BTC-C Height %d from btctx %s", blockHeight, btcTxObj.BtcTxID))
						return tx.Hash, blockHeight
					}
				}
			}
		}
	}
	return []byte{}, 0
}

// getAllJWKs gets all JWK TXs
func (rpc *RPC) GetAllJWKs() ([]types.Tx, error) {
	Txs := []types.Tx{}
	endPage := 2
	for i := 1; i <= endPage; i++ {
		txResult, err := rpc.client.TxSearch("JWK.CORE='NEW'", false, i, 100, "asc")
		if err != nil {
			return nil, err
		} else if txResult.TotalCount > 0 {
			rpc.logger.Info(fmt.Sprintf("Found %d JWK tx while loading", txResult.TotalCount))
			for _, tx := range txResult.Txs {
				decoded, err := util.DecodeTx(tx.Tx)
				if rpc.LogError(err) == nil {
					Txs = append(Txs, decoded)
				}
			}
		}
		endPage = (txResult.TotalCount / 100) + 1
	}
	return Txs, nil
}

// GetAllCHNGSTK gets all change stake txs
func (rpc *RPC) GetAllCHNGSTK() ([]types.Tx, error) {
	Txs := []types.Tx{}
	endPage := 2
	for i := 1; i <= endPage; i++ {
		txResult, err := rpc.client.TxSearch("CHNGSTK.CHANGE='STAKE'", false, i, 100, "")
		if err != nil {
			return nil, err
		} else if txResult.TotalCount > 0 {
			rpc.logger.Info(fmt.Sprintf("Found %d CHNGSTK tx while loading", txResult.TotalCount))
			for _, tx := range txResult.Txs {
				decoded, err := util.DecodeTx(tx.Tx)
				if rpc.LogError(err) == nil {
					Txs = append(Txs, decoded)
				}
			}
		}
		endPage = (txResult.TotalCount / 100) + 1
	}
	return Txs, nil
}
