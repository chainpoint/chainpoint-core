package abci

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/chainpoint/chainpoint-core/go-abci-service/types"

	types2 "github.com/chainpoint/tendermint/abci/types"

	"github.com/chainpoint/chainpoint-core/go-abci-service/util"
	"github.com/chainpoint/tendermint/abci/example/code"
	common "github.com/chainpoint/tendermint/libs/common"
	core_types "github.com/chainpoint/tendermint/rpc/core/types"
)

// incrementTxInt: Helper method to increment transaction integer
func (app *AnchorApplication) incrementTxInt(tags []common.KVPair) []common.KVPair {
	app.state.TxInt++ // no pre-increment :(
	return append(tags, common.KVPair{Key: []byte("TxInt"), Value: util.Int64ToByte(app.state.TxInt)})
}

func (app *AnchorApplication) validateGossip(rawTx []byte) types2.ResponseCheckTx {
	var tx types.Tx
	var err error
	if app.state.ChainSynced {
		tx, err = util.DecodeVerifyTx(rawTx, app.CoreKeys)
	} else {
		tx, err = util.DecodeTx(rawTx)
	}
	app.logger.Info(fmt.Sprintf("CheckTX: %v", tx))
	if util.LoggerError(app.logger, err) != nil {
		return types2.ResponseCheckTx{Code: code.CodeTypeUnauthorized, GasWanted: 1}
	}
	// this serves as a shim for CheckTx so transactions we don't want in the mempool can
	// still be gossipped to other Cores
	if util.Contains(GossipTxs, tx.TxType) {
		go app.rpc.BroadcastMsg(tx)
		return types2.ResponseCheckTx{Code: code.CodeTypeUnauthorized, GasWanted: 1}
	}
	return types2.ResponseCheckTx{Code: code.CodeTypeOK, GasWanted: 1}
}

