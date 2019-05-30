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

/*//MintCoreReward : mint rewards for cores
func (app *AnchorApplication) MintCoreReward(sig []string, rewardCandidates []common.Address, rewardHash []byte) error {
	leader, ids := app.ElectLeader(1)
	if len(ids) == 1 {
		app.state.LastMintCoreID = ids[0]
	}
	if leader {
		app.logger.Info("Mint: Elected Leader for Minting")
		if len(sig) > 6 {
			sig = sig[0:6]
		}
		app.logger.Info(fmt.Sprintf("Mint Signatures: %v\nReward Candidates: %v\nReward Hash: %x\n", sig, rewardCandidates, rewardHash))
		sigBytes := make([][]byte, len(sig))
		for i, sigStr := range sig {
			decodedSig, err := hex.DecodeString(sigStr)
			if util.LoggerError(app.logger, err) != nil {
				app.logger.Info("Mint Error: mint hex decoding failed")
				continue
			}
			sigBytes[i] = decodedSig
		}
		err := app.ethClient.MintCores(rewardCandidates, rewardHash, sigBytes)
		if util.LoggerError(app.logger, err) != nil {
			app.logger.Info("Mint Error: invoking smart contract failed")
			return err
		}
		app.logger.Info("Mint process complete")
	}
	return nil
}*/

//PollCoresFromContract : load all past node staking events and update events
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
