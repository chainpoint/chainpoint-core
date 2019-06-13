package abci

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/chainpoint/chainpoint-core/go-abci-service/rabbitmq"
	"github.com/chainpoint/chainpoint-core/go-abci-service/util"
	"github.com/streadway/amqp"

	"github.com/chainpoint/chainpoint-core/go-abci-service/types"
)

// AggregateCalendar : Aggregate submitted hashes into a calendar transaction
func (app *AnchorApplication) AggregateCalendar() error {
	app.logger.Debug("starting scheduled aggregation")

	// Get agg objects
	aggs := app.aggregator.Aggregate(app.state.LatestNistRecord)

	// Pass the agg objects to generate a calendar tree
	calAgg := app.calendar.GenerateCalendarTree(aggs)
	if calAgg.CalRoot != "" {
		app.logger.Info(fmt.Sprintf("Calendar Root: %s", calAgg.CalRoot))

		result, err := app.rpc.BroadcastTx("CAL", calAgg.CalRoot, 2, time.Now().Unix(), app.ID, &app.config.ECPrivateKey)
		if app.LogError(err) != nil {
			return err
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
	app.logger.Debug(fmt.Sprintf("starting scheduled anchor period for tx ranges %d to %d", startTxRange, endTxRange))

	// elect leader to do the actual anchoring
	iAmLeader, leaderIDs := app.ElectLeader(1)
	if len(leaderIDs) == 0 {
		return errors.New("Leader election error")
	}
	app.logger.Debug(fmt.Sprintf("Leaders: %v", leaderIDs))

	// Get CAL transactions between the latest BTCA tx and the current latest tx
	txLeaves, err := app.getCalTxRange(startTxRange, endTxRange)
	if app.LogError(err) != nil {
		return err
	}

	// Aggregate all txs in range into a new merkle tree in prep for BTC anchoring
	treeData := app.calendar.AggregateAnchorTx(txLeaves)
	app.logger.Info(fmt.Sprintf("treeData for current Anchor: %v", treeData))

	// If we have something to anchor, perform anchoring and proofgen functions
	if treeData.AnchorBtcAggRoot != "" {
		if iAmLeader {
			err := app.calendar.QueueBtcTxStateDataMessage(treeData)
			if app.LogError(err) != nil {
				app.resetAnchor(startTxRange)
				return err
			}
		}
		app.state.LatestBtcaHeight = app.state.Height //So no one will try to re-anchor while processing the btc tx

		// wait for a BTC-M tx
		deadline := time.Now().Add(2 * time.Minute)
		for app.state.LatestBtcmTxInt < startTxRange && !time.Now().After(deadline) {
			time.Sleep(10 * time.Second)
		}

		// A BTC-M tx should have hit by now
		if app.state.LatestBtcmTxInt < startTxRange { //If not, it'll be less than the start of the current range.
			app.resetAnchor(startTxRange)
		} else {
			err = app.calendar.QueueBtcaStateDataMessage(treeData)
			if app.LogError(err) != nil {
				app.resetAnchor(startTxRange)
				return err
			}
			app.state.EndCalTxInt = endTxRange
			if iAmLeader {
				BtcA := types.BtcA{
					AnchorBtcAggRoot: treeData.AnchorBtcAggRoot,
					BtcTxID:          app.state.LatestBtcTx,
				}
				BtcAData, err := json.Marshal(BtcA)
				if app.LogError(err) != nil {
					app.resetAnchor(startTxRange)
					return err
				}
				result, err := app.rpc.BroadcastTx("BTC-A", string(BtcAData), 2, time.Now().Unix(), app.ID, &app.config.ECPrivateKey)
				app.logger.Debug(fmt.Sprintf("Anchor result: %v", result))
				if app.LogError(err) != nil {
					if strings.Contains(err.Error(), "-32603") {
						app.logger.Debug(fmt.Sprintf("BTC-A block already committed; Leader is %v", leaderIDs))
						return err
					}
					app.resetAnchor(startTxRange)
					return err
				}
			}
		}
		return nil
	}
	return errors.New("no transactions to aggregate")
}

// ConsumeBtcTxMsg : Consumes a btctx RMQ message to initiate monitoring on all nodes
func (app *AnchorApplication) ConsumeBtcTxMsg(msgBytes []byte) error {
	var btcTxObj types.BtcTxMsg
	if err := json.Unmarshal(msgBytes, &btcTxObj); err != nil {
		return app.LogError(err)
	}
	app.state.LatestBtcTx = btcTxObj.BtcTxID // Update app state with txID so we can broadcast BTC-A
	stateObj := types.BtcTxProofState{
		AnchorBtcAggID: btcTxObj.AnchorBtcAggID,
		BtcTxID:        btcTxObj.BtcTxID,
		BtcTxState: types.BtcTxOpsState{
			Ops: []types.ProofLineItem{
				types.ProofLineItem{
					Left: btcTxObj.BtcTxBody[:strings.Index(btcTxObj.BtcTxBody, btcTxObj.AnchorBtcAggRoot)],
				},
				types.ProofLineItem{
					Right: btcTxObj.BtcTxBody[strings.Index(btcTxObj.BtcTxBody, btcTxObj.AnchorBtcAggRoot)+len(btcTxObj.AnchorBtcAggRoot):],
				},
				types.ProofLineItem{
					Op: "sha-256-x2",
				},
			},
		},
	}
	dataJSON, err := json.Marshal(stateObj)
	if app.LogError(err) != nil {
		return err
	}
	err = rabbitmq.Publish(app.config.RabbitmqURI, "work.proofstate", "btctx", dataJSON)
	if err != nil {
		rabbitmq.LogError(err, "rmq dial failure, is rmq connected?")
		return err
	}
	txIDBytes, err := json.Marshal(types.TxID{TxID: btcTxObj.BtcTxID})
	err = rabbitmq.Publish(app.config.RabbitmqURI, "work.btcmon", "", txIDBytes)
	if err != nil {
		rabbitmq.LogError(err, "rmq dial failure, is rmq connected?")
		return err
	}
	return nil
}

// ConsumeBtcMonMsg : consumes a btc mon message and issues a BTC-Confirm transaction along with completing btc proof generation
func (app *AnchorApplication) ConsumeBtcMonMsg(msg amqp.Delivery) error {
	var anchoringCoreID string
	var hash []byte
	var btcMonObj types.BtcMonMsg
	app.LogError(json.Unmarshal(msg.Body, &btcMonObj))
	// Get the CoreID that originally published the anchor TX using the btc tx ID we tagged it with
	queryLine := fmt.Sprintf("BTCTX='%s'", btcMonObj.BtcTxID)
	app.logger.Info("Anchor confirmation query: " + queryLine)
	txResult, err := app.rpc.client.TxSearch(queryLine, false, 1, 25)
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
		app.logger.Error(fmt.Sprintf("Anchor: Cannot retrieve BTCTX-tagged transaction for btc tx: %s", btcMonObj.BtcTxID))
	}
	// Broadcast the confirmation message with metadata
	result, err := app.rpc.BroadcastTxWithMeta("BTC-C", btcMonObj.BtcHeadRoot, 2, time.Now().Unix(), app.ID, anchoringCoreID+"|"+btcMonObj.BtcTxID, &app.config.ECPrivateKey)
	time.Sleep(1 * time.Minute) // wait until it hits the mempool
	if app.LogError(err) != nil {
		app.logger.Error(fmt.Sprintf("Anchor: Another core has probably already committed a BTCC tx: %s", err.Error()))
		txResult, err := app.rpc.GetTxByInt(app.state.LatestBtccTxInt)
		if app.LogError(err) != nil && len(txResult.Txs) > 0 {
			hash = txResult.Txs[0].Hash
		} else {
			return err
		}
	} else {
		hash = result.Hash
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

	err = rabbitmq.Publish(app.config.RabbitmqURI, "work.proofstate", "btcmon", stateObjBytes)
	if err != nil {
		rabbitmq.LogError(err, "rmq dial failure, is rmq connected?")
		return err
	}
	msg.Ack(false)
	return nil
}

func (app *AnchorApplication) processMessage(msg amqp.Delivery) error {
	switch msg.Type {
	case "btctx":
		time.Sleep(30 * time.Second)
		_, err := app.rpc.BroadcastTx("BTC-M", string(msg.Body), 2, time.Now().Unix(), app.ID, &app.config.ECPrivateKey)
		if app.LogError(err) != nil {
			return err
		}
		msg.Ack(false)
		break
	case "btcmon":
		err := app.ConsumeBtcMonMsg(msg)
		app.LogError(err)
		break
	case "reward":
		break
	default:
		msg.Ack(false)
	}
	return nil
}

// ReceiveCalRMQ : Continually consume the calendar work queue and
// process any resulting messages from the tx and monitor services
func (app *AnchorApplication) ReceiveCalRMQ() error {
	var session rabbitmq.Session
	var err error
	endConsume := false
	for {
		session, err = rabbitmq.ConnectAndConsume(app.config.RabbitmqURI, "work.cal")
		if err != nil {
			rabbitmq.LogError(err, "failed to dial for work.cal queue")
			time.Sleep(5 * time.Second)
			continue
		}
		for {
			select {
			case err = <-session.Notify:
				if endConsume {
					return err
				}
				time.Sleep(5 * time.Second)
				break //reconnect
			case msg := <-session.Msgs:
				if len(msg.Body) > 0 {
					go app.processMessage(msg)
				}
			}
		}
	}
}

// resetAnchor ensures that anchoring will begin again in the next block
func (app *AnchorApplication) resetAnchor(startTxRange int64) {
	app.logger.Debug("Anchoring failed, restarting anchor epoch")
	app.state.BeginCalTxInt = startTxRange
	app.state.LatestBtcaHeight = -1 //ensure election and anchoring reoccurs next block
}
