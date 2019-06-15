package abci

import (
	"crypto/ecdsa"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/chainpoint/chainpoint-core/go-abci-service/ethcontracts"
	"github.com/chainpoint/chainpoint-core/go-abci-service/postgres"
	"github.com/chainpoint/chainpoint-core/go-abci-service/util"
	"github.com/go-redis/redis"

	"github.com/chainpoint/tendermint/libs/log"

	"github.com/chainpoint/chainpoint-core/go-abci-service/aggregator"
	"github.com/chainpoint/chainpoint-core/go-abci-service/calendar"

	cron "gopkg.in/robfig/cron.v3"

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
	GossipTxs                        = []string{"TOKEN", "NIST", "BTC-M"}
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
	ethClient            *ethcontracts.EthClient
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
			continue
		} else {
			_, err = pgClient.GetNodeCount()
			if util.LoggerError(*config.Logger, err) != nil {
				(*config.Logger).Info("table 'staked_nodes' doesn't exist, did API start successfully?")
				time.Sleep(5 * time.Second)
				continue
			}
			_, err = pgClient.GetCoreCount()
			if util.LoggerError(*config.Logger, err) != nil {
				(*config.Logger).Info("table 'staked_cores' doesn't exist, did API start successfully?")
				time.Sleep(5 * time.Second)
				continue
			}
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

	for _, nodeIPString := range config.PrivateNodeIPs {
		node := types.Node{
			EthAddr:     "0",
			PublicIP:    sql.NullString{String: nodeIPString, Valid: true},
			BlockNumber: sql.NullInt64{Int64: 0, Valid: true},
		}
		inserted, err := pgClient.NodeUpsert(node)
		if inserted {
			(*config.Logger).Info(fmt.Sprintf("Inserted private node %s: %t", nodeIPString, inserted))
		}
		util.LoggerError(*config.Logger, err)
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

	//Declare ethereum Client
	var ethClient *ethcontracts.EthClient
	if config.DoNodeManagement {
		ethClient, err = ethcontracts.NewClient(config.EthConfig.EthereumURL, config.EthConfig.EthPrivateKey,
			config.EthConfig.TokenContractAddr, config.EthConfig.RegistryContractAddr,
			*config.Logger)
		if util.LoggerError(*config.Logger, err) != nil {
			panic(err)
		}
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
		ethClient:   ethClient,
		rpc:         NewRPCClient(config.TendermintRPC, *config.Logger),
		CoreKeys:    map[string]ecdsa.PublicKey{},
	}

	// Create cron scheduler
	scheduler := cron.New(cron.WithLocation(time.UTC))

	//Initialize and monitor node state
	if config.DoNodeManagement {
		go app.PollNodesFromContract()
		go app.PollCoresFromContract()
	}

	//Initialize calendar writing if enabled
	if config.DoCal {
		// Run calendar aggregation every minute with pointer to nist object
		scheduler.AddFunc("0/1 0-23 * * *", func() {
			if app.state.ChainSynced { // don't aggregate if Tendermint isn't synced or functioning correctly
				time.Sleep(30 * time.Second) //offset from nist loop by 30 seconds
				app.AggregateCalendar()
			}
		})
	}

	//Initialize anchoring to bitcoin if enabled
	if config.DoAnchor {
		go app.SyncMonitor()   //make sure we're synced before enabling anchoring
		go app.ReceiveCalRMQ() // Infinite loop to process btctx and btcmon rabbitMQ messages
	}

	//Start scheduled cron tasks
	scheduler.Start()

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
		Data:       string(infoJSON),
		Version:    version.ABCIVersion,
		AppVersion: ProtocolVersion.Uint64(),
	}
}

// DeliverTx : tx is url encoded json
func (app *AnchorApplication) DeliverTx(tx []byte) types2.ResponseDeliverTx {
	return app.updateStateFromTx(tx, false)
}

func (app *AnchorApplication) DeliverMsg(tx []byte) types2.ResponseDeliverMsg {
	resp := app.updateStateFromTx(tx, true)
	return types2.ResponseDeliverMsg{Code: resp.Code}
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
	// Anchor every anchorInterval of blocks
	if app.config.DoAnchor && (app.state.Height-app.state.LatestBtcaHeight) > int64(app.config.AnchorInterval) {
		if app.state.ChainSynced {
			go app.AnchorBTC(app.state.BeginCalTxInt, app.state.LatestCalTxInt) // aggregate and anchor these tx ranges
			if app.config.DoNodeAudit && !app.state.NodeMintPending {
				go app.AuditNodes() //retrieve, audit, and reward some nodes
				go app.StartNodeMintProcess()
			}
			if app.config.DoNodeAudit && !app.state.CoreMintPending {
				go app.StartCoreMintProcess()
			}
		} else {
			app.state.EndCalTxInt = app.state.LatestCalTxInt
		}
	}

	if app.state.ChainSynced {
		go app.NistBeaconMonitor() // update NIST beacon using deterministic leader election
		if app.config.DoNodeAudit && app.JWKSent {
			go app.MintMonitor()
		}
		if !app.JWKSent {
			go app.KeyMonitor() // send out this Core's JWK to the rest of the network
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
