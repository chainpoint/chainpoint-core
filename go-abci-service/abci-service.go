package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
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

	//Instantiate ABCI application
	config := initABCIConfig()
	app := abci.NewAnchorApplication(config)

	//Instantiate Tendermint Node
	defaultConfig, err := initTendermintConfig()
	if err != nil {
		util.LogError(err)
		return
	}
	logger := initTMLogger(defaultConfig)
	nodeKey, err := p2p.LoadOrGenNodeKey(defaultConfig.NodeKeyFile())
	if util.LogError(err) != nil {
		return
	}
	if tendermintPeers := util.GetEnv("PEERS", ""); tendermintPeers != "" {
		defaultConfig.P2P.PersistentPeers = tendermintPeers
	}
	if tendermintSeeds := util.GetEnv("SEEDS", ""); tendermintSeeds != "" {
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

	//declare connection to abci app
	appProxy := proxy.NewLocalClientCreator(app)

	/* Declare Tendermint Node with given config and abci app */
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

	// Wait forever, shutdown gracefully upon
	cmn.TrapSignal(*config.Logger, func() {
		if n.IsRunning() {
			logger.Info("Shutting down Core...")
			n.Stop()
		}
	})

	// Start Tendermint Node
	if err := n.Start(); err != nil {
		util.LogError(err)
		return
	}
	logger.Info("Started node", "nodeInfo", n.Switch().NodeInfo())

	// Wait forever
	select {}
	return
}

// initABCIConfig: receives ENV variables and initializes app config struct
func initABCIConfig() types.AnchorConfig {
	// Perform env type conversions
	doPrivateNetwork, _ := strconv.ParseBool(util.GetEnv("PRIVATE_NETWORK", "false"))
	nodeIPs := strings.Split(util.GetEnv("PRIVATE_NODE_IPS", ""), ",")
	doNodeManagement, _ := strconv.ParseBool(util.GetEnv("NODE_MANAGEMENT", "true"))
	doAuditLoop, _ := strconv.ParseBool(util.GetEnv("AUDIT", "true"))
	doNodeManagement = doNodeManagement && !doPrivateNetwork           //only allow node management if private networking is disabled
	doAuditLoop = doNodeManagement && doAuditLoop && !doPrivateNetwork //only allow auditing if node management enabled and private networking disabled
	doCalLoop, _ := strconv.ParseBool(util.GetEnv("AGGREGATE", "false"))
	doAnchorLoop, _ := strconv.ParseBool(util.GetEnv("ANCHOR", "false"))
	anchorInterval, _ := strconv.Atoi(util.GetEnv("ANCHOR_INTERVAL", "60"))
	ethInfuraApiKey := util.GetEnv("ETH_INFURA_API_KEY", "")
	ethereumURL := util.GetEnv("ETH_URI", fmt.Sprintf("https://ropsten.infura.io/v3/%s", ethInfuraApiKey))
	testMode := util.GetEnv("NETWORK", "testnet")
	useTestNets := (testMode == "testnet")
	ethTokenContract := ""
	if doAuditLoop {
		ethTokenContract = util.ReadContractJSON("/go/src/github.com/chainpoint/chainpoint-core/go-abci-service/ethcontracts/TierionNetworkToken.json", useTestNets)
		if ethTokenContract == "" {
			ethTokenContract = util.GetEnv("TokenContractAddr", "0x230829991362d2741f954D2827771ce7c320FeB2")
		}
	}
	ethRegistryContract := ""
	if doNodeManagement {
		ethRegistryContract = util.ReadContractJSON("/go/src/github.com/chainpoint/chainpoint-core/go-abci-service/ethcontracts/ChainpointRegistry.json", useTestNets)
		if ethRegistryContract == "" && doNodeManagement {
			ethRegistryContract = util.GetEnv("RegistryContractAddr", "0xbD6a355A72aa22723423795F383e8bDd8526D0De")
		}
	}
	ethPrivateKey := util.GetEnv("ETH_PRIVATE_KEY", "")
	if len(ethPrivateKey) > 0 && strings.Contains(ethPrivateKey, "0x") {
		ethPrivateKey = ethPrivateKey[2:]
	}
	tendermintRPC := types.TendermintURI{
		TMServer: util.GetEnv("TENDERMINT_HOST", "127.0.0.1"),
		TMPort:   util.GetEnv("TENDERMINT_PORT", "26657"),
	}
	postgresUser := util.GetEnv(" POSTGRES_CONNECT_USER", "chainpoint")
	postgresPw := util.GetEnv("POSTGRES_CONNECT_PW", "chainpoint")
	postgresHost := util.GetEnv("POSTGRES_CONNECT_HOST", "postgres")
	postgresPort := util.GetEnv("POSTGRES_CONNECT_PORT", "5432")
	postgresDb := util.GetEnv("POSTGRES_CONNECT_DB", "chainpoint")
	redisURI := util.GetEnv("REDIS", "redis://redis:6379")
	apiURI := util.GetEnv("API_URI", "http://api:8080")

	allowLevel, _ := log.AllowLevel(strings.ToLower(util.GetEnv("LOG_LEVEL", "DEBUG")))
	tmLogger := log.NewFilter(log.NewTMLogger(log.NewSyncWriter(os.Stdout)), allowLevel)

	ethConfig := types.EthConfig{
		EthereumURL:          ethereumURL,
		EthPrivateKey:        ethPrivateKey,
		TokenContractAddr:    ethTokenContract,
		RegistryContractAddr: ethRegistryContract,
	}

	store, err := pemutil.LoadFile("/run/secrets/ECDSA_PKPEM")
	if err != nil {
		util.LogError(err)
	}
	ecPrivKey, ok := store.ECPrivateKey()
	if !ok {
		util.LogError(errors.New("ecdsa key load failed"))
	}

	// Create config object
	return types.AnchorConfig{
		DBType:           "goleveldb",
		RabbitmqURI:      util.GetEnv("RABBITMQ_URI", "amqp://chainpoint:chainpoint@rabbitmq:5672/"),
		TendermintRPC:    tendermintRPC,
		PostgresURI:      fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", postgresUser, postgresPw, postgresHost, postgresPort, postgresDb),
		RedisURI:         redisURI,
		APIURI:           apiURI,
		EthConfig:        ethConfig,
		ECPrivateKey:     *ecPrivKey,
		DoNodeAudit:      doAuditLoop,
		DoNodeManagement: doNodeManagement,
		DoPrivateNetwork: doPrivateNetwork,
		PrivateNodeIPs:   nodeIPs,
		DoCal:            doCalLoop,
		DoAnchor:         doAnchorLoop,
		AnchorInterval:   anchorInterval,
		Logger:           &tmLogger,
	}
}

