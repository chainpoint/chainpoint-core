package abci

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

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
			app.ID = string(status.NodeInfo.ID())
		}
		if status.SyncInfo.CatchingUp {
			app.state.ChainSynced = false
		} else {
			app.state.ChainSynced = true
		}
	}
}

//KeyMonitor : updates active ECDSA public keys from all accessible peers
func (app *AnchorApplication) KeyMonitor() {
	for i := 0; i < 2; i++ {
		time.Sleep(30 * time.Second)
		selfStatusURL := fmt.Sprintf("%s/status", app.config.APIURI)
		response, err := http.Get(selfStatusURL)
		if app.LogError(err) != nil {
			continue
		}
		contents, err := ioutil.ReadAll(response.Body)
		if app.LogError(err) != nil {
			continue
		}
		var apiStatus types.CoreAPIStatus
		err = json.Unmarshal(contents, &apiStatus)
		if app.LogError(err) != nil {
			continue
		}
		app.JWK = apiStatus.Jwk
		jwkJson, err := json.Marshal(apiStatus.Jwk)
		if app.LogError(err) != nil {
			continue
		}
		_, err = app.rpc.BroadcastTxCommit("JWK", string(jwkJson), 2, time.Now().Unix(), app.ID, &app.config.ECPrivateKey)
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

//LoadJWK : load public keys derived from JWTs from redis
func (app *AnchorApplication) LoadJWK() error {
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
		app.CoreKeys[coreID] = pubKey
	}
	return nil
}

//SaveJWK : save the JWT value retrieved
func (app *AnchorApplication) SaveJWK(tx types.Tx) error {
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
	if app.LogError(err) == nil {
		app.CoreKeys[tx.CoreID] = *pubKey
		pubKeyBytes := elliptic.Marshal(pubKey.Curve, pubKey.X, pubKey.Y)
		util.LoggerError(app.logger, app.redisClient.Set("CoreID:"+tx.CoreID, base64.StdEncoding.EncodeToString(pubKeyBytes), 0).Err())
	}
	value, err := app.redisClient.Get(key).Result()
	if err == redis.Nil || value != string(jsonJwk) {
		err = app.redisClient.Set(key, value, 0).Err()
		if app.LogError(err) != nil {
			return err
		}
		app.logger.Info(fmt.Sprintf("Set JWK cache for kid %s", jwkType.Kid))
	}
	return nil
}
