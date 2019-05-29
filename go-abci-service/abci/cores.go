package abci

import (
	"database/sql"
	"encoding/hex"
	"fmt"
	"math/big"
	"time"

	"github.com/chainpoint/chainpoint-core/go-abci-service/types"
	"github.com/chainpoint/chainpoint-core/go-abci-service/util"
)

//PollNodesFromContract : load all past node staking events and update events
func (app *AnchorApplication) PollCoresFromContract() {
	highestBlock := big.NewInt(0)
	first := true
	for {
		app.logger.Info(fmt.Sprintf("Polling for Registry events after block %d", highestBlock.Int64()))
		if first {
			first = false
		} else {
			time.Sleep(30 * time.Second)
		}

		//Consume all past node events from this contract and import them into the local postgres instance
		coresStaked, err := app.ethClient.GetPastCoresStakedEvents(*highestBlock)
		if util.LoggerError(app.logger, err) != nil {
			app.logger.Info("error in finding past staked nodes")
			continue
		}
		for _, core := range coresStaked {
			newCore := types.Core{
				EthAddr:     core.Sender.Hex(),
				PublicIP:    sql.NullString{String: util.Int2Ip(core.CoreIp).String(), Valid: true},
				CoreId:      sql.NullString{String: hex.EncodeToString(core.CoreId)},
				BlockNumber: sql.NullInt64{Int64: int64(core.Raw.BlockNumber), Valid: true},
			}
			inserted, err := app.pgClient.CoreUpsert(newCore)
			if util.LoggerError(app.logger, err) != nil {
				continue
			}
			app.logger.Info(fmt.Sprintf("Inserted for %#v: %t", newCore, inserted))
		}

		//Consume all updated events and reconcile them with the previous states
		coresStakedUpdated, err := app.ethClient.GetPastCoresStakeUpdatedEvents(*highestBlock)
		if util.LoggerError(app.logger, err) != nil {
			continue
		}
		for _, core := range coresStakedUpdated {
			newCore := types.Core{
				EthAddr:     core.Sender.Hex(),
				PublicIP:    sql.NullString{String: util.Int2Ip(core.CoreIp).String(), Valid: true},
				CoreId:      sql.NullString{String: hex.EncodeToString(core.CoreId)},
				BlockNumber: sql.NullInt64{Int64: int64(core.Raw.BlockNumber), Valid: true},
			}
			inserted, err := app.pgClient.CoreUpsert(newCore)
			if util.LoggerError(app.logger, err) != nil {
				continue
			}
			app.logger.Info(fmt.Sprintf("Updated for %#v: %t", newCore, inserted))
		}

		//Consume unstake events and delete nodes where the blockNumber of this event is higher than the last stake or update
		coresUnstaked, err := app.ethClient.GetPastCoresUnstakeEvents(*highestBlock)
		if util.LoggerError(app.logger, err) != nil {
			continue
		}
		for _, core := range coresUnstaked {
			newCore := types.Core{
				EthAddr:     core.Sender.Hex(),
				PublicIP:    sql.NullString{String: util.Int2Ip(core.CoreIp).String(), Valid: true},
				CoreId:      sql.NullString{String: hex.EncodeToString(core.CoreId)},
				BlockNumber: sql.NullInt64{Int64: int64(core.Raw.BlockNumber), Valid: true},
			}
			deleted, err := app.pgClient.CoreDelete(newCore)
			if util.LoggerError(app.logger, err) != nil {
				continue
			}
			app.logger.Info(fmt.Sprintf("Deleted for %#v: %t", newCore, deleted))
		}

		highestBlock, err = app.ethClient.HighestBlock()
		if util.LoggerError(app.logger, err) != nil {
			continue
		}
	}
}
