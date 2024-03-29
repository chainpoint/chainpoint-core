package bitcoin

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	analytics2 "github.com/chainpoint/chainpoint-core/analytics"
	"github.com/chainpoint/chainpoint-core/calendar"
	"github.com/chainpoint/chainpoint-core/database"
	"github.com/chainpoint/chainpoint-core/database/level"
	"github.com/chainpoint/chainpoint-core/leaderelection"
	"github.com/chainpoint/chainpoint-core/proof"
	"github.com/chainpoint/chainpoint-core/tendermintrpc"
	"github.com/chainpoint/chainpoint-core/types"
	"github.com/chainpoint/chainpoint-core/util"
	lightning "github.com/chainpoint/lightning-go"
	merkletools "github.com/chainpoint/merkletools-go"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/tendermint/tendermint/libs/log"
	"strconv"
	"strings"
	"time"
)

const CONFIRMED_BTC_TX_IDS_KEY = "BTC_Mon:ConfirmedBTCTxIds"
const CHECK_BTC_TX_IDS_KEY = "BTC_Mon:CheckNewBTCTxIds"

type AnchorBTC struct {
	state         *types.AnchorState
	config        types.AnchorConfig
	tendermintRpc *tendermintrpc.RPC
	Cache         *level.KVStore
	Db            database.ChainpointDatabase
	LnClient      *lightning.LightningClient
	logger        log.Logger
	analytics     *analytics2.UniversalAnalytics
}

func NewBTCAnchorEngine(state *types.AnchorState, config types.AnchorConfig, tendermintRpc *tendermintrpc.RPC,
	database *database.ChainpointDatabase, cache *level.KVStore, LnClient *lightning.LightningClient, logger log.Logger, analytics *analytics2.UniversalAnalytics) *AnchorBTC {
	return &AnchorBTC{
		state:         state,
		config:        config,
		tendermintRpc: tendermintRpc,
		Cache:         cache,
		Db:            *database,
		LnClient:      LnClient,
		logger:        logger,
		analytics:     analytics,
	}
}

func (app *AnchorBTC) GetTreeFromCalRange(startTxRange int64, endTxRange int64) (types.BtcAgg, error) {
	// GetArray CAL transactions between the latest BTCA tx and the current latest tx
	txLeaves, err := app.tendermintRpc.GetCalTxRange(startTxRange, endTxRange)
	app.logger.Info(fmt.Sprintf("Retrieved %d CAL leaves from ranges %d to %d", len(txLeaves), startTxRange, endTxRange))
	if app.LogError(err) != nil {
		return types.BtcAgg{}, err
	}
	// Aggregate all txs in range into a new merkle tree in prep for BTC anchoring
	treeData := calendar.AggregateAnchorTx(txLeaves)
	return treeData, nil
}

