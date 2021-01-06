package abci

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/chainpoint/chainpoint-core/go-abci-service/merkletools"
	"github.com/lightningnetwork/lnd/lnrpc"
	"strconv"
	"strings"
	"time"

	"github.com/chainpoint/chainpoint-core/go-abci-service/lightning"

	"github.com/chainpoint/chainpoint-core/go-abci-service/validation"

	"github.com/chainpoint/chainpoint-core/go-abci-service/util"
	"github.com/go-redis/redis"

	beacon "github.com/chainpoint/chainpoint-core/go-abci-service/beacon"

	"github.com/chainpoint/chainpoint-core/go-abci-service/types"
)

const CONFIRMED_BTC_TX_IDS_KEY = "BTC_Mon:ConfirmedBTCTxIds"
const NEW_BTC_TX_IDS_KEY = "BTC_Mon:NewBTCTxIds"
const CHECK_BTC_TX_IDS_KEY = "BTC_Mon:CheckNewBTCTxIds"
const STATIC_FEE_AMT = 60000 // 12500 // 60k amounts to 240 sat/vbyte

//SyncMonitor : turns off anchoring if we're not synced. Not cron scheduled since we need it to start immediately.
func (app *AnchorApplication) SyncMonitor() {
	for {
		time.Sleep(30 * time.Second) // allow chain time to initialize
		//app.logger.Info("Syncing Chain status and validators")
		status, err := app.rpc.GetStatus()
		if app.LogError(err) != nil {
			time.Sleep(5 * time.Second)
			continue
		}
		if app.ID == "" {
			app.ID = string(status.ValidatorInfo.Address.String())
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
			app.lnClient.LocalSats = stakeAmt
			app.state.LnStakePerVal = stakeAmt
			app.state.LnStakePrice = stakeAmt * int64(len(validators.Validators)) //Total Stake Price includes the other 1/3 just in case
			//app.logger.Info(fmt.Sprintf("Stake Amt per Val: %d, total stake: %d", stakeAmt, app.state.LnStakePrice))
		}
		if app.LogError(err) != nil {
			continue
		}
		if status.SyncInfo.CatchingUp {
			app.state.ChainSynced = false
		} else {
			app.state.ChainSynced = true
		}
	}
}

//LNDMonitor : maintains unlock of wallet while abci is running
func (app *AnchorApplication) LNDMonitor() {
	for {
		app.lnClient.Unlocker()
		time.Sleep(60 * time.Second)
	}
}

