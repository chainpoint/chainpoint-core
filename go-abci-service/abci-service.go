package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/tendermint/tendermint/node"
	"github.com/tendermint/tendermint/proxy"

	"github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tendermint/privval"

	"github.com/knq/pemutil"
	"github.com/spf13/viper"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/chainpoint/chainpoint-core/go-abci-service/abci"
	"github.com/chainpoint/chainpoint-core/go-abci-service/types"
	"github.com/chainpoint/chainpoint-core/go-abci-service/util"
	cfg "github.com/tendermint/tendermint/config"
	tmflags "github.com/tendermint/tendermint/libs/cli/flags"
	cmn "github.com/tendermint/tendermint/libs/common"
)

func main() {
	// Perform env type conversions
	doNodeManagement, _ := strconv.ParseBool(util.GetEnv("NODE_MANAGEMENT", "true"))
	doAuditLoop, _ := strconv.ParseBool(util.GetEnv("AUDIT", "true"))
	doAuditLoop = doNodeManagement && doAuditLoop //only allow auditing if node management enabled
	doCalLoop, _ := strconv.ParseBool(util.GetEnv("AGGREGATE", "false"))
	doAnchorLoop, _ := strconv.ParseBool(util.GetEnv("ANCHOR", "false"))
	anchorInterval, _ := strconv.Atoi(util.GetEnv("ANCHOR_BLOCK_INTERVAL", "60"))
	ethInfuraApiKey := util.GetEnv("ETH_INFURA_API_KEY", "")
	ethereumURL := util.GetEnv("ETH_URI", fmt.Sprintf("https://ropsten.infura.io/v3/%s", ethInfuraApiKey))
	ethTokenContract := util.GetEnv("TokenContractAddr", "0xB439eBe79cAeaA92C8E8813cEF14411B80bB8ef0")
	ethRegistryContract := util.GetEnv("RegistryContractAddr", "0x2Cfa392F736C1f562C5aA3D62226a29b7D1517b6")
	ethPrivateKey := util.GetEnv("ETH_PRIVATE_KEY", "")
	tendermintRPC := types.TendermintURI{
		TMServer: util.GetEnv("TENDERMINT_HOST", "127.0.0.1"),
		TMPort:   util.GetEnv("TENDERMINT_PORT", "26657"),
	}
	tendermintPeers := util.GetEnv("PEERS", "")
	tendermintSeeds := util.GetEnv("SEEDS", "")
	postgresUser := util.GetEnv(" POSTGRES_CONNECT_USER", "chainpoint")
	postgresPw := util.GetEnv("POSTGRES_CONNECT_PW", "chainpoint")
	postgresHost := util.GetEnv("POSTGRES_CONNECT_HOST", "postgres")
	postgresPort := util.GetEnv("POSTGRES_CONNECT_PORT", "5432")
	postgresDb := util.GetEnv("POSTGRES_CONNECT_DB", "chainpoint")
	redisURI := util.GetEnv("REDIS", "redis://redis:6379")

	allowLevel, _ := log.AllowLevel(strings.ToLower(util.GetEnv("LOG_LEVEL", "DEBUG")))
	tmLogger := log.NewFilter(log.NewTMLogger(log.NewSyncWriter(os.Stdout)), allowLevel)

	ethConfig := types.EthConfig{
		EthereumURL:          ethereumURL,
		EthPrivateKey:        ethPrivateKey,
		TokenContractAddr:    ethTokenContract,
		RegistryContractAddr: ethRegistryContract,
	}

	store, err := pemutil.LoadFile("/run/secrets/ECDSA_KEYPAIR")
	if err != nil {
		util.LogError(err)
	}
	ecPrivKey, ok := store.ECPrivateKey()
	if !ok {
		util.LogError(errors.New("ecdsa key load failed"))
	}

	// Create config object
	config := types.AnchorConfig{
		DBType:           "goleveldb",
		RabbitmqURI:      util.GetEnv("RABBITMQ_URI", "amqp://chainpoint:chainpoint@rabbitmq:5672/"),
		TendermintRPC:    tendermintRPC,
		PostgresURI:      fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", postgresUser, postgresPw, postgresHost, postgresPort, postgresDb),
		RedisURI:         redisURI,
		EthConfig:        ethConfig,
		ECPrivateKey:     *ecPrivKey,
		DoNodeAudit:      doAuditLoop,
		DoNodeManagement: doNodeManagement,
		DoCal:            doCalLoop,
		DoAnchor:         doAnchorLoop,
		AnchorInterval:   anchorInterval,
		Logger:           &tmLogger,
	}

	//Instantiate ABCI application
	app := abci.NewAnchorApplication(config)
	defaultConfig := cfg.DefaultConfig()
	err = viper.Unmarshal(defaultConfig)
	if err != nil {
		return
	}
	defaultConfig.SetRoot("/tendermint")
	cfg.EnsureRoot(defaultConfig.RootDir)
	logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))
	if err != nil {
		util.LogError(err)
		return
	}
	if defaultConfig.LogFormat == cfg.LogFormatJSON {
		logger = log.NewTMJSONLogger(log.NewSyncWriter(os.Stdout))
	}
	logger, err = tmflags.ParseLogLevel(defaultConfig.LogLevel, logger, cfg.DefaultLogLevel())
	if err != nil {
		util.LogError(err)
		return
	}
	logger = logger.With("module", "main")

	nodeKey, err := p2p.LoadOrGenNodeKey(defaultConfig.NodeKeyFile())
	if err != nil {
		util.LogError(err)
		return
	}

	if tendermintPeers != "" {
		defaultConfig.P2P.PersistentPeers = tendermintPeers
	}
	if tendermintSeeds != "" {
		defaultConfig.P2P.Seeds = tendermintSeeds
	}

	// Convert old PrivValidator if it exists.
	oldPrivVal := defaultConfig.OldPrivValidatorFile()
	newPrivValKey := defaultConfig.PrivValidatorKeyFile()
	newPrivValState := defaultConfig.PrivValidatorStateFile()
	if _, err := os.Stat(oldPrivVal); !os.IsNotExist(err) {
		oldPV, err := privval.LoadOldFilePV(oldPrivVal)
		if err != nil {
			util.LogError(err)
			return
		}
		logger.Info("Upgrading PrivValidator file",
			"old", oldPrivVal,
			"newKey", newPrivValKey,
			"newState", newPrivValState,
		)
		oldPV.Upgrade(newPrivValKey, newPrivValState)
	}
	appProxy := proxy.NewLocalClientCreator(app)
	n, err := node.NewNode(defaultConfig,
		privval.LoadOrGenFilePV(newPrivValKey, newPrivValState),
		nodeKey,
		appProxy,
		node.DefaultGenesisDocProviderFunc(defaultConfig),
		node.DefaultDBProvider,
		node.DefaultMetricsProvider(defaultConfig.Instrumentation),
		logger,
	)
	if err != nil {
		util.LogError(err)
		return
	}

	// Wait forever
	cmn.TrapSignal(tmLogger, func() {
		if n.IsRunning() {
			n.Stop()
		}
	})

	if err := n.Start(); err != nil {
		util.LogError(err)
		return
	}
	logger.Info("Started node", "nodeInfo", n.Switch().NodeInfo())
	select {}

	return
}
