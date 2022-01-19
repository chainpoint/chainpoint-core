package main

import (
	"bufio"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/chainpoint/chainpoint-core/types"
	"github.com/knq/pemutil"
	"github.com/lightningnetwork/lnd/signal"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/tendermint/tendermint/node"
	"github.com/tendermint/tendermint/proxy"

	"github.com/lightningnetwork/lnd"
	"github.com/throttled/throttled/v2"
	"github.com/throttled/throttled/v2/store/memstore"

	"github.com/chainpoint/chainpoint-core/abci"
	"github.com/chainpoint/chainpoint-core/util"
	"github.com/common-nighthawk/go-figure"
	"github.com/gorilla/mux"
	"github.com/manifoldco/promptui"
	"github.com/sethvargo/go-password/password"
	tmos "github.com/tendermint/tendermint/libs/os"
)

var home string

func setup(config types.AnchorConfig) {

	if _, err := os.Stat(home); os.IsNotExist(err) {
		os.MkdirAll(home, os.ModePerm)
	}

	if _, err := os.Stat(home + "/data/keys/ecdsa_key.pem"); os.IsNotExist(err) {
		os.MkdirAll(home+"/data/keys", os.ModePerm)
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
		configs = append(configs, "chainpoint_core_base_uri=http://"+ipResult)
		config.CoreURI = ipResult

		promptNetwork := promptui.Select{
			Label: "Select Bitcoin Network Type",
			Items: []string{"mainnet", "testnet"},
		}
		_, networkResult, err := promptNetwork.Run()
		if err != nil {
			panic(err)
		}
		configs = append(configs, "network="+networkResult)
		config.BitcoinNetwork = networkResult

		promptPublic := promptui.Select{
			Label: "Will this node be joining the public Chainpoint Network or running standalone?",
			Items: []string{"Public Chainpoint Network", "Standalone Mode"},
		}
		_, publicResult, err := promptPublic.Run()
		if err != nil {
			panic(err)
		}
		configs = append(configs, "network="+publicResult)
		if publicResult == "Public Chainpoint Network" {
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
				stakeText := fmt.Sprintf("You will need at least %d Satoshis (%f BTC) to join the Chainpoint Network!\n", seedStatus.TotalStakePrice, inBtc)
				fmt.Printf(stakeText)
			}
			configs = append(configs, "seeds="+seed)
		}
		if _, err := os.Stat(home + "/.lnd"); os.IsNotExist(err) {
			config.LightningConfig.Testnet = config.BitcoinNetwork == "testnet"
			config.LightningConfig.MacPath = fmt.Sprintf("%s/.lnd/data/chain/bitcoin/%s/admin.macaroon", home, config.BitcoinNetwork)
			os.MkdirAll(home+"/.lnd", os.ModePerm)
			config.LightningConfig.NoMacaroons = true
			go runLnd(config)
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
			configs = append(configs, "hot_wallet_pass="+config.LightningConfig.WalletPass)
			config.LightningConfig.NoMacaroons = false
			err = config.LightningConfig.WaitForMacaroon(5 * time.Minute)
			if err != nil {
				fmt.Println("LND admin not ready after 5 minutes")
				panic(err)
			}
			address, err := config.LightningConfig.WaitForNewAddress(5 * time.Minute)
			if err != nil {
				fmt.Printf("Failed to create new address!")
				panic(err)
			}
			config.LightningConfig.WalletAddress = address
			configs = append(configs, "hot_wallet_address="+address)
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
				stakeText := fmt.Sprintf("Please fund your Lightning address with at least %d Satoshis (%f BTC) to join the Chainpoint Network!\n", seedStatus.TotalStakePrice, inBtc)
				fmt.Println(stakeText)
			}
			sessionBytes := make([]byte, 32)
			if _, err := rand.Read(sessionBytes); err != nil {
				panic(err)
			}
			configs = append(configs, "session_secret="+hex.EncodeToString(sessionBytes))
		}
		file, err := os.OpenFile(home+"/core.conf", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatalf("failed creating file: %s", err)
		}
		datawriter := bufio.NewWriter(file)
		for _, data := range configs {
			_, _ = datawriter.WriteString(data + "\n")
		}
		datawriter.Flush()
		file.Close()

		fmt.Printf("Chainpoint Core Setup Complete. Run with ./chainpoint-core -config %s\n", home+"/core.conf")
		os.Exit(0)
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

func runLnd(config types.AnchorConfig) {
	if config.LightningConfig.UseChainpointConfig {
		lndHome := home + "/.lnd"
		coreIPOnly := util.GetIPOnly(config.CoreURI)
		osArgs := []string{
			"--lnddir=" + lndHome,
			"--logdir=" + lndHome + "/logs",
			"--datadir=" + lndHome + "/data",
			"--bitcoin.active",
			"--bitcoin.node=neutrino",
			"--externalip=" + coreIPOnly + ":9735",
			"--listen=0.0.0.0:9735",
			"--restlisten=0.0.0.0:8080",
			"--rpclisten=0.0.0.0:10009",
			"--bitcoin.defaultchanconfs=3",
			"--tlsextradomain=lnd",
			"--tlsextraip=" + coreIPOnly,
			"--debuglevel=" + config.LightningConfig.LndLogLevel,
		}
		if config.BitcoinNetwork == "mainnet" {
			osArgs = append(osArgs, "--feeurl=https://nodes.lightning.computer/fees/v1/btc-fee-estimates.json")
			osArgs = append(osArgs, []string{"btcd-mainnet.lightning.computer", "mainnet1-btcd.zaphq.io", "mainnet2-btcd.zaphq.io", "24.155.196.246:8333", "75.103.209.147:8333"}...)
			osArgs = append(osArgs, "--bitcoin.mainnet")
			osArgs = append(osArgs, "--routing.assumechanvalid")
		} else if config.BitcoinNetwork == "testnet" {
			osArgs = append(osArgs, "--bitcoin.testnet")
			osArgs = append(osArgs, []string{"faucet.lightning.community:18333", "btcd-testnet.lightning.computer", "testnet1-btcd.zaphq.io", "testnet2-btcd.zaphq.io"}...)
		}
		os.Args = append(os.Args, osArgs...)
	}
	shutdownInterceptor, err := signal.Intercept()
	if err != nil {
		panic(err)
	}
	loadedConfig, err := lnd.LoadConfig(shutdownInterceptor)
	if err != nil {
		panic(err)
	}
	if err := lnd.Main(
		loadedConfig, lnd.ListenerCfg{}, loadedConfig.ImplementationConfig(shutdownInterceptor), shutdownInterceptor,
	); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
