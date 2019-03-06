package abci

import (
	"errors"
	"fmt"
	"time"

	"github.com/chainpoint/chainpoint-core/go-abci-service/aggregator"
	"github.com/chainpoint/chainpoint-core/go-abci-service/calendar"
	"github.com/chainpoint/chainpoint-core/go-abci-service/types"
	"github.com/chainpoint/chainpoint-core/go-abci-service/util"
	beacon "github.com/chainpoint/go-nist-beacon"
)

// AggregateCalendar : Aggregate submitted hashes into a calendar transaction
func (app *AnchorApplication) AggregateCalendar(nist *beacon.Record) error {
	fmt.Println("starting scheduled aggregation")
	rpc := GetHTTPClient(app.tendermintURI)
	defer rpc.Stop()

	// Get agg objects
	aggs := aggregator.Aggregate(app.rabbitmqURI, *nist)

	// Pass the agg objects to generate a calendar tree
	calAgg := calendar.GenerateCalendarTree(aggs)
	if calAgg.CalRoot != "" {
		fmt.Printf("Root: %s\n", calAgg.CalRoot)

		result, err := BroadcastTx(app.tendermintURI, "CAL", calAgg.CalRoot, 2, time.Now().Unix())
		if util.LogError(err) != nil {
			return err
		}
		fmt.Printf("CAL result: %v\n", result)
		if result.Code == 0 {
			var tx types.TxTm
			tx.Hash = result.Hash.Bytes()
			tx.Data = result.Data.Bytes()
			calendar.QueueCalStateMessage(app.rabbitmqURI, tx, calAgg)
			return nil
		}
	}
	return errors.New("No hashes to aggregate")
}

// AnchorBTC : Anchor scans all CAL transactions since last anchor epoch and writes the merkle root to the Calendar and to bitcoin
func (app *AnchorApplication) AnchorBTC(startTxRange int64, endTxRange int64) error {
	fmt.Println("starting scheduled anchor")

	rpc := GetHTTPClient(app.tendermintURI)
	defer rpc.Stop()
	status, err := rpc.Status()

	iAmLeader, leaderID := ElectLeader(app.tendermintURI)
	if err != nil && leaderID == "" && status.SyncInfo.CatchingUp {
		app.state.EndCalTxInt = endTxRange
		return errors.New("No leader- not caught up")
	}

	fmt.Printf("Leader: %s\n", leaderID)
	/* Get CAL transactions between the latest BTCA tx and the current latest tx */
	txLeaves, err := getTxRange(app.tendermintURI, startTxRange, endTxRange)
	if util.LogError(err) != nil {
		return err
	}
	treeData := calendar.AggregateAnchorTx(txLeaves)
	fmt.Printf("treeData for current anchor: %v\n", treeData)
	if treeData.AnchorBtcAggRoot != "" {
		if iAmLeader {
			result, err := BroadcastTx(app.tendermintURI, "BTC-A", treeData.AnchorBtcAggRoot, 2, time.Now().Unix())
			if util.LogError(err) != nil {
				return err
			}
			fmt.Printf("Anchor result: %v\n", result)
		}
		app.state.EndCalTxInt = endTxRange
		calendar.QueueBtcaStateDataMessage(app.rabbitmqURI, iAmLeader, treeData)
		return nil
	}
	return errors.New("no transactions to aggregate")
}
