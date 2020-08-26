package abci

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/chainpoint/chainpoint-core/go-abci-service/lightning"
	"strconv"
	"strings"
	"time"

	"github.com/chainpoint/chainpoint-core/go-abci-service/rabbitmq"
	"github.com/chainpoint/chainpoint-core/go-abci-service/types"
	"github.com/chainpoint/chainpoint-core/go-abci-service/util"
)

// AnchorCalendar : Aggregate submitted hashes into a calendar transaction
func (app *AnchorApplication) AnchorCalendar(height int64) error {
	app.logger.Debug("starting scheduled aggregation")

	// Get agg objects
	aggs := app.aggregator.AggregateAndReset()
	app.logger.Debug(fmt.Sprintf("Aggregated %d roots: ", len(aggs)))
	app.logger.Debug(fmt.Sprintf("Aggregation Tree: %#v", aggs))

	// Pass the agg objects to generate a calendar tree
	calAgg := app.calendar.GenerateCalendarTree(aggs)
	if calAgg.CalRoot != "" {
		app.logger.Info(fmt.Sprintf("Calendar Root: %s", calAgg.CalRoot))
		app.logger.Debug(fmt.Sprintf("Calendar Tree: %#v", calAgg))
		result, err := app.rpc.BroadcastTx("CAL", calAgg.CalRoot, 2, time.Now().Unix(), app.ID, &app.config.ECPrivateKey)
		if app.LogError(err) != nil {
			return err
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
			app.calendar.QueueCalStateMessage(tx, calAgg)
			return nil
		}
	}
	return errors.New("No hashes to aggregate")
}

// AnchorBTC : Anchor scans all CAL transactions since last anchor epoch and writes the merkle root to the Calendar and to bitcoin
func (app *AnchorApplication) AnchorBTC(startTxRange int64, endTxRange int64) error {
	// elect leader to do the actual anchoring
	iAmLeader, leaderIDs := app.ElectChainContributorAsLeader(1, []string{app.state.LastErrorCoreID})
	if len(leaderIDs) == 0 {
		return errors.New("Leader election error")
	}
	app.logger.Info(fmt.Sprintf("Anchor Leaders: %v", leaderIDs))

	// Get CAL transactions between the latest BTCA tx and the current latest tx
	txLeaves, err := app.getCalTxRange(startTxRange, endTxRange)
	if app.LogError(err) != nil {
		return err
	}

	// Aggregate all txs in range into a new merkle tree in prep for BTC anchoring
	treeData := app.calendar.AggregateAnchorTx(txLeaves)
	app.logger.Info(fmt.Sprintf("treeData for Anchor for tx ranges %d to %d for aggroot: %s: %v", startTxRange, endTxRange, treeData.AnchorBtcAggRoot, treeData))

	// If we have something to anchor, perform anchoring and proofgen functions
	if treeData.AnchorBtcAggRoot != "" {
		app.state.LastElectedCoreID = leaderIDs[0]

		if treeData.AnchorBtcAggRoot == app.state.LatestErrRoot {
			app.state.LatestErrRoot = ""
		}
		// elect anchorer
		if iAmLeader {
			err := app.SendBtcTx(treeData, app.state.Height, startTxRange, endTxRange)
			if app.LogError(err) != nil {
				_, err := app.rpc.BroadcastTx("BTC-E", treeData.AnchorBtcAggRoot, 2, time.Now().Unix(), app.ID, &app.config.ECPrivateKey)
				if app.LogError(err) != nil {
					panic(err)
				}
				return errors.New("no balance")
			}
		}
		// begin monitoring for anchor
		failedAnchorCheck := types.AnchorRange{
			AnchorBtcAggRoot: treeData.AnchorBtcAggRoot,
			CalBlockHeight:   app.state.Height,
			BeginCalTxInt:    startTxRange,
			EndCalTxInt:      endTxRange,
		}
		failedAnchorJSON, _ := json.Marshal(failedAnchorCheck)
		redisResult := app.redisClient.WithContext(context.Background()).SAdd(CHECK_BTC_TX_IDS_KEY, string(failedAnchorJSON))
		if app.LogError(redisResult.Err()) != nil {
			return redisResult.Err()
		}
		treeDataJSON, err := json.Marshal(treeData)
		if app.LogError(err) != nil {
			app.logger.Info(fmt.Sprintf("Anchor TreeData marshal failure for aggroot: %s", treeData.AnchorBtcAggRoot))
			return err
		}
		setResult := app.redisClient.WithContext(context.Background()).Set(treeData.AnchorBtcAggRoot, string(treeDataJSON), (6 * time.Hour))
		if app.LogError(setResult.Err()) != nil {
			app.logger.Info("Anchor TreeData save failure")
			return setResult.Err()
		} else {
			app.logger.Info(fmt.Sprintf("Saved Anchor TreeData under root %s", treeData.AnchorBtcAggRoot))
		}
		app.state.EndCalTxInt = endTxRange            // Ensure we update our range of CAL txs for next anchor period
		app.state.LatestBtcaHeight = app.state.Height // So no one will try to re-anchor while processing the btc tx
		return nil
	}
	return errors.New("no transactions to aggregate")
}