// AnchorToChain : StartAnchoring scans all CAL transactions since last anchor epoch and writes the merkle root to the Calendar and to bitcoin
func (app *AnchorBTC) AnchorToChain(startTxRange int64, endTxRange int64) error {
	// elect leader to do the actual anchoring
	if app.config.ElectionMode == "test" {
		app.state.LastErrorCoreID = ""
	}
	iAmLeader, leaderIDs := leaderelection.ElectChainContributorAsLeader(1, []string{app.state.LastErrorCoreID}, *app.state)
	if len(leaderIDs) == 0 {
		return errors.New("Leader election error")
	}
	app.logger.Info(fmt.Sprintf("StartAnchoring Leaders: %v", leaderIDs))

	treeData, err := app.GetTreeFromCalRange(startTxRange, endTxRange)
	if err != nil {
		return err
	}
	app.logger.Info(fmt.Sprintf("StartAnchoring tx ranges %d to %d at Height %d, latestBtcaHeight %d, for aggroot: %s", startTxRange, endTxRange, app.state.Height, app.state.LatestBtcaHeight, treeData.AnchorBtcAggRoot))
	app.logger.Info(fmt.Sprintf("treeData for StartAnchoring: %#v", treeData))

	// If we have something to anchor, perform anchoring and proofgen functions
	if treeData.AnchorBtcAggRoot != "" {
		app.state.LastElectedCoreID = leaderIDs[0]

		if treeData.AnchorBtcAggRoot == app.state.LatestErrRoot {
			app.state.LatestErrRoot = ""
		}
		// elect anchorer
		if iAmLeader {
			btcTx, btca, err := app.SendBtcTx(treeData, app.state.Height, startTxRange, endTxRange)
			if app.LogError(err) != nil {
				_, err := app.tendermintRpc.BroadcastTx("BTC-E", treeData.AnchorBtcAggRoot, 2, time.Now().Unix(), app.state.ID, app.config.ECPrivateKey)
				if app.LogError(err) != nil {
					panic(err)
				}
			}
			_, err = app.tendermintRpc.BroadcastTx("BTC-A", string(btca), 2, time.Now().Unix(), app.state.ID, app.config.ECPrivateKey)
			if app.LogError(err) != nil {
				app.logger.Info(fmt.Sprintf("failed sending BTC-A"))
				panic(err)
			} else {
				go app.analytics.SendEvent(app.state.LatestTimeRecord, "CreateAnchorTx", btcTx, time.Now().Format(time.RFC3339), "", strconv.FormatInt(app.state.LatestBtcFee*4/1000, 10), "")
			}
		}

		// begin monitoring for anchor
		failedAnchorCheck := types.AnchorRange{
			AnchorBtcAggRoot: treeData.AnchorBtcAggRoot,
			CalBlockHeight:   app.state.Height,
			BtcBlockHeight:   int64(app.state.LNState.BlockHeight),
			BeginCalTxInt:    startTxRange,
			EndCalTxInt:      endTxRange,
			AmLeader:         iAmLeader,
		}
		failedAnchorJSON, _ := json.Marshal(failedAnchorCheck)
		err := app.Cache.Append(CHECK_BTC_TX_IDS_KEY, string(failedAnchorJSON))
		if app.LogError(err) != nil {
			return err
		}
		app.state.BeginCalTxInt = endTxRange
		app.state.EndCalTxInt = endTxRange            // Ensure we update our range of CAL txs for next anchor period
		app.state.LatestBtcaHeight = app.state.Height // So no one will try to re-anchor while processing the btc tx
		return nil
	}
	return errors.New("no transactions to aggregate")
}

// SendBtcTx : sends btc tx to lnd and enqueues tx monitoring information
func (app *AnchorBTC) SendBtcTx(anchorDataObj types.BtcAgg, height int64, start int64, end int64) (string, []byte, error) {
	hexRoot, err := hex.DecodeString(anchorDataObj.AnchorBtcAggRoot)
	if util.LogError(err) != nil {
		return "", []byte{}, err
	}
	txid, rawtx, err := app.LnClient.SendOpReturn(hexRoot)
	if util.LogError(err) != nil {
		return "", []byte{}, err
	}
	msgBtcMon := types.BtcTxMsg{
		AnchorBtcAggID:   anchorDataObj.AnchorBtcAggID,
		AnchorBtcAggRoot: anchorDataObj.AnchorBtcAggRoot,
		BtcTxID:          txid,
		BtcTxBody:        rawtx,
		BtcTxHeight:      0,
		CalBlockHeight:   height,
		BeginCalTxInt:    start,
		EndCalTxInt:      end,
	}
	btcJSON, err := json.Marshal(msgBtcMon)
	app.logger.Info(fmt.Sprint("Sending BTC-A OP_RETURN: %#v", msgBtcMon))
	return txid, btcJSON, err
}

// AnchorReward : Send sats to last anchoring core
func (app *AnchorBTC) AnchorReward(CoreID string) error {
	if val, exists := app.state.LnUris[CoreID]; exists && app.config.AnchorReward > 0 {
		if !app.state.ChainSynced {
			return errors.New("Reward not sent; Chain not yet synced")
		}
		ip := lightning.GetIpFromUri(val.Peer)
		if len(ip) == 0 {
			return errors.New("Reward not sent; Can't obtain IP for peer")
		}
		status := util.GetAPIStatus(ip)
		if len(status.LightningAddress) == 0 {
			return errors.New("Reward not sent; Can't obtain status for peer")
		}
		resp, err := app.LnClient.SendCoins(status.LightningAddress, int64(app.config.AnchorReward), int32(app.LnClient.MinConfs))
		if app.LogError(err) != nil {
			return err
		}
		app.logger.Info(fmt.Sprintf("Reward Sent to %s with txid %s", CoreID, resp.Txid))
	}
	app.logger.Info(fmt.Sprintf("Reward of %d not sent to CoreID %s", app.config.AnchorReward, CoreID))
	return errors.New(fmt.Sprintf("Reward not sent; LnURI of CoreID %s not found in local database", CoreID))
}

