package abci

import (
	"crypto/ecdsa"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/chainpoint/tendermint/abci/example/code"

	"github.com/chainpoint/chainpoint-core/go-abci-service/postgres"
	"github.com/chainpoint/chainpoint-core/go-abci-service/util"
	"github.com/go-redis/redis"

	"github.com/chainpoint/tendermint/libs/log"

	"github.com/chainpoint/chainpoint-core/go-abci-service/aggregator"
	"github.com/chainpoint/chainpoint-core/go-abci-service/calendar"

	"github.com/chainpoint/chainpoint-core/go-abci-service/types"
	types2 "github.com/chainpoint/tendermint/abci/types"
	cmn "github.com/chainpoint/tendermint/libs/common"
	dbm "github.com/chainpoint/tendermint/libs/db"
	"github.com/chainpoint/tendermint/version"
)

// variables for protocol version and main db state key
var (
	stateKey                         = []byte("chainpoint")
	ProtocolVersion version.Protocol = 0x1
	GossipTxs                        = []string{"TOKEN", "NIST"}
)

const MINT_EPOCH = 6400

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
	ValUpdates           []types2.ValidatorUpdate
	NodeRewardSignatures []string
	CoreRewardSignatures []string
	Db                   dbm.DB
	state                types.AnchorState
	config               types.AnchorConfig
	logger               log.Logger
	calendar             *calendar.Calendar
	aggregator           *aggregator.Aggregator
	pgClient             *postgres.Postgres
	redisClient          *redis.Client
	rpc                  *RPC
	ID                   string
	JWK                  types.Jwk
	JWKSent              bool
	CoreKeys             map[string]ecdsa.PublicKey
}

//NewAnchorApplication is ABCI app constructor
func NewAnchorApplication(config types.AnchorConfig) *AnchorApplication {
	// Load state from disk
	name := "anchor"
	db := dbm.NewDB(name, dbm.DBBackendType(config.DBType), "/tendermint/data")
	state := loadState(db)
	state.ChainSynced = false // False until we finish syncing

	// Declare postgres connection
	var pgClient *postgres.Postgres
	var err error
	deadline := time.Now().Add(1 * time.Minute)
	for !time.Now().After(deadline) {
		pgClient, err = postgres.NewPGFromURI(config.PostgresURI, *config.Logger)
		if util.LoggerError(*config.Logger, err) != nil {
			time.Sleep(5 * time.Second)
			break
		}
	}
	if err != nil {
		fmt.Println("Postgres not ready after 1 minute")
		panic(err)
	} else if pgClient != nil {
		fmt.Println("Connection to Postgres established")
	}

	//Declare redis Client
	var redisClient *redis.Client
	deadline = time.Now().Add(1 * time.Minute)
	for !time.Now().After(deadline) {
		opt, err := redis.ParseURL(config.RedisURI)
		if util.LoggerError(*config.Logger, err) != nil {
			continue
		}
		redisClient = redis.NewClient(opt)
		_, err = redisClient.Ping().Result()
		if util.LoggerError(*config.Logger, err) != nil {
			continue
		} else {
			break
		}
	}
	if err != nil {
		fmt.Println("Redis not ready after 1 minute")
		panic(err)
	} else if redisClient != nil {
		fmt.Println("Connection to Redis established")
	}

	for _, coreIPString := range config.PrivateCoreIPs {
		coreDetails := strings.Split(coreIPString, "@")
		if len(coreDetails) != 2 {
			(*config.Logger).Error(fmt.Sprintf("Core list needs to be comma-delimited list of <Tendermint_ID>@<IP>"))
			continue
		}
		id := coreDetails[0]
		ip := coreDetails[1]
		core := types.Core{
			EthAddr:     "0",
			CoreId:      sql.NullString{String: id, Valid: true},
			PublicIP:    sql.NullString{String: ip, Valid: true},
			BlockNumber: sql.NullInt64{Int64: 0, Valid: true},
		}
		inserted, err := pgClient.CoreUpsert(core)
		if inserted {
			(*config.Logger).Info(fmt.Sprintf("Inserted private core %s: %t", coreIPString, inserted))
		}
		util.LoggerError(*config.Logger, err)
	}

	//Construct application
	app := AnchorApplication{
		Db:                   db,
		state:                state,
		config:               config,
		logger:               *config.Logger,
		NodeRewardSignatures: make([]string, 0),
		CoreRewardSignatures: make([]string, 0),
		calendar: &calendar.Calendar{
			RabbitmqURI: config.RabbitmqURI,
			Logger:      *config.Logger,
		},
		aggregator: &aggregator.Aggregator{
			RabbitmqURI: config.RabbitmqURI,
			Logger:      *config.Logger,
		},
		pgClient:    pgClient,
		redisClient: redisClient,
		rpc:         NewRPCClient(config.TendermintConfig, *config.Logger),
		CoreKeys:    map[string]ecdsa.PublicKey{},
	}

	//Initialize calendar writing if enabled
	if config.DoCal {
		go app.aggregator.StartAggregation()
	}

	//Initialize anchoring to bitcoin if enabled
	if config.DoAnchor {
		go app.SyncMonitor()   //make sure we're synced before enabling anchoring
		go app.ReceiveCalRMQ() // Infinite loop to process btctx and btcmon rabbitMQ messages
	}

	go app.KeyMonitor()

	// Load JWK into local mapping from redis
	app.LoadJWK()

	return &app
}

