package abci

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	beacon "github.com/chainpoint/go-nist-beacon"

	"github.com/chainpoint/chainpoint-core/go-abci-service/types"
)

//SyncMonitor : turns off anchoring if we're not synced. Not cron scheduled since we need it to start immediately.
func (app *AnchorApplication) SyncMonitor() {
	for {
		status, err := app.rpc.GetStatus()
		if app.LogError(err) != nil {
			time.Sleep(5 * time.Second)
			continue
		}
		if app.ID == "" {
			app.ID = string(status.NodeInfo.ID())
		}
		if status.SyncInfo.CatchingUp {
			app.state.ChainSynced = false
		} else {
			app.state.ChainSynced = true
		}
		time.Sleep(30 * time.Second)
	}
}

//KeyMonitor : updates active ECDSA public keys from all accessible peers
func (app *AnchorApplication) KeyMonitor() {
	selfStatusURL := fmt.Sprintf("%s/status", app.config.APIURI)
	response, err := http.Get(selfStatusURL)
	if app.LogError(err) != nil {
		return
	}
	contents, err := ioutil.ReadAll(response.Body)
	if app.LogError(err) != nil {
		return
	}
	var apiStatus types.CoreAPIStatus
	err = json.Unmarshal(contents, &apiStatus)
	if app.LogError(err) != nil {
		return
	}
	app.JWK = apiStatus.Jwk
	jwkJson, err := json.Marshal(apiStatus.Jwk)
	if app.LogError(err) != nil {
		return
	}
	_, err = app.rpc.BroadcastTx("JWK", string(jwkJson), 2, time.Now().Unix(), app.ID, &app.config.ECPrivateKey)
	if app.LogError(err) != nil {
		return
	}
}

// NistBeaconMonitor : elects a leader to poll and gossip NIST. Called every minute by ABCI.commit
func (app *AnchorApplication) NistBeaconMonitor() {
	time.Sleep(15 * time.Second) //sleep after commit for a few seconds
	if leader, leaders := app.ElectValidator(1); leader && app.state.ChainSynced {
		app.logger.Info(fmt.Sprintf("NIST: Elected as leader. Leaders: %v", leaders))
		nistRecord, err := beacon.LastRecord()
		if app.LogError(err) != nil {
			app.logger.Error("Unable to obtain new NIST beacon value")
			return
		}
		_, err = app.rpc.BroadcastTx("NIST", nistRecord.ChainpointFormat(), 2, time.Now().Unix(), app.ID, &app.config.ECPrivateKey) // elect a leader to send a NIST tx
		if app.LogError(err) != nil {
			app.logger.Debug(fmt.Sprintf("Failed to gossip NIST beacon value of %s", nistRecord.ChainpointFormat()))
		}
	}
}

//MintMonitor : efficiently monitor for new minting and gossip that block to other cores
func (app *AnchorApplication) MintMonitor() {
	if leader, _ := app.ElectValidator(1); leader && app.state.ChainSynced {
		lastNodeMintedAt, err := app.ethClient.GetNodeLastMintedAt()
		if app.LogError(err) != nil {
			app.logger.Error("Unable to obtain new NodeLastMintedAt value")
			return
		}
		if lastNodeMintedAt.Int64() != 0 && lastNodeMintedAt.Int64() >= app.state.LastNodeMintedAtBlock+MINT_EPOCH {
			app.logger.Info("Mint success, sending Node MINT tx")
			_, err = app.rpc.BroadcastTx("NODE-MINT", strconv.FormatInt(lastNodeMintedAt.Int64(), 10), 2, time.Now().Unix(), app.ID, &app.config.ECPrivateKey) // elect a leader to send a NIST tx
			if err != nil {
				app.logger.Debug("Failed to gossip Node MINT for LastNodeMintedAtBlock gossip")
			}
		}
		lastCoreMintedAt, err := app.ethClient.GetCoreLastMintedAt()
		if app.LogError(err) != nil {
			app.logger.Error("Unable to obtain new CoreLastMintedAt value")
			return
		}
		if lastCoreMintedAt.Int64() != 0 && lastCoreMintedAt.Int64() >= app.state.LastCoreMintedAtBlock+MINT_EPOCH {
			app.logger.Info("Mint success, sending Core MINT tx")
			_, err = app.rpc.BroadcastTx("CORE-MINT", strconv.FormatInt(lastCoreMintedAt.Int64(), 10), 2, time.Now().Unix(), app.ID, &app.config.ECPrivateKey) // elect a leader to send a NIST tx
			if err != nil {
				app.logger.Debug("Failed to gossip Core MINT for LastNodeMintedAtBlock gossip")
			}
		}
	}
}
