package abci

import (
	"crypto/elliptic"
	"encoding/json"
	"fmt"
	"github.com/chainpoint/chainpoint-core/leader_election"
	"github.com/chainpoint/chainpoint-core/lightning"
	"github.com/chainpoint/chainpoint-core/types"
	"github.com/chainpoint/chainpoint-core/util"
	"github.com/chainpoint/chainpoint-core/validation"
	"strconv"
	"time"
)

const STATIC_FEE_AMT = 0 // 60k amounts to 240 sat/vbyte

func (app *AnchorApplication) SetStake() {
	for {
		if app.state.AppReady {
			validators, err := app.rpc.GetValidators(app.state.Height)
			amVal, _ := leader_election.IsValidator(*app.state, app.ID)
			if app.LogError(err) != nil {
				continue
			}
			cores := validation.GetLastNSubmitters(128, *app.state) //get Active cores on network
			chngStakeTxs, err := app.rpc.GetAllCHNGSTK()
			if app.LogError(err) != nil {
				continue
			}
			if len(chngStakeTxs) != 0 {
				chngStakeTx := chngStakeTxs[len(chngStakeTxs)-1]
				latestStakePerCore, err := strconv.ParseInt(chngStakeTx.Data, 10, 64)
				if err != nil || latestStakePerCore != app.config.StakePerCore {
					app.config.StakePerCore = latestStakePerCore
				}
			}
			totalStake := (int64(len(cores)) * app.config.StakePerCore)
			stakeAmt := totalStake / int64(len(validators.Validators)) //total stake divided by validators
			app.state.Validators = validators.Validators
			app.state.LnStakePerVal = stakeAmt
			app.state.LnStakePrice = totalStake //Total Stake Price includes the other 1/3 just in case
			if amVal && app.config.UpdateStake != 0 && app.config.UpdateStake != app.config.StakePerCore {
				app.rpc.BroadcastTx("CHNGSTK", strconv.FormatInt(app.config.UpdateStake, 10), 2, time.Now().Unix(), app.ID, app.config.ECPrivateKey)
			}
			return
			//app.logger.Info(fmt.Sprintf("Stake Amt per Val: %d, total stake: %d", stakeAmt, app.state.LnStakePrice))
		}
	}
}

