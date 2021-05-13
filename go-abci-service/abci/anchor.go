package abci

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/chainpoint/chainpoint-core/go-abci-service/lightning"
	"time"

	"github.com/chainpoint/chainpoint-core/go-abci-service/proof"
	"github.com/chainpoint/chainpoint-core/go-abci-service/types"
	"github.com/chainpoint/chainpoint-core/go-abci-service/util"
)

// Anchor: Anchor calendar and btc blockchains
func (app *AnchorApplication) Anchor() {
	// Run AnchorCalendar and AnchorBTC one after another
	if app.state.ChainSynced && app.config.DoCal {
		go app.AnchorCalendar(app.state.Height)
	}
	if app.config.DoAnchor && (app.state.Height-app.state.LatestBtcaHeight) > int64(app.config.AnchorInterval) {
		if app.state.ChainSynced {
			// prevent current height, non-indexed cal roots from being anchored
			if app.state.LatestCalTxInt-app.state.BeginCalTxInt > app.state.CurrentCalInts {
				go app.AnchorBTC(app.state.BeginCalTxInt, app.state.LatestCalTxInt-app.state.CurrentCalInts)
			}
		} else {
			app.state.EndCalTxInt = app.state.LatestCalTxInt
		}
	}
	app.state.CurrentCalInts = 0
}

// AnchorCalendar : Aggregate submitted hashes into a calendar transaction
func (app *AnchorApplication) AnchorCalendar(height int64) (int, error) {
	app.logger.Debug("starting scheduled aggregation")

	// Get agg objects
	aggs := app.aggregator.AggregateAndReset()
	aggStates := make([]types.AggState, 0)
	for _, agg := range aggs {
		aggStates = append(aggStates, agg.AggStates...)
		app.LogError(app.PgClient.BulkInsertAggState(agg.AggStates))
	}
	app.logger.Debug(fmt.Sprintf("Aggregated %d roots: ", len(aggs)))
	app.logger.Debug(fmt.Sprintf("Aggregation Tree: %#v", aggs))

	// Pass the agg objects to generate a calendar tree
	calAgg := app.calendar.GenerateCalendarTree(aggs)
	if calAgg.CalRoot != "" {
		app.logger.Info(fmt.Sprintf("Calendar Root: %s", calAgg.CalRoot))
		app.logger.Debug(fmt.Sprintf("Calendar Tree: %#v", calAgg))
		result, err := app.rpc.BroadcastTx("CAL", calAgg.CalRoot, 2, time.Now().Unix(), app.ID, &app.config.ECPrivateKey)
		if app.LogError(err) != nil {
			return 0, err
		}
		deadline := height + 2
		for app.state.Height < deadline {
			time.Sleep(10 * time.Second)
		}
		app.logger.Debug(fmt.Sprintf("CAL result: %v", result))
		if result.Code == 0 {
			var tx types.TxTm
			tx.Hash = result.Hash.Bytes()
			tx.Data = result.Data.Bytes()
			calStates := app.calendar.CreateCalStateMessage(tx, calAgg)
			app.logger.Info(fmt.Sprintf("Cal States: %#v", len(calStates)))
			app.logger.Info("Generating Cal Batch")
			app.LogError(app.PgClient.BulkInsertCalState(calStates))
			app.LogError(app.GenerateCalBatch(aggStates, calStates))
			app.logger.Info("Generating Cal Batch Complete")
			return len(aggs), nil
		}
	}
	return 0, errors.New("No hashes to aggregate")
}

func (app *AnchorApplication) GenerateCalBatch(aggStates []types.AggState, calStates []types.CalStateObject) error {
	app.logger.Info(util.GetCurrentFuncName(1))
	calLookUp := make(map[string]string)
	for _, calState := range calStates {
		calLookUp[calState.AggID] = calState.CalState
	}
	proofs := []types.ProofState{}
	for _, aggStateRow := range aggStates {
		proof := proof.Proof()
		app.LogError(proof.AddChainpointHeader(aggStateRow.Hash, aggStateRow.ProofID))
		app.LogError(proof.AddCalendarBranch(aggStateRow, calLookUp[aggStateRow.AggID], app.config.BitcoinNetwork))
		proofBytes, err := json.Marshal(proof)
		app.logger.Info(fmt.Sprintf("Proof: %s", string(proofBytes)))
		if app.LogError(err) != nil {
			continue
		}
		proofState := types.ProofState{
			ProofID: proof["proof_id"].(string),
			Proof:   string(proofBytes),
		}
		proofs = append(proofs, proofState)
	}
	return app.LogError(app.PgClient.BulkInsertProofs(proofs))
}

