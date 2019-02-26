package main

import (
	"os"
	"strconv"

	"github.com/chainpoint/chainpoint-core/go-abci-service/abci"
	"github.com/chainpoint/chainpoint-core/go-abci-service/merkletools"
	"github.com/chainpoint/chainpoint-core/go-abci-service/types"
	"github.com/chainpoint/chainpoint-core/go-abci-service/util"

	"github.com/tendermint/tendermint/abci/server"
	cmn "github.com/tendermint/tendermint/libs/common"
	dbm "github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/libs/log"
	core_types "github.com/tendermint/tendermint/rpc/core/types"
)

var proofDB *dbm.GoLevelDB
var nodeStatus types.NodeStatus
var netInfo *core_types.ResultNetInfo
var currentCalTree merkletools.MerkleTree
var tendermintRPC types.TendermintURI
var rabbitmqUri string

func main() {
	tmServer := util.GetEnv("TENDERMINT_HOST", "tendermint")
	tmPort := util.GetEnv("TENDERMINT_PORT", "26657")
	rabbitmqUri = util.GetEnv("RABBITMQ_URI", "amqp://chainpoint:chainpoint@rabbitmq:5672/")
	//doCalLoop, _ := strconv.ParseBool(util.GetEnv("AGGREGATE", "false"))
	doAnchorLoop, _ := strconv.ParseBool(util.GetEnv("ANCHOR", "false"))

	tendermintRPC = types.TendermintURI{
		TMServer: tmServer,
		TMPort:   tmPort,
	}

	allowLevel, _ := log.AllowLevel("debug")
	logger := log.NewFilter(log.NewTMLogger(log.NewSyncWriter(os.Stdout)), allowLevel)

	/* Instantiate ABCI application */
	app := abci.NewAnchorApplication(rabbitmqUri, tendermintRPC, doAnchorLoop, 3)

	// Start the ABCI connection to the Tendermint Node
	srv, err := server.NewServer("tcp://0.0.0.0:26658", "socket", app)
	if err != nil {
		return
	}
	srv.SetLogger(logger.With("module", "abci-server"))
	if err := srv.Start(); err != nil {
		return
	}

	for {
		var err error
		if nodeStatus, err = abci.GetStatus(tendermintRPC); err != nil {
			continue
		} else {
			break
		}
	}

	// Infinite loop to process btctx and btcmon rabbitMQ messages
	if doAnchorLoop {
		go abci.ReceiveCalRMQ(rabbitmqUri, tendermintRPC)
	}

	// Wait forever
	cmn.TrapSignal(func() {
		// Cleanup
		srv.Stop()
	})
	return
}
