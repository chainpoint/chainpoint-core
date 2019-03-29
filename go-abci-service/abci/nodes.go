package abci

import (
	"database/sql"
	"fmt"
	"math/big"

	"github.com/chainpoint/chainpoint-core/go-abci-service/types"
	"github.com/chainpoint/chainpoint-core/go-abci-service/util"
)

//LoadNodesFromContract : load all past node staking events and update events
func (app *AnchorApplication) LoadNodesFromContract() error {
	//Consume all past node events from this contract and import them into the local postgres instance
	nodesStaked, err := app.ethClient.GetPastNodesStakedEvents()
	if util.LoggerError(app.logger, err) != nil {
		return err
	}
	for _, node := range nodesStaked {
		newNode := types.Node{
			EthAddr:         node.Sender.Hex(),
			PublicIP:        sql.NullString{String: util.BytesToIP(node.NodeIp[:]), Valid: true},
			AmountStaked:    sql.NullInt64{Int64: node.AmountStaked.Int64(), Valid: true},
			StakeExpiration: sql.NullInt64{Int64: node.Duration.Int64(), Valid: true},
			BlockNumber:     sql.NullInt64{Int64: int64(node.Raw.BlockNumber), Valid: true},
		}
		inserted, err := app.pgClient.NodeUpsert(newNode)
		if util.LoggerError(app.logger, err) != nil {
			return err
		}
		app.logger.Info(fmt.Sprintf("Inserted for %#v: %t\n", newNode, inserted))
	}

	//Consume all updated events and reconcile them with the previous states
	nodesStakedUpdated, err := app.ethClient.GetPastNodesStakeUpdatedEvents()
	if util.LoggerError(app.logger, err) != nil {
		return err
	}
	for _, node := range nodesStakedUpdated {
		newNode := types.Node{
			EthAddr:         node.Sender.Hex(),
			PublicIP:        sql.NullString{String: util.BytesToIP(node.NodeIp[:]), Valid: true},
			AmountStaked:    sql.NullInt64{Int64: node.AmountStaked.Int64(), Valid: true},
			StakeExpiration: sql.NullInt64{Int64: node.Duration.Int64(), Valid: true},
			BlockNumber:     sql.NullInt64{Int64: int64(node.Raw.BlockNumber), Valid: true},
		}
		inserted, err := app.pgClient.NodeUpsert(newNode)
		if util.LoggerError(app.logger, err) != nil {
			return err
		}
		fmt.Printf("Inserted Update for %#v: %t\n", newNode, inserted)
	}
	return nil
}

//WatchNodesFromContract : get all future node staking events and updates
func (app *AnchorApplication) WatchNodesFromContract() error {
	highestBlock, err := app.ethClient.HighestBlock()
	if util.LoggerError(app.logger, err) != nil {
		highestBlock = big.NewInt(0)
	}
	go app.ethClient.WatchNodeStakeEvents(app.pgClient.HandleNodeStaking, *highestBlock)
	go app.ethClient.WatchNodeStakeUpdatedEvents(app.pgClient.HandleNodeStakeUpdating, *highestBlock)
	return nil
}
