package abci

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/chainpoint/chainpoint-core/leaderelection"
	"github.com/tendermint/tendermint/abci/example/code"
	"github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"strconv"
	"strings"
	"time"
)

// constant prefix for a validator transaction
const (
	ValidatorSetChangePrefix string = "val:"
)

// MakeValSetChangeTx : TODO: describe this
func MakeValSetChangeTx(pubkey types.PubKey, power int64) []byte {
	return []byte(fmt.Sprintf("val:%X/%d", pubkey.Data, power))
}

func isValidatorTx(tx []byte) bool {
	return strings.HasPrefix(string(tx), ValidatorSetChangePrefix)
}

func (app *AnchorApplication) CheckVoteValidator() {
	var validatorValue string
	if app.config.ProposedVal != "" {
		err, _, _, _, blockHeight := ValidateValidatorTx(app.config.ProposedVal)
		if app.LogError(err) == nil && blockHeight == app.state.Height {
			app.PendingValidator = validatorValue
			amLeader, leaderId := leaderelection.ElectValidatorAsLeader(1, []string{}, *app.state, app.config)
			app.logger.Info(fmt.Sprintf("Validator Promotion: %s was elected to submit VAL tx", leaderId))
			if amLeader {
				go func() {
					time.Sleep(1 * time.Minute)
					app.rpc.BroadcastTx("VAL", validatorValue, 2, time.Now().Unix(), app.ID, app.config.ECPrivateKey)
				}()
			}
		}
	}
}

func ValidateValidatorTx(val string) (err error, id string, pubkey []byte, power int64, blockHeight int64) {
	//get the pubkey and power
	var idS, pubkeyS, powerS, blockS string
	var blockH int64
	idPubKeyAndPower := strings.Split(val, "!")
	if len(idPubKeyAndPower) == 4 {
		idS, pubkeyS, powerS, blockS = idPubKeyAndPower[0], idPubKeyAndPower[1], idPubKeyAndPower[2], idPubKeyAndPower[3]
	} else if len(idPubKeyAndPower) == 3 {
		idS, pubkeyS, powerS = idPubKeyAndPower[0], idPubKeyAndPower[1], idPubKeyAndPower[2]
	} else if len(idPubKeyAndPower) == 2 {
		pubkeyS, powerS = idPubKeyAndPower[1], idPubKeyAndPower[2]
	} else {
		return errors.New("Expected 'val:id!pubkey!power'"), "", []byte{}, 0, 0

	}

	// decode the pubkey
	id = strings.TrimPrefix(idS, "val:")
	pubkey, err = base64.StdEncoding.DecodeString(pubkeyS)
	if err != nil {
		return errors.New("pubkey is invalid base64"), "", []byte{}, 0, 0

	}
	// decode the power
	power, err = strconv.ParseInt(powerS, 10, 64)
	if err != nil {
		return errors.New("power isn't an integer"), "", []byte{}, 0, 0
	}
	if blockS != "" {
		blockH, err = strconv.ParseInt(blockS, 10, 64)
		if err != nil {
			return errors.New("block height isn't an integer"), "", []byte{}, 0, 0
		}
	}

	return nil, id, pubkey, power, blockH
}

// format is "id!pubkey!power"
// pubkey is a base64-encoded 32-byte ed25519 key
func (app *AnchorApplication) execValidatorTx(tx []byte) types.ResponseDeliverTx {
	err, _, pubkey, power, _ := ValidateValidatorTx(string(tx))
	if err != nil {
		return types.ResponseDeliverTx{
			Code: code.CodeTypeEncodingError,
			Log:  fmt.Sprintf(err.Error()),
		}
	}

	// update
	return app.updateValidator(types.Ed25519ValidatorUpdate(pubkey, power))
}

// add, update, or remove a validator
func (app *AnchorApplication) updateValidator(v types.ValidatorUpdate) types.ResponseDeliverTx {
	key := []byte("val:" + string(v.PubKey.Data))

	pubkey := ed25519.PubKeyEd25519{}
	copy(pubkey[:], v.PubKey.Data)

	if v.Power == 0 {
		// remove validator
		hasKey, err := app.Db.Has(key)
		if err != nil {
			panic(err)
		}
		if !hasKey {
			pubStr := base64.StdEncoding.EncodeToString(v.PubKey.Data)
			app.logger.Info(fmt.Sprintf("Cannot remove non-existent validator %s", pubStr))
			return types.ResponseDeliverTx{
				Code: code.CodeTypeUnauthorized,
				Log:  fmt.Sprintf("Cannot remove non-existent validator %s", pubStr)}
		}
		app.Db.Delete(key)
		delete(app.valAddrToPubKeyMap, string(pubkey.Address()))
	} else {
		// add or update validator
		value := bytes.NewBuffer(make([]byte, 0))
		if err := types.WriteMessage(&v, value); err != nil {
			app.logger.Info(fmt.Sprintf("Error encoding validator: %v", err))
			return types.ResponseDeliverTx{
				Code: code.CodeTypeEncodingError,
				Log:  fmt.Sprintf("Error encoding validator: %v", err)}
		}
		app.Db.Set(key, value.Bytes())
		app.valAddrToPubKeyMap[string(pubkey.Address())] = v.PubKey
	}

	// we only update the changes array if we successfully updated the tree
	app.ValUpdates = append(app.ValUpdates, v)
	app.logger.Info(fmt.Sprintf("Val Updates: %v", app.ValUpdates))
	return types.ResponseDeliverTx{Code: code.CodeTypeOK}
}
