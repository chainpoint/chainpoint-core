package abci

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/chainpoint/chainpoint-core/go-abci-service/util"
	"github.com/tendermint/tendermint/abci/example/code"
	"github.com/tendermint/tendermint/abci/types"
	cmn "github.com/tendermint/tendermint/libs/common"
)

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

func (app *AnchorApplication) incrementTx(tags []cmn.KVPair) []cmn.KVPair {
	app.state.txInt++ // no pre-increment :(
	return append(tags, cmn.KVPair{Key: []byte("txInt"), Value: util.Int64ToByte(app.state.txInt)})
}

/* Updates state based on type of transaction received. Used by DeliverTx */
func (app *AnchorApplication) updateStateFromTx(rawTx []byte) types.ResponseDeliverTx {
	tx, err := DecodeTx(rawTx)
	tags := []cmn.KVPair{}
	if err != nil {
		return types.ResponseDeliverTx{Code: code.CodeTypeEncodingError, Tags: tags}
	}
	var resp types.ResponseDeliverTx
	switch string(tx.TxType) {
	case "VAL":
		if isValidatorTx(tx.Data) {
			resp = app.execValidatorTx(tx.Data, tags)
		}
		app.incrementTx(tags)
		break
	case "CAL":
		resp = types.ResponseDeliverTx{Code: code.CodeTypeOK, Tags: tags}
		app.incrementTx(tags)
		break
	case "BTC-A":
		app.state.LatestBtcaTx = rawTx
		app.state.LatestBtcaHeight = app.state.Height + 1
		resp = types.ResponseDeliverTx{Code: code.CodeTypeOK, Tags: tags}
		app.incrementTx(tags)
		app.state.LatestBtcaTxInt = app.state.txInt
		break
	case "BTC-C":
		app.state.LatestBtccTx = rawTx
		app.state.LatestBtccHeight = app.state.Height + 1
		resp = types.ResponseDeliverTx{Code: code.CodeTypeOK, Tags: tags}
		app.incrementTx(tags)
		app.state.LatestBtccTxInt = app.state.txInt
		break
	case "NIST":
		resp = types.ResponseDeliverTx{Code: code.CodeTypeOK, Tags: tags}
		app.incrementTx(tags)
		break
	default:
		resp = types.ResponseDeliverTx{Code: code.CodeTypeUnauthorized, Tags: tags}
	}
	return resp
}

/*func GetBlockRangeRoots(tmServer string, tmPort string, minTxInt int64, maxTxInt int64) ([][]byte, error) {
	rpc := GetHTTPClient(tmServer, tmPort)
	defer rpc.Stop()
	Txs := types2.Txs{}
	for i := minTxInt; i < maxTxInt; i++ {
		txResult, err := rpc.TxSearch(fmt.Sprintf("txInt=%d", i))
		if err != nil {
			return nil, err
		} else {
			Txs = append(Txs, block.Block.Data.Txs...)
		}
	}
	calBytes := make([][]byte, len(Txs))
	for i, t := range Txs {
		calBytes[i] = t
	}
	return calBytes, nil
}*/