// SetOption : Method for runtime data transfer between other apps and ABCI
func (app *AnchorApplication) SetOption(req types2.RequestSetOption) (res types2.ResponseSetOption) {
	if req.Key == "TOKEN" {
		go app.pgClient.TokenHashUpsert(req.Value)
	}
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
		app.LogError(err)
		infoJSON = []byte("{}")
	}
	return types2.ResponseInfo{
		Data:             string(infoJSON),
		Version:          version.ABCIVersion,
		AppVersion:       ProtocolVersion.Uint64(),
		LastBlockAppHash: app.state.AppHash,
		LastBlockHeight:  app.state.Height,
	}
}

// DeliverTx : tx is url encoded json
func (app *AnchorApplication) DeliverTx(tx []byte) types2.ResponseDeliverTx {
	return app.updateStateFromTx(tx, false)
}

func (app *AnchorApplication) DeliverMsg(tx []byte) types2.ResponseDeliverMsg {
	code := code.CodeTypeUnknownError
	if app.state.ChainSynced {
		code = app.updateStateFromTx(tx, true).Code
	}
	return types2.ResponseDeliverMsg{Code: code}
}

// CheckTx : Pre-gossip validation
func (app *AnchorApplication) CheckTx(rawTx []byte) types2.ResponseCheckTx {
	return app.validateGossip(rawTx)
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
	// If the chain is synced, run all polling methods
	if app.state.ChainSynced {
		go app.NistBeaconMonitor() // update NIST beacon using deterministic leader election
		if app.config.DoCal {
			go app.AggregateCalendar()
		}
	}

	// Anchor every anchorInterval of blocks
	if app.config.DoAnchor && (app.state.Height-app.state.LatestBtcaHeight) > int64(app.config.AnchorInterval) {
		if app.state.ChainSynced {
			go app.AnchorBTC(app.state.BeginCalTxInt, app.state.LatestCalTxInt) // aggregate and anchor these tx ranges
		} else {
			app.state.EndCalTxInt = app.state.LatestCalTxInt
		}
	}

	// Finalize new block by calculating appHash and incrementing height
	appHash := make([]byte, 8)
	binary.PutVarint(appHash, app.state.Height)
	app.state.AppHash = appHash
	app.state.Height++
	saveState(app.Db, app.state)

	return types2.ResponseCommit{Data: appHash}
}

// Query : Custom ABCI query method. TODO: implement
func (app *AnchorApplication) Query(reqQuery types2.RequestQuery) (resQuery types2.ResponseQuery) {
	return
}

func (app *AnchorApplication) LogError(err error) error {
	if err != nil {
		app.logger.Error(fmt.Sprintf("Error: %s", err.Error()))
	}
	return err
}
