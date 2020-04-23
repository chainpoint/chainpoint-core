package abci

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/chainpoint/chainpoint-core/go-abci-service/merkletools"
	"github.com/lightningnetwork/lnd/lnrpc"
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

//SyncMonitor : turns off anchoring if we're not synced. Not cron scheduled since we need it to start immediately.
func (app *AnchorApplication) SyncMonitor() {
	for {
		time.Sleep(30 * time.Second) // allow chain time to initialize
		app.logger.Info("Syncing Chain status and validators")
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
			cores := validation.GetLastNistSubmitters(128, app.state) //get Active cores on network
			totalStake := (int64(len(cores)) * app.config.StakePerCore)
			stakeAmt := totalStake / int64(len(validators.Validators)) //total stake divided by 2/3 of validators
			app.Validators = validators.Validators
			app.lnClient.LocalSats = stakeAmt
			app.state.LnStakePerVal = stakeAmt
			app.state.LnStakePrice = stakeAmt * int64(len(validators.Validators))  //Total Stake Price includes the other 1/3 just in case
			app.logger.Info(fmt.Sprintf("Stake Amt per Val: %d, total stake: %d", stakeAmt, app.state.LnStakePrice))
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
			for _, validator := range app.Validators {
				valID := validator.Address.String()
				if lnID, exists := app.state.LnUris[valID]; exists {
					app.logger.Info(fmt.Sprintf("Adding Lightning Peer %s...", lnID.Peer))
					peerExists, err := app.lnClient.PeerExists(lnID.Peer)
					app.LogError(err)
					if peerExists || app.LogError(app.lnClient.AddPeer(lnID.Peer)) == nil {
						chanExists, err := app.lnClient.ChannelExists(lnID.Peer, app.lnClient.LocalSats)
						app.LogError(err)
						if !chanExists {
							app.logger.Info(fmt.Sprintf("Adding Lightning Channel of local balance %d for Peer %s...", lnID.RequiredChanAmt, lnID.Peer))
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

// NistBeaconMonitor : elects a leader to poll and gossip NIST. Called every minute by ABCI.commit
func (app *AnchorApplication) NistBeaconMonitor() {
	time.Sleep(15 * time.Second) //sleep after commit for a few seconds
	if app.state.Height > 2 && app.state.ChainSynced {
		if leader, leaders := app.ElectChainContributorAsLeaderNaive(1, []string{}); leader {
			app.logger.Info(fmt.Sprintf("NIST: Elected as leader. Leaders: %v", leaders))
			nistRecord, err := beacon.LastRecord()
			chainpointFormat := ""
			if app.LogError(err) != nil {
				if !strings.Contains(err.Error(), "stale") {
					chainpointFormat = app.state.LatestNistRecord // use the last "good" entropy beacon value known to this Core
				}
			}
			if chainpointFormat == "" {
				chainpointFormat = nistRecord.ChainpointFormat()
			}
			_, err = app.rpc.BroadcastTx("NIST", nistRecord.ChainpointFormat(), 2, time.Now().Unix(), app.ID, &app.config.ECPrivateKey) // elect a leader to send a NIST tx
			if app.LogError(err) != nil {
				app.logger.Debug(fmt.Sprintf("Failed to gossip NIST beacon value of %s", nistRecord.ChainpointFormat()))
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
	return nil
}

//VerifyIdentity : Verify that a channel exists only if we're a validator and the chain is synced
func (app *AnchorApplication) VerifyIdentity(tx types.Tx) bool {
	app.logger.Info(fmt.Sprintf("Verifying JWK Identity for %#v", tx))
	// Verification only matters to the chain if the chain is synced and we're a validator.
	// If we're the first validator, we accept by default.
	_, alreadyExists := app.state.CoreKeys[tx.CoreID]
	if app.state.ChainSynced && app.state.AmValidator && !alreadyExists && app.ID != tx.CoreID {
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
		chanExists, err := app.lnClient.RemoteChannelOpenAndFunded(lnID.Peer, app.lnClient.LocalSats)
		if app.LogError(err) == nil && chanExists {
			app.logger.Info("JWK Identity: Channel Open and Funded")
			return true
		} else {
			app.logger.Info("JWK Identity: Channel not open, rejecting")
			return false
		}
	} else if (!app.state.ChainSynced){
		// we're fast-syncing, so agree with the prior chainstate
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
	if jwkType.Kid != "" && app.JWK.Kid != "" && jwkType.Kid == app.JWK.Kid {
		app.logger.Info("JWK keysync tx committed")
		app.state.JWKStaked = true
	}
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
	return nil
}

func (app *AnchorApplication) CheckAnchor (btcmsg types.BtcTxMsg) (bool) {
	block, err := app.lnClient.GetBlock(btcmsg.BtcTxHeight)
	if app.LogError(err) != nil {
		return false
	}
	for _, t := range block.Transactions {
		if t == btcmsg.BtcTxID {
			app.logger.Info("BTC-A %s confirmed", t)
			return true
		}
	}
	return false
}

func (app *AnchorApplication) GetBlockTree (btcTx types.TxID) (lnrpc.BlockDetails, merkletools.MerkleTree, int, error) {
	block, err := app.lnClient.GetBlock(btcTx.BlockHeight)
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
		return lnrpc.BlockDetails{}, merkletools.MerkleTree{}, -1, errors.New("Transaction not found in block")
	}
	tree.MakeBTCTree()
	root := tree.GetMerkleRoot()
	reversedRoot := util.ReverseTxHex(hex.EncodeToString(root))
	reversedRootBytes, _ := hex.DecodeString(reversedRoot)
	if ! bytes.Equal(reversedRootBytes, block.MerkleRoot) {
		return block, merkletools.MerkleTree{}, -1, errors.New(fmt.Sprintf("%s does not equal block merkle root %s", hex.EncodeToString(reversedRootBytes), hex.EncodeToString(block.MerkleRoot)))
	}
	return block, tree, txIndex, nil
}

func (app *AnchorApplication) MonitorConfirmedTx () {
	results := app.redisClient.SMembers(CONFIRMED_BTC_TX_IDS_KEY)
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
			fmt.Sprintf(fmt.Sprintf("btc tx %s at %d confirmations", s, confirmCount))
			continue
		}
		block, tree, txIndex, err := app.GetBlockTree(tx)
		if app.LogError(err) != nil {
			return
		}
		var btcmsg types.BtcMonMsg
		btcmsg.BtcTxID = tx.TxID
		btcmsg.BtcHeadHeight = tx.BlockHeight
		btcmsg.BtcHeadRoot = hex.EncodeToString(block.MerkleRoot)
		proofs := tree.GetProof(txIndex)
		jsproofs := make([]types.JSProof, len(proofs))
		for i, proof := range proofs {
			if proof.Left {
				jsproofs[i] = types.JSProof{Left: hex.EncodeToString(proof.Value)}
			}else {
				jsproofs[i] = types.JSProof{Right: hex.EncodeToString(proof.Value)}
			}
		}
		err = app.ConsumeBtcMonMsg(btcmsg)
		app.logger.Info(fmt.Sprintf("btc tx %s confirmed", s))
		if app.LogError(err) == nil {
			delRes := app.redisClient.SRem(CONFIRMED_BTC_TX_IDS_KEY, s)
			if app.LogError(delRes.Err()) != nil {
				return
			}
		}
	}
}
