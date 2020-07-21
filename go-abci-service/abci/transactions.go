package abci

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/chainpoint/chainpoint-core/go-abci-service/validation"

	"github.com/chainpoint/chainpoint-core/go-abci-service/types"

	types2 "github.com/tendermint/tendermint/abci/types"

	"github.com/chainpoint/chainpoint-core/go-abci-service/util"
	"github.com/tendermint/tendermint/abci/example/code"
	"github.com/tendermint/tendermint/libs/kv"
	core_types "github.com/tendermint/tendermint/rpc/core/types"
)

// incrementTxInt: Helper method to increment transaction integer
func (app *AnchorApplication) incrementTxInt(tags []kv.Pair) []kv.Pair {
	app.state.TxInt++ // no pre-increment :(
	return append(tags, kv.Pair{Key: []byte("TxInt"), Value: util.Int64ToByte(app.state.TxInt)})
}

func (app *AnchorApplication) validateTx(rawTx []byte) types2.ResponseCheckTx {
	var tx types.Tx
	var err error
	var valid bool
	if app.state.ChainSynced {
		tx, valid, err = validation.Validate(rawTx, &app.state)
	} else {
		tx, err = util.DecodeTx(rawTx)
		valid = true
	}
	if app.LogError(err) != nil {
		return types2.ResponseCheckTx{Code: code.CodeTypeUnauthorized, GasWanted: 1}
	}
	if !valid && tx.CoreID != app.ID {
		app.LogError(errors.New(fmt.Sprintf("Validation of peer %s transaction rate failed", tx.CoreID)))
		return types2.ResponseCheckTx{Code: 66, GasWanted: 1} //CodeType for peer disconnection
	}
	if tx.TxType == "FEE" {
		//i, err := strconv.ParseInt(tx.Data, 10, 64)
		if app.LogError(err) != nil {
			return types2.ResponseCheckTx{Code: code.CodeTypeUnknownError, GasWanted: 1}
		}
	}
	if tx.TxType == "VAL" {
		components := strings.Split(tx.Data, "!")
		if len(components) == 3 {
			amVal, _ := app.IsValidator(app.ID)
			isSubmitterVal, _ := app.IsValidator(tx.CoreID)
			if !isSubmitterVal {
				if _, submitterRecord, err := validation.GetValidationRecord(tx.CoreID, app.state); err != nil {
					submitterRecord.UnAuthValSubmissions++
					validation.SetValidationRecord(tx.CoreID, submitterRecord, &app.state)
				}
			}
			id := components[0]
			if amVal {
				goodCandidate := false
				if _, record, err := validation.GetValidationRecord(id, app.state); err != nil {
					numValidators := len(app.Validators)
					power, err := strconv.ParseInt(components[2], 10, 32)
					if err != nil {
						return types2.ResponseCheckTx{Code: code.CodeTypeUnauthorized, GasWanted: 1}
					}
					if power == -1 {
						goodCandidate = true
					} else {
						goodCandidate = record.ConfirmedAnchors > int64(SUCCESSFUL_ANCHOR_CRITERIA+10*numValidators) || app.config.BitcoinNetwork == "testnet"
					}
				}
				if !(goodCandidate && app.PendingValidator == tx.Data) {
					app.logger.Info("Validator failed to validate VAL tx")
					return types2.ResponseCheckTx{Code: code.CodeTypeUnauthorized, GasWanted: 1}
				}
			}
		} else {
			app.logger.Info("Validator failed to validate VAL tx structure")
			return types2.ResponseCheckTx{Code: code.CodeTypeUnauthorized, GasWanted: 1}
		}
	}
	if tx.TxType == "JWK" && !app.VerifyIdentity(tx) {
		app.logger.Info("Unable to validate JWK Identity", "CoreID", tx.CoreID)
		return types2.ResponseCheckTx{Code: code.CodeTypeUnauthorized, GasWanted: 1}
	}else if tx.TxType == "JWK" {
		app.logger.Info("JWK Identity validated", "CoreID", tx.CoreID)
	}
	return types2.ResponseCheckTx{Code: code.CodeTypeOK, GasWanted: 1}
}

