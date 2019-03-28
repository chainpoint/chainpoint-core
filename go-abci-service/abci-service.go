package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/chainpoint/chainpoint-core/go-abci-service/ethcontracts"

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
	tendermintRPC := types.TendermintURI{
		TMServer: util.GetEnv("TENDERMINT_HOST", "tendermint"),
		TMPort:   util.GetEnv("TENDERMINT_PORT", "26657"),
	}

	allowLevel, _ := log.AllowLevel(strings.ToLower(util.GetEnv("LOG_LEVEL", "DEBUG")))
	tmLogger := log.NewFilter(log.NewTMLogger(log.NewSyncWriter(os.Stdout)), allowLevel)

	// Create config object
	config := types.AnchorConfig{
		DBType:         "goleveldb",
		RabbitmqURI:    util.GetEnv("RABBITMQ_URI", "amqp://chainpoint:chainpoint@rabbitmq:5672/"),
		TendermintRPC:  tendermintRPC,
		DoCal:          doCalLoop,
		DoAnchor:       doAnchorLoop,
		AnchorInterval: anchorInterval,
		Logger:         &tmLogger,
	}

	_, err := ethcontracts.NewClient(fmt.Sprintf("https://ropsten.infura.io/%s", ethInfuraApiKey),
		"0xC58f7d9a97bE0aC0084DBb2011Da67f36A0deD9F",
		"0x5AfdE9fFFf63FF1f883405615965422889B8dF29",
		tmLogger)
	/*	util.LoggerError(tmLogger, err)
		nodesStaked, err := ethClient.GetPastNodesStakedEvents()
		util.LoggerError(tmLogger, err)
		fmt.Printf("Node Stake Events: %#v\n", nodesStaked)*/

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
