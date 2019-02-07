package main

import (
	"os"

	"github.com/chainpoint/chainpoint-core/go-abci-service/abci"
	"github.com/chainpoint/merkletools-go"
	//"github.com/jasonlvhit/gocron"
	"github.com/tendermint/tendermint/abci/server"
	"github.com/tendermint/tendermint/abci/types"
	cmn "github.com/tendermint/tendermint/libs/common"
	dbm "github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/rpc/core/types"
)

var proofDB *dbm.GoLevelDB
var nodeStatus core_types.ResultStatus
var netInfo core_types.ResultNetInfo
var currentCalTree merkletools.MerkleTree

func main() {
	//tmServer := os.Getenv("TENDERMINT_HOST")
	//tmPort := os.Getenv("TENDERMINT_PORT")

	allowLevel, _ := log.AllowLevel("debug")
	logger := log.NewFilter(log.NewTMLogger(log.NewSyncWriter(os.Stdout)), allowLevel)

	/* Instantiate ABCI application */
	var app types.Application
	app = abci.NewAnchorApplication()

	currentCalTree = merkletools.MerkleTree{} //TODO: reload from storage?

	// Start the ABCI connection to the Tendermint Node
	srv, err := server.NewServer("tcp://0.0.0.0:26658", "socket", app)
	if err != nil {
		return
	}
	srv.SetLogger(logger.With("module", "abci-server"))
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


