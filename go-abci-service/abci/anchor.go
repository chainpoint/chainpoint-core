package abci

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/chainpoint/chainpoint-core/go-abci-service/types"
	"github.com/chainpoint/chainpoint-core/go-abci-service/util"
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
		if util.LogError(err) != nil {
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
	if util.LogError(err) != nil {
		return err
	}

	// Aggregate all txs in range into a new merkle tree in prep for BTC anchoring
	treeData := app.calendar.AggregateAnchorTx(txLeaves)
	app.logger.Info(fmt.Sprintf("treeData for current Anchor: %v", treeData))

	// If we have something to anchor, perform anchoring and proofgen functions
	if treeData.AnchorBtcAggRoot != "" {
		if iAmLeader {
			err := app.calendar.QueueBtcTxStateDataMessage(treeData)
			if util.LogError(err) != nil {
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
			if util.LogError(err) != nil {
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
				if util.LogError(err) != nil {
					app.resetAnchor(startTxRange)
					return err
				}
				result, err := app.rpc.BroadcastTx("BTC-A", string(BtcAData), 2, time.Now().Unix(), app.ID, &app.config.ECPrivateKey)
				app.logger.Debug(fmt.Sprintf("Anchor result: %v", result))
				if util.LogError(err) != nil {
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

// resetAnchor ensures that anchoring will begin again in the next block
func (app *AnchorApplication) resetAnchor(startTxRange int64) {
	app.logger.Debug("Anchoring failed, restarting anchor epoch")
	app.state.BeginCalTxInt = startTxRange
	app.state.LatestBtcaHeight = -1 //ensure election and anchoring reoccurs next block
}
