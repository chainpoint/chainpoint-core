package abci

import (
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"math/big"
	"sort"
	"time"

	"github.com/chainpoint/chainpoint-core/go-abci-service/ethcontracts"

	"github.com/ethereum/go-ethereum/common"

	"github.com/chainpoint/chainpoint-core/go-abci-service/types"
	"github.com/chainpoint/chainpoint-core/go-abci-service/util"
)

//MintCoreReward : mint rewards for cores
func (app *AnchorApplication) MintCoreReward(sig []string, rewardCandidates []common.Address, rewardHash []byte) error {
	app.logger.Info("CoreMint: Elected Leader for Minting")
	app.logger.Info(fmt.Sprintf("CoreMint: %v\nReward Candidates: %v\nReward Hash: %x\n", sig, rewardCandidates, rewardHash))
	sigBytes := make([][]byte, 126)
	for i, _ := range sigBytes {
		sigBytes[i] = make([]byte, 0)
	}
	for i, sigStr := range sig {
		var decodedSig []byte
		decodedSig, err := hex.DecodeString(sigStr)
		if util.LoggerError(app.logger, err) != nil {
			app.logger.Info("CoreMint: mint hex decoding failed")
			continue
		}
		sigBytes[i] = decodedSig
	}
	var sigFixedBytes [126][]byte
	copy(sigFixedBytes[:], sigBytes[:126])
	app.logger.Info("CoreMint: Sig Bytes: %v", sigFixedBytes)
	err := app.ethClient.MintCores(rewardCandidates, rewardHash, sigFixedBytes)
	if util.LoggerError(app.logger, err) != nil {
		app.logger.Info("CoreMint: invoking smart contract failed")
		return err
	}
	app.logger.Info("CoreMint process complete")

	return nil
}

//StartCoreMintProcess : wraps signing/minting process and handles state updates
func (app *AnchorApplication) StartCoreMintProcess() error {
	app.SetCoreMintPendingState(true) //needed since we can't do a blocking lock in commit
	err := app.SignCoreRewards()
	app.SetCoreMintPendingState(false)
	if util.LoggerError(app.logger, err) != nil {
		return err
	}
	return nil
}

//SetCoreMintPendingState : create a deferable method to set mint state
func (app *AnchorApplication) SetCoreMintPendingState(val bool) {
	app.state.CoreMintPending = val
	app.CoreRewardSignatures = make([]string, 0)
}

//CollectRewardNodes : collate and sign reward node list
func (app *AnchorApplication) SignCoreRewards() error {
	var candidates []common.Address
	var rewardHash []byte
	currentEthBlock, err := app.ethClient.HighestBlock()
	if util.LoggerError(app.logger, err) != nil {
		app.logger.Error("CoreMint Error: problem retrieving highest block for core minting")
		return err
	}
	if currentEthBlock.Int64()-app.state.LastCoreMintedAtBlock < MINT_EPOCH {
		app.logger.Info("CoreMint: Too soon for core minting")
		return errors.New("CoreMint: Too soon for minting")
	}
	candidates, rewardHash, err = app.GetCoreRewardCandidates()
	if util.LoggerError(app.logger, err) != nil {
		app.logger.Info("CoreMint Error: Error retrieving core reward candidates")
		return err
	}
	app.logger.Info(fmt.Sprintf("CoreMint: raw SHA3 hash: %x", rewardHash))
	rewardHash = signHash(rewardHash)
	app.logger.Info(fmt.Sprintf("CoreMint: with prefix: %x", rewardHash))
	signature, err := ethcontracts.SignMsg(rewardHash, app.ethClient.EthPrivateKey)
	signature[64] += 27
	if util.LoggerError(app.logger, err) != nil {
		app.logger.Info("CoreMint Error: Problem with signing message for minting")
		return err
	}
	_, err = app.rpc.BroadcastTx("CORE-SIGN", hex.EncodeToString(signature), 2, time.Now().Unix(), app.ID, &app.config.ECPrivateKey)
	if err != nil {
		app.logger.Info("CoreMint Error: Error issuing SIGN tx")
		return err
	}
	if leader, ids := app.ElectLeader(1); leader {
		peers := app.GetPeers()
		thresholdLenPeers := int(math.Ceil(float64(len(peers)) * 0.66))

		// wait for 6 SIGN tx
		deadline := time.Now().Add(4 * time.Minute)
		for len(app.CoreRewardSignatures) < thresholdLenPeers && !time.Now().After(deadline) {
			time.Sleep(10 * time.Second)
		}
		// Mint if 2/3+ SIGN txs are received
		if len(app.CoreRewardSignatures) >= thresholdLenPeers {
			app.logger.Info("CoreMint: Enough SIGN TXs received, calling mint")
			err := app.MintCoreReward(app.CoreRewardSignatures, candidates, rewardHash)
			if len(ids) == 1 {
				app.state.LastMintCoreID = ids[0]
			}
			if util.LoggerError(app.logger, err) != nil {
				return err
			}
		} else {
			app.logger.Info("CoreMint: Not enough SIGN TXs")
			return errors.New("CoreMint: Not enough SIGN TXs")
		}
	}
	return nil
}

//GetNodeRewardCandidates : scans for and collates the reward candidates in the current epoch
func (app *AnchorApplication) GetCoreRewardCandidates() ([]common.Address, []byte, error) {
	txResult, err := app.rpc.client.TxSearch(fmt.Sprintf("CORERC=%d", app.state.LastCoreMintedAtBlock), false, 1, 25)
	app.logger.Info(fmt.Sprintf("CoreMint for CORERC txResults: %#v", txResult))
	if util.LoggerError(app.logger, err) != nil {
		return []common.Address{}, []byte{}, err
	}
	coreArray := make([]common.Address, 0)
	for _, tx := range txResult.Txs {
		decoded, err := util.DecodeTx(tx.Tx)
		if util.LoggerError(app.logger, err) != nil {
			continue
		}
		core, err := app.pgClient.GetCoreByID(decoded.CoreID)
		app.logger.Info(fmt.Sprintf("CoreMint for core %#v", core))
		if util.LoggerError(app.logger, err) != nil {
			continue
		}
		coreArray = append(coreArray, common.HexToAddress(core.EthAddr))
	}
	if len(coreArray) == 0 {
		return []common.Address{}, []byte{}, errors.New("CoreMint: No CORE-RC tx from the last epoch have been found")
	}
	addresses := util.UniquifyAddresses(coreArray)
	sort.Slice(addresses[:], func(i, j int) bool {
		return addresses[i].Hex() > addresses[j].Hex()
	})
	app.logger.Info(fmt.Sprintf("CoreMint: input core addresses: %#v", addresses))
	rewardHash := ethcontracts.AddressesToHash(addresses)
	return addresses, rewardHash, nil
}

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
				CoreId:      sql.NullString{String: hex.EncodeToString(core.CoreId), Valid: true},
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
				CoreId:      sql.NullString{String: hex.EncodeToString(core.CoreId), Valid: true},
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
				CoreId:      sql.NullString{String: hex.EncodeToString(core.CoreId), Valid: true},
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
