package abci

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/chainpoint/chainpoint-core/go-abci-service/calendar"
	"time"

	"github.com/chainpoint/chainpoint-core/go-abci-service/proof"
	"github.com/chainpoint/chainpoint-core/go-abci-service/types"
	"github.com/chainpoint/chainpoint-core/go-abci-service/util"
)

// StartAnchoring: StartAnchoring calendar and btc blockchains
func (app *AnchorApplication) StartAnchoring() {
	// Run AnchorCalendar and AnchorToChain one after another
	if app.state.ChainSynced && app.config.DoCal {
		go app.AnchorCalendar(app.state.Height)
	}
	if app.config.DoAnchor && (app.state.Height-app.state.LatestBtcaHeight) > int64(app.config.AnchorInterval) {
		if app.state.ChainSynced {
			// prevent current height, non-indexed cal roots from being anchored
			if app.state.LatestCalTxInt-app.state.BeginCalTxInt > app.state.CurrentCalInts {
				go app.Anchor.AnchorToChain(app.state.BeginCalTxInt, app.state.LatestCalTxInt-app.state.CurrentCalInts)
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
		app.LogError(app.Cache.BulkInsertAggState(agg.AggStates))
	}
	app.logger.Debug(fmt.Sprintf("Aggregated %d roots: ", len(aggs)))
	app.logger.Debug(fmt.Sprintf("Aggregation Tree: %#v", aggs))

	// Pass the agg objects to generate a calendar tree
	calAgg := calendar.GenerateCalendarTree(aggs)
	if calAgg.CalRoot != "" {
		app.logger.Info(fmt.Sprintf("Calendar Root: %s", calAgg.CalRoot))
		app.logger.Debug(fmt.Sprintf("Calendar Tree: %#v", calAgg))
		result, err := app.rpc.BroadcastTx("CAL", calAgg.CalRoot, 2, time.Now().Unix(), app.ID, &app.config.ECPrivateKey)
		if app.LogError(err) != nil {
			return 0, err
		}
		go app.Analytics.SendEvent(app.state.LatestTimeRecord, "CreateCalTx", calAgg.CalRoot, time.Now().Format(time.RFC3339), "", "", "")
		app.logger.Debug(fmt.Sprintf("CAL result: %+v", result))
		if result.Code == 0 {
			var tx types.TxTm
			tx.Hash = result.Hash.Bytes()
			tx.Data = result.Data.Bytes()
			calStates := calendar.CreateCalStateMessage(tx, calAgg)
			app.logger.Info(fmt.Sprintf("Cal States: %#v", len(calStates)))
			app.logger.Info("Generating Cal Batch")
			app.LogError(app.Cache.BulkInsertCalState(calStates))
			app.LogError(app.GenerateCalBatch(aggStates, calStates))
			hashRoot := hex.EncodeToString(tx.Hash)
			app.Cache.Add(hashRoot, calAgg.CalRoot)
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
	return app.LogError(app.Cache.BulkInsertProofs(proofs))
}

