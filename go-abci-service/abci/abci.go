package abci

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"time"

	"github.com/chainpoint/chainpoint-core/go-abci-service/util"

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
	stateKey                         = []byte("chainpoint")
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
	ValUpdates     []types.ValidatorUpdate
	state          State
	rabbitmqUri    string
	tendermintURI  TendermintURI
	doAnchor       bool
	anchorInterval int64
}

func NewAnchorApplication(rabbitmqUri string, tendermintRPC TendermintURI, doAnchor bool, anchorInterval int64) *AnchorApplication {
	name := "anchor"
	db, err := dbm.NewGoLevelDB(name, "/tendermint/data")
	if err != nil {
		panic(err)
	}
	state := loadState(db)
	return &AnchorApplication{
		state:          state,
		rabbitmqUri:    rabbitmqUri,
		tendermintURI:  tendermintRPC,
		doAnchor:       doAnchor,
		anchorInterval: anchorInterval,
	}
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
	infoJSON, err := json.Marshal(app.state)
	if err != nil {
		fmt.Println(err)
		infoJSON = []byte("{}")
	}
	return types.ResponseInfo{
		Data:       string(infoJSON),
		Version:    version.ABCIVersion,
		AppVersion: ProtocolVersion.Uint64(),
	}
}

// tx is url encoded json
func (app *AnchorApplication) DeliverTx(tx []byte) types.ResponseDeliverTx {
	resp := app.updateStateFromTx(tx)
	return resp
}

func (app *AnchorApplication) CheckTx(tx []byte) types.ResponseCheckTx {
	return types.ResponseCheckTx{Code: code.CodeTypeOK, GasWanted: 1}
}

func (app *AnchorApplication) BeginBlock(req types.RequestBeginBlock) types.ResponseBeginBlock {
	if app.state.PendingBtcaTx != (Tx{}) {
		result, err := BroadcastTx(app.tendermintURI, "BTC-A", app.state.PendingBtcaTx.Data, 2, time.Now().Unix())
		util.LogError(err)
		fmt.Printf("Anchor result: %v\n", result)
		app.state.PendingBtcaTx = Tx{}
	}
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
	if app.doAnchor && (app.state.Height-app.state.LatestBtcaHeight) > app.anchorInterval {
		err := app.Anchor()
		util.LogError(err)
	}
	return types.ResponseCommit{Data: appHash}
}

func (app *AnchorApplication) Query(reqQuery types.RequestQuery) (resQuery types.ResponseQuery) {
	return
}