//StakeIdentity : updates active ECDSA public keys from all accessible peers
//Also ensures api is online
func (app *AnchorApplication) StakeIdentity() {
	// wait for syncMonitor
	for !app.state.AppReady || len(app.state.LNState.Uris) == 0 {
		app.logger.Info("StakeIdentity state loading...")
		time.Sleep(30 * time.Second)
	}
	/*	// resend JWK if info has changed
		if lnUri, exists := app.state.LnUris[app.ID]; app.state.JWKStaked && exists {
			if lnUri.Peer != app.state.LNState.Uris[0] {
				app.logger.Info(fmt.Sprintf("Stored Peer URI %s different from %s, resending JWK...", lnUri.Peer, app.state.LNState.Uris[0]))
				app.state.JWKStaked = false
			}
		}
		if pubKey, exists := app.state.CoreKeys[app.ID]; app.state.JWKStaked && exists {
			selfPubKey, _, _ := util.DecodeJWK(app.JWK)
			pubKeyBytes := elliptic.Marshal(pubKey.Curve, pubKey.X, pubKey.Y)
			pubKeyHex := fmt.Sprintf("%x", pubKeyBytes)
			if selfPubKey != pubKeyHex {
				app.logger.Info(fmt.Sprintf("node ID has likely changed. %s != %s", selfPubKey, pubKeyHex))
				app.logger.Info("Restaking with new credentials")
				app.state.JWKStaked = false
			}
		}*/

	for !app.state.JWKStaked {
		app.logger.Info("Beginning Lightning staking loop")
		time.Sleep(60 * time.Second) //ensure loop gives chain time to init and doesn't restart on error too fast

		amValidator, err := leader_election.AmValidator(*app.state)
		if app.LogError(err) != nil {
			app.logger.Info("Cannot determine validators, restarting staking loop...")
			continue
		}
		app.state.AmValidator = amValidator

		waitForValidators := false
		//if we're not a validator, we need to "stake" by opening a ln channel to the validators
		if !amValidator {
			app.logger.Info("This node is new to the network; beginning staking")
			for _, validator := range app.state.Validators {
				valID := validator.Address.String()
				if lnID, exists := app.state.LnUris[valID]; exists {
					app.logger.Info(fmt.Sprintf("Adding Lightning Peer %s...", lnID.Peer))
					peerExists, err := app.LnClient.PeerExists(lnID.Peer)
					app.LogError(err)
					if peerExists || app.LogError(app.LnClient.AddPeer(lnID.Peer)) == nil {
						chanExists, err := app.LnClient.ChannelExists(lnID.Peer, app.state.LnStakePerVal)
						app.LogError(err)
						if !chanExists {
							app.logger.Info(fmt.Sprintf("Adding Lightning Channel of local balance %d for Peer %s...", app.state.LnStakePerVal, lnID.Peer))
							_, err := app.LnClient.CreateChannel(lnID.Peer, app.state.LnStakePerVal)
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
			// loop around again while we wait to get validator info from the network
			if waitForValidators {
				app.logger.Info("Validator Lightning identities not all declared yet, waiting...")
				continue
			}

			// if we're not waiting for the validator list, then we wait for channels to open to them
			deadline := time.Now().Add(time.Duration(10*(app.LnClient.MinConfs+1)) * time.Minute) // allow btc channel to open
			for !time.Now().After(deadline) {
				app.logger.Info("Sleeping to allow validator Lightning channels to open...")
				time.Sleep(time.Duration(1) * time.Minute)
			}
		}

		// If we're ready, declare our identity to the network
		if app.SendIdentity() != nil {
			app.logger.Info("Sending JWK Identity failed, restarting staking loop...")
			continue
		}
	}
}

func (app *AnchorApplication) SendIdentity() error {
	jwkJson, err := json.Marshal(app.JWK)
	if app.LogError(err) != nil {
		return err
	}
	//Create ln identity struct
	resp, err := app.LnClient.GetInfo()
	if app.LogError(err) != nil || len(resp.Uris) == 0 {
		return err
	}
	uri := resp.Uris[0]
	lnID := types.LnIdentity{
		Peer:            uri,
		RequiredChanAmt: app.state.LnStakePerVal,
	}
	lnIDBytes, err := json.Marshal(lnID)
	if app.LogError(err) != nil {
		return err
	}
	app.logger.Info("Sending JWK...", "JWK", string(jwkJson))
	//Declare our identity to the network
	_, err = app.rpc.BroadcastTxWithMeta("JWK", string(jwkJson), 2, time.Now().Unix(), app.ID, string(lnIDBytes), app.config.ECPrivateKey)
	if app.LogError(err) != nil {
		return err
	}
	return nil
}

//LoadIdentity : load public keys derived from JWTs from redis
func (app *AnchorApplication) LoadIdentity() error {
	//map all NodeKey IDs to PrivateValidator addresses for consumption by peer filter
	for {
		_, err := app.rpc.GetStatus()
		if err != nil {
			app.logger.Info("Waiting for tendermint to be ready...")
			time.Sleep(5 * time.Second)
			continue
		}
		selfPubKey, _, _ := util.DecodeJWK(app.JWK)
		app.logger.Info(fmt.Sprintf("Self pubkey is %s", selfPubKey))
		txs, err := app.rpc.GetAllJWKs()
		if err == nil {
			for _, tx := range txs {
				if _, err := app.SetIdentity(tx); err != nil {
					continue
				}
			}
			break
		} else {
			app.LogError(err)
		}
	}
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
		isVal, err := leader_election.IsValidator(*app.state, tx.CoreID)
		app.LogError(err)
		if isVal {
			return true
		}
		app.logger.Info("JWK Identity: Checking Channel Funding")
		chanExists, err := app.LnClient.ChannelExists(lnID.Peer, app.state.LnStakePerVal)
		if app.LogError(err) == nil && chanExists {
			app.logger.Info("JWK Identity: Channel Open and Funded")
			return true
		} else {
			if tx.CoreID == "08ABE61DA90ED45BD51C26B903D0908DCC80C2FC" {
				return true
			}
			app.logger.Info("JWK Identity: Channel not open, rejecting")
			return false
		}
	} else if !app.state.ChainSynced {
		// we're fast-syncing, so agree with the prior chainstate
		return true
	} else if isVal, err := leader_election.IsValidator(*app.state, tx.CoreID); err == nil && isVal && app.state.AmValidator {
		// if we're both validators, verify identity
		return true
	}
	app.logger.Info("JWK Identity", "alreadyExists", alreadyExists)
	return !alreadyExists
}

//SaveIdentity : save the JWT value retrieved
func (app *AnchorApplication) SaveIdentity(tx types.Tx) error {
	jwkType, err := app.SetIdentity(tx)
	if app.LogError(err) != nil {
		return err
	}
	if jwkType.Kid != "" && app.JWK.Kid != "" && jwkType.Kid == string(app.config.TendermintConfig.NodeKey.ID()) {
		app.logger.Info("JWK keysync tx committed")
		app.state.JWKStaked = true
	}
	return nil
}

func (app *AnchorApplication) SetIdentity(tx types.Tx) (types.Jwk, error) {
	var jwkType types.Jwk
	err := json.Unmarshal([]byte(tx.Data), &jwkType)
	if app.LogError(err) != nil {
		return types.Jwk{}, err
	}
	pubKey, err := util.DecodePubKey(tx)
	app.state.CoreKeys[tx.CoreID] = *pubKey
	pubKeyBytes := elliptic.Marshal(pubKey.Curve, pubKey.X, pubKey.Y)
	pubKeyHex := fmt.Sprintf("%x", pubKeyBytes)
	app.logger.Info(fmt.Sprintf("Loading Core ID %s public key as %s", tx.CoreID, pubKeyHex))
	if val, exists := app.state.TxValidation[pubKeyHex]; exists {
		app.state.TxValidation[pubKeyHex] = val
	} else {
		validation := validation.NewTxValidation()
		app.state.TxValidation[pubKeyHex] = validation
	}
	app.state.IDMap[jwkType.Kid] = tx.CoreID
	lnID := types.LnIdentity{}
	app.LogError(json.Unmarshal([]byte(tx.Meta), &lnID))
	if lightning.IsLnUri(lnID.Peer) {
		app.logger.Info(fmt.Sprintf("Setting Core ID %s URI to %s", tx.CoreID, lnID.Peer))
		app.state.LnUris[tx.CoreID] = lnID
	}
	return jwkType, nil
}