// BeginTxMonitor : Consumes a btctx message to initiate monitoring on all nodes
func (app *AnchorBTC) BeginTxMonitor(msgBytes []byte) error {
	var btcTxObj types.BtcTxMsg
	if err := json.Unmarshal(msgBytes, &btcTxObj); err != nil {
		return app.LogError(err)
	}
	app.state.LatestBtcTx = btcTxObj.BtcTxID // Update app state with txID so we can broadcast BTC-A
	app.state.LatestBtcAggRoot = btcTxObj.AnchorBtcAggRoot
	stateObj := calendar.GenerateAnchorBtcTxState(btcTxObj)
	app.logger.Info(fmt.Sprintf("BTC-A BtcTx State Obj: %#v", stateObj))
	err := app.Db.BulkInsertBtcTxState([]types.AnchorBtcTxState{stateObj})
	if app.LogError(err) != nil {
		return err
	}

	txIDBytes, err := json.Marshal(types.TxID{TxID: btcTxObj.BtcTxID, AnchorBtcAggRoot: btcTxObj.AnchorBtcAggRoot})
	err = app.Cache.Append(CONFIRMED_BTC_TX_IDS_KEY, string(txIDBytes))
	if err != nil {
		return err
	}
	// end monitoring for failed anchor
	app.FindAndRemoveBtcCheck(btcTxObj.AnchorBtcAggRoot)

	btcAgg, err := app.GetTreeFromCalRange(btcTxObj.BeginCalTxInt, btcTxObj.EndCalTxInt)
	if app.LogError(err) != nil {
		return err
	}
	if btcAgg.AnchorBtcAggRoot != btcTxObj.AnchorBtcAggRoot {
		app.logger.Info(fmt.Sprintf("BTC-A StartAnchoring TreeData calculation failure for BTC-A aggroot: %s, local treeData result was %s", btcTxObj.AnchorBtcAggRoot, btcAgg.AnchorBtcAggRoot))
		app.logger.Info(fmt.Sprintf("BTC-A treeData for StartAnchoring comparison: %#v", btcAgg))
		return errors.New("StartAnchoring failure, AggRoot mismatch")
	}
	anchorBTCAggStateObjects := calendar.PrepareBtcaStateData(btcAgg)
	anchorBTCAggStateObjectsJson, _ := json.Marshal(anchorBTCAggStateObjects)
	app.logger.Info(fmt.Sprintf("anchorBTCAggStateObjects for AggRoot %s via BtcTx %s: %s", btcTxObj.AnchorBtcAggRoot, btcTxObj.BtcTxID, anchorBTCAggStateObjectsJson))
	err = app.Db.BulkInsertBtcAggState(anchorBTCAggStateObjects)
	if app.LogError(err) != nil {
		app.logger.Info(fmt.Sprintf("StartAnchoring TreeData save failure, resetting anchor: %s", btcAgg.AnchorBtcAggRoot))
		return err
	}
	app.logger.Info(fmt.Sprintf("BTC-A StartAnchoring Success for %s", btcTxObj.AnchorBtcAggRoot))
	return nil
}

