package abci

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	fee2 "github.com/chainpoint/chainpoint-core/go-abci-service/fee"
	"github.com/chainpoint/chainpoint-core/go-abci-service/merkletools"
	"github.com/lightningnetwork/lnd/lnrpc"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/chainpoint/chainpoint-core/go-abci-service/validation"

	"github.com/chainpoint/chainpoint-core/go-abci-service/util"

	beacon "github.com/chainpoint/chainpoint-core/go-abci-service/beacon"

	"github.com/chainpoint/chainpoint-core/go-abci-service/types"
)

const CONFIRMED_BTC_TX_IDS_KEY = "BTC_Mon:ConfirmedBTCTxIds"
const NEW_BTC_TX_IDS_KEY = "BTC_Mon:NewBTCTxIds"
const CHECK_BTC_TX_IDS_KEY = "BTC_Mon:CheckNewBTCTxIds"
const STATIC_FEE_AMT = 40000 // 12500 // 60k amounts to 240 sat/vbyte

//SyncMonitor : turns off anchoring if we're not synced. Not cron scheduled since we need it to start immediately.
func (app *AnchorApplication) SyncMonitor() {
	for {
		time.Sleep(30 * time.Second) // allow chain time to initialize
		//app.logger.Info("Syncing Chain status and validators")
		var err error
		app.state.TMState, err = app.rpc.GetStatus()
		if app.LogError(err) != nil {
			time.Sleep(5 * time.Second)
			continue
		}
		app.state.TMNetInfo, err = app.rpc.GetNetInfo()
		if app.LogError(err) != nil {
			time.Sleep(5 * time.Second)
			continue
		}
		if app.ID == "" {
			app.ID = app.state.TMState.ValidatorInfo.Address.String()
			app.logger.Info("Core ID set ", "ID", app.ID)
		}
		if app.state.Height != 0 && app.state.ChainSynced {
			validators, err := app.rpc.GetValidators(app.state.Height)
			if app.LogError(err) != nil {
				continue
			}
			cores := validation.GetLastCalSubmitters(128, app.state) //get Active cores on network
			totalStake := (int64(len(cores)) * app.config.StakePerCore)
			stakeAmt := totalStake / int64(len(validators.Validators)) //total stake divided by 2/3 of validators
			app.state.Validators = validators.Validators
			app.LnClient.LocalSats = stakeAmt
			app.state.LnStakePerVal = stakeAmt
			app.state.LnStakePrice = stakeAmt * int64(len(validators.Validators)) //Total Stake Price includes the other 1/3 just in case
			//app.logger.Info(fmt.Sprintf("Stake Amt per Val: %d, total stake: %d", stakeAmt, app.state.LnStakePrice))
		}
		if app.state.TMState.SyncInfo.CatchingUp {
			app.state.ChainSynced = false
		} else {
			app.state.ChainSynced = true
		}
	}
}

//LNDMonitor : maintains unlock of wallet while abci is running, updates height, runs confirmation loop
func (app *AnchorApplication) LNDMonitor() {
	app.LnClient.Unlocker()
	state, err := app.LnClient.GetInfo()
	if app.LogError(err) == nil {
		app.state.LNState = *state
		if app.state.BtcHeight != int64(app.state.LNState.BlockHeight) {
			currBlockHeightInt64 := int64(app.state.LNState.BlockHeight)
			if currBlockHeightInt64 != 0 {
				err = app.MonitorBlocksForConfirmation(app.state.BtcHeight, currBlockHeightInt64)
				if app.LogError(err) != nil {
					return
				}
			}
			app.state.BtcHeight = int64(app.state.LNState.BlockHeight)
			app.logger.Info(fmt.Sprintf("New BTC Block %d", app.state.BtcHeight))
		}
	}
}

