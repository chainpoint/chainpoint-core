package abci

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/tendermint/tendermint/abci/example/code"
	"github.com/tendermint/tendermint/abci/types"
	cmn "github.com/tendermint/tendermint/libs/common"
	dbm "github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/version"
)

const (
	ValidatorSetChangePrefix string = "val:"
)

var (
	stateKey        = []byte("stateKey")
	ProtocolVersion version.Protocol = 0x1
)

func loadState(db dbm.DB) State {
	stateBytes := db.Get(stateKey)
	var state State
	if len(stateBytes) != 0 {
		err := json.Unmarshal(stateBytes, &state)
		if err != nil {
			panic(err)
		}
	}
	state.db = db
	return state
}

func saveState(state State) {
	stateBytes, err := json.Marshal(state)
	if err != nil {
		panic(err)
	}
	state.db.Set(stateKey, stateBytes)
}

//---------------------------------------------------

var _ types.Application = (*AnchorApplication)(nil)

type AnchorApplication struct {
	types.BaseApplication
	ValUpdates    []types.ValidatorUpdate
	state         State
}

func NewAnchorApplication() *AnchorApplication {
	name := "anchor"
	db, err := dbm.NewGoLevelDB(name, "/tendermint/data")
	if err != nil {
		panic(err)
	}

	state := loadState(db)
	return &AnchorApplication{state: state}
}

func (app *AnchorApplication) SetOption(req types.RequestSetOption) (res types.ResponseSetOption) {
	return
}

// Save the validators in the merkle tree
func (app *AnchorApplication) InitChain(req types.RequestInitChain) types.ResponseInitChain {
	for _, v := range req.Validators {
		r := app.updateValidator(v, []cmn.KVPair{})
		if r.IsErr() {
			fmt.Println(r)
		}
	}
	return types.ResponseInitChain{}
}

func (app *AnchorApplication) Info(req types.RequestInfo) (resInfo types.ResponseInfo) {
	return types.ResponseInfo{
		Data: "",
		Version:    version.ABCIVersion,
		AppVersion: ProtocolVersion.Uint64(),
	}
}

// tx is url encoded json
func (app *AnchorApplication) DeliverTx(tx []byte) types.ResponseDeliverTx {
	tags := []cmn.KVPair{}
	resp := types.ResponseDeliverTx{Code: code.CodeTypeUnauthorized, Tags: tags}
	return resp
}

func (app *AnchorApplication) CheckTx(tx []byte) types.ResponseCheckTx {
	return types.ResponseCheckTx{Code: code.CodeTypeOK, GasWanted: 1}
}

func (app *AnchorApplication) BeginBlock(req types.RequestBeginBlock) types.ResponseBeginBlock {
	app.ValUpdates = make([]types.ValidatorUpdate, 0)
	return types.ResponseBeginBlock{}
}

func (app *AnchorApplication) EndBlock(req types.RequestEndBlock) types.ResponseEndBlock {
	return types.ResponseEndBlock{ValidatorUpdates: app.ValUpdates}
}

func (app *AnchorApplication) Commit() types.ResponseCommit {
	appHash := make([]byte, 8)
	binary.PutVarint(appHash, app.state.Size)
	app.state.AppHash = appHash
	app.state.Height += 1
	saveState(app.state)
	return types.ResponseCommit{Data: appHash}
}

func (app *AnchorApplication) Query(reqQuery types.RequestQuery) (resQuery types.ResponseQuery) {
	return
}

func (app *AnchorApplication) Validators() (validators []types.ValidatorUpdate) {
	itr := app.state.db.Iterator(nil, nil)
	for ; itr.Valid(); itr.Next() {
		if isValidatorTx(itr.Key()) {
			validator := new(types.ValidatorUpdate)
			err := types.ReadMessage(bytes.NewBuffer(itr.Value()), validator)
			if err != nil {
				panic(err)
			}
			validators = append(validators, *validator)
		}
	}
	return
}

func MakeValSetChangeTx(pubkey types.PubKey, power int64) []byte {
	return []byte(fmt.Sprintf("val:%X/%d", pubkey.Data, power))
}

func isValidatorTx(tx []byte) bool {
	return strings.HasPrefix(string(tx), ValidatorSetChangePrefix)
}

func (app *AnchorApplication) execValidatorTx(tx []byte, tags []cmn.KVPair) types.ResponseDeliverTx {
	tx = tx[len(ValidatorSetChangePrefix):]
	//get the pubkey and power
	pubKeyAndPower := strings.Split(string(tx), "/")
	if len(pubKeyAndPower) != 2 {
		return types.ResponseDeliverTx{
			Code: code.CodeTypeEncodingError,
			Log:  fmt.Sprintf("Expected 'pubkey/power'. Got %v", pubKeyAndPower)}
	}
	pubkeyS, powerS := pubKeyAndPower[0], pubKeyAndPower[1]

	// decode the pubkey
	pubkey, err := base64.StdEncoding.DecodeString(pubkeyS)
	if err != nil {
		return types.ResponseDeliverTx{
			Code: code.CodeTypeEncodingError,
			Log:  fmt.Sprintf("Pubkey (%s) is invalid hex", pubkeyS)}
	}

	// decode the power
	power, err := strconv.ParseInt(powerS, 10, 64)
	if err != nil {
		return types.ResponseDeliverTx{
			Code: code.CodeTypeEncodingError,
			Log:  fmt.Sprintf("Power (%s) is not an int", powerS)}
	}

	// update
	return app.updateValidator(types.Ed25519ValidatorUpdate(pubkey, int64(power)), tags)
}

// add, update, or remove a validator
func (app *AnchorApplication) updateValidator(v types.ValidatorUpdate, tags []cmn.KVPair) types.ResponseDeliverTx {
	key := []byte("val:" + string(v.PubKey.Data))
	if v.Power == 0 {
		// remove validator
		if !app.state.db.Has(key) {
			return types.ResponseDeliverTx{
				Code: code.CodeTypeUnauthorized,
				Log:  fmt.Sprintf("Cannot remove non-existent validator %X", key)}
		}
		app.state.db.Delete(key)
	} else {
		// add or update validator
		value := bytes.NewBuffer(make([]byte, 0))
		if err := types.WriteMessage(&v, value); err != nil {
			return types.ResponseDeliverTx{
				Code: code.CodeTypeEncodingError,
				Log:  fmt.Sprintf("Error encoding validator: %v", err)}
		}
		app.state.db.Set(key, value.Bytes())
	}

	// we only update the changes array if we successfully updated the tree
	app.ValUpdates = append(app.ValUpdates, v)

	return types.ResponseDeliverTx{Code: code.CodeTypeOK, Tags: tags}
}
