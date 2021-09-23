package main

import (
	"bufio"
	"crypto/elliptic"
	"errors"
	"fmt"
	"github.com/chainpoint/chainpoint-core/types"
	"github.com/knq/pemutil"
	"github.com/lightningnetwork/lnd/signal"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"


	"github.com/tendermint/tendermint/node"
	"github.com/tendermint/tendermint/proxy"

	"github.com/throttled/throttled/v2"
	"github.com/throttled/throttled/v2/store/memstore"
	"github.com/lightningnetwork/lnd"
	"github.com/jessevdk/go-flags"

	"github.com/chainpoint/chainpoint-core/abci"
	"github.com/chainpoint/chainpoint-core/util"
	"github.com/gorilla/mux"
	tmos "github.com/tendermint/tendermint/libs/os"
	"github.com/common-nighthawk/go-figure"
	"github.com/manifoldco/promptui"
	"github.com/sethvargo/go-password/password"
)

var home string

func setup(config types.AnchorConfig) {

	if _, err := os.Stat(home); os.IsNotExist(err) {
		os.MkdirAll(home, os.ModePerm)
	}

	if _, err := os.Stat(home + "/data/keys/ecdsa_key.pem"); os.IsNotExist(err) {
		os.MkdirAll(home + "/data/keys", os.ModePerm)
		st, _ := pemutil.GenerateECKeySet(elliptic.P256())
		st.WriteFile(home + "/data/keys/ecdsa_key.pem")
	}

	if _, err := os.Stat(home + "/core.conf"); os.IsNotExist(err) {
		configs := []string{}
		var seed, seedIp string
		var seedStatus types.CoreAPIStatus
		promptIp := promptui.Prompt{
			Label:    "What is this server's public IP?",
			Validate: util.ValidateIPAddress,
		}
		ipResult, err := promptIp.Run()
		if err != nil {
			panic(err)
		}
		configs = append(configs, "chainpoint_core_base_uri=http://" + ipResult)

		promptNetwork := promptui.Select{
			Label: "Select Bitcoin Network Type",
			Items: []string{"mainnet", "testnet"},
		}
		_, networkResult, err := promptNetwork.Run()
		if err != nil {
			panic(err)
		}
		configs = append(configs, "network=" + networkResult)


		promptPublic := promptui.Select{
			Label: "Will this node be joining the public Chainpoint Network or running standalone?",
			Items: []string{"Public Chainpoint Network", "Standalone Mode"},
		}
		_, publicResult, err := promptPublic.Run()
		if err != nil {
			panic(err)
		}
		configs = append(configs, "network=" + publicResult)
		if publicResult == "Public Chainpoint Network"{
			if networkResult == "mainnet" {
				seed = "24ba3a2556ebae073b42d94815836b29594a2456@18.220.31.138:26656"
				seedIp = "18.220.31.138"
			}
			if networkResult == "testnet" {
				seed = "5c285f74977733ea970ac2c66e515cc767837644@3.135.54.225:26656"
				seedIp = "3.135.54.225"
			}
			seedStatus = util.GetAPIStatus(seedIp)
			if seedStatus.TotalStakePrice != 0 {
				inBtc := float64(seedStatus.TotalStakePrice) / float64(100000000)
				stakeText := fmt.Sprintf("You will need at least %s Satoshis (%f BTC) to join the Chainpoint Network!\n", seedStatus.TotalStakePrice, inBtc)
				fmt.Printf(stakeText)
			}
			configs = append(configs, "seeds=" + seed)
		}
		if _, err := os.Stat(home + "/.lnd"); os.IsNotExist(err) {
			os.MkdirAll(home + "/.lnd", os.ModePerm)
			go runLnd(config)
			config.LightningConfig.NoMacaroons = true
			err = config.LightningConfig.WaitForConnection(5 * time.Minute)
			if err != nil {
				fmt.Println("LND not ready after 5 minutes")
				panic(err)
			}
			if len(config.LightningConfig.WalletSeed) != 24 {
				seed, err := config.LightningConfig.GenSeed()
				if err != nil {
					panic(err)
				}
				config.LightningConfig.WalletSeed = seed
			}
			if len(config.LightningConfig.WalletPass) == 0 {
				res, err := password.Generate(20, 10, 0, false, false)
				if err != nil {
					panic(err)
				}
				config.LightningConfig.WalletPass = res
			}
			err := config.LightningConfig.InitWallet()
			if err != nil {
				fmt.Sprintf("Failed to initialize lnd wallet!")
				panic(err)
			}
			configs = append(configs, "hot_wallet_pass=" + config.LightningConfig.WalletPass)
			config.LightningConfig.NoMacaroons = false
			address, err := config.LightningConfig.NewAddress()
			if err != nil {
				fmt.Printf("Failed to create new address!")
				panic(err)
			}
			config.LightningConfig.WalletAddress = address
			configs = append(configs, "hot_wallet_address=" + address)
			fmt.Printf("****************************************************\n")
			fmt.Printf("Lightning initialization has completed successfully.\n")
			fmt.Printf("****************************************************\n")
			fmt.Printf("Lightning Wallet Password: %s\n", config.LightningConfig.WalletPass)
			fmt.Printf("Lightning Wallet Seed: %s\n", strings.Join(config.LightningConfig.WalletSeed, ","))
			fmt.Printf("Lightning Wallet Address: %s\n", config.LightningConfig.WalletAddress)
			fmt.Printf("****************************************************\n")
			fmt.Printf("You should back this information up in a secure place\n")
			fmt.Printf("****************************************************\n\n")
			if publicResult == "Public Chainpoint Network" && seedStatus.TotalStakePrice != 0 {
				inBtc := float64(seedStatus.TotalStakePrice) / float64(100000000)
				stakeText := fmt.Sprintf("You will need to fund your new address with at least %s Satoshis (%f BTC) to join the Chainpoint Network!\n", seedStatus.TotalStakePrice, inBtc)
				fmt.Printf(stakeText)
			}
		}
		file, err := os.OpenFile(home + "/core.conf", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatalf("failed creating file: %s", err)
		}
		datawriter := bufio.NewWriter(file)
		for _, data := range configs {
			_, _ = datawriter.WriteString(data + "\n")
		}
		datawriter.Flush()
		file.Close()

		fmt.Printf("Chainpoint Core Setup Complete. Run with ./chainpoint-core -config $s", home + "/core.conf")
		return
	}
}

