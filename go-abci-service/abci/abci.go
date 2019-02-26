package abci

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"time"

	cron "gopkg.in/robfig/cron.v3"

	types2 "github.com/tendermint/tendermint/abci/types"

	"github.com/chainpoint/chainpoint-core/go-abci-service/types"
	"github.com/tendermint/tendermint/abci/example/code"
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

func loadState(db dbm.DB) types.State {
	stateBytes := db.Get(stateKey)
	var state types.State
	if len(stateBytes) != 0 {
		err := json.Unmarshal(stateBytes, &state)
		if err != nil {
			panic(err)
		}
	}
	state.Db = db
	return state
}

func saveState(state types.State) {
	stateBytes, err := json.Marshal(state)
	if err != nil {
		panic(err)
	}
	state.Db.Set(stateKey, stateBytes)
}

//---------------------------------------------------

var _ types2.Application = (*AnchorApplication)(nil)

type AnchorApplication struct {
	types2.BaseApplication
	ValUpdates     []types2.ValidatorUpdate
	state          types.State
	rabbitmqUri    string
	tendermintURI  types.TendermintURI
	doAnchor       bool
	anchorInterval int64
}

func NewAnchorApplication(rabbitmqUri string, tendermintRPC types.TendermintURI, doAnchor bool, anchorInterval int64) *AnchorApplication {
	name := "anchor"
	db, err := dbm.NewGoLevelDB(name, "/tendermint/data")
	if err != nil {
		panic(err)
	}
	state := loadState(db)

	// start calendar aggregation loop
	scheduler := cron.New(cron.WithLocation(time.UTC))
	scheduler.AddFunc("0/1 0-23 * * *", func() {
		AggregateCalendar(tendermintRPC, rabbitmqUri)
	})
	scheduler.Start()

	return &AnchorApplication{
		state:          state,
		rabbitmqUri:    rabbitmqUri,
		tendermintURI:  tendermintRPC,
		doAnchor:       doAnchor,
		anchorInterval: anchorInterval,
	}
}

func (app *AnchorApplication) SetOption(req types2.RequestSetOption) (res types2.ResponseSetOption) {
	return
}

// Save the validators in the merkle tree
func (app *AnchorApplication) InitChain(req types2.RequestInitChain) types2.ResponseInitChain {
	for _, v := range req.Validators {
		r := app.updateValidator(v, []cmn.KVPair{})
		if r.IsErr() {
			fmt.Println(r)
		}
	}
	return types2.ResponseInitChain{}
}

func (app *AnchorApplication) Info(req types2.RequestInfo) (resInfo types2.ResponseInfo) {
	infoJSON, err := json.Marshal(app.state)
	if err != nil {
		fmt.Println(err)
		infoJSON = []byte("{}")
	}
	return types2.ResponseInfo{
		Data:       string(infoJSON),
		Version:    version.ABCIVersion,
		AppVersion: ProtocolVersion.Uint64(),
	}
}

// tx is url encoded json
func (app *AnchorApplication) DeliverTx(tx []byte) types2.ResponseDeliverTx {
	resp := app.updateStateFromTx(tx)
	return resp
}

func (app *AnchorApplication) CheckTx(tx []byte) types2.ResponseCheckTx {
	return types2.ResponseCheckTx{Code: code.CodeTypeOK, GasWanted: 1}
}

func (app *AnchorApplication) BeginBlock(req types2.RequestBeginBlock) types2.ResponseBeginBlock {
	app.ValUpdates = make([]types2.ValidatorUpdate, 0)
	return types2.ResponseBeginBlock{}
}

func (app *AnchorApplication) EndBlock(req types2.RequestEndBlock) types2.ResponseEndBlock {
	return types2.ResponseEndBlock{ValidatorUpdates: app.ValUpdates}
}

func (app *AnchorApplication) Commit() types2.ResponseCommit {

	// Anchor every anchorInterval of blocks
	if app.doAnchor && (app.state.Height-app.state.LatestBtcaHeight) > app.anchorInterval {
		go AnchorBTC(app.tendermintURI, app.rabbitmqUri, &app.state.PrevCalTxInt, app.state.LatestCalTxInt)
	}

	// Finalize new block by calculating appHash and incrementing height
	appHash := make([]byte, 8)
	binary.PutVarint(appHash, app.state.Size)
	app.state.AppHash = appHash
	app.state.Height += 1
	saveState(app.state)

	return types2.ResponseCommit{Data: appHash}
}

func (app *AnchorApplication) Query(reqQuery types2.RequestQuery) (resQuery types2.ResponseQuery) {
	return
}