// ConfirmAnchor : consumes a btc mon message and issues a BTC-Confirm transaction along with completing btc proof generation
func (app *AnchorBTC) ConfirmAnchor(btcMonObj types.BtcMonMsg) error {
	app.logger.Info(fmt.Sprintf("Creating BTC-C for %s", btcMonObj.BtcTxID))
	var hash []byte
	anchoringCoreID, err := app.tendermintRpc.GetAnchoringCore(fmt.Sprintf("BTC-A.BTCTX='%s'", btcMonObj.BtcTxID))
	if len(anchoringCoreID) == 0 {
		app.logger.Error(fmt.Sprintf("StartAnchoring confirmation: Cannot retrieve BTCTX-tagged transaction for btc tx: %s", btcMonObj.BtcTxID))
	} else {
		if app.config.ElectionMode == "test" {
			anchoringCoreID = ""
		}
		app.logger.Info(fmt.Sprintf("Retrieved confirmation query for core %s", anchoringCoreID))
	}

	deadline := time.Now().Add(time.Duration(5) * time.Minute)
	for !time.Now().After(deadline) {
		//only start BTC-C leader election process if someone else hasn't
		if btcMonObj.BtcHeadRoot != string(app.state.LatestBtccTx) {
			// Broadcast the confirmation message with metadata
			amLeader, _ := leaderelection.ElectValidatorAsLeader(1, []string{anchoringCoreID}, *app.state, app.config)
			if amLeader {
				btcc, err := json.Marshal(types.BtcMonMsg{
					BtcTxID:       btcMonObj.BtcTxID,
					BtcHeadHeight: btcMonObj.BtcHeadHeight,
					BtcHeadRoot:   btcMonObj.BtcHeadRoot,
					Path:          nil,
				})
				result, err := app.tendermintRpc.BroadcastTxWithMeta("BTC-C", string(btcc), 3, time.Now().Unix(), app.state.ID, anchoringCoreID, app.config.ECPrivateKey)
				app.LogError(err)
				app.logger.Info(fmt.Sprint("BTC-C confirmation Hash: %v", result.Hash))
			}
		}
		time.Sleep(70 * time.Second) // wait until next block to query for btc-c
		hash = app.tendermintRpc.GetBTCCForBtcRoot(btcMonObj)
		if len(hash) > 0 {
			break
		}
		app.logger.Info(fmt.Sprintf("Restarting confirmation process for %s", btcMonObj.BtcTxID))
	}
	headStateObj := calendar.GenerateHeadStateObject(app.config.CoreURI, hash, btcMonObj)
	proofIds, err := app.Db.GetProofIdsByBtcTxId(btcMonObj.BtcTxID)
	app.logger.Info(fmt.Sprintf("BTC ProofIds: %#v", proofIds))
	app.LogError(err)
	app.logger.Info(fmt.Sprintf("BtcHeadState: %#v", headStateObj))
	app.LogError(app.GenerateBtcBatch(proofIds, headStateObj))
	go app.analytics.SendEvent(app.state.LatestTimeRecord, "CreateConfirmTx", btcMonObj.BtcTxID, time.Now().Format(time.RFC3339), "", "", "")
	return nil
}

func (app *AnchorBTC) ConstructProof(txid string) (proof.P, error) {
	btca, err := app.tendermintRpc.GetBtcaForCalTx(txid)
	if app.LogError(err) != nil {
		return proof.Proof(), err
	}
	btcAgg, err := app.GetTreeFromCalRange(btca.BeginCalTxInt, btca.EndCalTxInt)
	if app.LogError(err) != nil {
		return proof.Proof(), err
	}
	if btcAgg.AnchorBtcAggRoot != btca.AnchorBtcAggRoot {
		app.logger.Info(fmt.Sprintf("ConstructProof TreeData calculation failure for BTC-A aggroot: %s, local treeData result was %s", btca.AnchorBtcAggRoot, btcAgg.AnchorBtcAggRoot))
		return proof.Proof(), errors.New("StartAnchoring failure, AggRoot mismatch")
	}
	anchorBtcTxState := calendar.GenerateAnchorBtcTxState(btca)
	anchorBTCAggStateObjects := calendar.PrepareBtcaStateData(btcAgg)
	/*	err = app.Db.BulkInsertBtcAggState(anchorBTCAggStateObjects)
		if app.LogError(err) != nil {
			app.logger.Info(fmt.Sprintf("ConstructProof TreeData save failure, resetting anchor: %s", btcAgg.AnchorBtcAggRoot))
			return proof.Proof(), err
		}*/
	btccHash, blockHeight := app.tendermintRpc.GetAnchorHeight(btca)
	if blockHeight == 0 {
		//TODO: sad path: contact anchoringCore for tx/height, reconstruct blocktree
		//TODO: or we're querying too soon
		txBytes, err := hex.DecodeString(btca.BtcTxID)
		if app.LogError(err) != nil {
			return proof.Proof(), err
		}
		txDetails, err := app.LnClient.GetTransaction(txBytes)
		if app.LogError(err) != nil {
			return proof.Proof(), err
		}
		if len(txDetails.Transactions) > 0 {
			blockHeight = int64(txDetails.Transactions[0].BlockHeight)
		}
		blockDetails, err := app.LnClient.GetBlockByHeight(blockHeight)
		if app.LogError(err) != nil {
			return proof.Proof(), err
		}
		btccHash = app.tendermintRpc.GetBTCCForBtcRoot(types.BtcMonMsg{
			BtcTxID:       "",
			BtcHeadHeight: 0,
			BtcHeadRoot:   util.ReverseTxHex(hex.EncodeToString(blockDetails.MerkleRoot)),
			Path:          nil,
		})
		if len(btccHash) == 0 {
			return proof.Proof(), errors.New(fmt.Sprintf("Could not retrieve btcc"))
		}
	}
	//TODO: happy path: query for btcc, extract height, reconstruct blocktree
	btcProofMsg, err := app.GenerateBtcHeaderProof(types.TxID{
		TxID:             btca.BtcTxID,
		BlockHeight:      blockHeight,
		AnchorBtcAggRoot: "",
	})
	if err != nil {
		return proof.Proof(), err
	}
	headStateObj := calendar.GenerateHeadStateObject(app.config.CoreURI, btccHash, btcProofMsg)

	anchorBtcAggStateLookup := make(map[string]types.AnchorBtcAggState)
	for _, anchorAggState := range anchorBTCAggStateObjects {
		anchorBtcAggStateLookup[anchorAggState.CalId] = anchorAggState
	}
	btcProof := proof.Proof()
	err = btcProof.AddChainBranch(anchorBtcAggStateLookup[txid], anchorBtcTxState, headStateObj, btcProof.SetProofType(app.config.BitcoinNetwork, "btc"))
	if app.LogError(err) != nil {
		return proof.Proof(), err
	}
	return btcProof, nil
}

