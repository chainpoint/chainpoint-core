package abci

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/chainpoint/chainpoint-core/leaderelection"
	"github.com/tendermint/tendermint/crypto/tmhash"
	"strconv"
	"strings"
	"time"

	"github.com/chainpoint/chainpoint-core/validation"

	"github.com/chainpoint/chainpoint-core/types"

	types2 "github.com/tendermint/tendermint/abci/types"

	"github.com/chainpoint/chainpoint-core/util"
	"github.com/tendermint/tendermint/abci/example/code"
	"github.com/tendermint/tendermint/libs/kv"
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
		tx, valid, err = validation.Validate(rawTx, app.state)
	} else {
		tx, err = util.DecodeTx(rawTx)
		valid = true
		app.logger.Info("Syncing, tx validation skipped")
	}
	if app.LogError(err) != nil {
		return types2.ResponseCheckTx{Code: code.CodeTypeUnauthorized, GasWanted: 1}
	}
	if !valid && tx.CoreID != app.ID {
		app.LogError(errors.New(fmt.Sprintf("Validation of peer %s transaction rate failed for tx %+v", tx.CoreID, tx)))
		return types2.ResponseCheckTx{Code: code.CodeTypeUnauthorized, GasWanted: 1} //CodeType for peer disconnection
	}
	amVal, _ := leaderelection.IsValidator(*app.state, app.ID)
	isSubmitterVal, _ := leaderelection.IsValidator(*app.state, tx.CoreID)
	switch string(tx.TxType) {
	case "BTC-A":
		var btcTxObj types.BtcTxMsg
		if err := json.Unmarshal([]byte(tx.Data), &btcTxObj); app.LogError(err) != nil {
			return types2.ResponseCheckTx{Code: code.CodeTypeUnauthorized, GasWanted: 1}
		}
		if matchErr := app.Anchor.CheckAnchor(btcTxObj); app.LogError(matchErr) != nil {
			return types2.ResponseCheckTx{Code: code.CodeTypeUnauthorized, GasWanted: 1}
		}
	case "VAL":
		err, id, _, power, _ := ValidateValidatorTx(tx.Data)
		if app.LogError(err) != nil {
			return types2.ResponseCheckTx{
				Code: code.CodeTypeEncodingError,
				Log:  fmt.Sprintf(err.Error()),
			}
		}
		if !isSubmitterVal {
			if _, submitterRecord, err := validation.GetValidationRecord(tx.CoreID, *app.state); err != nil {
				submitterRecord.UnAuthValSubmissions++
				validation.SetValidationRecord(tx.CoreID, submitterRecord, app.state)
			}
		}
		if amVal {
			goodCandidate := false
			if _, record, err := validation.GetValidationRecord(id, *app.state); err != nil {
				numValidators := len(app.state.Validators)
				if power <= 0 { //make it easier to get rid of a validator than to promote one
					goodCandidate = true
				} else {
					goodCandidate = record.ConfirmedAnchors > int64(SUCCESSFUL_ANCHOR_CRITERIA+10*numValidators) || app.config.BitcoinNetwork == "testnet"
				}
			}
			if !(goodCandidate && app.PendingValidator == tx.Data)  {
				if id != "182267DF3316F8C487F68214CC2EA42256B26F07" {
					app.logger.Info("Validator failed to validate VAL tx", "id", id, "goodCandidate", goodCandidate, "PendingValidator", app.PendingValidator, "tx.Data", tx.Data)
					return types2.ResponseCheckTx{Code: code.CodeTypeUnauthorized, GasWanted: 1}
				}
			}
		}
	case "CHNGSTK":
		newStakePerCore, err := strconv.ParseInt(tx.Data, 10, 64)
		if err != nil || newStakePerCore != app.PendingChangeStake {
			app.logger.Info("Stake proposal does not match configuration")
			return types2.ResponseCheckTx{Code: code.CodeTypeUnauthorized, GasWanted: 1}
		}
	case "JWK":
		if !app.VerifyIdentity(tx) {
			app.logger.Info("Unable to validate JWK Identity", "CoreID", tx.CoreID)
			return types2.ResponseCheckTx{Code: code.CodeTypeUnauthorized, GasWanted: 1}
		} else {
			app.logger.Info("JWK Identity validated", "CoreID", tx.CoreID)
		}
	}

	return types2.ResponseCheckTx{Code: code.CodeTypeOK, GasWanted: 1}
}

