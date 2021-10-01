package abci

import (
	"crypto/elliptic"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/chainpoint/chainpoint-core/leader_election"
	"github.com/chainpoint/chainpoint-core/lightning"
	"github.com/chainpoint/chainpoint-core/types"
	"github.com/chainpoint/chainpoint-core/util"
	"github.com/chainpoint/chainpoint-core/validation"
	"time"
)

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
	_, err = app.rpc.BroadcastTxWithMeta("JWK", string(jwkJson), 2, time.Now().Unix(), app.ID, string(lnIDBytes), &app.config.ECPrivateKey)
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
		txs, err := app.rpc.GetAllJWKs()
		if err == nil {
			for _, tx := range txs {
				var jwkType types.Jwk
				err := json.Unmarshal([]byte(tx.Data), &jwkType)
				if app.LogError(err) != nil {
					continue
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
			}
			break;
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
	var jwkType types.Jwk
	json.Unmarshal([]byte(tx.Data), &jwkType)
	app.logger.Info("JWK kid", "JWK Tx kid", jwkType.Kid, "app JWK kid", app.JWK.Kid)
	pubKey, err := util.DecodePubKey(tx)
	var pubKeyBytes []byte
	if app.LogError(err) == nil {
		app.state.CoreKeys[tx.CoreID] = *pubKey
		pubKeyBytes = elliptic.Marshal(pubKey.Curve, pubKey.X, pubKey.Y)
		util.LoggerError(app.logger, app.Cache.Add("CoreIDs", tx.CoreID))
		util.LoggerError(app.logger, app.Cache.Set("CoreID:" + tx.CoreID, base64.StdEncoding.EncodeToString(pubKeyBytes)))
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
		app.logger.Info(fmt.Sprintf("Setting Core ID %s URI to %s", tx.CoreID, lnID.Peer))
		app.state.LnUris[tx.CoreID] = lnID
	}
	if jwkType.Kid != "" && app.JWK.Kid != "" && jwkType.Kid == app.JWK.Kid {
		app.logger.Info("JWK keysync tx committed")
		app.state.JWKStaked = true
	}
	return nil
}
