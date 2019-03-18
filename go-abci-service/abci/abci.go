package abci

import (
	"encoding/binary"
	"encoding/json"
	"time"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/chainpoint/chainpoint-core/go-abci-service/aggregator"
	"github.com/chainpoint/chainpoint-core/go-abci-service/calendar"

	cron "gopkg.in/robfig/cron.v3"

	"github.com/chainpoint/chainpoint-core/go-abci-service/types"
	"github.com/tendermint/tendermint/abci/example/code"
	types2 "github.com/tendermint/tendermint/abci/types"
	cmn "github.com/tendermint/tendermint/libs/common"
	dbm "github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/version"
)

// variables for protocol version and main db state key
var (
	stateKey                         = []byte("chainpoint")
	ProtocolVersion version.Protocol = 0x1
)

// loadState loads the AnchorState struct from a database instance
func loadState(db dbm.DB) types.AnchorState {
	stateBytes := db.Get(stateKey)
	var state types.AnchorState
	if len(stateBytes) != 0 {
		err := json.Unmarshal(stateBytes, &state)
		if err != nil {
			panic(err)
		}
	}
	return state
}

//loadState saves the AnchorState struct to disk
func saveState(Db dbm.DB, state types.AnchorState) {
	stateBytes, err := json.Marshal(state)
	if err != nil {
		panic(err)
	}
	Db.Set(stateKey, stateBytes)
}

//---------------------------------------------------

var _ types2.Application = (*AnchorApplication)(nil)

// AnchorApplication : AnchorState and config variables for the abci app
type AnchorApplication struct {
	types2.BaseApplication
	ValUpdates []types2.ValidatorUpdate
	Db         dbm.DB
	state      types.AnchorState
	config     types.AnchorConfig
	logger     log.Logger
	calendar   *calendar.Calendar
	aggregator *aggregator.Aggregator
}

//NewAnchorApplication is ABCI app constructor
func NewAnchorApplication(config types.AnchorConfig) *AnchorApplication {
	// Load state from disk
	name := "anchor"
	db := dbm.NewDB(name, dbm.DBBackendType(config.DBType), "/tendermint/data")
	state := loadState(db)
	state.ChainSynced = false // False until we finish syncing

	app := AnchorApplication{
		Db:     db,
		state:  state,
		config: config,
		logger: *config.Logger,
		calendar: &calendar.Calendar{
			RabbitmqURI: config.RabbitmqURI,
			Logger:      *config.Logger,
		},
		aggregator: &aggregator.Aggregator{
			RabbitmqURI: config.RabbitmqURI,
			Logger:      *config.Logger,
		},
	}

	if config.DoCal {
		// Create cron scheduler
		scheduler := cron.New(cron.WithLocation(time.UTC))

		// Update NIST beacon record and gossip it to ensure everyone has the same aggregator state
		scheduler.AddFunc("0/1 0-23 * * *", app.NistBeaconMonitor)

		// Run calendar aggregation every minute with pointer to nist object
		scheduler.AddFunc("0/1 0-23 * * *", func() {
			if app.state.ChainSynced { // don't aggregate if Tendermint isn't synced or functioning correctly
				time.Sleep(30 * time.Second) //offset from nist loop by 30 seconds
				app.AggregateCalendar()
			}
		})
		scheduler.Start()
	}

	if config.DoAnchor {
		go app.SyncMonitor()   //make sure we're synced before enabling anchoring
		go app.ReceiveCalRMQ() // Infinite loop to process btctx and btcmon rabbitMQ messages
	}

	return &app
}

// SetOption : Method for runtime data transfer between other apps and ABCI
func (app *AnchorApplication) SetOption(req types2.RequestSetOption) (res types2.ResponseSetOption) {
	return
}

// InitChain : Save the validators in the merkle tree
func (app *AnchorApplication) InitChain(req types2.RequestInitChain) types2.ResponseInitChain {
	for _, v := range req.Validators {
		r := app.updateValidator(v, []cmn.KVPair{})
		if r.IsErr() {
			app.logger.Error("Init Chain failed", r)
		}
	}
	return types2.ResponseInitChain{}
}

// Info : Return the state of the current application in JSON
func (app *AnchorApplication) Info(req types2.RequestInfo) (resInfo types2.ResponseInfo) {
	infoJSON, err := json.Marshal(app.state)
	if err != nil {
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

// CheckTx : Pre-gossip validation
func (app *AnchorApplication) CheckTx(tx []byte) types2.ResponseCheckTx {
	return types2.ResponseCheckTx{Code: code.CodeTypeOK, GasWanted: 1}
}

// BeginBlock : Handler that runs at the beginning of every block
func (app *AnchorApplication) BeginBlock(req types2.RequestBeginBlock) types2.ResponseBeginBlock {
	app.ValUpdates = make([]types2.ValidatorUpdate, 0)
	return types2.ResponseBeginBlock{}
}

// EndBlock : Handler that runs at the end of every block, validators can be updated here
func (app *AnchorApplication) EndBlock(req types2.RequestEndBlock) types2.ResponseEndBlock {
	return types2.ResponseEndBlock{ValidatorUpdates: app.ValUpdates}
}

//Commit is called at the end of every block to finalize and save chain state
func (app *AnchorApplication) Commit() types2.ResponseCommit {

	// Anchor every anchorInterval of blocks
	if app.config.DoAnchor && (app.state.Height-app.state.LatestBtcaHeight) > int64(app.config.AnchorInterval) {
		if app.state.ChainSynced {
			go app.AnchorBTC(app.state.BeginCalTxInt, app.state.LatestCalTxInt)
		} else {
			app.state.EndCalTxInt = app.state.LatestCalTxInt
		}
	}

	// Finalize new block by calculating appHash and incrementing height
	appHash := make([]byte, 8)
	binary.PutVarint(appHash, app.state.LatestCalTxInt) // most frequent tx, best tracks app state
	app.state.AppHash = appHash
	app.state.Height++
	saveState(app.Db, app.state)

	return types2.ResponseCommit{Data: appHash}
}

// Query : Custom ABCI query method. TODO: implement
func (app *AnchorApplication) Query(reqQuery types2.RequestQuery) (resQuery types2.ResponseQuery) {
	return
}