// updateStateFromTx: Updates state based on type of transaction received. Used by DeliverTx
func (app *AnchorApplication) updateStateFromTx(rawTx []byte, gossip bool) types2.ResponseDeliverTx {
	var tx types.Tx
	var err error
	var resp types2.ResponseDeliverTx
	tags := []common.KVPair{}
	if app.state.ChainSynced {
		tx, err = util.DecodeVerifyTx(rawTx, app.CoreKeys)
	} else {
		tx, err = util.DecodeTx(rawTx)
	}
	app.logger.Info(fmt.Sprintf("Received Tx: %s, Gossip: %t", tx.TxType, gossip))
	util.LoggerError(app.logger, err)
	switch string(tx.TxType) {
	case "VAL":
		tags = app.incrementTxInt(tags)
		if isValidatorTx([]byte(tx.Data)) {
			resp = app.execValidatorTx([]byte(tx.Data), tags)
		}
		break
	case "CAL":
		tags = app.incrementTxInt(tags)
		app.state.LatestCalTxInt = app.state.TxInt
		resp = types2.ResponseDeliverTx{Code: code.CodeTypeOK, Tags: tags}
		break
	case "BTC-M":
		//Begin monitoring using the data contained in this gossiped (but ultimately nacked) transaction
		app.state.LatestBtcmHeight = app.state.Height + 1
		app.state.LatestBtcmTxInt = app.state.TxInt
		app.state.LatestBtcmTx = base64.StdEncoding.EncodeToString(rawTx)
		app.ConsumeBtcTxMsg([]byte(tx.Data))
		app.logger.Info(fmt.Sprintf("BTC-M Anchor Data: %s", tx.Data))
		if leader, _ := app.ElectValidator(1); leader && app.state.ChainSynced {
			go app.rpc.BroadcastTx("CORE-RC", tx.CoreID, 2, time.Now().Unix(), app.ID, &app.config.ECPrivateKey)
		}
		resp = types2.ResponseDeliverTx{Code: code.CodeTypeOK, Tags: tags}
		break
	case "BTC-A":
		app.state.LatestBtcaTx = rawTx
		app.state.LatestBtcaHeight = app.state.Height + 1
		tags = app.incrementTxInt(tags)
		app.state.LatestBtcaTxInt = app.state.TxInt
		app.state.BeginCalTxInt = app.state.EndCalTxInt // Keep a placeholder in case a CAL Tx is sent in between the time of a BTC-A broadcast and its handling
		app.state.LastAnchorCoreID = tx.CoreID
		resp = types2.ResponseDeliverTx{Code: code.CodeTypeOK, Tags: tags}
		break
	case "BTC-C":
		app.state.LatestBtccTx = rawTx
		app.state.LatestBtccHeight = app.state.Height + 1
		tags = app.incrementTxInt(tags)
		app.state.LatestBtccTxInt = app.state.TxInt
		resp = types2.ResponseDeliverTx{Code: code.CodeTypeOK, Tags: tags}
		break
	case "NIST":
		app.state.LatestNistRecord = tx.Data
		resp = types2.ResponseDeliverTx{Code: code.CodeTypeOK, Tags: tags}
		break
	case "NODE-MINT":
		lastMintedAtBlock, err := strconv.ParseInt(tx.Data, 10, 64)
		if err != nil {
			app.logger.Debug("Parsing Node MINT tx failed")
		} else {
			app.state.PrevNodeMintedAtBlock = app.state.LastNodeMintedAtBlock
			app.state.LastNodeMintedAtBlock = lastMintedAtBlock
		}
		resp = types2.ResponseDeliverTx{Code: code.CodeTypeOK, Tags: tags}
		break
	case "CORE-MINT":
		lastMintedAtBlock, err := strconv.ParseInt(tx.Data, 10, 64)
		if err != nil {
			app.logger.Debug("Parsing Core MINT tx failed")
		} else {
			app.state.PrevCoreMintedAtBlock = app.state.LastCoreMintedAtBlock
			app.state.LastCoreMintedAtBlock = lastMintedAtBlock
		}
		resp = types2.ResponseDeliverTx{Code: code.CodeTypeOK, Tags: tags}
		break
	case "JWK":
		app.SaveJWK(tx)
		resp = types2.ResponseDeliverTx{Code: code.CodeTypeOK, Tags: tags}
		break
	case "CORE-SIGN":
		app.CoreRewardSignatures = util.UniquifyStrings(append(app.CoreRewardSignatures, tx.Data))
		resp = types2.ResponseDeliverTx{Code: code.CodeTypeOK, Tags: tags}
		break
	case "NODE-SIGN":
		app.NodeRewardSignatures = util.UniquifyStrings(append(app.NodeRewardSignatures, tx.Data))
		resp = types2.ResponseDeliverTx{Code: code.CodeTypeOK, Tags: tags}
		break
	case "NODE-RC":
		tags = app.incrementTxInt(tags)
		tags = append(tags, common.KVPair{Key: []byte("NODERC"), Value: util.Int64ToByte(app.state.LastNodeMintedAtBlock)})
		resp = types2.ResponseDeliverTx{Code: code.CodeTypeOK, Tags: tags}
		break
	case "CORE-RC":
		tags = app.incrementTxInt(tags)
		tags = append(tags, common.KVPair{Key: []byte("CORERC"), Value: util.Int64ToByte(app.state.LastCoreMintedAtBlock)})
		resp = types2.ResponseDeliverTx{Code: code.CodeTypeOK, Tags: tags}
		break
	case "TOKEN":
		go app.pgClient.TokenHashUpsert(tx.Data)
		resp = types2.ResponseDeliverTx{Code: code.CodeTypeOK, Tags: tags}
		break
	default:
		resp = types2.ResponseDeliverTx{Code: code.CodeTypeUnauthorized, Tags: tags}
	}
	return resp
}

// GetTxRange gets all CAL TXs within a particular range
func (app *AnchorApplication) getCalTxRange(minTxInt int64, maxTxInt int64) ([]core_types.ResultTx, error) {
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