// SendBtcTx : sends btc tx to lnd and enqueues tx monitoring information
func (app *AnchorApplication) SendBtcTx(anchorDataObj types.BtcAgg, height int64, start int64, end int64) error {
	hexRoot, err := hex.DecodeString(anchorDataObj.AnchorBtcAggRoot)
	if util.LogError(err) != nil {
		return err
	}
	txid, rawtx, err := app.lnClient.SendOpReturn(hexRoot)
	if util.LogError(err) != nil {
		return err
	}
	msgBtcMon := types.BtcTxMsg{
		AnchorBtcAggID:   anchorDataObj.AnchorBtcAggID,
		AnchorBtcAggRoot: anchorDataObj.AnchorBtcAggRoot,
		BtcTxBody:        rawtx,
		BtcTxID:          txid,
		CalBlockHeight:   height,
		BeginCalTxInt:    start,
		EndCalTxInt:      end,
	}
	btcJSON, err := json.Marshal(msgBtcMon)
	app.logger.Info(fmt.Sprint("Sending BTC-A OP_RETURN: %#v", msgBtcMon))
	if util.LogError(err) != nil {
		return err
	}
	result := app.redisClient.WithContext(context.Background()).SAdd(NEW_BTC_TX_IDS_KEY, string(btcJSON))
	if util.LogError(result.Err()) != nil {
		return result.Err()
	}
	app.logger.Info("Added BTC-A message to redis")
	return nil
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
		resp, err := app.lnClient.SendCoins(status.LightningAddress, int64(app.config.AnchorReward), int32(app.lnClient.MinConfs))
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
	stateObj := types.BtcTxProofState{
		AnchorBtcAggID: btcTxObj.AnchorBtcAggID,
		BtcTxID:        btcTxObj.BtcTxID,
		BtcTxState: types.BtcTxOpsState{
			Ops: []types.ProofLineItem{
				{
					Left: btcTxObj.BtcTxBody[:strings.Index(btcTxObj.BtcTxBody, btcTxObj.AnchorBtcAggRoot)],
				},
				{
					Right: btcTxObj.BtcTxBody[strings.Index(btcTxObj.BtcTxBody, btcTxObj.AnchorBtcAggRoot)+len(btcTxObj.AnchorBtcAggRoot):],
				},
				{
					Op: "sha-256-x2",
				},
			},
		},
	}
	app.logger.Info(fmt.Sprintf("BtcTx State Obj: %#v", stateObj))
	dataJSON, err := json.Marshal(stateObj)
	if app.LogError(err) != nil {
		return err
	}
	err = rabbitmq.Publish(app.config.RabbitmqURI, "work.proofstate", "btctx", dataJSON)
	if err != nil {
		rabbitmq.LogError(err, "rmq dial failure, is rmq connected?")
		return err
	}
	txIDBytes, err := json.Marshal(types.TxID{TxID: btcTxObj.BtcTxID, BlockHeight: btcTxObj.BtcTxHeight})
	result := app.redisClient.WithContext(context.Background()).SAdd(CONFIRMED_BTC_TX_IDS_KEY, string(txIDBytes))

	// end monitoring for failed anchor
	failedAnchorCheck := types.AnchorRange{
		AnchorBtcAggRoot: btcTxObj.AnchorBtcAggRoot,
		CalBlockHeight:   btcTxObj.CalBlockHeight,
		BeginCalTxInt:    btcTxObj.BeginCalTxInt,
		EndCalTxInt:      btcTxObj.EndCalTxInt,
	}
	failedAnchorJSON, _ := json.Marshal(failedAnchorCheck)
	redisResult := app.redisClient.WithContext(context.Background()).SRem(CHECK_BTC_TX_IDS_KEY, string(failedAnchorJSON))
	if app.LogError(redisResult.Err()) != nil {
		return redisResult.Err()
	}

	// Create agg state messages
	getResult := app.redisClient.WithContext(context.Background()).Get(btcTxObj.AnchorBtcAggRoot)
	var btcAgg types.BtcAgg
	if err := json.Unmarshal([]byte(getResult.Val()), &btcAgg); err != nil {
		//app.resetAnchor(failedAnchorCheck.BeginCalTxInt)
		app.LogError(getResult.Err())
		app.LogError(err)
		app.logger.Info(fmt.Sprintf("Anchor TreeData retrieval failure for aggroot: %s, result was %s", btcTxObj.AnchorBtcAggRoot, getResult.Val()))
		return err
	}
	err = app.calendar.QueueBtcaStateDataMessage(btcAgg)
	if app.LogError(err) != nil {
		app.logger.Info(fmt.Sprintf("Anchor TreeData queue failure, resetting anchor: %s", btcAgg.AnchorBtcAggRoot))
		//app.resetAnchor(failedAnchorCheck.BeginCalTxInt)
		return err
	}
	delResult := app.redisClient.WithContext(context.Background()).Del(btcTxObj.AnchorBtcAggRoot)
	if app.LogError(delResult.Err()) != nil {
		return err
	}
	app.logger.Info("Anchor Success")
	if app.LogError(result.Err()) != nil {
		return err
	}
	return nil
}