//StakeIdentity : updates active ECDSA public keys from all accessible peers
//Also ensures api is online
func (app *AnchorApplication) StakeIdentity() {
	// wait for syncMonitor
	for app.ID == "" || len(app.state.LNState.Uris) == 0 {
		app.logger.Info("StakeIdentity state loading...")
		time.Sleep(30 * time.Second)
	}
	// resend JWK if info has changed
	if lnUri, exists := app.state.LnUris[app.ID]; exists {
		if lnUri.Peer != app.state.LNState.Uris[0] {
			app.logger.Info(fmt.Sprintf("Stored Peer URI %s different from %s, resending JWK...", lnUri.Peer, app.state.LNState.Uris[0]))
			app.state.JWKStaked = false
		}
	}

	for !app.state.JWKStaked {
		app.logger.Info("Beginning Lightning staking loop")
		time.Sleep(60 * time.Second) //ensure loop gives chain time to init and doesn't restart on error too fast
		if !app.state.ChainSynced || app.state.Height < 2 || app.ID == "" {
			app.logger.Info("Chain not synced, restarting staking loop...")
			continue
		}
		amValidator, err := app.AmValidator()
		if app.LogError(err) != nil {
			app.logger.Info("Cannot determin validators, restarting staking loop...")
			continue
		}

		//if we're not a validator, we need to "stake" by opening a ln channel to the validators
		if !amValidator {
			app.logger.Info("This node is new to the network; beginning staking")
			waitForValidators := false
			for _, validator := range app.state.Validators {
				valID := validator.Address.String()
				if lnID, exists := app.state.LnUris[valID]; exists {
					app.logger.Info(fmt.Sprintf("Adding Lightning Peer %s...", lnID.Peer))
					peerExists, err := app.LnClient.PeerExists(lnID.Peer)
					app.LogError(err)
					if peerExists || app.LogError(app.LnClient.AddPeer(lnID.Peer)) == nil {
						chanExists, err := app.LnClient.ChannelExists(lnID.Peer, app.LnClient.LocalSats)
						app.LogError(err)
						if !chanExists {
							app.logger.Info(fmt.Sprintf("Adding Lightning Channel of local balance %d for Peer %s...", app.LnClient.LocalSats, lnID.Peer))
							_, err := app.LnClient.CreateChannel(lnID.Peer, app.LnClient.LocalSats)
							app.LogError(err)
						} else {
							app.logger.Info(fmt.Sprintf("Lightning Channel %s exists, skipping...", lnID.Peer))
							continue
						}
					}
				} else {
					waitForValidators = true
					continue
				}
			}
			if waitForValidators {
				app.logger.Info("Validator Lightning identities not all declared yet, waiting...")
				continue
			}
			deadline := time.Now().Add(time.Duration(10*(app.LnClient.MinConfs+1)) * time.Minute) // allow btc channel to open
			for !time.Now().After(deadline) {
				app.logger.Info("Sleeping to allow validator Lightning channels to open...")
				time.Sleep(time.Duration(1) * time.Minute)
			}
		} else {
			app.logger.Info("This node is a validator, skipping Lightning staking")
			app.state.AmValidator = true
		}
		if app.SendIdentity() != nil {
			app.logger.Info("Sending JWK Identity failed, restarting staking loop...")
			continue
		}
	}
}

// BeaconMonitor : elects a leader to poll DRAND. Called every minute by ABCI.commit
func (app *AnchorApplication) BeaconMonitor() {
	time.Sleep(30 * time.Second) //sleep after commit for a few seconds
	if app.state.Height > 2 {
		//round, randomness, err := beacon.GetPublicRandomness()
		round, randomness, err := beacon.GetCloudflareRandomness()
		chainpointFormat := fmt.Sprintf("%d:%s", round, randomness)
		if app.LogError(err) != nil {
			chainpointFormat = app.state.LatestTimeRecord // use the last "good" entropy beacon value known to this Core
		} else {
			app.state.LatestTimeRecord = chainpointFormat
			app.aggregator.LatestTime = app.state.LatestTimeRecord
		}
		if app.LogError(err) != nil {
			app.logger.Debug(fmt.Sprintf("Failed to gossip DRAND beacon value of %s", chainpointFormat))
		}
	}
}