func main() {
	figure.NewColorFigure("Chainpoint Core", "colossal", "red", false).Print()
	homedirname, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	home = fmt.Sprintf("%s/.chainpoint/core", homedirname)

	//runtime.GOMAXPROCS(runtime.NumCPU() * 2)
	//Instantiate Tendermint Node Config


	//Instantiate ABCI application
	config := abci.InitConfig(home)
	if config.BitcoinNetwork == "mainnet" {
		config.ChainId = "mainnet-chain-32"
	}
	logger := config.TendermintConfig.Logger

	setup(config)

	go runLnd(config) //start lnd


	app := abci.NewAnchorApplication(config)

	//declare connection to abci app
	appProxy := proxy.NewLocalClientCreator(app)

	/* Instantiate Tendermint Node with given config and abci app */
	n, err := node.NewNode(config.TendermintConfig.Config,
		&config.TendermintConfig.FilePV,
		config.TendermintConfig.NodeKey,
		appProxy,
		node.DefaultGenesisDocProviderFunc(config.TendermintConfig.Config),
		node.DefaultDBProvider,
		node.DefaultMetricsProvider(config.TendermintConfig.Config.Instrumentation),
		logger,
	)
	if err != nil {
		panic(err)
	}

	// Wait forever, shutdown gracefully upon
	tmos.TrapSignal(*config.Logger, func() {
		if n.IsRunning() {
			app.Cache.LevelDb.Close()
			logger.Info("Shutting down Core...")
			n.Stop()
		}
	})

	// Start Tendermint Node
	if err := n.Start(); err != nil {
		panic(err)
	}
	logger.Info("Started node", "nodeInfo", n.Switch().NodeInfo())

	time.Sleep(10 * time.Second) //prevent API from blocking tendermint init

	hashStore, err := memstore.New(65536)
	apiStore, err := memstore.New(65536)
	proofStore, err := memstore.New(65536)
	if err != nil {
		panic(err)
	}

	hashQuota := throttled.RateQuota{throttled.PerMin(3), 5}
	apiQuota := throttled.RateQuota{throttled.PerSec(15), 50}
	proofQuota := throttled.RateQuota{throttled.PerSec(25), 100}
	hashLimiter, err := throttled.NewGCRARateLimiter(hashStore, hashQuota)
	apiLimiter, err := throttled.NewGCRARateLimiter(apiStore, apiQuota)
	proofLimiter, err := throttled.NewGCRARateLimiter(proofStore, proofQuota)
	if err != nil {
		panic(err)
	}

	hashRateLimiter := throttled.HTTPRateLimiter{
		RateLimiter: hashLimiter,
		VaryBy:      &throttled.VaryBy{RemoteAddr: true},
	}
	apiRateLimiter := throttled.HTTPRateLimiter{
		RateLimiter: apiLimiter,
		VaryBy:      &throttled.VaryBy{RemoteAddr: true},
	}
	proofRateLimiter := throttled.HTTPRateLimiter{
		RateLimiter: proofLimiter,
		VaryBy:      &throttled.VaryBy{RemoteAddr: true},
	}

	r := mux.NewRouter()
	r.Handle("/", apiRateLimiter.RateLimit(http.HandlerFunc(app.HomeHandler)))
	r.Handle("/hash", hashRateLimiter.RateLimit(http.HandlerFunc(app.HashHandler)))
	r.Handle("/proofs", proofRateLimiter.RateLimit(http.HandlerFunc(app.ProofHandler)))
	r.Handle("/calendar/{txid}", apiRateLimiter.RateLimit(http.HandlerFunc(app.CalHandler)))
	r.Handle("/calendar/{txid}/data", apiRateLimiter.RateLimit(http.HandlerFunc(app.CalDataHandler)))
	r.Handle("/status", apiRateLimiter.RateLimit(http.HandlerFunc(app.StatusHandler)))
	r.Handle("/peers", apiRateLimiter.RateLimit(http.HandlerFunc(app.PeerHandler)))
	r.Handle("/gateways/public", apiRateLimiter.RateLimit(http.HandlerFunc(app.GatewaysHandler)))


	server := &http.Server{
		Handler:      r,
		Addr:         ":" + config.APIPort,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	util.LogError(server.ListenAndServe())

	return
}

func runLnd(config types.AnchorConfig){
	loadedConfig, err := lnd.LoadConfig()
	if config.LightningConfig.UseChainpointConfig {
		//defaults
		loadedConfig.LndDir = "home" + "/.lnd"
		loadedConfig.LogDir = loadedConfig.LndDir + "/logs"
		loadedConfig.DataDir = loadedConfig.LndDir + "/data"
		loadedConfig.Bitcoin.Node = "neutrino"
		loadedConfig.Bitcoin.Active = true
		loadedConfig.DebugLevel = "error"
		coreIPOnly := util.GetIPOnly(config.CoreURI)
		ip, err := net.ResolveIPAddr("ip", coreIPOnly+":9735")
		if err != nil {
			panic(errors.New("Invalid IP in CoreURI"))
		}
		loadedConfig.ExternalIPs = []net.Addr{ip}
		p2p, _ := net.ResolveIPAddr("ip", "0.0.0.0:9735")
		loadedConfig.Listeners = []net.Addr{p2p}
		rest, _ := net.ResolveIPAddr("ip", "0.0.0.0:8080")
		loadedConfig.RESTListeners = []net.Addr{rest}
		rpc, _ := net.ResolveIPAddr("ip", "0.0.0.0:10009")
		loadedConfig.RPCListeners = []net.Addr{rpc}
		loadedConfig.Bitcoin.DefaultNumChanConfs = 3
		loadedConfig.TLSExtraDomains = []string{"lnd"}
		loadedConfig.TLSExtraIPs = []string{coreIPOnly}
		if config.BitcoinNetwork == "mainnet" {
			loadedConfig.Bitcoin.MainNet = true
			loadedConfig.NeutrinoMode.AddPeers = []string{"btcd-mainnet.lightning.computer", "mainnet1-btcd.zaphq.io", "mainnet2-btcd.zaphq.io", "24.155.196.246:8333","75.103.209.147:8333"}
			loadedConfig.FeeURL = "https://nodes.lightning.computer/fees/v1/btc-fee-estimates.json"
		} else if config.BitcoinNetwork == "testnet" {
			loadedConfig.Bitcoin.TestNet3 = true
			loadedConfig.NeutrinoMode.AddPeers = []string{"faucet.lightning.community:18333", "btcd-testnet.lightning.computer", "testnet1-btcd.zaphq.io", "testnet2-btcd.zaphq.io"}
		}
	}
	if err != nil {
		if e, ok := err.(*flags.Error); !ok || e.Type != flags.ErrHelp {
			// Print error if not due to help request.
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		// Help was requested, exit normally.
		os.Exit(0)
	}
	if err := lnd.Main(
		loadedConfig, lnd.ListenerCfg{}, signal.ShutdownChannel(),
	); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}


