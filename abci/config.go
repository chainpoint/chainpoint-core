package abci

import (
	"errors"
	"fmt"
	"github.com/chainpoint/chainpoint-core/lightning"
	"github.com/chainpoint/chainpoint-core/tendermint_rpc"
	"github.com/chainpoint/chainpoint-core/types"
	"github.com/chainpoint/chainpoint-core/util"
	"github.com/jacohend/flag"
	"github.com/knq/pemutil"
	"github.com/spf13/viper"
	cfg "github.com/tendermint/tendermint/config"
	tmflags "github.com/tendermint/tendermint/libs/cli/flags"
	"github.com/tendermint/tendermint/libs/log"
	tmos "github.com/tendermint/tendermint/libs/os"
	"github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tendermint/privval"
	types2 "github.com/tendermint/tendermint/types"
	tmtime "github.com/tendermint/tendermint/types/time"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// initConfig: receives ENV variables and initializes app config struct
func InitConfig(home string) types.AnchorConfig {

	// Perform env type conversions
	var proposedValidator string
	var listenAddr, tendermintPeers, tendermintSeeds, tendermintLogFilter, lndLogFilter string
	var bitcoinNetwork, walletAddress, walletPass, walletSeed, secretKeyPath, aggregatorAllowStr, blockCIDRStr, apiPort string
	var tlsCertPath, macaroonPath, lndSocket, electionMode, sessionSecret, tmServer, tmPort string
	var coreName, analyticsID, logLevel string
	var feeMultiplier float64
	var anchorInterval, anchorTimeout, anchorReward, hashPrice, feeInterval, stakePerCore, updateStake int
	var useAggregatorAllowlist, doCalLoop, doAnchorLoop, useChpLndConfig bool
	flag.String(flag.DefaultConfigFlagname, "", "path to config file")
	flag.StringVar(&bitcoinNetwork, "network", "mainnet", "bitcoin network")
	flag.BoolVar(&useAggregatorAllowlist, "aggregator_public", false, "use aggregator allow list")
	flag.StringVar(&aggregatorAllowStr, "aggregator_whitelist", "", "prevent whitelisted IPs from needing to pay invoices")
	flag.BoolVar(&doCalLoop, "aggregate", true, "whether to submit calendar transactions to Chainpoint Calendar")
	flag.BoolVar(&doAnchorLoop, "anchor", true, "whether to participate in bitcoin anchoring elections")
	flag.StringVar(&electionMode, "election", "reputation", "mode for leader election")
	flag.IntVar(&anchorInterval, "anchor_interval", 60, "interval to use for bitcoin anchoring")
	flag.IntVar(&anchorTimeout, "anchor_timeout", 20, "timeout use for bitcoin anchoring")
	flag.IntVar(&anchorReward, "anchor_reward", 0, "reward for cores that anchor")
	flag.IntVar(&hashPrice, "submit_hash_price_sat", 2, "cost in satoshis for non-whitelisted gateways to submit a hash")
	flag.StringVar(&blockCIDRStr, "cidr_blocklist", "", "comma-delimited list of IPs to block")
	flag.StringVar(&proposedValidator, "propose_validator", "", "propose the promotion of a core to validator")

	//lightning settings
	flag.StringVar(&walletAddress, "hot_wallet_address", "", "birthday address for lnd account")
	flag.StringVar(&walletPass, "hot_wallet_pass", "", "hot wallet password")
	flag.StringVar(&walletSeed, "hot_wallet_seed", "", "hot wallet seed phrase")
	flag.StringVar(&macaroonPath, "macaroon_path", "", "path to lnd admin macaroon")
	flag.StringVar(&tlsCertPath, "ln_tls_path", fmt.Sprintf("%s/.lnd/tls.cert", home), "path to lnd tls certificate")
	flag.StringVar(&lndSocket, "lnd_socket", "127.0.0.1:10009", "url to lnd grpc server")
	flag.StringVar(&lndLogFilter, "lnd_log_level", "error", "log level for lnd")
	flag.BoolVar(&useChpLndConfig, "chainpoint_lnd_config", true, "whether to use chainpoint's default lnd config")
	flag.Float64Var(&feeMultiplier, "btc_fee_multiplier", 2.2, "multiply anchoring fee by this constant when mempool is congested")
	flag.IntVar(&feeInterval, "fee_interval", 10, "interval in minutes to check for new bitcoin tx fee")
	flag.IntVar(&stakePerCore, "stake_per_core", 1000000, "minimum amount staked per channel to permit the addition of a Core")
	flag.IntVar(&updateStake, "update_stake", 0, "a validator may change this value to adjust the stake_per_core")
	flag.StringVar(&sessionSecret, "session_secret", "", "mutual LSAT macaroon secret for cores and gateways")
	flag.StringVar(&tmServer, "tendermint_host", "127.0.0.1", "tendermint api url")
	flag.StringVar(&tmPort, "tendermint_port", "26657", "tendermint api port")
	flag.StringVar(&apiPort, "api_port", "80", "core api port")
	flag.StringVar(&coreName, "chainpoint_core_name", "", "core Name")
	flag.StringVar(&analyticsID, "google_ua_id", "", "google analytics id")
	flag.StringVar(&logLevel, "log_level", "info", "log level")
	flag.StringVar(&secretKeyPath, "secret_key_path", home+"/data/keys/ecdsa_key.pem", "path to ECDSA secret key")
	flag.StringVar(&listenAddr, "chainpoint_core_base_uri", "http://0.0.0.0:26656", "tendermint base uri")
	flag.StringVar(&tendermintPeers, "peers", "", "comma-delimited list of peers")
	flag.StringVar(&tendermintSeeds, "seeds", "", "comma-delimited list of seeds")
	flag.StringVar(&tendermintLogFilter, "log_filter", "main:debug,state:info,*:error", "log level for tendermint")
	flag.Parse()
	aggregatorAllowlist := strings.Split(aggregatorAllowStr, ",")
	blockCIDRs := strings.Split(blockCIDRStr, ",")
	/*	if walletAddress == "" {
		content, err := ioutil.ReadFile("/run/secrets/HOT_WALLET_ADDRESS")
		if err != nil {
			panic(err)
		}
		walletAddress = string(content)
	}*/
	if walletSeed != "" && len(strings.Split(walletSeed, ",")) != 24 {
		panic(errors.New("Provided wallet seed is not the required 24 words"))
	}
	if macaroonPath == "" {
		macaroonPath = fmt.Sprintf("%s/.lnd/data/chain/bitcoin/%s/admin.macaroon", home, strings.ToLower(bitcoinNetwork))
	}

	tmConfig, err := initTendermintConfig(home, bitcoinNetwork, listenAddr, tendermintSeeds, tendermintPeers, tendermintLogFilter)
	if util.LogError(err) != nil {
		panic(err)
	}
	tmConfig.TMServer = tmServer
	tmConfig.TMPort = tmPort

	if len(coreName) == 0 {
		coreName = listenAddr
	}

	allowLevel, _ := log.AllowLevel(strings.ToLower(logLevel))
	tmLogger := log.NewFilter(log.NewTMLogger(log.NewSyncWriter(os.Stdout)), allowLevel)

	store, err := pemutil.LoadFile(secretKeyPath)
	if err != nil {
		util.LogError(err)
	}
	ecPrivKey, ok := store.ECPrivateKey()
	if !ok {
		util.LogError(errors.New("ecdsa key load failed"))
	}

	var blocklist []string
	blocklist, err = util.ReadLines(home + "/ip_blocklist.txt")
	if util.LogError(err) != nil {
		blocklist = []string{}
	}

	// Create config object
	return types.AnchorConfig{
		HomePath:         home,
		APIPort:          apiPort,
		DBType:           "goleveldb",
		BitcoinNetwork:   bitcoinNetwork,
		ElectionMode:     electionMode,
		TendermintConfig: tmConfig,
		LightningConfig: lightning.LnClient{
			TlsPath:             tlsCertPath,
			MacPath:             macaroonPath,
			ServerHostPort:      lndSocket,
			Logger:              tmLogger,
			LndLogLevel:         lndLogFilter,
			MinConfs:            3,
			Testnet:             bitcoinNetwork == "testnet",
			WalletAddress:       walletAddress,
			WalletPass:          walletPass,
			WalletSeed:          strings.Split(walletSeed, ","),
			HashPrice:           int64(hashPrice),
			SessionSecret:       sessionSecret,
			UseChainpointConfig: useChpLndConfig,
		},
		ECPrivateKey:     ecPrivKey,
		CIDRBlockList:    blockCIDRs,
		IPBlockList:      blocklist,
		DoCal:            doCalLoop,
		DoAnchor:         doAnchorLoop,
		AnchorInterval:   anchorInterval,
		Logger:           &tmLogger,
		FilePV:           tmConfig.FilePV,
		AnchorTimeout:    anchorTimeout,
		AnchorReward:     anchorReward,
		StakePerCore:     1000000,
		FeeInterval:      int64(feeInterval),
		FeeMultiplier:    feeMultiplier,
		HashPrice:        hashPrice,
		UseAllowlist:     useAggregatorAllowlist,
		GatewayAllowlist: aggregatorAllowlist,
		CoreURI:          listenAddr,
		CoreName:         coreName,
		AnalyticsID:      analyticsID,
		ProposedVal:      proposedValidator,
	}
}

// initTendermintConfig : imports tendermint config.toml and initializes config variables
func initTendermintConfig(home string, network string, listenAddr string, tendermintPeers string, tendermintSeeds string, tendermintLogFilter string) (types.TendermintConfig, error) {
	var TMConfig types.TendermintConfig
	initEnv("TM")
	homeFlag := os.ExpandEnv(filepath.Join("$HOME", cfg.DefaultTendermintDir))
	homeDir := home
	viper.Set(homeFlag, homeDir)
	viper.SetConfigName("config")            // name of config file (without extension)
	viper.AddConfigPath(homeDir + "/config") // search root directory /config

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		// stderr, so if we redirect output to json file, this doesn't appear
		// fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	} else if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
		fmt.Sprintf("Config File Not Found, err: $s", err.Error())
		// ignore not found error, return other errors
		return TMConfig, err
	}
	defaultConfig := cfg.DefaultConfig()
	err := viper.Unmarshal(defaultConfig)
	if err != nil {
		return TMConfig, err
	}
	defaultConfig.SetRoot(homeDir)
	defaultConfig.DBPath = homeDir + "/data"
	defaultConfig.DBBackend = "cleveldb"
	defaultConfig.Consensus.TimeoutCommit = time.Duration(60 * time.Second)
	defaultConfig.RPC.TimeoutBroadcastTxCommit = time.Duration(65 * time.Second) // allows us to wait for tx to commit + 5 sec latency margin
	defaultConfig.RPC.ListenAddress = "tcp://0.0.0.0:26657"
	defaultConfig.P2P.ListenAddress = "tcp://0.0.0.0:26656"

	ipOnly := util.GetIPOnly(listenAddr)
	defaultConfig.P2P.ExternalAddress = ipOnly + ":26656"
	defaultConfig.P2P.MaxNumInboundPeers = 300
	defaultConfig.P2P.MaxNumOutboundPeers = 75
	defaultConfig.TxIndex.IndexAllKeys = true
	peers := []string{}
	if tendermintPeers != "" {
		peers = strings.Split(tendermintPeers, ",")
		defaultConfig.P2P.PersistentPeers = tendermintPeers
	}
	if tendermintSeeds != "" {
		peers = strings.Split(tendermintSeeds, ",")
		defaultConfig.P2P.Seeds = tendermintSeeds
	}
	//fmt.Printf("Config : %#v\n", defaultConfig)
	cfg.EnsureRoot(defaultConfig.RootDir)

	//initialize logger
	tmlogger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))
	if defaultConfig.LogFormat == cfg.LogFormatJSON {
		tmlogger = log.NewTMJSONLogger(log.NewSyncWriter(os.Stdout))
	}
	logger, err := tmflags.ParseLogLevel(tendermintLogFilter, tmlogger, cfg.DefaultLogLevel())
	if err != nil {
		panic(err)
	}
	logger = logger.With("module", "main")
	TMConfig.Logger = logger
	peerGenesisFound := false
	peersOrSeedsExist := len(peers) != 0
	// The following initializes an rpc client for a peer and pulls its genesis file
	if peersOrSeedsExist {
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
				rpc := tendermint_rpc.NewRPCClient(peerRPC, logger)
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
						peerGenesisFound = true
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
	if tmos.FileExists(genFile) || peerGenesisFound {
		logger.Info("Found genesis file", "path", genFile)
	} else if !peersOrSeedsExist {
		panic(errors.New("Can't retrieve Genesis File from Seed- check firewall on both ends"))
	} else {
		genDoc := types2.GenesisDoc{
			ChainID:         fmt.Sprintf(network+"-chain-%d", time.Now().Second()),
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
