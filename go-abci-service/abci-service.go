package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/chainpoint/chainpoint-core/go-abci-service/lightning"

	types2 "github.com/tendermint/tendermint/types"

	"github.com/tendermint/tendermint/node"
	"github.com/tendermint/tendermint/proxy"

	"github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tendermint/privval"

	"github.com/knq/pemutil"
	"github.com/spf13/viper"

	"github.com/tendermint/tendermint/libs/log"
	"github.com/gorilla/mux"
	"github.com/chainpoint/chainpoint-core/go-abci-service/abci"
	"github.com/chainpoint/chainpoint-core/go-abci-service/types"
	"github.com/chainpoint/chainpoint-core/go-abci-service/util"
	cfg "github.com/tendermint/tendermint/config"
	tmflags "github.com/tendermint/tendermint/libs/cli/flags"
	tmos "github.com/tendermint/tendermint/libs/os"
	tmtime "github.com/tendermint/tendermint/types/time"
)

func main() {
	//Instantiate Tendermint Node Config
	tmConfig, err := initTendermintConfig()
	if util.LogError(err) != nil {
		return
	}
	logger := tmConfig.Logger

	//Instantiate ABCI application
	config := initABCIConfig(tmConfig.FilePV, tmConfig.NodeKey)
	if config.BitcoinNetwork == "mainnet" {
		config.ChainId = "mainnet-chain-32"
	}
	app := abci.NewAnchorApplication(config)

	//declare connection to abci app
	appProxy := proxy.NewLocalClientCreator(app)

	/* Instantiate Tendermint Node with given config and abci app */
	n, err := node.NewNode(tmConfig.Config,
		&tmConfig.FilePV,
		tmConfig.NodeKey,
		appProxy,
		node.DefaultGenesisDocProviderFunc(tmConfig.Config),
		node.DefaultDBProvider,
		node.DefaultMetricsProvider(tmConfig.Config.Instrumentation),
		logger,
	)
	if err != nil {
		util.LogError(err)
		return
	}

	// Wait forever, shutdown gracefully upon
	tmos.TrapSignal(*config.Logger, func() {
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

	/* /hash, /proof, /calendar, /status, /peers, /gateways/public, /boltwall/invoice, /boltwall/node */
	r := mux.NewRouter()
	r.HandleFunc("/", app.HomeHandler)
	r.HandleFunc("/hash", HashHandler)
	r.HandleFunc("/proof", ProofHandler)
	r.HandleFunc("/calendar", CalHandler)
	r.HandleFunc("/status", StatusHandler)
	r.HandleFunc("/peers", PeersHandler)
	r.HandleFunc("/gateways/public", GatewaysHandler)
	r.HandleFunc("/boltwall/invoice", BoltwallInvoiceHandler)
	r.HandleFunc("/boltwall/node", BoltwallNodeHandler)
	http.Handle("/", r)

	return
}

// initABCIConfig: receives ENV variables and initializes app config struct
func initABCIConfig(pv privval.FilePV, nodeKey *p2p.NodeKey) types.AnchorConfig {
	// Perform env type conversions
	bitcoinNetwork := util.GetEnv("NETWORK", "testnet")
	doPrivateNetwork, _ := strconv.ParseBool(util.GetEnv("PRIVATE_NETWORK", "false"))
	nodeIPs := strings.Split(util.GetEnv("PRIVATE_NODE_IPS", ""), ",")
	coreIPs := strings.Split(util.GetEnv("PRIVATE_CORE_IPS", ""), ",")
	doCalLoop, _ := strconv.ParseBool(util.GetEnv("AGGREGATE", "false"))
	doAnchorLoop, _ := strconv.ParseBool(util.GetEnv("ANCHOR", "false"))
	anchorInterval, _ := strconv.Atoi(util.GetEnv("ANCHOR_INTERVAL", "60"))
	anchorTimeout, _ := strconv.Atoi(util.GetEnv("ANCHOR_TIMEOUT", "20"))
	anchorReward, _ := strconv.Atoi(util.GetEnv("ANCHOR_REWARD", "0"))
	blockCIDRs := strings.Split(util.GetEnv("CIDR_BLOCKLIST", ""), ",")
	hashPrice, _ := strconv.Atoi(util.GetEnv("SUBMIT_HASH_PRICE_SAT", "2"))

	walletAddress := util.GetEnv("HOT_WALLET_ADDRESS", "")
	if walletAddress == "" {
		content, err := ioutil.ReadFile("/run/secrets/HOT_WALLET_ADDRESS")
		if err != nil {
			panic(err)
		}
		walletAddress = string(content)
	}

	//lightning settings
	tlsCertPath := util.GetEnv("LN_TLS_CERT", "/root/.lnd/tls.cert")
	macaroonPath := util.GetEnv("MACAROON_PATH", fmt.Sprintf("/root/.lnd/data/chain/bitcoin/%s/admin.macaroon", strings.ToLower(bitcoinNetwork)))
	lndSocket := util.GetEnv("LND_SOCKET", "lnd:10009")
	feeMultiplier, _ := strconv.ParseFloat(util.GetEnv("BTC_FEE_MULTIPLIER", "2.2"), 64)
	feeInterval, _ := strconv.Atoi(util.GetEnv("FEE_INTERVAL", "10"))

	//testMode := util.GetEnv("NETWORK", "testnet")
	tendermintRPC := types.TendermintConfig{
		TMServer: util.GetEnv("TENDERMINT_HOST", "127.0.0.1"),
		TMPort:   util.GetEnv("TENDERMINT_PORT", "26657"),
		NodeKey:  nodeKey,
	}
	postgresUser := util.GetEnv(" POSTGRES_CONNECT_USER", "chainpoint")
	postgresPw := util.GetEnv("POSTGRES_CONNECT_PW", "chainpoint")
	postgresHost := util.GetEnv("POSTGRES_CONNECT_HOST", "postgres")
	postgresPort := util.GetEnv("POSTGRES_CONNECT_PORT", "5432")
	postgresDb := util.GetEnv("POSTGRES_CONNECT_DB", "chainpoint")
	redisURI := util.GetEnv("REDIS", "redis://redis:6379")
	apiURI := util.GetEnv("API_URI", "http://api:8080")

	allowLevel, _ := log.AllowLevel(strings.ToLower(util.GetEnv("LOG_LEVEL", "info")))
	tmLogger := log.NewFilter(log.NewTMLogger(log.NewSyncWriter(os.Stdout)), allowLevel)

	store, err := pemutil.LoadFile("/run/secrets/ECDSA_PKPEM")
	if err != nil {
		util.LogError(err)
	}
	ecPrivKey, ok := store.ECPrivateKey()
	if !ok {
		util.LogError(errors.New("ecdsa key load failed"))
	}

	var blocklist []string
	blocklist, err = util.ReadLines("/go/src/github.com/chainpoint/chainpoint-core/go-abci-service/ip_blocklist.txt")
	if util.LogError(err) != nil {
		blocklist = []string{}
	}

	// Create config object
	return types.AnchorConfig{
		DBType:           "goleveldb",
		BitcoinNetwork:   bitcoinNetwork,
		RabbitmqURI:      util.GetEnv("RABBITMQ_URI", "amqp://chainpoint:chainpoint@rabbitmq:5672/"),
		TendermintConfig: tendermintRPC,
		LightningConfig: lightning.LnClient{
			TlsPath:        tlsCertPath,
			MacPath:        macaroonPath,
			ServerHostPort: lndSocket,
			Logger:         tmLogger,
			MinConfs:       3,
			Testnet:        bitcoinNetwork == "testnet",
			WalletAddress:  walletAddress,
			FeeMultiplier:  feeMultiplier,
		},
		PostgresURI:      fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", postgresUser, postgresPw, postgresHost, postgresPort, postgresDb),
		RedisURI:         redisURI,
		APIURI:           apiURI,
		ECPrivateKey:     *ecPrivKey,
		DoPrivateNetwork: doPrivateNetwork,
		PrivateNodeIPs:   nodeIPs,
		PrivateCoreIPs:   coreIPs,
		CIDRBlockList:    blockCIDRs,
		IPBlockList:      blocklist,
		DoCal:            doCalLoop,
		DoAnchor:         doAnchorLoop,
		AnchorInterval:   anchorInterval,
		Logger:           &tmLogger,
		FilePV:           pv,
		AnchorTimeout:    anchorTimeout,
		AnchorReward:     anchorReward,
		StakePerCore:     1000000,
		FeeInterval:      int64(feeInterval),
		HashPrice:        hashPrice,
	}
}

// initTendermintConfig : imports tendermint config.toml and initializes config variables
func initTendermintConfig() (types.TendermintConfig, error) {
	var TMConfig types.TendermintConfig
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
		return TMConfig, err
	}
	defaultConfig := cfg.DefaultConfig()
	err := viper.Unmarshal(defaultConfig)
	if err != nil {
		return TMConfig, err
	}
	defaultConfig.SetRoot(homeDir)
	defaultConfig.DBBackend = "cleveldb"
	defaultConfig.Consensus.TimeoutCommit = time.Duration(60 * time.Second)
	defaultConfig.RPC.TimeoutBroadcastTxCommit = time.Duration(65 * time.Second) // allows us to wait for tx to commit + 5 sec latency margin
	defaultConfig.RPC.ListenAddress = "tcp://0.0.0.0:26657"
	defaultConfig.P2P.ListenAddress = "tcp://0.0.0.0:26656"
	listenAddr := util.GetEnv("CHAINPOINT_CORE_BASE_URI", "http://0.0.0.0:26656")
	if strings.Contains(listenAddr, "//") {
		listenAddr = listenAddr[strings.LastIndex(listenAddr, "/")+1:]
	}
	if strings.Contains(listenAddr, ":") {
		listenAddr = listenAddr[:strings.LastIndex(listenAddr, ":")]
	}
	defaultConfig.P2P.ExternalAddress = listenAddr + ":26656"
	defaultConfig.P2P.MaxNumInboundPeers = 300
	defaultConfig.P2P.MaxNumOutboundPeers = 75
	defaultConfig.TxIndex.IndexAllKeys = true
	peers := []string{}
	if tendermintPeers := util.GetEnv("PEERS", ""); tendermintPeers != "" {
		peers = strings.Split(tendermintPeers, ",")
		defaultConfig.P2P.PersistentPeers = tendermintPeers
	}
	if tendermintSeeds := util.GetEnv("SEEDS", ""); tendermintSeeds != "" {
		peers = strings.Split(tendermintSeeds, ",")
		defaultConfig.P2P.Seeds = tendermintSeeds
	}
	fmt.Printf("Config : %#v\n", defaultConfig)
	cfg.EnsureRoot(defaultConfig.RootDir)

	//initialize logger
	tmlogger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))
	if defaultConfig.LogFormat == cfg.LogFormatJSON {
		tmlogger = log.NewTMJSONLogger(log.NewSyncWriter(os.Stdout))
	}
	logger, err := tmflags.ParseLogLevel(util.GetEnv("LOG_FILTER", "main:debug,state:info,*:error"), tmlogger, cfg.DefaultLogLevel())
	if err != nil {
		panic(err)
	}
	logger = logger.With("module", "main")
	TMConfig.Logger = logger
	peerGenesis := false
	// The following initializes an rpc client for a peer and pulls its genesis file
	if len(peers) != 0 {
		peer := peers[0]                    // get first peer
		nodeUri := strings.Split(peer, "@") // separate to get IP
		if len(nodeUri) == 2 {
			peerUri := strings.Split(nodeUri[1], ":") // split port from IP
			if len(peerUri) == 2 {
				peerIP := peerUri[0]
				//initialize RPC
				peerRPC := types.TendermintConfig{
					TMServer: peerIP,
					TMPort:   "26657",
				}
				rpc := abci.NewRPCClient(peerRPC, logger)
				// Pull and save genesis file
				genesis, err := rpc.GetGenesis()
				if err == nil {
					genFile := defaultConfig.GenesisFile()
					genDoc := types2.GenesisDoc{
						ChainID:         genesis.Genesis.ChainID,
						GenesisTime:     genesis.Genesis.GenesisTime,
						ConsensusParams: genesis.Genesis.ConsensusParams,
					}
					genDoc.Validators = genesis.Genesis.Validators
					if err := genDoc.SaveAs(genFile); err != nil {
						panic(err)
					} else {
						peerGenesis = true
					}
					logger.Info("Saved genesis file from peer", "path", genFile)
				}
			}
		}
	}

	// initialize private validator key
	newPrivValKey := defaultConfig.PrivValidatorKeyFile()
	newPrivValState := defaultConfig.PrivValidatorStateFile()
	if !tmos.FileExists(newPrivValState) {
		filePV := privval.GenFilePV(newPrivValKey, newPrivValState)
		filePV.LastSignState.Save()
	}
	TMConfig.FilePV = *privval.LoadOrGenFilePV(newPrivValKey, newPrivValState)

	//initialize this node's keys
	nodeKey, err := p2p.LoadOrGenNodeKey(defaultConfig.NodeKeyFile())
	TMConfig.NodeKey = nodeKey

	// initialize genesis file
	genFile := defaultConfig.GenesisFile()
	if tmos.FileExists(genFile) || peerGenesis {
		logger.Info("Found genesis file", "path", genFile)
	} else {
		genDoc := types2.GenesisDoc{
			ChainID:         fmt.Sprintf(util.GetEnv("NETWORK", "testnet")+"-chain-%d", time.Now().Second()),
			GenesisTime:     tmtime.Now(),
			ConsensusParams: types2.DefaultConsensusParams(),
		}
		key, _ := TMConfig.FilePV.GetPubKey()
		genDoc.Validators = []types2.GenesisValidator{{
			Address: key.Address(),
			PubKey:  key,
			Power:   10,
		}}
		if err := genDoc.SaveAs(genFile); err != nil {
			panic(err)
		}
		logger.Info("Generated genesis file", "path", genFile)
	}
	TMConfig.Config = defaultConfig

	return TMConfig, nil
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
