package abci

import (
	"crypto/ecdsa"
	"encoding/binary"
	"encoding/json"
	"fmt"
	analytics2 "github.com/chainpoint/chainpoint-core/analytics"
	"github.com/chainpoint/chainpoint-core/anchor"
	"github.com/chainpoint/chainpoint-core/anchor/bitcoin"
	"github.com/chainpoint/chainpoint-core/level"
	"github.com/chainpoint/chainpoint-core/tendermint_rpc"
	"github.com/tendermint/tendermint/abci/example/code"
	"net"
	"path"
	"strings"
	"time"

	"github.com/chainpoint/chainpoint-core/lightning"

	"github.com/chainpoint/chainpoint-core/validation"

	"github.com/chainpoint/chainpoint-core/util"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/chainpoint/chainpoint-core/aggregator"

	"github.com/chainpoint/chainpoint-core/types"
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
	Anchor               anchor.AnchorEngine
	state                *types.AnchorState
	config               types.AnchorConfig
	logger               log.Logger
	aggregator           *aggregator.Aggregator
	Cache                *level.Cache
	LnClient             *lightning.LnClient
	rpc                  *tendermint_rpc.RPC
	ID                   string
	JWK                  types.Jwk
	Analytics            *analytics2.UniversalAnalytics
}

//NewAnchorApplication is ABCI app constructor
func NewAnchorApplication(config types.AnchorConfig) *AnchorApplication {
	// Load state from disk
	name := "anchor"
	db := dbm.NewDB(name, dbm.CLevelDBBackend, config.HomePath+"/data")
	load_state := loadState(db)
	state := &load_state
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

	var err error

	//Wait for lightning connection
	err = config.LightningConfig.WaitForConnection(5 * time.Minute)
	if err != nil {
		fmt.Println("LND not ready after 5 minutes")
		panic(err)
	}
	err = nil

	jwkType := util.GenerateKey(config.ECPrivateKey, string(config.TendermintConfig.NodeKey.ID()))

	rpcClient := tendermint_rpc.NewRPCClient(config.TendermintConfig, *config.Logger)

	analytics := analytics2.NewClient(config.CoreName, config.AnalyticsID, *config.Logger)

	cache := level.NewCache(&db, *config.Logger)

	var anchorEngine anchor.AnchorEngine = bitcoin.NewBTCAnchorEngine(state, config, rpcClient, cache, &config.LightningConfig, *config.Logger, &analytics)

	//Construct application
	app := AnchorApplication{
		valAddrToPubKeyMap:   map[string]types2.PubKey{},
		Db:                   db,
		Anchor:               anchorEngine,
		state:                state,
		config:               config,
		logger:               *config.Logger,
		NodeRewardSignatures: make([]string, 0),
		CoreRewardSignatures: make([]string, 0),
		aggregator: &aggregator.Aggregator{
			Logger: *config.Logger,
		},
		Cache:     cache,
		LnClient:  &config.LightningConfig,
		rpc:       rpcClient,
		JWK:       jwkType,
		Analytics: &analytics,
	}

	app.logger.Info("Tendermint Block Height", "block_height", app.state.Height)

	app.logger.Info("Lightning Staking", "JWKStaked", state.JWKStaked, "JWK Kid", jwkType.Kid)

	//Initialize calendar writing if enabled
	if config.DoCal {
		go app.aggregator.StartAggregation()
	}

	go app.SyncMonitor() //make sure we're synced

	// Load JWK into local mapping from redis
	go app.LoadIdentity()

	// Execute any necessary logic to change the staking amount
	go app.SetStake()

	// Stake and transmit identity
	go app.StakeIdentity()

	//Migrations
	/*	if _, exists := app.state.Migrations[1]; !exists && config.ChainId == "mainnet-chain-32" {
			app.state.BeginCalTxInt = 3096
			app.state.Migrations[1] = "BeginCalTxInt=3096"
		}
		if _, exists := app.state.Migrations[2]; !exists && config.ChainId == "mainnet-chain-32" {
			app.state.LatestBtcaHeight = 17399
			app.state.Migrations[2] = "LatestBtcaHeight=17399"
		}*/

	// execute validator promotion logic

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
	return app.updateStateFromTx(tx.Tx)
}

// CheckTx : Pre-gossip validation
func (app *AnchorApplication) CheckTx(rawTx types2.RequestCheckTx) types2.ResponseCheckTx {
	return app.validateTx(rawTx.Tx)
}

// BeginBlock : Handler that runs at the beginning of every block
func (app *AnchorApplication) BeginBlock(req types2.RequestBeginBlock) types2.ResponseBeginBlock {
	app.ValUpdates = make([]types2.ValidatorUpdate, 0)
	return types2.ResponseBeginBlock{}
}

// EndBlock : Handler that runs at the end of every block, validators can be updated here
func (app *AnchorApplication) EndBlock(req types2.RequestEndBlock) types2.ResponseEndBlock {
	// If the chain is synced, run all polling methods
	if app.state.ChainSynced {
		go app.BeaconMonitor() // update time beacon using deterministic leader election
		go app.FeeMonitor()
	}
	// StartAnchoring blockchain
	app.StartAnchoring()

	// monitor confirmed tx. Run on a separate thread but in order
	if app.state.ChainSynced {
		go func() {
			app.Anchor.BlockSyncMonitor()
			if app.config.DoAnchor {
				app.Anchor.MonitorConfirmedTx()
				app.Anchor.MonitorFailedAnchor() //must be roughly synchronous with chain operation in order to recover from failed anchors
			}
		}()
		if app.config.DoAnchor {
			go app.Cache.PruneOldState()
		}
	}
	// check if we need to vote on a pending validator proposal
	app.CheckVoteValidator()
	return types2.ResponseEndBlock{ValidatorUpdates: app.ValUpdates}
}

//Commit is called at the end of every block to finalize and save chain state
func (app *AnchorApplication) Commit() types2.ResponseCommit {
	// Finalize new block by calculating appHash and incrementing height
	appHash := make([]byte, 8)
	binary.PutVarint(appHash, app.state.Height)
	app.state.AppHash = appHash
	app.state.Height++
	saveState(app.Db, *app.state)

	return types2.ResponseCommit{Data: appHash}
}

// Query : Custom ABCI query method.
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
		JWKChanges, _ := validation.GetJWKChanges(coreID, app.state)
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