//StakeIdentity : updates active ECDSA public keys from all accessible peers
//Also ensures api is online
func (app *AnchorApplication) StakeIdentity() {
	for !app.state.JWKStaked {
		app.logger.Info("Beginning Lightning staking loop")
		time.Sleep(60 * time.Second) //ensure loop gives chain time to init and doesn't restart on error too fast
		if !app.state.ChainSynced || app.state.Height < 2 || app.ID == "" {
			continue
		}
		amValidator, err := app.AmValidator()
		if app.LogError(err) != nil {
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
					peerExists, err := app.lnClient.PeerExists(lnID.Peer)
					app.LogError(err)
					if peerExists || app.LogError(app.lnClient.AddPeer(lnID.Peer)) == nil {
						chanExists, err := app.lnClient.ChannelExists(lnID.Peer, app.lnClient.LocalSats)
						app.LogError(err)
						if !chanExists {
							app.logger.Info(fmt.Sprintf("Adding Lightning Channel of local balance %d for Peer %s...", app.lnClient.LocalSats, lnID.Peer))
							_, err := app.lnClient.CreateChannel(lnID.Peer, app.lnClient.LocalSats)
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
			deadline := time.Now().Add(time.Duration(10*(app.lnClient.MinConfs+1)) * time.Minute) // allow btc channel to open
			for !time.Now().After(deadline) {
				app.logger.Info("Sleeping to allow validator Lightning channels to open...")
				time.Sleep(time.Duration(1) * time.Minute)
			}
		} else {
			app.logger.Info("This node is a validator, skipping Lightning staking")
			app.state.AmValidator = true
		}
		jwkJson, err := json.Marshal(app.JWK)
		if app.LogError(err) != nil {
			continue
		}
		//Create ln identity struct
		resp, err := app.lnClient.GetInfo()
		if app.LogError(err) != nil || len(resp.Uris) == 0 {
			continue
		}
		uri := resp.Uris[0]
		lnID := types.LnIdentity{
			Peer:            uri,
			RequiredChanAmt: app.lnClient.LocalSats,
		}
		lnIDBytes, err := json.Marshal(lnID)
		if app.LogError(err) != nil {
			continue
		}
		app.logger.Info("Sending JWK...", "JWK", string(jwkJson))
		//Declare our identity to the network
		_, err = app.rpc.BroadcastTxWithMeta("JWK", string(jwkJson), 2, time.Now().Unix(), app.ID, string(lnIDBytes), &app.config.ECPrivateKey)
		if app.LogError(err) != nil {
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
			fee, err := app.lnClient.GetLndFeeEstimate()
			app.lnClient.Logger.Info(fmt.Sprintf("FEE from LND: %d", fee))
			if err != nil || fee <= STATIC_FEE_AMT {
				app.logger.Info("Attempting to use third party FEE....")
				fee, err = app.lnClient.GetThirdPartyFeeEstimate()
				if fee < STATIC_FEE_AMT {
					fee = STATIC_FEE_AMT
				}
				if err != nil || app.lnClient.Testnet {
					app.logger.Info("falling back to static FEE")
					fee = int64(app.lnClient.FeeMultiplier * float64(STATIC_FEE_AMT))
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

//LoadIdentity : load public keys derived from JWTs from redis
func (app *AnchorApplication) LoadIdentity() error {
	var cursor uint64
	var idKeys []string
	for {
		var keys []string
		var err error
		keys, cursor, err = app.redisClient.Scan(cursor, "CoreID:*", 10).Result()
		if err != nil {
			return err
		}
		idKeys = append(idKeys, keys...)
		if cursor == 0 {
			break
		}
	}
	if len(idKeys) == 0 {
		return util.LoggerError(app.logger, errors.New("no JWT keys found in redis"))
	}
	for _, k := range idKeys {
		var coreID string
		idStr := strings.Split(k, ":")
		if len(idStr) == 2 {
			coreID = idStr[1]
		} else {
			continue
		}
		b64Str, err := app.redisClient.Get(k).Result()
		if app.LogError(err) != nil {
			continue
		}
		pubKeyBytes, err := base64.StdEncoding.DecodeString(b64Str)
		if app.LogError(err) != nil {
			continue
		}
		x, y := elliptic.Unmarshal(elliptic.P256(), pubKeyBytes)
		pubKey := ecdsa.PublicKey{
			Curve: elliptic.P256(),
			X:     x,
			Y:     y,
		}
		app.logger.Info(fmt.Sprintf("Setting JWK Identity for Core %s: %s", coreID, b64Str))
		app.state.CoreKeys[coreID] = pubKey
		app.state.TxValidation[fmt.Sprintf("%x", pubKeyBytes)] = validation.NewTxValidation()
	}

	//map all NodeKey IDs to PrivateValidator addresses for consumption by peer filter
	go func() {
		for i := 1; i < 5; i++ {
			_, err := app.rpc.GetStatus()
			if err == nil {
				break
			} else {
				app.logger.Info("Waiting for tendermint to be ready...")
				time.Sleep(5 * time.Second)
			}
		}
		txs, err := app.getAllJWKs()
		if err == nil {
			for _, tx := range txs {
				var jwkType types.Jwk
				err := json.Unmarshal([]byte(tx.Data), &jwkType)
				if app.LogError(err) != nil {
					continue
				}
				app.state.IDMap[jwkType.Kid] = tx.CoreID
			}
		} else {
			app.LogError(err)
		}
	}()
	return nil
}

//VerifyIdentity : Verify that a channel exists only if we're a validator and the chain is synced
func (app *AnchorApplication) VerifyIdentity(tx types.Tx) bool {
	app.logger.Info(fmt.Sprintf("Verifying JWK Identity for %#v", tx))
	// Verification only matters to the chain if the chain is synced and we're a validator.
	// If we're the first validator, we accept by default.
	_, alreadyExists := app.state.CoreKeys[tx.CoreID]
	if app.ID == tx.CoreID {
		app.logger.Info("Validated JWK since we're the proposer")
		return true
	} else if app.state.ChainSynced && app.state.AmValidator && !alreadyExists {
		lnID := types.LnIdentity{}
		if app.LogError(json.Unmarshal([]byte(tx.Meta), &lnID)) != nil {
			return false
		}
		app.logger.Info("Checking if the incoming JWK Identity is from a validator")
		isVal, err := app.IsValidator(tx.CoreID)
		app.LogError(err)
		if isVal {
			return true
		}
		app.logger.Info("JWK Identity: Checking Channel Funding")
		chanExists, err := app.lnClient.ChannelExists(lnID.Peer, app.lnClient.LocalSats)
		if app.LogError(err) == nil && chanExists {
			app.logger.Info("JWK Identity: Channel Open and Funded")
			return true
		} else {
			app.logger.Info("JWK Identity: Channel not open, rejecting")
			return false
		}
	} else if !app.state.ChainSynced {
		// we're fast-syncing, so agree with the prior chainstate
		return true
	} else if isVal, err := app.IsValidator(tx.CoreID); err == nil && isVal && app.state.AmValidator {
		// if we're both validators, verify identity
		return true
	}
	app.logger.Info("JWK Identity", "alreadyExists", alreadyExists)
	return !alreadyExists
}

//SaveIdentity : save the JWT value retrieved
func (app *AnchorApplication) SaveIdentity(tx types.Tx) error {
	var jwkType types.Jwk
	json.Unmarshal([]byte(tx.Data), &jwkType)
	key := fmt.Sprintf("CorePublicKey:%s", jwkType.Kid)
	app.logger.Info("JWK kid", "JWK Tx kid", jwkType.Kid, "app JWK kid", app.JWK.Kid)
	jsonJwk, err := json.Marshal(jwkType)
	if app.LogError(err) != nil {
		return err
	}
	pubKey, err := util.DecodePubKey(tx)
	var pubKeyBytes []byte
	if app.LogError(err) == nil {
		app.state.CoreKeys[tx.CoreID] = *pubKey
		pubKeyBytes = elliptic.Marshal(pubKey.Curve, pubKey.X, pubKey.Y)
		util.LoggerError(app.logger, app.redisClient.Set("CoreID:"+tx.CoreID, base64.StdEncoding.EncodeToString(pubKeyBytes), 0).Err())
	}
	value, err := app.redisClient.Get(key).Result()
	if app.LogError(err) == redis.Nil || value != string(jsonJwk) {
		err = app.redisClient.Set(key, value, 0).Err()
		if app.LogError(err) != nil {
			return err
		}
		app.logger.Info(fmt.Sprintf("Set JWK cache for kid %s", jwkType.Kid))
	}
	pubKeyHex := fmt.Sprintf("%x", pubKeyBytes)
	if val, exists := app.state.TxValidation[pubKeyHex]; exists {
		app.state.TxValidation[pubKeyHex] = val
	} else {
		validation := validation.NewTxValidation()
		app.state.TxValidation[pubKeyHex] = validation
	}
	lnID := types.LnIdentity{}
	app.LogError(json.Unmarshal([]byte(tx.Meta), &lnID))
	if lightning.IsLnUri(lnID.Peer) {
		app.state.LnUris[tx.CoreID] = lnID
	}
	if jwkType.Kid != "" && app.JWK.Kid != "" && jwkType.Kid == app.JWK.Kid {
		app.logger.Info("JWK keysync tx committed")
		app.state.JWKStaked = true
	}
	return nil
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

func (app *AnchorApplication) FailedAnchorMonitor() {
	results := app.redisClient.WithContext(context.Background()).SMembers(CHECK_BTC_TX_IDS_KEY)
	if app.LogError(results.Err()) != nil {
		return
	}
	for _, s := range results.Val() {
		var anchor types.AnchorRange
		if app.LogError(json.Unmarshal([]byte(s), &anchor)) != nil {
			app.logger.Error("cannot unmarshal json for Failed BTC check")
			continue
		}
		if app.state.Height-anchor.CalBlockHeight >= int64(app.config.AnchorTimeout) || app.state.LatestErrRoot == anchor.AnchorBtcAggRoot {
			if app.state.Height-anchor.CalBlockHeight >= int64(app.config.AnchorTimeout) {
				app.logger.Info(fmt.Sprintf("Anchor Failure (timeout), Resetting state for aggroot %s from cal range %d to %d", anchor.AnchorBtcAggRoot, anchor.BeginCalTxInt, anchor.EndCalTxInt))
			} else {
				app.logger.Info(fmt.Sprintf("Anchor Failure (BTC-E), Resetting state for aggroot %s from cal range %d to %d", anchor.AnchorBtcAggRoot, anchor.BeginCalTxInt, anchor.EndCalTxInt))
			}
			app.resetAnchor(anchor.BeginCalTxInt)
			validation.IncrementFailedAnchor(app.state.LastElectedCoreID, &app.state)
			delRes := app.redisClient.WithContext(context.Background()).SRem(CHECK_BTC_TX_IDS_KEY, s)
			if app.LogError(delRes.Err()) != nil {
				continue
			}
			app.logger.Info("Checking if we were leader and need to remove New BTC Check....")
			results := app.redisClient.WithContext(context.Background()).SMembers(NEW_BTC_TX_IDS_KEY)
			if app.LogError(results.Err()) != nil {
				continue
			}
			for _, a := range results.Val() {
				var tx types.BtcTxMsg
				if app.LogError(json.Unmarshal([]byte(a), &tx)) != nil {
					app.logger.Error("cannot unmarshal json for New BTC check")
					continue
				}
				if tx.AnchorBtcAggRoot == anchor.AnchorBtcAggRoot {
					app.logger.Info("Removing New BTC Check", "AnchorBtcAggRoot", anchor.AnchorBtcAggRoot)
					delRes = app.redisClient.WithContext(context.Background()).SRem(NEW_BTC_TX_IDS_KEY, a)
					if app.LogError(delRes.Err()) != nil {
						continue
					}
					break
				}
			}
		}
	}
}

func (app *AnchorApplication) GetBlockTree(btcTx types.TxID) (lnrpc.BlockDetails, merkletools.MerkleTree, int, error) {
	block, err := app.lnClient.GetBlockByHeight(btcTx.BlockHeight)
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

func (app *AnchorApplication) MonitorNewTx() {
	app.logger.Info("Starting New BTC Check")
	results := app.redisClient.WithContext(context.Background()).SMembers(NEW_BTC_TX_IDS_KEY)
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
		txDetails, err := app.lnClient.GetTransaction(txBytes)
		if app.LogError(err) != nil {
			app.logger.Info("New BTC Check: Cannot find transaction")
			continue
		}
		if len(txDetails.GetTransactions()) == 0 {
			app.logger.Info("New BTC Check: No transactions found")
			continue
		}
		txData := txDetails.Transactions[0]
		if txData.NumConfirmations < 1 {
			app.logger.Info(fmt.Sprintf("New BTC Check: %s not yet confirmed", tx.BtcTxID))
			continue
		} else {
			app.logger.Info(fmt.Sprintf("New BTC Check: %s has been confirmed", tx.BtcTxID))
		}
		app.logger.Info(fmt.Sprintf("New BTC Check: block height is %d", int64(txData.BlockHeight)))
		tx.BtcTxHeight = int64(txData.BlockHeight)
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
			delRes := app.redisClient.WithContext(context.Background()).SRem(NEW_BTC_TX_IDS_KEY, res)
			app.LogError(delRes.Err())
		}(btcMonBytes, s)
	}
}

func (app *AnchorApplication) MonitorConfirmedTx() {
	results := app.redisClient.WithContext(context.Background()).SMembers(CONFIRMED_BTC_TX_IDS_KEY)
	if app.LogError(results.Err()) != nil {
		return
	}
	for _, s := range results.Val() {
		app.logger.Info(fmt.Sprintf("Checking btc tx %s", s))
		var tx types.TxID
		if app.LogError(json.Unmarshal([]byte(s), &tx)) != nil {
			return
		}
		info, err := app.lnClient.GetInfo()
		if app.LogError(err) != nil {
			return
		}
		confirmCount := info.BlockHeight - uint32(tx.BlockHeight) + 1
		if confirmCount < 6 {
			app.logger.Info(fmt.Sprintf("btc tx %s at %d confirmations", s, confirmCount))
			continue
		}
		block, tree, txIndex, err := app.GetBlockTree(tx)
		if app.LogError(err) != nil {
			if strings.Contains(err.Error(), "not found in block") {
				app.redisClient.WithContext(context.Background()).SRem(CONFIRMED_BTC_TX_IDS_KEY, s)
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
		delRes := app.redisClient.WithContext(context.Background()).SRem(CONFIRMED_BTC_TX_IDS_KEY, s)
		if app.LogError(delRes.Err()) != nil {
			continue
		}
	}
}
