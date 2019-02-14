package abci

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/chainpoint/chainpoint-core/go-abci-service/util"
	"github.com/tendermint/tendermint/abci/example/code"
	"github.com/tendermint/tendermint/abci/types"
	cmn "github.com/tendermint/tendermint/libs/common"
	types2 "github.com/tendermint/tendermint/types"
)


func decodeTx(incoming []byte) (Tx, error){
	decoded, err := base64.StdEncoding.DecodeString(string(incoming))
	var calendar Tx
	if err != nil {
		fmt.Println(err)
		return calendar, err
	}
	json.Unmarshal([]byte(decoded), &calendar)
	return calendar, nil
}

func (app *AnchorApplication) updateStateFromTx(rawTx []byte) (types.ResponseDeliverTx){
	tx, err := decodeTx(rawTx)
	tags := []cmn.KVPair{}
	if err != nil{
		return types.ResponseDeliverTx{Code: code.CodeTypeEncodingError, Tags: tags}
	}
	var resp types.ResponseDeliverTx
	switch string(tx.TxType) {
	case "VAL":
		if isValidatorTx(tx.Data) {
			resp = app.execValidatorTx(tx.Data, tags)
		}
		break
	case "CAL":
		resp = types.ResponseDeliverTx{Code: code.CodeTypeOK, Tags: tags}
		break
	case "BTC-A":
		app.state.db.Set([]byte("latest_btca"), rawTx)
		app.state.db.Set([]byte("latest_btca_height"), util.Int64ToByte(app.state.Height+1))
		resp = types.ResponseDeliverTx{Code: code.CodeTypeOK, Tags: tags}
		break
	case "BTC-C":
		app.state.db.Set([]byte("latest_btcc"), rawTx)
		app.state.db.Set([]byte("latest_btcc_height"), util.Int64ToByte(app.state.Height+1))
		resp = types.ResponseDeliverTx{Code: code.CodeTypeOK, Tags: tags}
		break
	case "NIST":
		resp = types.ResponseDeliverTx{Code: code.CodeTypeOK, Tags: tags}
		break
	default:
		resp = types.ResponseDeliverTx{Code: code.CodeTypeUnauthorized, Tags: tags}
	}
	return resp
}

func GetBlockRangeRoots(tmServer string, tmPort string, minHeight int64, maxHeight int64) ([][]byte, error) {
	rpc := GetHTTPClient(tmServer, tmPort)
	defer rpc.Stop()
	Txs := types2.Txs{}
	for i := minHeight; i < maxHeight; i++ {
		block, err := rpc.Block(&i)
		if block != nil {
			Txs = append(Txs, block.Block.Data.Txs...)
		} else {
			return nil, err
		}
	}
	calBytes := make([][]byte, len(Txs))
	for i, t := range Txs {
		calBytes[i] = t
	}
	return calBytes, nil
}