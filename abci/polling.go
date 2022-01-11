package abci

import (
	"encoding/base64"
	"fmt"
	"github.com/chainpoint/chainpoint-core/beacon"
	fee2 "github.com/chainpoint/chainpoint-core/fee"
	"github.com/chainpoint/chainpoint-core/util"
	"time"
)

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
			key, err := app.config.TendermintConfig.FilePV.GetPubKey()
			if err != nil {
				app.logger.Info("Core Tendermint Publickey set", "Key", base64.StdEncoding.EncodeToString(key.Bytes()))
			}
		}
		if app.state.TMState.SyncInfo.CatchingUp {
			app.state.ChainSynced = false
		} else {
			app.state.ChainSynced = true
		}
		if app.state.ChainSynced && app.state.Height > 2 && app.ID != "" {
			app.state.AppReady = true
		}
	}
}

// BeaconMonitor : elects a leader to poll DRAND. Called every minute by ABCI.commit
func (app *AnchorApplication) BeaconMonitor() {
	time.Sleep(30 * time.Second) //sleep after commit for a few seconds
	if app.state.AppReady {
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
	if app.state.AppReady && app.state.Height-app.state.LastBtcFeeHeight >= app.config.FeeInterval {
		var fee int64
		lndFee, _ := app.LnClient.GetLndFeeEstimate()
		app.LnClient.Logger.Info(fmt.Sprintf("FEE from LND: %d", lndFee))
		thirdPartyFee, _ := fee2.GetThirdPartyFeeEstimate()
		app.LnClient.Logger.Info(fmt.Sprintf("FEE from Third Party: %d", thirdPartyFee))
		fee = util.MaxInt64(lndFee, thirdPartyFee)
		fee = util.MaxInt64(fee, STATIC_FEE_AMT)
		app.logger.Info(fmt.Sprintf("Ln Wallet EstimateFEE: %v", fee))
		app.state.LatestBtcFee = fee
		app.state.LastBtcFeeHeight = app.state.Height
		app.LnClient.LastFee = app.state.LatestBtcFee
	}
}
