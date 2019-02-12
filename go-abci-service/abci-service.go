package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/jasonlvhit/gocron"
	"github.com/tendermint/tendermint/rpc/client"
	types2 "github.com/tendermint/tendermint/types"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/chainpoint/chainpoint-core/go-abci-service/abci"
	"github.com/chainpoint/chainpoint-core/go-abci-service/merkletools"
	//"github.com/jasonlvhit/gocron"
	"github.com/tendermint/tendermint/abci/server"
	"github.com/tendermint/tendermint/abci/types"
	cmn "github.com/tendermint/tendermint/libs/common"
	dbm "github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/rpc/core/types"
	"github.com/chainpoint/chainpoint-aggregator-go/aggregator"
)

var proofDB *dbm.GoLevelDB
var nodeStatus core_types.ResultStatus
var netInfo core_types.ResultNetInfo
var currentCalTree merkletools.MerkleTree

func main() {
	tmServer := os.Getenv("TENDERMINT_HOST")
	tmPort := os.Getenv("TENDERMINT_PORT")

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

	// Begin scheduled methods
	go func(){
		gocron.Every(1).Minutes().Do(loopCAL, tmServer, tmPort)
		<-gocron.Start()
	}()

	// Wait forever
	cmn.TrapSignal(func() {
		// Cleanup
		srv.Stop()
	})
	return
}

func loopCAL(tmServer string, tmPort string) error {
	rpc := getHTTPClient(tmServer, tmPort)
	defer rpc.Stop()
	var agg aggregator.Aggregation
	agg.Aggregate()
	if agg.AggRoot != "" {
		tx := abci.Tx{TxType: []byte("CAL"), Data: []byte(agg.AggRoot), Version: 2, Time: time.Now().Unix()}
		txJSON, _ := json.Marshal(tx)
		params := base64.StdEncoding.EncodeToString(txJSON)
		result, err := rpc.BroadcastTxSync([]byte(params))
		if err != nil {
			return err
		}
		if result.Code == 0 {
			return nil
		}
	}
	return error.New("No hashes to aggregate")
}

func getHTTPClient(tmServer string, tmPort string) *client.HTTP {
	return client.NewHTTP(fmt.Sprintf("http://%s:%s", tmServer, tmPort), "/websocket")
}


