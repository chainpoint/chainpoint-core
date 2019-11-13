package validation

import (
	"crypto/elliptic"
	"fmt"

	"github.com/chainpoint/chainpoint-core/go-abci-service/types"
	"github.com/chainpoint/chainpoint-core/go-abci-service/util"
)

// Holds state for validating Transactions
type TxValidation struct {
	LastJWKTx           types.Tx
	LastJWKTxBlock      int64
	CurrJWKViolationExp int64
	JWKViolations       int

	LastCalTx           types.Tx
	LastCalTxBlock      int64
	CurrCalViolationExp int64
	CalViolations       int

	LastBtcaTx           types.Tx
	LastBtcaTxBlock      int64
	CurrBtcaViolationExp int64
	BtcaViolations       int

	LastBtccTx           types.Tx
	LastBtccTxBlock      int64
	CurrBtccViolationExp int64
	BtccViolations       int

	LastNISTTx           types.Tx
	LastNISTTxBlock      int64
	CurrNISTViolationExp int64
	NISTViolations       int
}

func NewTxValidatorMap() map[string]TxValidation {
	return make(map[string]TxValidation)
}

func Validate(incoming []byte, state *types.AnchorState) (types.Tx, bool, error) {
	tx, err := util.DecodeVerifyTx(incoming, state.CoreKeys)
	if err != nil {
		return tx, false, nil
	}
	txType := string(tx.TxType)
	coreID := string(tx.CoreID)
	pubKey := state.CoreKeys[coreID]
	pubKeyBytes := elliptic.Marshal(pubKey.Curve, pubKey.X, pubKey.Y)
	pubKeyHex := fmt.Sprintf("%x", pubKeyBytes)
	validationRecord := state.TxValidation[pubKeyHex]
	validated := false
	switch string(txType) {
	case "CAL":
		if (state.Height - validationRecord.LastCalTxBlock) < 1 {
			validationRecord.CalViolations++
		} else {
			validated = true
			validationRecord.LastCalTx = tx
			validationRecord.LastCalTxBlock = state.Height
		}
		break
	case "BTC-A":
		if (state.Height - validationRecord.LastBtcaTxBlock) < 60 {
			validationRecord.BtcaViolations++
		} else {
			validated = true
			validationRecord.LastBtcaTx = tx
			validationRecord.LastBtcaTxBlock = state.Height
		}
		break
	case "BTC-C":
		if (state.Height - validationRecord.LastBtccTxBlock) < 60 {
			validationRecord.BtccViolations++
		} else {
			validated = true
			validationRecord.LastBtccTx = tx
			validationRecord.LastBtccTxBlock = state.Height
		}
		break
	case "NIST":
		timeRecord := util.GetNISTTimestamp(tx.Data)
		lastTimeRecord := util.GetNISTTimestamp(state.LatestNistRecord)
		timeDiff := timeRecord - lastTimeRecord
		if timeDiff == 0 || timeDiff < 0 {
			validationRecord.NISTViolations++
		} else {
			validated = true
			validationRecord.LastNISTTx = tx
			validationRecord.LastNISTTxBlock = state.Height
		}
		break
	case "JWT":
	}
	state.TxValidation[pubKeyHex] = validationRecord
	return tx, validated, nil
}
