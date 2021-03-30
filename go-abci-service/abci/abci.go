package abci

import (
	"crypto/ecdsa"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/chainpoint/tendermint/abci/example/code"
	"net"
	"path"
	"strings"
	"time"

	types3 "github.com/tendermint/tendermint/types"

	"github.com/chainpoint/chainpoint-core/go-abci-service/lightning"

	"github.com/chainpoint/chainpoint-core/go-abci-service/validation"

	"github.com/chainpoint/chainpoint-core/go-abci-service/postgres"
	"github.com/chainpoint/chainpoint-core/go-abci-service/util"
	"github.com/go-redis/redis"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/chainpoint/chainpoint-core/go-abci-service/aggregator"
	"github.com/chainpoint/chainpoint-core/go-abci-service/calendar"

	"github.com/chainpoint/chainpoint-core/go-abci-service/types"
	types2 "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/version"
	dbm "github.com/tendermint/tm-db"
)

// variables for protocol version and main db state key
var (
	stateKey                         = []byte("chainpoint")
	ProtocolVersion version.Protocol = 0x1
	GossipTxs                        = []string{"NIST"}
)

const SUCCESSFUL_ANCHOR_CRITERIA = 100

// loadState loads the AnchorState struct from a database instance
func loadState(db dbm.DB) types.AnchorState {
	stateBytes, err := db.Get(stateKey)
	if util.LogError(err) != nil {
		panic(err)
	}
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
	// validator set
	ValUpdates           []types2.ValidatorUpdate
	valAddrToPubKeyMap   map[string]types2.PubKey
	PendingValidator     string
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
	lnClient             *lightning.LnClient
	rpc                  *RPC
	ID                   string
	JWK                  types.Jwk
}

