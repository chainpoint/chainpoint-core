package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/chainpoint/chainpoint-core/go-abci-service/abci"
	"github.com/chainpoint/chainpoint-core/go-abci-service/types"
	"github.com/chainpoint/chainpoint-core/go-abci-service/util"

	"github.com/tendermint/tendermint/abci/server"
	cmn "github.com/tendermint/tendermint/libs/common"
)

func main() {
	// Perform env type conversions
	doCalLoop, _ := strconv.ParseBool(util.GetEnv("AGGREGATE", "false"))
	doAnchorLoop, _ := strconv.ParseBool(util.GetEnv("ANCHOR", "false"))
	anchorInterval, _ := strconv.Atoi(util.GetEnv("ANCHOR_BLOCK_INTERVAL", "60"))
	ethInfuraApiKey := util.GetEnv("ETH_INFURA_API_KEY", "")
	ethTokenContract := util.GetEnv("TokenContractAddr", "0xC58f7d9a97bE0aC0084DBb2011Da67f36A0deD9F")
	ethRegistryContract := util.GetEnv("RegistryContractAddr", "0x5AfdE9fFFf63FF1f883405615965422889B8dF29")
	tendermintRPC := types.TendermintURI{
		TMServer: util.GetEnv("TENDERMINT_HOST", "tendermint"),
		TMPort:   util.GetEnv("TENDERMINT_PORT", "26657"),
	}
	POSTGRES_USER := util.GetEnv(" POSTGRES_CONNECT_USER", "chainpoint")
	POSTGRES_PW := util.GetEnv("POSTGRES_CONNECT_PW", "chainpoint")
	POSTGRES_HOST := util.GetEnv("POSTGRES_CONNECT_HOST", "postgres")
	POSTGRES_PORT := util.GetEnv("POSTGRES_CONNECT_PORT", "5432")
	POSTGRES_DB := util.GetEnv("POSTGRES_CONNECT_DB", "chainpoint")

	allowLevel, _ := log.AllowLevel(strings.ToLower(util.GetEnv("LOG_LEVEL", "DEBUG")))
	tmLogger := log.NewFilter(log.NewTMLogger(log.NewSyncWriter(os.Stdout)), allowLevel)

	// Create config object
	config := types.AnchorConfig{
		DBType:               "goleveldb",
		RabbitmqURI:          util.GetEnv("RABBITMQ_URI", "amqp://chainpoint:chainpoint@rabbitmq:5672/"),
		TendermintRPC:        tendermintRPC,
		PostgresURI:          fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", POSTGRES_USER, POSTGRES_PW, POSTGRES_HOST, POSTGRES_PORT, POSTGRES_DB),
		EthereumURL:          fmt.Sprintf("https://ropsten.infura.io/%s", ethInfuraApiKey),
		TokenContractAddr:    ethTokenContract,
		RegistryContractAddr: ethRegistryContract,
		DoCal:                doCalLoop,
		DoAnchor:             doAnchorLoop,
		AnchorInterval:       anchorInterval,
		Logger:               &tmLogger,
	}

	//Instantiate ABCI application
	app := abci.NewAnchorApplication(config)

	// Start the ABCI connection to the Tendermint Node
	srv, err := server.NewServer("tcp://0.0.0.0:26658", "socket", app)
	if err != nil {
		return
	}
	srv.SetLogger(tmLogger.With("module", "abci-server"))
	if err := srv.Start(); err != nil {
		return
	}

	// Wait forever
	cmn.TrapSignal(func() {
		// Cleanup
		srv.Stop()
	})
	return
}
