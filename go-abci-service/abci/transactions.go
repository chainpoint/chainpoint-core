package abci

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	core_types "github.com/tendermint/tendermint/rpc/core/types"

	"github.com/chainpoint/chainpoint-core/go-abci-service/util"
	"github.com/tendermint/tendermint/abci/example/code"
	"github.com/tendermint/tendermint/abci/types"
	cmn "github.com/tendermint/tendermint/libs/common"
)

// DecodeTx accepts a Chainpoint Calendar transaction in base64 and decodes it into abci.Tx struct
func DecodeTx(incoming []byte) (Tx, error) {
	decoded, err := base64.StdEncoding.DecodeString(string(incoming))
	var calendar Tx
	if err != nil {
		fmt.Println(err)
		return calendar, err
	}
	json.Unmarshal([]byte(decoded), &calendar)
	return calendar, nil
}

// Helper method to increment transaction integer
func (app *AnchorApplication) incrementTx(tags []cmn.KVPair) []cmn.KVPair {
	app.state.TxInt++ // no pre-increment :(
	return append(tags, cmn.KVPair{Key: []byte("TxInt"), Value: util.Int64ToByte(app.state.TxInt)})
}

// Updates state based on type of transaction received. Used by DeliverTx
func (app *AnchorApplication) updateStateFromTx(rawTx []byte) types.ResponseDeliverTx {
	tx, err := DecodeTx(rawTx)
	tags := []cmn.KVPair{}
	if err != nil {
		return types.ResponseDeliverTx{Code: code.CodeTypeEncodingError, Tags: tags}
	}
	var resp types.ResponseDeliverTx
	switch string(tx.TxType) {
	case "VAL":
		tags := app.incrementTx(tags)
		if isValidatorTx(tx.Data) {
			resp = app.execValidatorTx(tx.Data, tags)
		}
		break
	case "CAL":
		tags := app.incrementTx(tags)
		app.state.LatestCalTxInt = app.state.TxInt
		resp = types.ResponseDeliverTx{Code: code.CodeTypeOK, Tags: tags}
		break
	case "BTC-A":
		app.state.LatestBtcaTx = rawTx
		app.state.LatestBtcaHeight = app.state.Height + 1
		tags := app.incrementTx(tags)
		app.state.LatestBtcaTxInt = app.state.TxInt
		resp = types.ResponseDeliverTx{Code: code.CodeTypeOK, Tags: tags}
		break
	case "BTC-C":
		app.state.LatestBtccTx = rawTx
		app.state.LatestBtccHeight = app.state.Height + 1
		tags := app.incrementTx(tags)
		app.state.LatestBtccTxInt = app.state.TxInt
		resp = types.ResponseDeliverTx{Code: code.CodeTypeOK, Tags: tags}
		break
	case "NIST":
		tags := app.incrementTx(tags)
		resp = types.ResponseDeliverTx{Code: code.CodeTypeOK, Tags: tags}
		break
	default:
		resp = types.ResponseDeliverTx{Code: code.CodeTypeUnauthorized, Tags: tags}
	}
	return resp
}

// GetTxRange gets all TXs within a particular range
func GetTxRange(tendermintRPC TendermintURI, minTxInt int64, maxTxInt int64) ([]core_types.ResultTx, error) {
	rpc := GetHTTPClient(tendermintRPC)
	defer rpc.Stop()
	Txs := []core_types.ResultTx{}
	for i := minTxInt; i < maxTxInt; i++ {
		txResult, err := rpc.TxSearch(fmt.Sprintf("TxInt=%d", i), false, 1, 1)
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
