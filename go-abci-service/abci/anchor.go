package abci

import (
	"errors"
	"fmt"
	"time"

	"github.com/chainpoint/chainpoint-core/go-abci-service/types"
	"github.com/chainpoint/chainpoint-core/go-abci-service/util"
)

// AggregateCalendar : Aggregate submitted hashes into a calendar transaction
func (app *AnchorApplication) AggregateCalendar() error {
	app.logger.Debug("starting scheduled aggregation")
	rpc := GetHTTPClient(app.config.TendermintRPC)
	defer rpc.Stop()

	// Get agg objects
	aggs := app.aggregator.Aggregate(app.state.LatestNistRecord)

	// Pass the agg objects to generate a calendar tree
	calAgg := app.calendar.GenerateCalendarTree(aggs)
	if calAgg.CalRoot != "" {
		app.logger.Debug(fmt.Sprintf("Calendar Root: %s", calAgg.CalRoot))

		result, err := app.BroadcastTx("CAL", calAgg.CalRoot, 2, time.Now().Unix())
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
	app.logger.Debug(fmt.Sprintf("starting scheduled anchor for tx ranges %d to %d", startTxRange, endTxRange))

	iAmLeader, leaderID := app.ElectLeader()
	if leaderID == "" {
		return errors.New("Leader election error")
	}

	app.logger.Debug(fmt.Sprintf("Leader: %s", leaderID))
	/* Get CAL transactions between the latest BTCA tx and the current latest tx */
	txLeaves, err := app.getTxRange(startTxRange, endTxRange)
	if util.LogError(err) != nil {
		return err
	}
	// Aggregate all txs in rage into a new merkle tree in prep for BTC anchoring
	treeData := app.calendar.AggregateAnchorTx(txLeaves)
	app.logger.Debug(fmt.Sprintf("treeData for current anchor: %v", treeData))
	if treeData.AnchorBtcAggRoot != "" {
		if iAmLeader {
			result, err := app.BroadcastTx("BTC-A", treeData.AnchorBtcAggRoot, 2, time.Now().Unix())
			if util.LogError(err) != nil {
				return err
			}
			app.logger.Debug(fmt.Sprintf("Anchor result: %v", result))
		}
		app.state.EndCalTxInt = endTxRange
		err := app.calendar.QueueBtcaStateDataMessage(iAmLeader, treeData)
		if util.LogError(err) != nil {
			return err
		}
		time.Sleep(60 * time.Second) // wait for a BTC-M tx
		// A BTC-M tx should have hit by now.
		if app.state.LatestBtcmTxInt < startTxRange { //If not, it'll be less than the start of the current range.
			app.logger.Debug("Anchoring failed, restarting anchor epoch")
			app.state.BeginCalTxInt = startTxRange
			app.state.LatestBtcaHeight = -1 //ensure election and anchoring reoccurs next block
		}
		return nil
	}
	return errors.New("no transactions to aggregate")
}