// updateStateFromTx: Updates state based on type of transaction received. Used by DeliverTx
func (app *AnchorApplication) updateStateFromTx(rawTx []byte, gossip bool) types2.ResponseDeliverTx {
	var tx types.Tx
	var err error
	var resp types2.ResponseDeliverTx
	tags := []kv.Pair{}
	if app.state.ChainSynced {
		tx, err = util.DecodeTxAndVerifySig(rawTx, app.state.CoreKeys)
	} else {
		tx, err = util.DecodeTx(rawTx)
	}
	app.logger.Info(fmt.Sprintf("Received Tx: %s, Gossip: %t", tx.TxType, gossip))
	app.LogError(err)
	switch string(tx.TxType) {
	case "VAL":
		components := strings.Split(tx.Data, "!")
		if len(components) == 3 {
			data := components[1] + "!" + components[2]
			tags = app.incrementTxInt(tags)
			app.logger.Info(fmt.Sprintf("Val tx: %s", data))
			if isValidatorTx([]byte(data)) {
				app.logger.Info("Executing VAL tx")
				resp = app.execValidatorTx([]byte(data))
			}
		}
		break
	case "CAL":
		tags = app.incrementTxInt(tags)
		app.state.LatestCalTxInt = app.state.TxInt
		resp = types2.ResponseDeliverTx{Code: code.CodeTypeOK}
		break
	case "BTC-E":
		app.state.LatestErrRoot = tx.Data
		app.state.LastErrorCoreID = tx.CoreID
		resp = types2.ResponseDeliverTx{Code: code.CodeTypeOK}
	case "BTC-A":
		var btca types.BtcTxMsg
		if util.LoggerError(app.logger, json.Unmarshal([]byte(tx.Data), &btca)) != nil {
			break
		}
		//Begin monitoring using the data contained in this transaction
		if app.state.ChainSynced {
			go app.ConsumeBtcTxMsg([]byte(tx.Data))
			app.logger.Info(fmt.Sprintf("BTC-A Anchor Data: %s", tx.Data))
		}
		app.state.LatestBtcaTx = rawTx
		app.state.LatestBtcaHeight = app.state.Height + 1
		tags = app.incrementTxInt(tags)
		app.state.LatestBtcaTxInt = app.state.TxInt
		app.state.BeginCalTxInt = btca.EndCalTxInt // Keep a placeholder in case a CAL Tx is sent in between the time of a BTC-A broadcast and its handling
		tags = append(tags, kv.Pair{Key: []byte("BTCTX"), Value: []byte(btca.BtcTxID)})
		resp = types2.ResponseDeliverTx{Code: code.CodeTypeOK}
		break
	case "BTC-C":
		if tx.Data == string(app.state.LatestBtccTx) {
			app.logger.Info(fmt.Sprintf("We've already seen this BTC-C confirmation tx: %s", tx.Data))
			resp = types2.ResponseDeliverTx{Code: code.CodeTypeUnauthorized}
			break
		}
		app.state.LatestBtccTx = []byte(tx.Data)
		app.state.LatestBtccHeight = app.state.Height + 1
		tags = app.incrementTxInt(tags)
		app.state.LatestBtccTxInt = app.state.TxInt
		tags = append(tags, kv.Pair{Key: []byte("BTCC"), Value: []byte(tx.Data)})
		meta := strings.Split(tx.Meta, "|") // first part of meta is core ID that issued TX, second part is BTC TX ID
		if len(meta) > 0 {
			app.state.LastAnchorCoreID = meta[0]
			if app.state.ChainSynced {
				go app.AnchorReward(app.state.LastAnchorCoreID)
			}
			validation.IncrementSuccessAnchor(app.state.LastAnchorCoreID, &app.state)
		}
		resp = types2.ResponseDeliverTx{Code: code.CodeTypeOK}
		break
	case "NIST":
		app.state.LatestNistRecord = tx.Data
		resp = types2.ResponseDeliverTx{Code: code.CodeTypeOK}
		if app.config.DoCal {
			app.aggregator.LatestTime = app.state.LatestNistRecord
		}
		break
	case "DRAND":
		app.state.LatestTimeRecord = tx.Data
		resp = types2.ResponseDeliverTx{Code: code.CodeTypeOK}
		if app.config.DoCal {
			app.aggregator.LatestTime = app.state.LatestTimeRecord
		}
		break
	case "FEE":
		i, err := strconv.ParseInt(tx.Data, 10, 64)
		if app.LogError(err) != nil {
			resp = types2.ResponseDeliverTx{Code: code.CodeTypeEncodingError}
			break
		}
		app.state.LatestBtcFee = i
		app.state.LastBtcFeeHeight = app.state.Height
		app.lnClient.LastFee = app.state.LatestBtcFee
		resp = types2.ResponseDeliverTx{Code: code.CodeTypeOK}
		break
	case "JWK":
		if app.LogError(app.SaveIdentity(tx)) == nil {
			tags = app.incrementTxInt(tags)
			tags = append(tags, kv.Pair{Key: []byte("CORE"), Value: []byte("NEW")})
			resp = types2.ResponseDeliverTx{Code: code.CodeTypeOK}
			break
		}
		fallthrough
	default:
		resp = types2.ResponseDeliverTx{Code: code.CodeTypeUnauthorized}
	}
	events := []types2.Event{
		{
			Type: tx.TxType,
			Attributes: tags,
		},
	}
	resp.Events = events
	return resp
}

// GetTxRange gets all CAL TXs within a particular range
func (app *AnchorApplication) getCalTxRange(minTxInt int64, maxTxInt int64) ([]core_types.ResultTx, error) {
	if maxTxInt <= minTxInt {
		return nil, errors.New("max of tx range is less than or equal to min")
	}
	Txs := []core_types.ResultTx{}
	for i := minTxInt; i <= maxTxInt; i++ {
		txResult, err := app.rpc.client.TxSearch(fmt.Sprintf("CAL.TxInt=%d", i), false, 1, 1, "")
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

// getAllJWKs gets all JWK TXs
func (app *AnchorApplication) getAllJWKs() ([]types.Tx, error) {
	Txs := []types.Tx{}
	txResult, err := app.rpc.client.TxSearch("JWK.CORE='NEW'", false, 1, 200, "")
	if err != nil {
		return nil, err
	} else if txResult.TotalCount > 0 {
		app.logger.Info(fmt.Sprintf("Found %d JWK tx while loading", txResult.TotalCount))
		for _, tx := range txResult.Txs {
			decoded, err := util.DecodeTx(tx.Tx)
			if app.LogError(err) == nil {
				Txs = append(Txs, decoded)
			}
		}
	}
	return Txs, nil
}