func (app *AnchorBTC) GenerateBtcBatch(proofIds []string, btcHeadState types.AnchorBtcHeadState) error {
	app.logger.Info(util.GetCurrentFuncName(1))
	aggStates, err := app.Db.GetAggStateObjectsByProofIds(proofIds)
	if err != nil {
		return err
	}
	aggIds := []string{}
	for _, aggState := range aggStates {
		aggIds = append(aggIds, aggState.AggID)
	}
	calStates, err := app.Db.GetCalStateObjectsByAggIds(aggIds)
	if err != nil {
		return err
	}
	calIds := []string{}
	for _, calState := range calStates {
		calIds = append(calIds, calState.CalId)
	}

	anchorBtcAggStates, err := app.Db.GetAnchorBTCAggStateObjectsByCalIds(calIds)
	if err != nil {
		return err
	}
	if len(anchorBtcAggStates) == 0 {
		return errors.New("no anchorbtcggstate to retrieve")
	}

	btcTxState, err := app.Db.GetBTCTxStateObjectByBtcHeadState(btcHeadState.BtcTxId)
	if err != nil {
		return err
	}
	if len(btcTxState.BtcTxId) == 0 {
		return errors.New(fmt.Sprintf("btcTxState cannot be located for %s", btcHeadState.BtcTxId))
	}

	calLookUp := make(map[string]types.CalStateObject)
	for _, calState := range calStates {
		calLookUp[calState.AggID] = calState
	}

	anchorBtcAggStateLookup := make(map[string]types.AnchorBtcAggState)
	for _, anchorAggState := range anchorBtcAggStates {
		anchorBtcAggStateLookup[anchorAggState.CalId] = anchorAggState
	}
	proofs := []types.ProofState{}
	//associate calendar merkle tree aggregations with corresponding btc merkle tree, then generate final proof
	for _, aggStateRow := range aggStates {
		proof := proof.Proof()
		app.LogError(proof.AddChainpointHeader("https://w3id.org/chainpoint/v5", "Chainpoint", aggStateRow.Hash, aggStateRow.ProofID))

		app.LogError(proof.AddCalendarBranch(aggStateRow, calLookUp[aggStateRow.AggID].CalState, proof.SetProofType(app.config.BitcoinNetwork, "cal")))

		if calVal, exists := calLookUp[aggStateRow.AggID]; exists {
			if _, exists2 := anchorBtcAggStateLookup[calVal.CalId]; !exists2 {
				app.logger.Error("Error: can't find anchorBTCAggState for", "CalId", calVal.CalId)
				continue
			}
		} else {
			app.logger.Error("Error: can't find calState for", "aggStateRow.AggID", aggStateRow.AggID)
			continue
		}
		app.logger.Info(fmt.Sprintf("Assembling proof %s:\n BtcAggState: %+v\n TxState: %+v\n, HeadState: %+v", aggStateRow.ProofID, anchorBtcAggStateLookup[calLookUp[aggStateRow.AggID].CalId], btcTxState, btcHeadState))
		app.LogError(proof.AddChainBranch(anchorBtcAggStateLookup[calLookUp[aggStateRow.AggID].CalId], btcTxState, btcHeadState, proof.SetProofType(app.config.BitcoinNetwork, "btc")))
		proofBytes, err := json.Marshal(proof)
		if app.LogError(err) != nil {
			continue
		}
		proofState := types.ProofState{
			ProofID: proof["proof_id"].(string),
			Proof:   string(proofBytes),
		}
		proofs = append(proofs, proofState)
	}
	app.logger.Info(fmt.Sprintf("btc proofs: %#v", proofs))
	return app.LogError(app.Db.BulkInsertProofs(proofs))
}

