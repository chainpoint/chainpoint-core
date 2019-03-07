package abci

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"time"

	"github.com/chainpoint/chainpoint-core/go-abci-service/util"

	beacon "github.com/chainpoint/go-nist-beacon"

	cron "gopkg.in/robfig/cron.v3"

	"github.com/chainpoint/chainpoint-core/go-abci-service/types"
	"github.com/tendermint/tendermint/abci/example/code"
	types2 "github.com/tendermint/tendermint/abci/types"
	cmn "github.com/tendermint/tendermint/libs/common"
	dbm "github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/version"
)

// TODO: Describe this
const (
	ValidatorSetChangePrefix string = "val:"
)

// TODO: Describe this
var (
	stateKey                         = []byte("chainpoint")
	ProtocolVersion version.Protocol = 0x1
)

// loadState loads the State struct from a database instance
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

//loadState saves the State struct to disk
func saveState(state types.State) {
	stateBytes, err := json.Marshal(state)
	if err != nil {
		panic(err)
	}
	state.Db.Set(stateKey, stateBytes)
}

//---------------------------------------------------

var _ types2.Application = (*AnchorApplication)(nil)

// AnchorApplication : TODO: describe this
type AnchorApplication struct {
	types2.BaseApplication
	ValUpdates     []types2.ValidatorUpdate
	state          types.State
	rabbitmqURI    string
	tendermintURI  types.TendermintURI
	anchorInterval int
}

//NewAnchorApplication is ABCI app constructor
func NewAnchorApplication(rabbitmqURI string, tendermintRPC types.TendermintURI, doCal bool, doAnchor bool, anchorInterval int) *AnchorApplication {
	// Load state from disk
	name := "anchor"
	db, err := dbm.NewGoLevelDB(name, "/tendermint/data")
	if err != nil {
		panic(err)
	}
	state := loadState(db)
	state.AnchorEnabled = doAnchor

	app := AnchorApplication{
		state:          state,
		rabbitmqURI:    rabbitmqURI,
		tendermintURI:  tendermintRPC,
		anchorInterval: anchorInterval,
	}

	if doCal {
		// Create cron scheduler
		scheduler := cron.New(cron.WithLocation(time.UTC))

		// Update NIST beacon record and gossip it to ensure everyone has the same aggregator state
		scheduler.AddFunc("0/1 0-23 * * *", func() {
			nistRecord, err := beacon.LastRecord()
			if util.LogError(err) != nil {
				return
			}
			if leader, _ := ElectLeader(tendermintRPC); leader {
				_, err := BroadcastTx(tendermintRPC, "NIST", nistRecord.ChainpointFormat(), 2, time.Now().Unix()) // elect a leader to send a NIST tx
				util.LogError(err)
			}
		})

		// Run calendar aggregation every minute with pointer to nist object
		scheduler.AddFunc("0/1 0-23 * * *", func() {
			time.Sleep(30 * time.Second) //offset from nist loop by 30 seconds
			app.AggregateCalendar()
		})
		scheduler.Start()
	}

	// Infinite loop to process btctx and btcmon rabbitMQ messages
	if doAnchor {
		go app.SyncMonitor()
		go ReceiveCalRMQ(rabbitmqURI, tendermintRPC)
	}

	return &app
}

// SetOption : TODO: describe this
func (app *AnchorApplication) SetOption(req types2.RequestSetOption) (res types2.ResponseSetOption) {
	return
}

// InitChain : Save the validators in the merkle tree
func (app *AnchorApplication) InitChain(req types2.RequestInitChain) types2.ResponseInitChain {
	for _, v := range req.Validators {
		r := app.updateValidator(v, []cmn.KVPair{})
		if r.IsErr() {
			fmt.Println(r)
		}
	}
	return types2.ResponseInitChain{}
}

// Info : TODO: describe this
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

// DeliverTx : tx is url encoded json
func (app *AnchorApplication) DeliverTx(tx []byte) types2.ResponseDeliverTx {
	resp := app.updateStateFromTx(tx)
	return resp
}

// CheckTx : TODO: describe this
func (app *AnchorApplication) CheckTx(tx []byte) types2.ResponseCheckTx {
	return types2.ResponseCheckTx{Code: code.CodeTypeOK, GasWanted: 1}
}

// BeginBlock : TODO: describe this
func (app *AnchorApplication) BeginBlock(req types2.RequestBeginBlock) types2.ResponseBeginBlock {
	app.ValUpdates = make([]types2.ValidatorUpdate, 0)
	return types2.ResponseBeginBlock{}
}

// EndBlock : TODO: describe this
func (app *AnchorApplication) EndBlock(req types2.RequestEndBlock) types2.ResponseEndBlock {
	return types2.ResponseEndBlock{ValidatorUpdates: app.ValUpdates}
}

//Commit is called at the end of every block to finalize and save chain state
func (app *AnchorApplication) Commit() types2.ResponseCommit {

	// Anchor every anchorInterval of blocks
	if app.state.AnchorEnabled && (app.state.Height-app.state.LatestBtcaHeight) > int64(app.anchorInterval) {
		go app.AnchorBTC(app.state.BeginCalTxInt, app.state.LatestCalTxInt)
	} else if !app.state.AnchorEnabled {
		app.state.EndCalTxInt = app.state.LatestCalTxInt
	}

	// Finalize new block by calculating appHash and incrementing height
	appHash := make([]byte, 8)
	binary.PutVarint(appHash, app.state.Size)
	app.state.AppHash = appHash
	app.state.Height++
	saveState(app.state)

	return types2.ResponseCommit{Data: appHash}
}

// Query : TODO: describe this
func (app *AnchorApplication) Query(reqQuery types2.RequestQuery) (resQuery types2.ResponseQuery) {
	return
}
