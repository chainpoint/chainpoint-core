package validation

import (
	"crypto/elliptic"
	"errors"
	"fmt"

	"github.com/chainpoint/chainpoint-core/go-abci-service/types"
	"github.com/chainpoint/chainpoint-core/go-abci-service/util"
)

//NewTxValidationMap : initialize record keeping for validations
func NewTxValidationMap() map[string]types.TxValidation {
	return make(map[string]types.TxValidation)
}

//NewTxValidation : initialize values for validation of tx
func NewTxValidation() types.TxValidation {
	permittedViolations := types.ViolationRateLimit{
		AllowedRate: 2,
		PerBlocks:   60,
		LastCheck:   0,
		Bucket:      0.0,
	}
	return types.TxValidation{
		JWKViolationRate:  permittedViolations,
		CalViolationRate:  permittedViolations,
		BtcaViolationRate: permittedViolations,
		BtccViolationRate: permittedViolations,
		NISTViolationRate: permittedViolations,
	}
}

func ViolationRateLimitUpdate(currHeight int64, limit *types.ViolationRateLimit) {
	delta := currHeight - limit.LastCheck
	limit.LastCheck = currHeight
	limit.Bucket += float32(delta) * (float32(limit.AllowedRate) / float32(limit.PerBlocks))
	if limit.Bucket > float32(limit.AllowedRate) {
		limit.Bucket = float32(limit.AllowedRate)
	}
}

func NonViolationRateLimitUpdate(limit *types.ViolationRateLimit) {
	limit.Bucket -= 1.0
}

func Validate(incoming []byte, state *types.AnchorState) (types.Tx, bool, error) {
	tx, err := util.DecodeTxAndVerifySig(incoming, state.CoreKeys)
	if err != nil {
		return tx, false, nil
	}
	txType := string(tx.TxType)
	coreID := string(tx.CoreID)

	// Allow a Core to transmit JWK for the first time
	if _, exists := state.CoreKeys[coreID]; !exists {
		if txType == "JWK" {
			return tx, true, nil
		} else {
			return tx, false, errors.New("Transmitting Core has not yet declared its keys")
		}
	}

	// Obtain pubkey in hex format from our record of cores, keyed by coreID
	pubKey := state.CoreKeys[coreID]
	pubKeyBytes := elliptic.Marshal(pubKey.Curve, pubKey.X, pubKey.Y)
	pubKeyHex := fmt.Sprintf("%x", pubKeyBytes)

	// Find the transmitting core's validation record from the pub key
	var validationRecord types.TxValidation
	if _, exists := state.TxValidation[pubKeyHex]; !exists {
		validationRecord = NewTxValidation()
	}
	validationRecord = state.TxValidation[pubKeyHex]

	validated := false // default to invalid tx

	switch string(txType) {
	case "CAL":
		if (state.Height - validationRecord.LastCalTxHeight) < 1 {
			ViolationRateLimitUpdate(state.Height, &validationRecord.CalViolationRate)
		} else {
			validated = true
			NonViolationRateLimitUpdate(&validationRecord.CalViolationRate)
			validationRecord.LastCalTxHeight = state.Height
		}
		break
	case "BTC-A":
		if ((state.Height - validationRecord.LastBtcaTxHeight) < 61) || (state.Height-state.LatestBtcaHeight < 61) {
			ViolationRateLimitUpdate(state.Height, &validationRecord.BtcaViolationRate)
		} else {
			validated = true
			NonViolationRateLimitUpdate(&validationRecord.BtcaViolationRate)
			validationRecord.LastBtcaTxHeight = state.Height
		}
		break
	case "BTC-C":
		if ((state.Height - validationRecord.LastBtccTxHeight) < 61) || (state.Height-state.LatestBtccHeight < 61) {
			ViolationRateLimitUpdate(state.Height, &validationRecord.BtccViolationRate)
		} else {
			validated = true
			NonViolationRateLimitUpdate(&validationRecord.BtccViolationRate)
			validationRecord.LastBtccTxHeight = state.Height
		}
		break
	case "NIST":
		timeRecord := util.GetNISTTimestamp(tx.Data)
		lastTimeRecord := util.GetNISTTimestamp(state.LatestNistRecord)
		timeDiff := timeRecord - lastTimeRecord
		if timeDiff == 0 || timeDiff < 0 {
			ViolationRateLimitUpdate(state.Height, &validationRecord.NISTViolationRate)
		} else {
			validated = true
			NonViolationRateLimitUpdate(&validationRecord.NISTViolationRate)
			validationRecord.LastNISTTxHeight = state.Height
		}
		break
	case "JWT":
		if (state.Height - validationRecord.LastJWKTxHeight) < 1440 {
			ViolationRateLimitUpdate(state.Height, &validationRecord.JWKViolationRate)
		} else {
			validated = true
			NonViolationRateLimitUpdate(&validationRecord.JWKViolationRate)
			validationRecord.LastJWKTxHeight = state.Height
		}
	}
	state.TxValidation[pubKeyHex] = validationRecord
	return tx, validated, nil
}
