package abci

import (
	"errors"
	"fmt"
	"time"

	"github.com/chainpoint/chainpoint-core/go-abci-service/aggregator"
	"github.com/chainpoint/chainpoint-core/go-abci-service/calendar"
	"github.com/chainpoint/chainpoint-core/go-abci-service/types"
	"github.com/chainpoint/chainpoint-core/go-abci-service/util"
)

/* Aggregate submitted hashes into a calendar transaction */
func AggregateCalendar(tendermintRPC types.TendermintURI, rabbitmqUri string) error {
	fmt.Println("starting scheduled aggregation")
	rpc := GetHTTPClient(tendermintRPC)
	defer rpc.Stop()
	aggs := aggregator.Aggregate(rabbitmqUri)
	// Because there is a 1 : 1 calendar/aggregation interval, there is only one  root here
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
			calendar.QueueCalStateMessage(rabbitmqUri, tx, calAgg)
			return nil
		}
	}
	return errors.New("No hashes to aggregate")
}

// Anchor scans all CAL transactions since last anchor epoch and writes the merkle root to the Calendar and to bitcoin
func AnchorBTC(tendermintURI types.TendermintURI, rabbitmqUri string, startTxRange *int64, endTxRange int64) error {
	fmt.Println("starting scheduled anchor")
	iAmLeader, _ := ElectLeader(tendermintURI)
	fmt.Printf("Leader: %t", iAmLeader)
	/* Get CAL transactions between the latest BTCA tx and the current latest tx */
	txLeaves, err := getTxRange(tendermintURI, *startTxRange, endTxRange)
	if util.LogError(err) != nil {
		return err
	}
	treeData := calendar.AggregateAnchorTx(txLeaves)
	fmt.Printf("treeData for current anchor: %v\n", treeData)
	if treeData.AggRoot != "" {
		if iAmLeader {
			result, err := BroadcastTx(tendermintURI, "BTC-A", treeData.AggRoot, 2, time.Now().Unix())
			if util.LogError(err) != nil {
				return err
			} else {
				*startTxRange = endTxRange
			}
			fmt.Printf("Anchor result: %v\n", result)
		}
		calendar.QueueBtcaStateDataMessage(rabbitmqUri, iAmLeader, treeData)
		return nil
	}
	return errors.New("no transactions to aggregate")
}