//NewAnchorApplication is ABCI app constructor
func NewAnchorApplication(config types.AnchorConfig) *AnchorApplication {
	// Load state from disk
	name := "anchor"
	db := dbm.NewDB(name, dbm.CLevelDBBackend, "/tendermint/data")
	state := loadState(db)
	if state.TxValidation == nil {
		state.TxValidation = validation.NewTxValidationMap()
	}
	if state.LnUris == nil {
		state.LnUris = map[string]types.LnIdentity{}
	}
	if state.Migrations == nil {
		state.Migrations = make(map[int]string)
	}
	if state.IDMap == nil {
		state.IDMap = make(map[string]string)
	}
	state.CoreKeys = map[string]ecdsa.PublicKey{}
	state.ChainSynced = false // False until we finish syncing

	// Declare postgres connection
	var pgClient *postgres.Postgres
	var err error
	deadline := time.Now().Add(1 * time.Minute)
	for !time.Now().After(deadline) {
		pgClient, err = postgres.NewPGFromURI(config.PostgresURI, *config.Logger)
		if util.LoggerError(*config.Logger, err) != nil {
			time.Sleep(5 * time.Second)
		} else {
			break
		}
	}
	if err != nil {
		fmt.Println("Postgres not ready after 1 minute")
		panic(err)
	} else if pgClient != nil {
		if !pgClient.SchemaExists() {
			pgClient.CreateSchema()
		} else {
			fmt.Println("Schema exists")
		}
		fmt.Println("Connection to Postgres established")
	}

	//Declare redis Client
	var redisClient *redis.Client
	deadline = time.Now().Add(1 * time.Minute)
	for !time.Now().After(deadline) {
		opt, err := redis.ParseURL(config.RedisURI)
		if util.LoggerError(*config.Logger, err) != nil {
			time.Sleep(5 * time.Second)
			continue
		}
		redisClient = redis.NewClient(opt)
		_, err = redisClient.Ping().Result()
		if util.LoggerError(*config.Logger, err) != nil {
			time.Sleep(5 * time.Second)
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

	//Wait for lightning connection
	deadline = time.Now().Add(5 * time.Minute)
	for !time.Now().After(deadline) {
		conn, err := config.LightningConfig.CreateConn()
		if util.LoggerError(*config.Logger, err) != nil {
			time.Sleep(5 * time.Second)
			continue
		} else {
			conn.Close()
			break
		}
	}
	if err != nil {
		fmt.Println("LND not ready after 1 minute")
		panic(err)
	} else if redisClient != nil {
		fmt.Println("Connection to LND established")
	}

	jwkType := util.GenerateKey(&config.ECPrivateKey, string(config.TendermintConfig.NodeKey.ID()))

	//Construct application
	app := AnchorApplication{
		valAddrToPubKeyMap:   map[string]types2.PubKey{},
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
			Logger: *config.Logger,
		},
		pgClient:    pgClient,
		redisClient: redisClient,
		lnClient:    &config.LightningConfig,
		rpc:         NewRPCClient(config.TendermintConfig, *config.Logger),
		JWK:         jwkType,
	}

	app.logger.Info("Lightning Staking", "JWKStaked", state.JWKStaked, "JWK Kid", jwkType.Kid)

	//Initialize calendar writing if enabled
	if config.DoCal {
		go app.aggregator.StartAggregation()
	}

	//Initialize anchoring to bitcoin if enabled
	if config.DoAnchor {
		go app.SyncMonitor() //make sure we're synced before enabling anchoring
	}

	// Load JWK into local mapping from redis
	app.LoadIdentity()

	// Stake and transmit identity
	go app.StakeIdentity()

	// Ensure LND Wallet stays unlocked
	go app.LNDMonitor()

	//Migrations
	/*	if _, exists := app.state.Migrations[1]; !exists && config.ChainId == "mainnet-chain-32" {
			app.state.BeginCalTxInt = 3096
			app.state.Migrations[1] = "BeginCalTxInt=3096"
		}
		if _, exists := app.state.Migrations[2]; !exists && config.ChainId == "mainnet-chain-32" {
			app.state.LatestBtcaHeight = 17399
			app.state.Migrations[2] = "LatestBtcaHeight=17399"
		}*/

	key := util.GetEnv("SET_OPTION_KEY", "")
	val := util.GetEnv("SET_OPTION_VAL", "")
	if key == "VOTE" || key == "VAL" {
		go func() {
			time.Sleep(1 * time.Minute)
			components := strings.Split(val, "!")
			if len(components) != 3 {
				app.logger.Error("VAL or VOTE data is malformed")
				return
			}
			if key == "VAL" {
				app.PendingValidator = val
				app.rpc.BroadcastTx("VAL", val, 2, time.Now().Unix(), app.ID, &app.config.ECPrivateKey)
			}
			if key == "VOTE" {
				app.PendingValidator = val
			}
		}()
	}

	return &app
}

// SetOption : Method for runtime data transfer between other apps and ABCI
func (app *AnchorApplication) SetOption(req types2.RequestSetOption) (res types2.ResponseSetOption) {
	//req.Value must be <base64ValidatorPubKey>!<VotingPower>

	return
}

// InitChain : Save the validators in the merkle tree
func (app *AnchorApplication) InitChain(req types2.RequestInitChain) types2.ResponseInitChain {
	for _, v := range req.Validators {
		r := app.updateValidator(v)
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
func (app *AnchorApplication) DeliverTx(tx types2.RequestDeliverTx) types2.ResponseDeliverTx {
	return app.updateStateFromTx(tx.Tx, false)
}

// CheckTx : Pre-gossip validation
func (app *AnchorApplication) CheckTx(rawTx types2.RequestCheckTx) types2.ResponseCheckTx {
	return app.validateTx(rawTx.Tx)
}

// BeginBlock : Handler that runs at the beginning of every block
func (app *AnchorApplication) BeginBlock(req types2.RequestBeginBlock) types2.ResponseBeginBlock {
	app.ValUpdates = make([]types2.ValidatorUpdate, 0)
	for _, ev := range req.ByzantineValidators {
		if ev.Type == types3.ABCIEvidenceTypeDuplicateVote {
			// decrease voting power by 1
			if ev.TotalVotingPower == 0 {
				continue
			}
			app.updateValidator(types2.ValidatorUpdate{
				PubKey: app.valAddrToPubKeyMap[string(ev.Validator.Address)],
				Power:  ev.TotalVotingPower - 1,
			})
		}
	}
	return types2.ResponseBeginBlock{}
}

// EndBlock : Handler that runs at the end of every block, validators can be updated here
func (app *AnchorApplication) EndBlock(req types2.RequestEndBlock) types2.ResponseEndBlock {
	// If the chain is synced, run all polling methods
	if app.state.ChainSynced {
		go app.BeaconMonitor() // update time beacon using deterministic leader election
		go app.FeeMonitor()
	}

	// Anchor blockchain
	app.Anchor()

	// monitor confirmed tx
	if app.state.ChainSynced && app.config.DoAnchor {
		go func() {
			app.MonitorNewTx()
			app.MonitorConfirmedTx()
			app.FailedAnchorMonitor() //must be roughly synchronous with chain operation in order to recover from failed anchors
			app.pgClient.PruneProofStateTables()
		}()
	}
	return types2.ResponseEndBlock{ValidatorUpdates: app.ValUpdates}
}

//Commit is called at the end of every block to finalize and save chain state
func (app *AnchorApplication) Commit() types2.ResponseCommit {
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
	urlPath := reqQuery.Path
	base := path.Base(urlPath)
	resQuery.Code = code.CodeTypeOK
	if strings.Contains(urlPath, "/p2p/filter/addr") {
		ipStr := base
		if strings.Contains(base, ":") {
			ipStr = strings.Split(base, ":")[0]
		}
		app.logger.Info(fmt.Sprintf("Looking up peer info for ip %s", ipStr))
		ip := net.ParseIP(ipStr)
		for _, blockCIDR := range app.config.CIDRBlockList {
			if blockCIDR == "" {
				continue
			}
			_, ipNet, _ := net.ParseCIDR(blockCIDR)
			if ipNet.Contains(ip) {
				app.logger.Info(fmt.Sprintf("ip %s unauthorized", ipStr))
				resQuery.Code = code.CodeTypeUnauthorized
				return
			}
		}
		for _, blockIP := range app.config.IPBlockList {
			if strings.Contains(blockIP, ipStr) {
				app.logger.Info(fmt.Sprintf("ip %s unauthorized", ipStr))
				resQuery.Code = code.CodeTypeUnauthorized
				return
			}
		}
		app.logger.Info(fmt.Sprintf("connection allowed for ip %s", base))
	} else if strings.Contains(urlPath, "/p2p/filter/id") {
		coreID, exists := app.state.IDMap[base]
		if !exists {
			app.logger.Info(fmt.Sprintf("id record does not exist for %s", base))
			return
		}
		app.logger.Info(fmt.Sprintf("Looking up peer info for id %s", base))
/*		_, validationRecord, err := validation.GetValidationRecord(coreID, app.state)
		if app.LogError(err) != nil {
			app.logger.Info(fmt.Sprintf("validation record does not exist for %s", base))
			return
		}
		anchorRatio, _ := validation.GetAnchorSuccessRatio(coreID, &app.state)
		if anchorRatio < 0.3 && app.state.Height-validationRecord.LastBtcaTxHeight > 10000 {
			app.logger.Info(fmt.Sprintf("id %s unauthorized", coreID))
			resQuery.Code = code.CodeTypeUnauthorized
			return
		}
		calRatio, _ := validation.GetCalSuccessRatio(coreID, &app.state)
		if calRatio < 0.3 && app.state.Height-validationRecord.LastCalTxHeight > 10000 {
			app.logger.Info(fmt.Sprintf("id %s unauthorized", coreID))
			resQuery.Code = code.CodeTypeUnauthorized
			return
		}*/
		JWKChanges, _ := validation.GetJWKChanges(coreID, &app.state)
		if JWKChanges > 3 {
			app.logger.Info(fmt.Sprintf("id %s unauthorized", coreID))
			resQuery.Code = code.CodeTypeUnauthorized
			return
		}
		app.logger.Info(fmt.Sprintf("connection allowed for id %s", base))
	}
	return
}

func (app *AnchorApplication) LogError(err error) error {
	if err != nil {
		app.logger.Error(fmt.Sprintf("Error in %s: %s", util.GetCurrentFuncName(2), err.Error()))
	}
	return err
}
