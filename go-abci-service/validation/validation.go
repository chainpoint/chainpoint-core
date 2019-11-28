package validation

import (
	"crypto/elliptic"
	"errors"
	"fmt"
	"strings"

	"github.com/chainpoint/chainpoint-core/go-abci-service/types"
	"github.com/chainpoint/chainpoint-core/go-abci-service/util"
)

//NewTxValidationMap : initialize record keeping for validations
func NewTxValidationMap() map[string]types.TxValidation {
	return make(map[string]types.TxValidation)
}

//NewTxValidation : initialize values for validation of tx
func NewTxValidation() types.TxValidation {
	permittedCalRate := types.RateLimit{
		AllowedRate: 70,
		PerBlocks:   60,
		LastCheck:   0,
		Bucket:      0.0,
	}
	permittedJWKRate := types.RateLimit{
		AllowedRate: 2,
		PerBlocks:   1440,
		LastCheck:   0,
		Bucket:      0.0,
	}
	permittedBtcRate := types.RateLimit{
		AllowedRate: 2,
		PerBlocks:   60,
		LastCheck:   0,
		Bucket:      0.0,
	}
	return types.TxValidation{
		JWKAllowedRate:  permittedJWKRate,
		CalAllowedRate:  permittedCalRate,
		BtcaAllowedRate: permittedBtcRate,
		BtccAllowedRate: permittedBtcRate,
		NISTAllowedRate: permittedCalRate,
	}
}

// RateLimitUpdate : simple token bucket rate limiter
func RateLimitUpdate(currHeight int64, limit *types.RateLimit) {
	delta := currHeight - limit.LastCheck
	limit.LastCheck = currHeight
	limit.Bucket += float32(delta) * (float32(limit.AllowedRate) / float32(limit.PerBlocks))
	if limit.Bucket > float32(limit.AllowedRate) {
		limit.Bucket = float32(limit.AllowedRate)
	}
}

// IsHabitualViolator : find out if the core has been violating rat elimits
func IsHabitualViolator(limit types.RateLimit) bool {
	return limit.Bucket < 1.0
}

// UpdateAcceptTx : Update successful acceptance of Tx for rate limiting
func UpdateAcceptTx(limit *types.RateLimit) {
	limit.Bucket -= 1.0
}

// IsValidBtcc : Check if BTCC tx corresponds to a previous BTC-A
func IsValidBtcc(tx types.Tx, state types.AnchorState) bool {
	meta := strings.Split(tx.Meta, "|") // first part of meta is core ID that issued TX, second part is BTC TX ID
	btcaTx, err := util.DecodeTx(state.LatestBtcaTx)
	return err != nil && len(meta) > 0 && meta[0] == btcaTx.CoreID
}

// GetPubKeyHex : Gets the public key of a core, given the CoreID string
func GetPubKeyHex(coreID string, state types.AnchorState) string {
	if _, exists := state.CoreKeys[coreID]; !exists {
		return ""
	}
	// Obtain pubkey in hex format from our record of cores, keyed by coreID
	pubKey := state.CoreKeys[coreID]
	pubKeyBytes := elliptic.Marshal(pubKey.Curve, pubKey.X, pubKey.Y)
	pubKeyHex := fmt.Sprintf("%x", pubKeyBytes)
	return pubKeyHex
}

// GetValidationRecord : Gets a validation record for a Core, given the CoreID
func GetValidationRecord(coreID string, state types.AnchorState) (string, types.TxValidation, error) {
	pubKeyHex := GetPubKeyHex(coreID, state)
	if pubKeyHex == "" {
		return "", NewTxValidation(), errors.New("no pubkey for core id")
	}

	// Find the transmitting core's validation record from the pub key
	var validationRecord types.TxValidation
	if _, exists := state.TxValidation[pubKeyHex]; !exists {
		validationRecord = NewTxValidation()
	}
	validationRecord = state.TxValidation[pubKeyHex]
	return pubKeyHex, validationRecord, nil
}

// SetValidationRecord : sets a validation record on the state db pointer
func SetValidationRecord(coreID string, validationRecord types.TxValidation, state *types.AnchorState) error {
	pubKeyHex := GetPubKeyHex(coreID, *state)
	if pubKeyHex == "" {
		return errors.New("no pubkey for core id")
	}
	state.TxValidation[pubKeyHex] = validationRecord
	return nil
}

// IncrementSuccessAnchor : increments the successful anchor record, given a coreID string and a pointer to state db
func IncrementSuccessAnchor(coreID string, state *types.AnchorState) error {
	_, validationRecord, err := GetValidationRecord(coreID, *state)
	if err != nil {
		return err
	}
	validationRecord.ConfirmedAnchors++
	err = SetValidationRecord(coreID, validationRecord, state)
	return err
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

	pubKeyHex, validationRecord, err := GetValidationRecord(coreID, *state)

	validated := true

	switch string(txType) {
	case "CAL":
		RateLimitUpdate(state.Height, &validationRecord.CalAllowedRate)
		if IsHabitualViolator(validationRecord.CalAllowedRate) {
			validated = false
		} else {
			UpdateAcceptTx(&validationRecord.CalAllowedRate)
			validationRecord.LastCalTxHeight = state.Height
		}
		break
	case "BTC-A":
		RateLimitUpdate(state.Height, &validationRecord.BtcaAllowedRate)
		if IsHabitualViolator(validationRecord.BtcaAllowedRate) || (state.Height-state.LatestBtcaHeight < 61) {
			validated = false
		} else {
			UpdateAcceptTx(&validationRecord.BtcaAllowedRate)
			validationRecord.LastBtcaTxHeight = state.Height
		}
		break
	case "BTC-C":
		RateLimitUpdate(state.Height, &validationRecord.BtccAllowedRate)
		if IsHabitualViolator(validationRecord.BtccAllowedRate) || (state.Height-state.LatestBtccHeight < 61) /*|| !IsValidBtcc(tx, *state)*/ {
			validated = false
		} else {
			UpdateAcceptTx(&validationRecord.BtccAllowedRate)
			validationRecord.LastBtccTxHeight = state.Height
		}
		break
	case "NIST":
		timeRecord := util.GetNISTTimestamp(tx.Data)
		lastTimeRecord := util.GetNISTTimestamp(state.LatestNistRecord)
		timeDiff := timeRecord - lastTimeRecord
		RateLimitUpdate(state.Height, &validationRecord.NISTAllowedRate)
		if IsHabitualViolator(validationRecord.NISTAllowedRate) || timeDiff == 0 || timeDiff < 0 {
			validated = false
		} else {
			UpdateAcceptTx(&validationRecord.NISTAllowedRate)
			validationRecord.LastNISTTxHeight = state.Height
		}
		break
	case "JWT":
		RateLimitUpdate(state.Height, &validationRecord.JWKAllowedRate)
		if IsHabitualViolator(validationRecord.JWKAllowedRate) {
		} else {
			UpdateAcceptTx(&validationRecord.JWKAllowedRate)
			validationRecord.LastJWKTxHeight = state.Height
		}
	}
	fmt.Printf("Tx Validation: %#v\nTx:%#v\nValidated:%t", validationRecord, tx, validated)
	state.TxValidation[pubKeyHex] = validationRecord
	return tx, validated, err
}