// initTendermintConfig : imports tendermint config.toml and initializes config variables
func initTendermintConfig() (*cfg.Config, error) {
	initEnv("TM")
	homeFlag := os.ExpandEnv(filepath.Join("$HOME", cfg.DefaultTendermintDir))
	homeDir := "/tendermint"
	viper.Set(homeFlag, homeDir)
	viper.SetConfigName("config")                         // name of config file (without extension)
	viper.AddConfigPath(homeDir)                          // search root directory
	viper.AddConfigPath(filepath.Join(homeDir, "config")) // search root directory /config

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		// stderr, so if we redirect output to json file, this doesn't appear
		// fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	} else if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
		// ignore not found error, return other errors
		return nil, err
	}
	defaultConfig := cfg.DefaultConfig()
	err := viper.Unmarshal(defaultConfig)
	if err != nil {
		return nil, err
	}
	defaultConfig.SetRoot(homeDir)
	fmt.Printf("Config : %#v\n", defaultConfig)
	cfg.EnsureRoot(defaultConfig.RootDir)
	return defaultConfig, nil
}

// initTMLogger : initializes
func initTMLogger(defaultConfig *cfg.Config) log.Logger {
	logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))

	if defaultConfig.LogFormat == cfg.LogFormatJSON {
		logger = log.NewTMJSONLogger(log.NewSyncWriter(os.Stdout))
	}
	logger, err := tmflags.ParseLogLevel(defaultConfig.LogLevel, logger, cfg.DefaultLogLevel())
	if err != nil {
		return nil
	}
	logger = logger.With("module", "main")
	return logger
}

// initEnv sets to use ENV variables if set.
func initEnv(prefix string) {
	copyEnvVars(prefix)

	// env variables with TM prefix (eg. TM_ROOT)
	viper.SetEnvPrefix(prefix)
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.AutomaticEnv()
}

// This copies all variables like TMROOT to TM_ROOT,
// so we can support both formats for the user
func copyEnvVars(prefix string) {
	prefix = strings.ToUpper(prefix)
	ps := prefix + "_"
	for _, e := range os.Environ() {
		kv := strings.SplitN(e, "=", 2)
		if len(kv) == 2 {
			k, v := kv[0], kv[1]
			if strings.HasPrefix(k, prefix) && !strings.HasPrefix(k, ps) {
				k2 := strings.Replace(k, prefix, ps, 1)
				os.Setenv(k2, v)
			}
		}
	}
}
