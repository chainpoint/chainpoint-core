package abci

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/chainpoint/chainpoint-core/go-abci-service/util"
	"github.com/tendermint/tendermint/abci/example/code"
	"github.com/tendermint/tendermint/abci/types"
	cmn "github.com/tendermint/tendermint/libs/common"
	core_types "github.com/tendermint/tendermint/rpc/core/types"
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

func EncodeTx(outgoing Tx) string {
	txJSON, _ := json.Marshal(outgoing)
	return base64.StdEncoding.EncodeToString(txJSON)
}

func BroadcastTx(rpcUri TendermintURI, txType string, data string, version int64, time int64) (core_types.ResultBroadcastTx, error) {
	rpc := GetHTTPClient(rpcUri)
	defer rpc.Stop()
	tx := Tx{TxType: txType, Data: data, Version: version, Time: time}
	result, err := rpc.BroadcastTxSync([]byte(EncodeTx(tx)))
	if util.LogError(err) != nil {
		return *result, err
	}
	return *result, nil
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
		if isValidatorTx([]byte(tx.Data)) {
			resp = app.execValidatorTx([]byte(tx.Data), tags)
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

// GetTxRange gets all CAL TXs within a particular range
func GetTxRange(tendermintRPC TendermintURI, minTxInt int64, maxTxInt int64) ([]core_types.ResultTx, error) {
	fmt.Printf("minTxInt: %d, maxTxINt: %d\n", minTxInt, maxTxInt)
	if maxTxInt < minTxInt {
		return nil, errors.New("max of tx range is less than min")
	}
	rpc := GetHTTPClient(tendermintRPC)
	defer rpc.Stop()
	Txs := []core_types.ResultTx{}
	for i := minTxInt; i <= maxTxInt; i++ {
		txResult, err := rpc.TxSearch(fmt.Sprintf("TxInt=%d", i), false, 1, 1)
		if err != nil {
			fmt.Println("RPC error: " + err.Error())
			return nil, err
		} else if txResult.TotalCount > 0 {
			for _, tx := range txResult.Txs {

				Txs = append(Txs, *tx)

			}
		}
	}
	return Txs, nil
}