// ConsumeBtcMonMsg : consumes a btc mon message and issues a BTC-Confirm transaction along with completing btc proof generation
func (app *AnchorApplication) ConsumeBtcMonMsg(btcMonObj types.BtcMonMsg) error {
	var anchoringCoreID string
	var hash []byte
	// Get the CoreID that originally published the anchor TX using the btc tx ID we tagged it with
	queryLine := fmt.Sprintf("BTC-A.BTCTX='%s'", btcMonObj.BtcTxID)
	app.logger.Info("Anchor confirmation query: " + queryLine)
	txResult, err := app.rpc.client.TxSearch(queryLine, false, 1, 25, "")
	if app.LogError(err) == nil {
		for _, tx := range txResult.Txs {
			decoded, err := util.DecodeTx(tx.Tx)
			if app.LogError(err) != nil {
				continue
			}
			anchoringCoreID = decoded.CoreID
		}
	}
	if len(anchoringCoreID) == 0 {
		app.logger.Error(fmt.Sprintf("Anchor confirmation: Cannot retrieve BTCTX-tagged transaction for btc tx: %s", btcMonObj.BtcTxID))
	} else {
		app.logger.Info(fmt.Sprintf("Retrieved confirmation query for core %s", anchoringCoreID))
	}

	deadline := time.Now().Add(time.Duration(5) * time.Minute)
	for !time.Now().After(deadline) {
		if btcMonObj.BtcHeadRoot == string(app.state.LatestBtccTx) {
			return errors.New(fmt.Sprintf("Already seen BTC-C confirmation for root %s", btcMonObj.BtcHeadRoot))
		}
		// Broadcast the confirmation message with metadata
		amLeader, _ := app.ElectValidatorAsLeader(1, []string{anchoringCoreID})
		if amLeader {
			result, err := app.rpc.BroadcastTxWithMeta("BTC-C", btcMonObj.BtcHeadRoot, 2, time.Now().Unix(), app.ID, anchoringCoreID+"|"+btcMonObj.BtcTxID, &app.config.ECPrivateKey)
			app.LogError(err)
			app.logger.Info(fmt.Sprint("BTC-C confirmation Hash: %v", result.Hash))
		}
		time.Sleep(70 * time.Second) // wait until next block to query for btc-c
		btccQueryLine := fmt.Sprintf("BTC-C.BTCC='%s'", btcMonObj.BtcHeadRoot)
		txResult, err := app.rpc.client.TxSearch(btccQueryLine, false, 1, 25, "")
		if app.LogError(err) == nil {
			for _, tx := range txResult.Txs {
				hash = tx.Hash
				app.logger.Info(fmt.Sprint("Found BTC-C Hash from confirmation leader: %v", hash))
			}
		}
		if len(hash) > 0 {
			break
		}
		app.logger.Info("Restarting confirmation process")
	}

	var btccStateObj types.BtccStateObj
	btccStateObj.BtcTxID = btcMonObj.BtcTxID
	btccStateObj.BtcHeadHeight = btcMonObj.BtcHeadHeight
	btccStateObj.BtcHeadState.Ops = make([]types.ProofLineItem, 0)
	for _, p := range btcMonObj.Path {
		if p.Left != "" {
			btccStateObj.BtcHeadState.Ops = append(btccStateObj.BtcHeadState.Ops, types.ProofLineItem{Left: string(p.Left)})
		}
		if p.Right != "" {
			btccStateObj.BtcHeadState.Ops = append(btccStateObj.BtcHeadState.Ops, types.ProofLineItem{Right: string(p.Right)})
		}
		btccStateObj.BtcHeadState.Ops = append(btccStateObj.BtcHeadState.Ops, types.ProofLineItem{Op: "sha-256-x2"})
	}
	baseURI := util.GetEnv("CHAINPOINT_CORE_BASE_URI", "https://tendermint.chainpoint.org")
	uri := strings.ToLower(fmt.Sprintf("%s/calendar/%x/data", baseURI, hash))
	btccStateObj.BtcHeadState.Anchor = types.AnchorObj{
		AnchorID: strconv.FormatInt(btcMonObj.BtcHeadHeight, 10),
		Uris:     []string{uri},
	}
	stateObjBytes, err := json.Marshal(btccStateObj)
	app.logger.Info("Completed AnchorStateObj: %s", string(stateObjBytes))
	err = rabbitmq.Publish(app.config.RabbitmqURI, "work.proofstate", "btcmon", stateObjBytes)
	if err != nil {
		rabbitmq.LogError(err, "rmq dial failure, is rmq connected?")
		return err
	}
	return nil
}

// resetAnchor ensures that anchoring will begin again in the next block
func (app *AnchorApplication) resetAnchor(startTxRange int64) {
	app.logger.Debug(fmt.Sprintf("Anchor failed, restarting anchor epoch from tx %d", startTxRange))
	app.state.BeginCalTxInt = startTxRange
	app.state.LatestBtcaHeight = -1 //ensure election and anchoring reoccurs next block
}