func (app *AnchorApplication) GetTreeFromCalRange(startTxRange int64, endTxRange int64) (types.BtcAgg, error) {
	// Get CAL transactions between the latest BTCA tx and the current latest tx
	txLeaves, err := app.getCalTxRange(startTxRange, endTxRange)
	app.logger.Info(fmt.Sprintf("Retrieved %d CAL leaves from ranges %d to %d", len(txLeaves), startTxRange, endTxRange))
	if app.LogError(err) != nil {
		return types.BtcAgg{}, err
	}
	// Aggregate all txs in range into a new merkle tree in prep for BTC anchoring
	treeData := app.calendar.AggregateAnchorTx(txLeaves)
	return treeData, nil
}

// AnchorBTC : Anchor scans all CAL transactions since last anchor epoch and writes the merkle root to the Calendar and to bitcoin
func (app *AnchorApplication) AnchorBTC(startTxRange int64, endTxRange int64) error {
	// elect leader to do the actual anchoring
	if app.config.ElectionMode == "test" {
		app.state.LastErrorCoreID = ""
	}
	iAmLeader, leaderIDs := app.ElectChainContributorAsLeader(1, []string{app.state.LastErrorCoreID})
	if len(leaderIDs) == 0 {
		return errors.New("Leader election error")
	}
	app.logger.Info(fmt.Sprintf("Anchor Leaders: %v", leaderIDs))

	treeData, err := app.GetTreeFromCalRange(startTxRange, endTxRange)
	if err != nil {
		return err
	}
	app.logger.Info(fmt.Sprintf("Anchoring tx ranges %d to %d at Height %d, latestBtcaHeight %d, for aggroot: %s", startTxRange, endTxRange, app.state.Height, app.state.LatestBtcaHeight, treeData.AnchorBtcAggRoot))
	app.logger.Info(fmt.Sprintf("treeData for Anchor: %#v", treeData))

	// If we have something to anchor, perform anchoring and proofgen functions
	if treeData.AnchorBtcAggRoot != "" {
		app.state.LastElectedCoreID = leaderIDs[0]

		if treeData.AnchorBtcAggRoot == app.state.LatestErrRoot {
			app.state.LatestErrRoot = ""
		}
		// elect anchorer
		if iAmLeader {
			btca, err := app.SendBtcTx(treeData, app.state.Height, startTxRange, endTxRange)
			if app.LogError(err) != nil {
				_, err := app.rpc.BroadcastTx("BTC-E", treeData.AnchorBtcAggRoot, 2, time.Now().Unix(), app.ID, &app.config.ECPrivateKey)
				if app.LogError(err) != nil {
					panic(err)
				}
			}
			_, err = app.rpc.BroadcastTx("BTC-A", string(btca), 2, time.Now().Unix(), app.ID, &app.config.ECPrivateKey)
			if app.LogError(err) != nil {
				app.logger.Info(fmt.Sprintf("failed sending BTC-A"))
				panic(err)
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
		redisResult := app.RedisClient.WithContext(context.Background()).SAdd(CHECK_BTC_TX_IDS_KEY, string(failedAnchorJSON))
		if app.LogError(redisResult.Err()) != nil {
			return redisResult.Err()
		}
		app.state.BeginCalTxInt = endTxRange
		app.state.EndCalTxInt = endTxRange            // Ensure we update our range of CAL txs for next anchor period
		app.state.LatestBtcaHeight = app.state.Height // So no one will try to re-anchor while processing the btc tx
		return nil
	}
	return errors.New("no transactions to aggregate")
}

// SendBtcTx : sends btc tx to lnd and enqueues tx monitoring information
func (app *AnchorApplication) SendBtcTx(anchorDataObj types.BtcAgg, height int64, start int64, end int64) ([]byte, error) {
	hexRoot, err := hex.DecodeString(anchorDataObj.AnchorBtcAggRoot)
	if util.LogError(err) != nil {
		return []byte{}, err
	}
	txid, rawtx, err := app.LnClient.SendOpReturn(hexRoot)
	if util.LogError(err) != nil {
		return []byte{}, err
	}
	msgBtcMon := types.BtcTxMsg{
		AnchorBtcAggID:   anchorDataObj.AnchorBtcAggID,
		AnchorBtcAggRoot: anchorDataObj.AnchorBtcAggRoot,
		BtcTxID:          txid,
		BtcTxBody:        rawtx,
		BtcTxHeight:	  0,
		CalBlockHeight:   height,
		BeginCalTxInt:    start,
		EndCalTxInt:      end,
	}
	btcJSON, err := json.Marshal(msgBtcMon)
	app.logger.Info(fmt.Sprint("Sending BTC-A OP_RETURN: %#v", msgBtcMon))
	return btcJSON, err
}

// AnchorReward : Send sats to last anchoring core
func (app *AnchorApplication) AnchorReward(CoreID string) error {
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

// ConsumeBtcTxMsg : Consumes a btctx RMQ message to initiate monitoring on all nodes
func (app *AnchorApplication) ConsumeBtcTxMsg(msgBytes []byte) error {
	var btcTxObj types.BtcTxMsg
	if err := json.Unmarshal(msgBytes, &btcTxObj); err != nil {
		return app.LogError(err)
	}
	app.state.LatestBtcTx = btcTxObj.BtcTxID // Update app state with txID so we can broadcast BTC-A
	app.state.LatestBtcAggRoot = btcTxObj.AnchorBtcAggRoot
	stateObj := app.calendar.GenerateAnchorBtcTxState(btcTxObj)
	app.logger.Info(fmt.Sprintf("BTC-A BtcTx State Obj: %#v", stateObj))
	err := app.PgClient.BulkInsertBtcTxState([]types.AnchorBtcTxState{stateObj})
	if app.LogError(err) != nil {
		return err
	}

	txIDBytes, err := json.Marshal(types.TxID{TxID: btcTxObj.BtcTxID, AnchorBtcAggRoot: btcTxObj.AnchorBtcAggRoot})
	result := app.RedisClient.WithContext(context.Background()).SAdd(CONFIRMED_BTC_TX_IDS_KEY, string(txIDBytes))
	if app.LogError(result.Err()) != nil {
		return err
	}
	// end monitoring for failed anchor
	app.FindAndRemoveBtcCheck(btcTxObj.AnchorBtcAggRoot)

	btcAgg, err := app.GetTreeFromCalRange(btcTxObj.BeginCalTxInt, btcTxObj.EndCalTxInt)
	if app.LogError(err) != nil {
		return err
	}
	if btcAgg.AnchorBtcAggRoot != btcTxObj.AnchorBtcAggRoot {
		app.logger.Info(fmt.Sprintf("BTC-A Anchor TreeData calculation failure for BTC-A aggroot: %s, local treeData result was %s", btcTxObj.AnchorBtcAggRoot, btcAgg.AnchorBtcAggRoot))
		app.logger.Info(fmt.Sprintf("BTC-A treeData for Anchor comparison: %#v", btcAgg))
		return errors.New("Anchor failure, AggRoot mismatch")
	}
	anchorBTCAggStateObjects := app.calendar.PrepareBtcaStateData(btcAgg)
	err = app.PgClient.BulkInsertBtcAggState(anchorBTCAggStateObjects)
	if app.LogError(err) != nil {
		app.logger.Info(fmt.Sprintf("Anchor TreeData save failure, resetting anchor: %s", btcAgg.AnchorBtcAggRoot))
		return err
	}
	app.logger.Info(fmt.Sprintf("BTC-A Anchor Success for %s", btcTxObj.AnchorBtcAggRoot))
	if app.LogError(result.Err()) != nil {
		return err
	}
	return nil
}

// ConsumeBtcMonMsg : consumes a btc mon message and issues a BTC-Confirm transaction along with completing btc proof generation
func (app *AnchorApplication) ConsumeBtcMonMsg(btcMonObj types.BtcMonMsg) error {
	app.logger.Info(fmt.Sprintf("Consuming BTC-C for %s", btcMonObj.BtcTxID))
	var hash []byte
	anchoringCoreID, err := app.getAnchoringCore(fmt.Sprintf("BTC-A.BTCTX='%s'", btcMonObj.BtcTxID))
	if len(anchoringCoreID) == 0 {
		app.logger.Error(fmt.Sprintf("Anchor confirmation: Cannot retrieve BTCTX-tagged transaction for btc tx: %s", btcMonObj.BtcTxID))
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
			amLeader, _ := app.ElectValidatorAsLeader(1, []string{anchoringCoreID})
			if amLeader {
				result, err := app.rpc.BroadcastTxWithMeta("BTC-C", btcMonObj.BtcHeadRoot, 2, time.Now().Unix(), app.ID, anchoringCoreID+"|"+btcMonObj.BtcTxID, &app.config.ECPrivateKey)
				app.LogError(err)
				app.logger.Info(fmt.Sprint("BTC-C confirmation Hash: %v", result.Hash))
			}
		}
		time.Sleep(70 * time.Second) // wait until next block to query for btc-c
		hash = app.GetBTCCTx(btcMonObj)
		if len(hash) > 0 {
			break
		}
		app.logger.Info(fmt.Sprintf("Restarting confirmation process for %s", btcMonObj.BtcTxID))
	}
	headStateObj := app.calendar.GenerateHeadStateObject(hash, btcMonObj)
	proofIds, err := app.PgClient.GetProofIdsByBtcTxId(btcMonObj.BtcTxID)
	app.logger.Info(fmt.Sprintf("BTC ProofIds: %#v", proofIds))
	app.LogError(err)
	app.logger.Info(fmt.Sprintf("BtcHeadState: %#v", headStateObj))
	app.LogError(app.PgClient.BulkInsertBtcHeadState([]types.AnchorBtcHeadState{headStateObj}))
	app.LogError(app.GenerateBtcBatch(proofIds, headStateObj))
	return nil
}

func (app *AnchorApplication) GenerateBtcBatch(proofIds []string, btcHeadState types.AnchorBtcHeadState) error {
	app.logger.Info(util.GetCurrentFuncName(1))
	aggStates, err := app.PgClient.GetAggStateObjectsByProofIds(proofIds)
	if err != nil {
		return err
	}
	aggIds := []string{}
	for _, aggState := range aggStates {
		aggIds = append(aggIds, aggState.AggID)
	}
	calStates, err := app.PgClient.GetCalStateObjectsByAggIds(aggIds)
	if err != nil {
		return err
	}
	calIds := []string{}
	for _, calState := range calStates {
		calIds = append(calIds, calState.CalId)
	}

	anchorBtcAggStates, err := app.PgClient.GetAnchorBTCAggStateObjectsByCalIds(calIds)
	if err != nil {
		return err
	}
	if len(anchorBtcAggStates) == 0 {
		return errors.New("no anchorbtcggstate to retrieve")
	}
	anchorBTCAggIds := []string{}
	for _, anchorBtcAggState := range anchorBtcAggStates {
		anchorBTCAggIds = append(anchorBTCAggIds, anchorBtcAggState.AnchorBtcAggId)
	}
	btcTxState, err := app.PgClient.GetBTCTxStateObjectByAnchorBTCAggId(anchorBTCAggIds[0])
	if err != nil {
		return err
	}
	if len(btcTxState.BtcTxId) == 0 {
		return errors.New(fmt.Sprintf("btcTxState cannot be located for %s", anchorBTCAggIds[0]))
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
		app.LogError(proof.AddChainpointHeader(aggStateRow.Hash, aggStateRow.ProofID))
		app.LogError(proof.AddCalendarBranch(aggStateRow, calLookUp[aggStateRow.AggID].CalState, app.config.BitcoinNetwork))

		if calVal, exists := calLookUp[aggStateRow.AggID]; exists {
			if _, exists2 := anchorBtcAggStateLookup[calVal.CalId]; !exists2 {
				app.logger.Error("Error: can't find anchorBTCAggState for", "CalId", calVal.CalId)
				continue
			}
		} else {
			app.logger.Error("Error: can't find calState for", "aggStateRow.AggID", aggStateRow.AggID)
			continue
		}
		app.LogError(proof.AddBtcBranch(anchorBtcAggStateLookup[calLookUp[aggStateRow.AggID].CalId], btcTxState, btcHeadState, app.config.BitcoinNetwork))
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
	return app.LogError(app.PgClient.BulkInsertProofs(proofs))
}

// resetAnchor ensures that anchoring will begin again in the next block
func (app *AnchorApplication) resetAnchor(startTxRange int64) {
	app.logger.Info(fmt.Sprintf("Anchor failure, restarting anchor epoch from tx %d", startTxRange))
	app.state.BeginCalTxInt = startTxRange
	app.state.LatestBtcaHeight = -1 //ensure election and anchoring reoccurs next block
}
