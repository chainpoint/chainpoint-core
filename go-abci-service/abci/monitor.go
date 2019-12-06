package abci

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/chainpoint/chainpoint-core/go-abci-service/lightning"

	"github.com/lestrrat-go/jwx/jwk"

	"github.com/chainpoint/chainpoint-core/go-abci-service/validation"

	"github.com/chainpoint/chainpoint-core/go-abci-service/util"
	"github.com/go-redis/redis"

	beacon "github.com/chainpoint/go-nist-beacon"

	"github.com/chainpoint/chainpoint-core/go-abci-service/types"
)

//SyncMonitor : turns off anchoring if we're not synced. Not cron scheduled since we need it to start immediately.
func (app *AnchorApplication) SyncMonitor() {
	for {
		time.Sleep(30 * time.Second)
		status, err := app.rpc.GetStatus()
		if app.LogError(err) != nil {
			time.Sleep(5 * time.Second)
			continue
		}
		if app.ID == "" {
			app.ID = string(status.ValidatorInfo.Address.String())
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
	for app.JWKSent != true {
		time.Sleep(60 * time.Second)
		if _, exists := app.state.CoreKeys[app.ID]; exists {
			return
		}
		if !app.state.ChainSynced {
			continue
		}
		validators, err := app.rpc.GetValidators(app.state.Height)
		if app.LogError(err) != nil {
			continue
		}
		for _, validator := range validators.Validators {
			valID := validator.Address.String()
			if lnUri, exists := app.state.LnUris[valID]; exists {
				app.logger.Info(fmt.Sprintf("Adding Lightning Peer %s...", lnUri))
				if app.LogError(app.lnClient.AddPeer(lnUri)) == nil {
					app.logger.Info(fmt.Sprintf("Adding Lightning Channel for Peer %s...", lnUri))
					_, err := app.lnClient.CreateChannel(lnUri)
					app.LogError(err)
				}
			}
		}
		jwk, err := jwk.New(app.config.ECPrivateKey.Public())
		if app.LogError(err) != nil {
			continue
		}
		jwkJson, err := json.MarshalIndent(jwk, "", "  ")
		if app.LogError(err) != nil {
			continue
		}
		resp, err := app.lnClient.GetInfo()
		if app.LogError(err) != nil || len(resp.Uris) == 0 {
			continue
		}
		uri := resp.Uris[0]
		_, err = app.rpc.BroadcastTxWithMeta("JWK", string(jwkJson), 2, time.Now().Unix(), app.ID, uri, &app.config.ECPrivateKey)
		if app.LogError(err) != nil {
			continue
		} else {
			return
		}
	}
	panic(errors.New("Cannot broadcast Core public key"))
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
		app.logger.Info(fmt.Sprintf("Setting JWK for Core %s: %s", coreID, b64Str))
		app.state.CoreKeys[coreID] = pubKey
		app.state.TxValidation[fmt.Sprintf("%x", pubKeyBytes)] = validation.NewTxValidation()
	}
	return nil
}

//SaveIdentity : save the JWT value retrieved
func (app *AnchorApplication) SaveIdentity(tx types.Tx) error {
	var jwkType types.Jwk
	json.Unmarshal([]byte(tx.Data), &jwkType)
	key := fmt.Sprintf("CorePublicKey:%s", jwkType.Kid)
	if jwkType.Kid == app.JWK.Kid {
		app.logger.Info("JWK keysync tx committed")
		app.JWKSent = true
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
	if lightning.IsLnUri(tx.Meta) {
		app.state.LnUris[tx.CoreID] = tx.Meta
	}
	return nil
}