func (app *AnchorBTC) LogError(err error) error {
	if err != nil {
		app.logger.Error(fmt.Sprintf("Error in %s: %s", util.GetCurrentFuncName(2), err.Error()))
	}
	return err
}

//FindAndRemoveBtcCheck : remove all checks in case of btc tx failure
func (app *AnchorBTC) FindAndRemoveBtcCheck(aggRoot string) error {
	checkResults, err := app.Cache.GetArray(CHECK_BTC_TX_IDS_KEY)
	if app.LogError(err) != nil {
		return err
	}
	for _, s := range checkResults {
		var anchor types.AnchorRange
		if app.LogError(json.Unmarshal([]byte(s), &anchor)) != nil {
			app.logger.Error("cannot unmarshal json for Failed BTC check")
			continue
		}
		if anchor.AnchorBtcAggRoot != aggRoot {
			continue
		}
		err = app.Cache.Del(CHECK_BTC_TX_IDS_KEY, s)
		if app.LogError(err) != nil {
			return err
		}
	}
	return nil
}

func (app *AnchorBTC) CheckAnchor(btcmsg types.BtcTxMsg) error {
	btcBodyBytes, _ := hex.DecodeString(btcmsg.BtcTxBody)
	var msgTx wire.MsgTx
	msgTx.DeserializeNoWitness(bytes.NewReader(btcBodyBytes))
	b := txscript.NewScriptBuilder()
	b.AddOp(txscript.OP_RETURN)
	rootBytes, _ := hex.DecodeString(btcmsg.AnchorBtcAggRoot)
	b.AddData(rootBytes)
	outputScript, err := b.Script()
	if app.LogError(err) != nil {
		return err
	}
	for _, out := range msgTx.TxOut {
		if bytes.Compare(out.PkScript, outputScript) == 0 && msgTx.TxHash().String() == btcmsg.BtcTxID {
			app.logger.Info(fmt.Sprintf("BTC-A %s confirmed", btcmsg.BtcTxID))
			return nil
		}
		app.logger.Info(fmt.Sprintf("BTC-A Confirmation loop %s != %s", btcmsg.BtcTxID, msgTx.TxHash().String()))
	}
	return errors.New("unable to verify BTC-A")
}

//BlockSyncMonitor : maintains unlock of wallet while abci is running, updates height, runs confirmation loop
func (app *AnchorBTC) BlockSyncMonitor() {
	app.logger.Info("Starting LND Monitor...")
	app.LnClient.Unlocker()
	state, err := app.LnClient.GetInfo()
	if app.LogError(err) == nil {
		app.state.LNState = *state
		app.logger.Info(fmt.Sprintf("LND state retrieved currHeight: %d vs newHeight: %d", app.state.BtcHeight, app.state.LNState.BlockHeight))
		if app.state.BtcHeight != int64(app.state.LNState.BlockHeight) {
			app.logger.Info("New Blocks detected from LND")
			currBlockHeightInt64 := int64(app.state.LNState.BlockHeight)
			isSynced := currBlockHeightInt64-app.state.BtcHeight < 36 // core should have a gap of less than 6 hours
			if currBlockHeightInt64 != 0 && isSynced {
				app.logger.Info("Monitoring Blocks from LND for Txs")
				err = app.MonitorBlocksForConfirmation(app.state.BtcHeight, currBlockHeightInt64)
				if app.LogError(err) != nil {
					return
				}
			}
			app.state.BtcHeight = int64(app.state.LNState.BlockHeight)
			app.logger.Info(fmt.Sprintf("New BTC Block %d", app.state.BtcHeight))
		}
	}
	app.logger.Info("Finished LND Monitor")
}

