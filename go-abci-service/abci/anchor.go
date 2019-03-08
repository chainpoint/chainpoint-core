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

		result, err := BroadcastTx(app.config.TendermintRPC, "CAL", calAgg.CalRoot, 2, time.Now().Unix())
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

	iAmLeader, leaderID := ElectLeader(app.config.TendermintRPC)
	if leaderID == "" {
		return errors.New("Leader election error")
	}

	app.logger.Debug(fmt.Sprintf("Leader: %s", leaderID))
	/* Get CAL transactions between the latest BTCA tx and the current latest tx */
	txLeaves, err := app.getTxRange(startTxRange, endTxRange)
	if util.LogError(err) != nil {
		return err
	}
	treeData := app.calendar.AggregateAnchorTx(txLeaves)
	app.logger.Debug(fmt.Sprintf("treeData for current anchor: %v", treeData))
	if treeData.AnchorBtcAggRoot != "" {
		if iAmLeader {
			result, err := BroadcastTx(app.config.TendermintRPC, "BTC-A", treeData.AnchorBtcAggRoot, 2, time.Now().Unix())
			if util.LogError(err) != nil {
				return err
			}
			app.logger.Debug(fmt.Sprintf("Anchor result: %v", result))
		}
		app.state.EndCalTxInt = endTxRange
		app.calendar.QueueBtcaStateDataMessage(iAmLeader, treeData)
		return nil
	}
	return errors.New("no transactions to aggregate")
}