// updateStateFromTx: Updates state based on type of transaction received. Used by DeliverTx
func (app *AnchorApplication) updateStateFromTx(rawTx []byte) types2.ResponseDeliverTx {
	var tx types.Tx
	var err error
	var resp = types2.ResponseDeliverTx{Code: code.CodeTypeUnauthorized}
	tags := []kv.Pair{}
	if app.state.ChainSynced {
		tx, err = util.DecodeTxAndVerifySig(rawTx, app.state.CoreKeys)
	} else {
		tx, err = util.DecodeTx(rawTx)
	}
	app.logger.Info(fmt.Sprintf("DeliverTx: %s", tx.TxType))
	app.LogError(err)
	switch string(tx.TxType) {
	case "VAL":
		tags = app.incrementTxInt(tags)
		app.logger.Info(fmt.Sprintf("Val tx: %s", tx.Data))
		resp = app.execValidatorTx([]byte(tx.Data))
	case "CHNGSTK":
		newStakePerCore, err := strconv.ParseInt(tx.Data, 10, 64)
		if app.LogError(err) != nil {
			break
		}
		app.config.StakePerCore = newStakePerCore
		tags = app.incrementTxInt(tags)
		tags = append(tags, kv.Pair{Key: []byte("CHANGE"), Value: []byte("STAKE")})
		resp = types2.ResponseDeliverTx{Code: code.CodeTypeOK}
	case "CAL":
		go func() {
			time.Sleep(1 * time.Minute)
			txHash := tmhash.Sum(rawTx)
			app.logger.Info("Removing Cal Hash from Db:", "hash", hex.EncodeToString(txHash))
			app.Cache.Del(hex.EncodeToString(txHash), "")
		}()
		tags = app.incrementTxInt(tags)
		app.state.LatestCalTxInt = app.state.TxInt
		app.state.CurrentCalInts++
		resp = types2.ResponseDeliverTx{Code: code.CodeTypeOK}
	case "BTC-E":
		app.state.LatestErrRoot = tx.Data
		app.state.LastErrorCoreID = tx.CoreID
		app.logger.Info(fmt.Sprintf("BTC-E from %s", tx.CoreID))
		resp = types2.ResponseDeliverTx{Code: code.CodeTypeOK}
	case "BTC-A":
		var btca types.BtcTxMsg
		if util.LoggerError(app.logger, json.Unmarshal([]byte(tx.Data), &btca)) != nil {
			break
		}
		//Begin monitoring using the data contained in this transaction
		if app.state.ChainSynced {
			go app.Anchor.BeginTxMonitor([]byte(tx.Data))
			app.logger.Info(fmt.Sprintf("BTC-A StartAnchoring Data: %s", tx.Data))
		} else {
			app.state.BeginCalTxInt = btca.EndCalTxInt
		}
		app.state.LatestBtcaTx = rawTx
		app.state.LatestBtcaHeight = app.state.Height + 1
		tags = app.incrementTxInt(tags)
		app.state.LatestBtcaTxInt = app.state.TxInt
		// Keep a placeholder in case a CAL Tx is sent in between the time of a BTC-A broadcast and its handling
		tags = append(tags, kv.Pair{Key: []byte("BTCTX"), Value: []byte(btca.BtcTxID)})
		resp = types2.ResponseDeliverTx{Code: code.CodeTypeOK}
	case "BTC-C":
		if tx.Data == string(app.state.LatestBtccTx) {
			app.logger.Info(fmt.Sprintf("We've already seen this BTC-C confirmation tx: %s", tx.Data))
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
				go app.Anchor.AnchorReward(app.state.LastAnchorCoreID)
			}
			validation.IncrementSuccessAnchor(app.state.LastAnchorCoreID, app.state)
		}
		resp = types2.ResponseDeliverTx{Code: code.CodeTypeOK}
	case "NIST":
		resp = types2.ResponseDeliverTx{Code: code.CodeTypeOK}
	case "DRAND":
		resp = types2.ResponseDeliverTx{Code: code.CodeTypeOK}
	case "FEE":
		i, err := strconv.ParseInt(tx.Data, 10, 64)
		if app.LogError(err) != nil {
			break
		}
		app.state.LatestBtcFee = i
		app.state.LastBtcFeeHeight = app.state.Height
		app.LnClient.LastFee = app.state.LatestBtcFee
		resp = types2.ResponseDeliverTx{Code: code.CodeTypeOK}
	case "JWK":
		if app.LogError(app.SaveIdentity(tx)) == nil {
			tags = app.incrementTxInt(tags)
			tags = append(tags, kv.Pair{Key: []byte("CORE"), Value: []byte("NEW")})
			resp = types2.ResponseDeliverTx{Code: code.CodeTypeOK}
		}
	}
	events := []types2.Event{
		{
			Type:       tx.TxType,
			Attributes: tags,
		},
	}
	resp.Events = events
	return resp
}