//FailedAnchorMonitor: ensures transactions reach btc chain within certain time limit
func (app *AnchorBTC) MonitorFailedAnchor() {
	if app.state.LNState.BlockHeight == 0 {
		app.logger.Info("BTC Height record is 0, waiting for update from btc chain...")
		return
	}
	btcHeight := int64(app.state.LNState.BlockHeight)
	checkResults, err := app.Cache.GetArray(CHECK_BTC_TX_IDS_KEY)
	if app.LogError(err) != nil {
		return
	}
	//iterate through pending tx looking for timeouts
	for _, s := range checkResults {
		var anchor types.AnchorRange
		if app.LogError(json.Unmarshal([]byte(s), &anchor)) != nil {
			app.logger.Error("cannot unmarshal json for Failed BTC check")
			continue
		}
		app.logger.Info(fmt.Sprintf("Checking root %s at %d for failure", anchor.AnchorBtcAggRoot, anchor.BtcBlockHeight))
		//A core reported a lack of balance for anchoring
		if anchor.AnchorBtcAggRoot == app.state.LastErrorCoreID {
			app.logger.Info(fmt.Sprintf("BTC-E for aggroot %s from cal range %d to %d", anchor.AnchorBtcAggRoot, anchor.BeginCalTxInt, anchor.EndCalTxInt))
			err := app.Cache.Del(CHECK_BTC_TX_IDS_KEY, s)
			if app.LogError(err) != nil {
				continue
			}
			app.ResetAnchor(anchor.BeginCalTxInt)
			continue
		}
		hasBeen10CalBlocks := app.state.Height-anchor.CalBlockHeight > 10
		hasBeen144BtcBlocks := anchor.BtcBlockHeight != 0 && btcHeight-anchor.BtcBlockHeight >= int64(144)

		if hasBeen10CalBlocks { // if we have no confirmation of mempool inclusion after 10 minutes
			// this usually means there's something seriously wrong with LND
			app.logger.Info("StartAnchoring Timeout while waiting for mempool", "AnchorBtcAggRoot", anchor.AnchorBtcAggRoot)
			// if there are subsequent anchors, we try to re-anchor just that range, else reset for a new anchor period
			if app.state.BeginCalTxInt >= anchor.EndCalTxInt {
				go app.AnchorToChain(anchor.BeginCalTxInt, anchor.EndCalTxInt)
			} else {
				app.ResetAnchor(anchor.BeginCalTxInt)
			}
			app.Cache.Del(CHECK_BTC_TX_IDS_KEY, s)
		}
		if hasBeen144BtcBlocks {
			app.Cache.Del(CHECK_BTC_TX_IDS_KEY, s)
		}
	}
}

func (app *AnchorBTC) IsInConfirmedTxs(anchorRoot string) (bool, types.TxID) {
	results, err := app.Cache.GetArray(CONFIRMED_BTC_TX_IDS_KEY)
	if app.LogError(err) != nil {
		return false, types.TxID{}
	}
	for _, s := range results {
		var tx types.TxID
		if app.LogError(json.Unmarshal([]byte(s), &tx)) != nil {
			continue
		}
		if tx.AnchorBtcAggRoot == anchorRoot {
			return true, tx
		}
	}
	return false, types.TxID{}
}

// MonitorConfirmedTx : Begins anchor confirmation process when a Tx is in the mempool
func (app *AnchorBTC) MonitorConfirmedTx() {
	results, err := app.Cache.GetArray(CONFIRMED_BTC_TX_IDS_KEY)
	if app.LogError(err) != nil {
		return
	}
	for _, s := range results {
		app.logger.Info(fmt.Sprintf("Checking confirmed btc tx %s", s))
		var tx types.TxID
		if app.LogError(json.Unmarshal([]byte(s), &tx)) != nil {
			continue
		}
		if tx.BlockHeight == 0 {
			app.logger.Info(fmt.Sprintf("btc tx %s not yet in block", s))
			continue
		}
		confirmCount := app.state.BtcHeight - tx.BlockHeight + 1
		if confirmCount < 6 {
			app.logger.Info(fmt.Sprintf("btc tx %s at %d confirmations", s, confirmCount))
			continue
		}
		btcmsg, err := app.GenerateBtcHeaderProof(tx)
		if err != nil && strings.Contains(err.Error(), "not found in block") {
			app.LogError(app.Cache.Del(CONFIRMED_BTC_TX_IDS_KEY, s))
		}
		go app.ConfirmAnchor(btcmsg)
		app.logger.Info(fmt.Sprintf("btc tx msg %+v confirmed", btcmsg))
		if app.LogError(app.Cache.Del(CONFIRMED_BTC_TX_IDS_KEY, s)) != nil {
			continue
		}
	}
}

