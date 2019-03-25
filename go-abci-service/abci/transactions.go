package abci

import (
	"errors"
	"fmt"

	types2 "github.com/tendermint/tendermint/abci/types"

	"github.com/chainpoint/chainpoint-core/go-abci-service/util"
	"github.com/tendermint/tendermint/abci/example/code"
	cmn "github.com/tendermint/tendermint/libs/common"
	core_types "github.com/tendermint/tendermint/rpc/core/types"
)

// incrementTxInt: Helper method to increment transaction integer
func (app *AnchorApplication) incrementTxInt(tags []cmn.KVPair) []cmn.KVPair {
	app.state.TxInt++ // no pre-increment :(
	return append(tags, cmn.KVPair{Key: []byte("TxInt"), Value: util.Int64ToByte(app.state.TxInt)})
}

// updateStateFromTx: Updates state based on type of transaction received. Used by DeliverTx
func (app *AnchorApplication) updateStateFromTx(rawTx []byte) types2.ResponseDeliverTx {
	tx, err := util.DecodeTx(rawTx)
	tags := []cmn.KVPair{}
	if err != nil {
		return types2.ResponseDeliverTx{Code: code.CodeTypeEncodingError, Tags: tags}
	}
	var resp types2.ResponseDeliverTx
	switch string(tx.TxType) {
	case "VAL":
		tags := app.incrementTxInt(tags)
		if isValidatorTx([]byte(tx.Data)) {
			resp = app.execValidatorTx([]byte(tx.Data), tags)
		}
		break
	case "CAL":
		tags := app.incrementTxInt(tags)
		app.state.LatestCalTxInt = app.state.TxInt
		resp = types2.ResponseDeliverTx{Code: code.CodeTypeOK, Tags: tags}
		break
	case "BTC-M":
		//Begin monitoring using the data contained in this gossiped (but ultimately nacked) transaction
		app.state.LatestBtcmHeight = app.state.Height + 1
		tags := app.incrementTxInt(tags)
		app.state.LatestBtcmTxInt = app.state.TxInt
		app.ConsumeBtcTxMsg([]byte(tx.Data))
		resp = types2.ResponseDeliverTx{Code: code.CodeTypeOK, Tags: tags}
		break
	case "BTC-A":
		app.state.LatestBtcaTx = rawTx
		app.state.LatestBtcaHeight = app.state.Height + 1
		tags := app.incrementTxInt(tags)
		app.state.LatestBtcaTxInt = app.state.TxInt
		app.state.BeginCalTxInt = app.state.EndCalTxInt // Keep a placeholder in case a CAL Tx is sent in between the time of a BTC-A broadcast and its handling
		resp = types2.ResponseDeliverTx{Code: code.CodeTypeOK, Tags: tags}
		break
	case "BTC-C":
		app.state.LatestBtccTx = rawTx
		app.state.LatestBtccHeight = app.state.Height + 1
		tags := app.incrementTxInt(tags)
		app.state.LatestBtccTxInt = app.state.TxInt
		resp = types2.ResponseDeliverTx{Code: code.CodeTypeOK, Tags: tags}
		break
	case "NIST":
		app.state.LatestNistRecord = tx.Data
		resp = types2.ResponseDeliverTx{Code: code.CodeTypeUnknownError, Tags: tags}
		break
	default:
		resp = types2.ResponseDeliverTx{Code: code.CodeTypeUnauthorized, Tags: tags}
	}
	return resp
}

// GetTxRange gets all CAL TXs within a particular range
func (app *AnchorApplication) getTxRange(minTxInt int64, maxTxInt int64) ([]core_types.ResultTx, error) {
	if maxTxInt <= minTxInt {
		return nil, errors.New("max of tx range is less than or equal to min")
	}
	Txs := []core_types.ResultTx{}
	for i := minTxInt; i <= maxTxInt; i++ {
		txResult, err := app.rpc.client.TxSearch(fmt.Sprintf("TxInt=%d", i), false, 1, 1)
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