// FeeMonitor : elects a leader to poll and gossip Fee. Called every n minutes by ABCI.commit
func (app *AnchorApplication) FeeMonitor() {
	time.Sleep(15 * time.Second) //sleep after commit for a few seconds
	if app.state.Height > 2 && app.state.Height-app.state.LastBtcFeeHeight >= app.config.FeeInterval {
		if leader, leaders := app.ElectValidatorAsLeader(1, []string{}); leader {
			app.logger.Info(fmt.Sprintf("FEE: Elected as leader. Leaders: %v", leaders))
			var fee int64
			fee, err := app.LnClient.GetLndFeeEstimate()
			app.LnClient.Logger.Info(fmt.Sprintf("FEE from LND: %d", fee))
			if app.LogError(err) != nil || fee <= STATIC_FEE_AMT {
				fee, err = fee2.GetThirdPartyFeeEstimate()
				app.LnClient.Logger.Info(fmt.Sprintf("FEE from Third Party: %d", fee))
				if fee < STATIC_FEE_AMT {
					fee = STATIC_FEE_AMT
				}
				if app.LogError(err) != nil || app.LnClient.Testnet {
					fee = int64(app.LnClient.FeeMultiplier * float64(fee))
					app.LnClient.Logger.Info(fmt.Sprintf("Static FEE: %d", fee))
				}
			}
			app.logger.Info(fmt.Sprintf("Ln Wallet EstimateFEE: %v", fee))
			_, err = app.rpc.BroadcastTx("FEE", strconv.FormatInt(fee, 10), 2, time.Now().Unix(), app.ID, &app.config.ECPrivateKey) // elect a leader to send a NIST tx
			if app.LogError(err) != nil {
				app.logger.Debug(fmt.Sprintf("Failed to gossip Fee value of %d", fee))
			}
		}
	}
}

func (app *AnchorApplication) CheckAnchor(btcmsg types.BtcTxMsg) error {
	btcBodyBytes, _ := hex.DecodeString(btcmsg.BtcTxBody)
	var msgTx wire.MsgTx
	msgTx.DeserializeNoWitness(bytes.NewReader(btcBodyBytes))
	b := txscript.NewScriptBuilder()
	b.AddOp(txscript.OP_RETURN)
	rootBytes, _ := hex.DecodeString(btcmsg.AnchorBtcAggRoot)
	b.AddData(rootBytes)
	outputScript, err := b.Script()
	if app.LogError(err) != nil {
		return err
	}
	for _, out := range msgTx.TxOut {
		if bytes.Compare(out.PkScript, outputScript) == 0 && msgTx.TxHash().String() == btcmsg.BtcTxID {
			app.logger.Info(fmt.Sprintf("BTC-A %s confirmed", btcmsg.BtcTxID))
			return nil
		}
		app.logger.Info(fmt.Sprintf("BTC-A Confirmation loop %s != %s", btcmsg.BtcTxID, msgTx.TxHash().String()))
	}
	return errors.New("unable to verify BTC-A")
}