func (app *AnchorBTC) GenerateBtcHeaderProof(tx types.TxID) (types.BtcMonMsg, error) {
	block, tree, txIndex, err := app.GetBlockTree(tx)
	if app.LogError(err) != nil {
		return types.BtcMonMsg{}, err
	}
	var btcmsg types.BtcMonMsg
	btcmsg.BtcTxID = tx.TxID
	btcmsg.BtcHeadHeight = tx.BlockHeight
	btcmsg.BtcHeadRoot = util.ReverseTxHex(hex.EncodeToString(block.MerkleRoot))
	proofs := tree.GetProof(txIndex)
	jsproofs := make([]types.JSProof, len(proofs))
	for i, proof := range proofs {
		if proof.Left {
			jsproofs[i] = types.JSProof{Left: hex.EncodeToString(proof.Value)}
		} else {
			jsproofs[i] = types.JSProof{Right: hex.EncodeToString(proof.Value)}
		}
	}
	btcmsg.Path = jsproofs
	return btcmsg, nil
}

// GetBlockTree : constructs block merkel tree with transaction as index
func (app *AnchorBTC) GetBlockTree(btcTx types.TxID) (lnrpc.BlockDetails, merkletools.MerkleTree, int, error) {
	block, err := app.LnClient.GetBlockByHeight(btcTx.BlockHeight)
	if app.LogError(err) != nil {
		return lnrpc.BlockDetails{}, merkletools.MerkleTree{}, -1, err
	}
	var tree merkletools.MerkleTree
	txIndex := -1
	for i, t := range block.Transactions {
		if t == btcTx.TxID {
			txIndex = i
		}
		tx := util.ReverseTxHex(t)
		hexTx, _ := hex.DecodeString(tx)
		tree.AddLeaf(hexTx)
	}
	if txIndex == -1 {
		return lnrpc.BlockDetails{}, merkletools.MerkleTree{}, -1, errors.New(fmt.Sprintf("Transaction %s not found in block %d", btcTx.TxID, btcTx.BlockHeight))
	}
	tree.MakeBTCTree()
	root := tree.GetMerkleRoot()
	reversedRoot := util.ReverseTxHex(hex.EncodeToString(root))
	reversedRootBytes, _ := hex.DecodeString(reversedRoot)
	reversedMerkleRoot, _ := hex.DecodeString(util.ReverseTxHex(hex.EncodeToString(block.MerkleRoot)))
	if !bytes.Equal(reversedRootBytes, reversedMerkleRoot) {
		return block, merkletools.MerkleTree{}, -1, errors.New(fmt.Sprintf("%s does not equal block merkle root %s", hex.EncodeToString(reversedRootBytes), hex.EncodeToString(block.MerkleRoot)))
	}
	return block, tree, txIndex, nil
}

// MonitorBlocksForConfirmation : since LND can't retrieve confirmed Txs, search block by block
func (app *AnchorBTC) MonitorBlocksForConfirmation(startHeight int64, endHeight int64) error {
	confirmationTxs := make([]types.TxID, 0)
	txsIdStrings := make([]string, 0)
	txsStrings := make([]string, 0)
	results, err := app.Cache.GetArray(CONFIRMED_BTC_TX_IDS_KEY)
	if err != nil {
		return err
	}
	for _, s := range results {
		var tx types.TxID
		if app.LogError(json.Unmarshal([]byte(s), &tx)) != nil {
			continue
		}
		if tx.BlockHeight != 0 {
			continue
		}
		confirmationTxs = append(confirmationTxs, tx)
		txsIdStrings = append(txsIdStrings, tx.TxID)
		txsStrings = append(txsStrings, s)
	}
	for i := startHeight; i < endHeight+1; i++ {
		block, err := app.LnClient.GetBlockByHeight(i)
		if app.LogError(err) != nil {
			return err
		}
		for _, t := range block.Transactions {
			if contains, index := util.ArrayContainsIndex(txsIdStrings, t); contains {
				confirmationTx := txsStrings[index]
				tx := confirmationTxs[index]
				app.Cache.Del(CONFIRMED_BTC_TX_IDS_KEY, confirmationTx)
				tx.BlockHeight = i
				txIDBytes, _ := json.Marshal(tx)
				app.Cache.Append(CONFIRMED_BTC_TX_IDS_KEY, string(txIDBytes))
				app.logger.Info(fmt.Sprintf("Found tx %s in block %d", tx.TxID, i))
			}
		}
	}
	return nil
}

// ResetAnchor ensures that anchoring will begin again in the next block
func (app *AnchorBTC) ResetAnchor(startTxRange int64) {
	app.logger.Info(fmt.Sprintf("StartAnchoring failure, restarting anchor epoch from tx %d", startTxRange))
	app.state.BeginCalTxInt = startTxRange
	app.state.LatestBtcaHeight = -1 //ensure election and anchoring reoccurs next block
}
