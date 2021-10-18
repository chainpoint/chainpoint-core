package abci

import (
	"crypto/elliptic"
	"fmt"
	"github.com/chainpoint/chainpoint-core/leader_election"
	"github.com/chainpoint/chainpoint-core/util"

	fee2 "github.com/chainpoint/chainpoint-core/fee"
	"strconv"
	"time"

	"github.com/chainpoint/chainpoint-core/validation"

	beacon "github.com/chainpoint/chainpoint-core/beacon"
)

const STATIC_FEE_AMT = 0 // 60k amounts to 240 sat/vbyte

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
			app.state.ID = app.ID
			app.logger.Info("Core ID set ", "ID", app.ID)
		}
		if app.state.Height != 0 && app.state.ChainSynced {
			validators, err := app.rpc.GetValidators(app.state.Height)
			if app.LogError(err) != nil {
				continue
			}
			cores := validation.GetLastNSubmitters(128, *app.state) //get Active cores on network
			totalStake := (int64(len(cores)) * app.config.StakePerCore)
			stakeAmt := totalStake / int64(len(validators.Validators)) //total stake divided by validators
			app.state.Validators = validators.Validators
			app.state.LnStakePerVal = stakeAmt
			app.state.LnStakePrice = totalStake //Total Stake Price includes the other 1/3 just in case
			//app.logger.Info(fmt.Sprintf("Stake Amt per Val: %d, total stake: %d", stakeAmt, app.state.LnStakePrice))
		}
		if app.state.TMState.SyncInfo.CatchingUp {
			app.state.ChainSynced = false
		} else {
			app.state.ChainSynced = true
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
	selfPubKey, _, _:= util.DecodeJWK(app.JWK)
	pubKey := app.state.CoreKeys[app.ID]
	pubKeyBytes := elliptic.Marshal(pubKey.Curve, pubKey.X, pubKey.Y)
	pubKeyHex := fmt.Sprintf("%x", pubKeyBytes)
	if selfPubKey != pubKeyHex {
		app.logger.Info(fmt.Sprintf("node ID has likely changed. %s != %s", selfPubKey, pubKeyHex))
		app.logger.Info("Restaking with new credentials")
		app.state.JWKStaked = false
	}

	for !app.state.JWKStaked {
		app.logger.Info("Beginning Lightning staking loop")
		time.Sleep(60 * time.Second) //ensure loop gives chain time to init and doesn't restart on error too fast
		if !app.state.ChainSynced || app.state.Height < 2 || app.ID == "" {
			app.logger.Info("Chain not synced, restarting staking loop...")
			continue
		}
		amValidator, err := leader_election.AmValidator(*app.state)
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
			app.logger.Debug(fmt.Sprintf("Failed to obtain DRAND beacon value of %s", chainpointFormat))
		}
	}
}

// FeeMonitor : elects a leader to poll and gossip Fee. Called every n minutes by ABCI.commit
func (app *AnchorApplication) FeeMonitor() {
	time.Sleep(15 * time.Second) //sleep after commit for a few seconds
	if app.state.Height > 2 && app.state.Height-app.state.LastBtcFeeHeight >= app.config.FeeInterval {
		if leader, leaders := leader_election.ElectValidatorAsLeader(1, []string{}, *app.state, app.config); leader {
			app.logger.Info(fmt.Sprintf("FEE: Elected as leader. Leaders: %v", leaders))
			var fee int64
			lndFee, _ := app.LnClient.GetLndFeeEstimate()
			app.LnClient.Logger.Info(fmt.Sprintf("FEE from LND: %d", lndFee))
			thirdPartyFee, _ := fee2.GetThirdPartyFeeEstimate()
			app.LnClient.Logger.Info(fmt.Sprintf("FEE from Third Party: %d", thirdPartyFee))
			fee = util.MaxInt64(lndFee, thirdPartyFee)
			fee = util.MaxInt64(fee, STATIC_FEE_AMT)
			app.logger.Info(fmt.Sprintf("Ln Wallet EstimateFEE: %v", fee))
			_, err := app.rpc.BroadcastTx("FEE", strconv.FormatInt(fee, 10), 2, time.Now().Unix(), app.ID, app.config.ECPrivateKey) // elect a leader to send a NIST tx
			if app.LogError(err) != nil {
				app.logger.Debug(fmt.Sprintf("Failed to gossip Fee value of %d", fee))
			}
		}
	}
}

