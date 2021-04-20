package abci

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/chainpoint/chainpoint-core/go-abci-service/lightning"
	"github.com/chainpoint/chainpoint-core/go-abci-service/types"
	"github.com/chainpoint/chainpoint-core/go-abci-service/util"
	"github.com/chainpoint/chainpoint-core/go-abci-service/validation"
	"github.com/go-redis/redis"
	"strings"
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
		RequiredChanAmt: app.LnClient.LocalSats,
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
	var cursor uint64
	var idKeys []string
	for {
		var keys []string
		var err error
		keys, cursor, err = app.RedisClient.Scan(cursor, "CoreID:*", 10).Result()
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
		b64Str, err := app.RedisClient.Get(k).Result()
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
		chanExists, err := app.LnClient.ChannelExists(lnID.Peer, app.LnClient.LocalSats)
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
		util.LoggerError(app.logger, app.RedisClient.Set("CoreID:"+tx.CoreID, base64.StdEncoding.EncodeToString(pubKeyBytes), 0).Err())
	}
	value, err := app.RedisClient.Get(key).Result()
	if app.LogError(err) == redis.Nil || value != string(jsonJwk) {
		err = app.RedisClient.Set(key, value, 0).Err()
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
		app.logger.Info(fmt.Sprintf("Setting Core ID %s URI to %s", tx.CoreID, lnID.Peer))
		app.state.LnUris[tx.CoreID] = lnID
	}
	if jwkType.Kid != "" && app.JWK.Kid != "" && jwkType.Kid == app.JWK.Kid {
		app.logger.Info("JWK keysync tx committed")
		app.state.JWKStaked = true
	}
	return nil
}