//FailedAnchorMonitor: ensures transactions reach btc chain within certain time limit
func (app *AnchorApplication) FailedAnchorMonitor() {
	if app.state.LNState.BlockHeight == 0 {
		app.logger.Info("BTC Height record is 0, waiting for update from btc chain...")
		return
	}
	btcHeight := int64(app.state.LNState.BlockHeight)
	checkResults := app.RedisClient.WithContext(context.Background()).SMembers(CHECK_BTC_TX_IDS_KEY)
	if app.LogError(checkResults.Err()) != nil {
		return
	}
	//iterate through pending tx looking for timeouts
	for _, s := range checkResults.Val() {
		var anchor types.AnchorRange
		if app.LogError(json.Unmarshal([]byte(s), &anchor)) != nil {
			app.logger.Error("cannot unmarshal json for Failed BTC check")
			continue
		}
		app.logger.Info(fmt.Sprintf("Checking root %s at %d for failure", anchor.AnchorBtcAggRoot, anchor.BtcBlockHeight))
		//A core reported a lack of balance for anchoring
		if anchor.AnchorBtcAggRoot == app.state.LastErrorCoreID {
			app.logger.Info(fmt.Sprintf("BTC-E for aggroot %s from cal range %d to %d", anchor.AnchorBtcAggRoot, anchor.BeginCalTxInt, anchor.EndCalTxInt))
			delRes := app.RedisClient.WithContext(context.Background()).SRem(CHECK_BTC_TX_IDS_KEY, s)
			if app.LogError(delRes.Err()) != nil {
				continue
			}
			app.resetAnchor(anchor.BeginCalTxInt)
			continue
		}
		hasBeen10CalBlocks := app.state.Height - anchor.CalBlockHeight > 10
		hasBeen3BtcBlocks := anchor.BtcBlockHeight != 0 && btcHeight-anchor.BtcBlockHeight >= int64(3)

		newTx, tx := app.IsInNewTx(anchor.AnchorBtcAggRoot)                     // Is this a new tx (issuing core)?
		confirmed, confirmedTx := app.IsInConfirmedTxs(anchor.AnchorBtcAggRoot) // Is this a confirmed tx (all cores)?
		mempoolButNoBlock := confirmed && confirmedTx.BlockHeight == 0

		// if our tx is in the mempool but late, rbf
		if hasBeen3BtcBlocks && mempoolButNoBlock {
			app.logger.Info("RBF for", "AnchorBtcAggRoot", anchor.AnchorBtcAggRoot)
			newFee := math.Round(float64(app.state.LatestBtcFee*4/1000) * app.LnClient.FeeMultiplier)
			_, err := app.LnClient.ReplaceByFee(tx.BtcTxBody, false, int(newFee))
			if app.LogError(err) != nil {
				continue
			}
			app.logger.Info("RBF Success for", "AnchorBtcAggRoot", anchor.AnchorBtcAggRoot)
			//Remove old anchor check
			delRes := app.RedisClient.WithContext(context.Background()).SRem(CHECK_BTC_TX_IDS_KEY, s)
			if app.LogError(delRes.Err()) != nil {
				continue
			}
			//Add new anchor check
			anchor.BtcBlockHeight = btcHeight // give ourselves extra time
			failedAnchorJSON, _ := json.Marshal(anchor)
			redisResult := app.RedisClient.WithContext(context.Background()).SAdd(CHECK_BTC_TX_IDS_KEY, string(failedAnchorJSON))
			if app.LogError(redisResult.Err()) != nil {
				continue
			}
		}
		if hasBeen10CalBlocks && !confirmed { // if we have no confirmation of mempool inclusion after 10 minutes
			if newTx {
				app.logger.Info(fmt.Sprintf("Anchor Timeout: tx never transmitted, maybe check if lnd has peers?"))
				continue
			}
			app.logger.Info("Anchor Timeout", "AnchorBtcAggRoot", anchor.AnchorBtcAggRoot, "Tx", confirmedTx.TxID)
			// if there are subsequent anchors, we try to re-anchor just that range, else reset for a new anchor period
			if app.state.EndCalTxInt > anchor.EndCalTxInt {
				go app.AnchorBTC(anchor.BeginCalTxInt, anchor.EndCalTxInt)
			} else {
				app.resetAnchor(anchor.BeginCalTxInt)
			}
			app.RedisClient.WithContext(context.Background()).SRem(CHECK_BTC_TX_IDS_KEY, s)

		}
	}
}

//FindAndRemoveBtcCheck : remove all checks in case of btc tx failure
func (app *AnchorApplication) FindAndRemoveBtcCheck(aggRoot string) error {
	checkResults := app.RedisClient.WithContext(context.Background()).SMembers(CHECK_BTC_TX_IDS_KEY)
	if app.LogError(checkResults.Err()) != nil {
		return checkResults.Err()
	}
	for _, s := range checkResults.Val() {
		var anchor types.AnchorRange
		if app.LogError(json.Unmarshal([]byte(s), &anchor)) != nil {
			app.logger.Error("cannot unmarshal json for Failed BTC check")
			continue
		}
		if anchor.AnchorBtcAggRoot != aggRoot {
			continue
		}
		delRes := app.RedisClient.WithContext(context.Background()).SRem(CHECK_BTC_TX_IDS_KEY, s)
		if app.LogError(delRes.Err()) != nil {
			return delRes.Err()
		}
	}
	return nil
}

func (app *AnchorApplication) IsInNewTx(anchorRoot string) (bool, types.BtcTxMsg) {
	results := app.RedisClient.WithContext(context.Background()).SMembers(NEW_BTC_TX_IDS_KEY)
	if app.LogError(results.Err()) != nil {
		return false, types.BtcTxMsg{}
	}
	for _, s := range results.Val() {
		var tx types.BtcTxMsg
		if app.LogError(json.Unmarshal([]byte(s), &tx)) != nil {
			continue
		}
		if tx.AnchorBtcAggRoot == anchorRoot {
			return true, tx
		}
	}
	return false, types.BtcTxMsg{}
}


