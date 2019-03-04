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
func AggregateCalendar(tendermintRPC types.TendermintURI, rabbitmqURI string, nist *beacon.Record) error {
	fmt.Println("starting scheduled aggregation")
	rpc := GetHTTPClient(tendermintRPC)
	defer rpc.Stop()

	// Get agg objects
	aggs := aggregator.Aggregate(rabbitmqURI, *nist)

	// Pass the agg objects to generate a calendar tree
	calAgg := calendar.GenerateCalendarTree(aggs)
	if calAgg.CalRoot != "" {
		fmt.Printf("Root: %s\n", calAgg.CalRoot)

		result, err := BroadcastTx(tendermintRPC, "CAL", calAgg.CalRoot, 2, time.Now().Unix())
		if util.LogError(err) != nil {
			return err
		}
		fmt.Printf("CAL result: %v\n", result)
		if result.Code == 0 {
			var tx types.TxTm
			tx.Hash = result.Hash.Bytes()
			tx.Data = result.Data.Bytes()
			calendar.QueueCalStateMessage(rabbitmqURI, tx, calAgg)
			return nil
		}
	}
	return errors.New("No hashes to aggregate")
}

// AnchorBTC : Anchor scans all CAL transactions since last anchor epoch and writes the merkle root to the Calendar and to bitcoin
func AnchorBTC(tendermintURI types.TendermintURI, rabbitmqURI string, startTxRange *int64, endTxRange int64) error {
	fmt.Println("starting scheduled anchor")
	iAmLeader, leaderID := ElectLeader(tendermintURI)
	fmt.Printf("Leader: %s\n", leaderID)
	/* Get CAL transactions between the latest BTCA tx and the current latest tx */
	txLeaves, err := getTxRange(tendermintURI, *startTxRange, endTxRange)
	if util.LogError(err) != nil {
		return err
	}
	treeData := calendar.AggregateAnchorTx(txLeaves)
	fmt.Printf("treeData for current anchor: %v\n", treeData)
	if treeData.AnchorBtcAggRoot != "" {
		if iAmLeader {
			result, err := BroadcastTx(tendermintURI, "BTC-A", treeData.AnchorBtcAggRoot, 2, time.Now().Unix())
			if util.LogError(err) != nil {
				return err
			}
			*startTxRange = endTxRange
			fmt.Printf("Anchor result: %v\n", result)
		}
		calendar.QueueBtcaStateDataMessage(rabbitmqURI, iAmLeader, treeData)
		return nil
	}
	return errors.New("no transactions to aggregate")
}