//MonitorNewTx: issues BTC-A upon new transaction confirmation. TODO: fold into FailedAnchorMonitor
func (app *AnchorApplication) MonitorNewTx() {
	app.logger.Info("Starting New BTC Check")
	results := app.RedisClient.WithContext(context.Background()).SMembers(NEW_BTC_TX_IDS_KEY)
	app.logger.Info(fmt.Sprintf("New BTC Check: Starting count for %d txns", len(results.Val())))
	if app.LogError(results.Err()) != nil {
		return
	}
	for _, s := range results.Val() {
		var tx types.BtcTxMsg
		if app.LogError(json.Unmarshal([]byte(s), &tx)) != nil {
			app.logger.Error("cannot unmarshal json for New BTC check")
			continue
		}
		txBytes, _ := hex.DecodeString(tx.BtcTxID)
		txDetails, err := app.LnClient.GetTransaction(txBytes)
		if app.LogError(err) != nil {
			app.logger.Info("New BTC Check: Cannot find transaction")
			continue
		}
		if len(txDetails.GetTransactions()) == 0 {
			app.logger.Info("New BTC Check: No transactions found")
			continue
		}
		app.logger.Info(fmt.Sprintf("Retrieved New Transaction %+v", txDetails.GetTransactions()))
		btcMonBytes, err := json.Marshal(tx)
		if app.LogError(err) != nil {
			app.logger.Info(fmt.Sprintf("New BTC Check: cannot marshal json"))
			continue
		}
		go func(monBytes []byte, res string) {
			app.logger.Info(fmt.Sprintf("New BTC Check: sending BTC-A %s", string(monBytes)))
			_, err = app.rpc.BroadcastTx("BTC-A", string(monBytes), 2, time.Now().Unix(), app.ID, &app.config.ECPrivateKey)
			if app.LogError(err) != nil {
				app.logger.Info(fmt.Sprintf("New BTC Check: failed sending BTC-A"))
			}
			delRes := app.RedisClient.WithContext(context.Background()).SRem(NEW_BTC_TX_IDS_KEY, res)
			app.LogError(delRes.Err())
		}(btcMonBytes, s)
	}
}

func (app *AnchorApplication) IsInConfirmedTxs(anchorRoot string) (bool, types.TxID) {
	results := app.RedisClient.WithContext(context.Background()).SMembers(CONFIRMED_BTC_TX_IDS_KEY)
	if app.LogError(results.Err()) != nil {
		return false, types.TxID{}
	}
	for _, s := range results.Val() {
		var tx types.TxID
		if app.LogError(json.Unmarshal([]byte(s), &tx)) != nil {
			continue
		}
		if tx.AnchorBtcAggRoot == anchorRoot {
			return true, tx
		}
	}
	return false, types.TxID{}
}

// MonitorConfirmedTx : Begins anchor confirmation process when a Tx is 6 btc blocks deep
func (app *AnchorApplication) MonitorConfirmedTx() {
	results := app.RedisClient.WithContext(context.Background()).SMembers(CONFIRMED_BTC_TX_IDS_KEY)
	if app.LogError(results.Err()) != nil {
		return
	}
	for _, s := range results.Val() {
		app.logger.Info(fmt.Sprintf("Checking confirmed btc tx %s", s))
		var tx types.TxID
		if app.LogError(json.Unmarshal([]byte(s), &tx)) != nil {
			continue
		}
		if tx.BlockHeight == 0 {
			app.logger.Info(fmt.Sprintf("btc tx %s not yet in block", s))
			continue
		}
		confirmCount := app.state.BtcHeight - tx.BlockHeight + 1
		if confirmCount < 6 {
			app.logger.Info(fmt.Sprintf("btc tx %s at %d confirmations", s, confirmCount))
			continue
		}
		block, tree, txIndex, err := app.GetBlockTree(tx)
		if app.LogError(err) != nil {
			if strings.Contains(err.Error(), "not found in block") {
				app.RedisClient.WithContext(context.Background()).SRem(CONFIRMED_BTC_TX_IDS_KEY, s)
			}
			continue
		}
		var btcmsg types.BtcMonMsg
		btcmsg.BtcTxID = tx.TxID
		btcmsg.BtcHeadHeight = tx.BlockHeight
		btcmsg.BtcHeadRoot = util.ReverseTxHex(hex.EncodeToString(block.MerkleRoot))
		proofs := tree.GetProof(txIndex)
		jsproofs := make([]types.JSProof, len(proofs))
		for i, proof := range proofs {
			if proof.Left {
				jsproofs[i] = types.JSProof{Left: hex.EncodeToString(proof.Value)}
			} else {
				jsproofs[i] = types.JSProof{Right: hex.EncodeToString(proof.Value)}
			}
		}
		btcmsg.Path = jsproofs
		go app.ConsumeBtcMonMsg(btcmsg)
		app.logger.Info(fmt.Sprintf("btc tx msg %+v confirmed from proof index %d", btcmsg, txIndex))
		delRes := app.RedisClient.WithContext(context.Background()).SRem(CONFIRMED_BTC_TX_IDS_KEY, s)
		if app.LogError(delRes.Err()) != nil {
			continue
		}
	}
}

// GetBlockTree : constructs block merkel tree with transaction as index
func (app *AnchorApplication) GetBlockTree(btcTx types.TxID) (lnrpc.BlockDetails, merkletools.MerkleTree, int, error) {
	block, err := app.LnClient.GetBlockByHeight(btcTx.BlockHeight)
	if app.LogError(err) != nil {
		return lnrpc.BlockDetails{}, merkletools.MerkleTree{}, -1, err
	}
	var tree merkletools.MerkleTree
	txIndex := -1
	for i, t := range block.Transactions {
		if t == btcTx.TxID {
			txIndex = i
		}
		tx := util.ReverseTxHex(t)
		hexTx, _ := hex.DecodeString(tx)
		tree.AddLeaf(hexTx)
	}
	if txIndex == -1 {
		return lnrpc.BlockDetails{}, merkletools.MerkleTree{}, -1, errors.New(fmt.Sprintf("Transaction %s not found in block %d", btcTx.TxID, btcTx.BlockHeight))
	}
	tree.MakeBTCTree()
	root := tree.GetMerkleRoot()
	reversedRoot := util.ReverseTxHex(hex.EncodeToString(root))
	reversedRootBytes, _ := hex.DecodeString(reversedRoot)
	reversedMerkleRoot, _ := hex.DecodeString(util.ReverseTxHex(hex.EncodeToString(block.MerkleRoot)))
	if !bytes.Equal(reversedRootBytes, reversedMerkleRoot) {
		return block, merkletools.MerkleTree{}, -1, errors.New(fmt.Sprintf("%s does not equal block merkle root %s", hex.EncodeToString(reversedRootBytes), hex.EncodeToString(block.MerkleRoot)))
	}
	return block, tree, txIndex, nil
}

// MonitorBlocksForConfirmation : since LND can't retrieve confirmed Txs, search block by block
func (app *AnchorApplication) MonitorBlocksForConfirmation(startHeight int64, endHeight int64) error {
	confirmationTxs := make([]types.TxID, 0)
	txsIdStrings := make([]string, 0)
	txsStrings := make([]string, 0)
	results := app.RedisClient.WithContext(context.Background()).SMembers(CONFIRMED_BTC_TX_IDS_KEY)
	for _, s := range results.Val() {
		var tx types.TxID
		if app.LogError(json.Unmarshal([]byte(s), &tx)) != nil {
			continue
		}
		if tx.BlockHeight != 0 {
			continue
		}
		confirmationTxs = append(confirmationTxs, tx)
		txsIdStrings = append(txsIdStrings, tx.TxID)
		txsStrings = append(txsStrings, s)
	}
	for i := startHeight; i < endHeight + 1; i++ {
		block, err := app.LnClient.GetBlockByHeight(i)
		if app.LogError(err) != nil {
			return err
		}
		for _, t := range block.Transactions {
			if contains, index := util.ArrayContainsIndex(txsIdStrings, t); contains {
				confirmationTx := txsStrings[index]
				tx := confirmationTxs[index]
				app.RedisClient.WithContext(context.Background()).SRem(CONFIRMED_BTC_TX_IDS_KEY, confirmationTx)
				tx.BlockHeight = i
				txIDBytes, _ := json.Marshal(tx)
				app.RedisClient.WithContext(context.Background()).SAdd(CONFIRMED_BTC_TX_IDS_KEY, string(txIDBytes))
				app.logger.Info(fmt.Sprintf("Found tx %s in block %d", tx.TxID, i))
			}
		}
	}
}
