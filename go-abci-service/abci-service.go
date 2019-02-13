package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jasonlvhit/gocron"
	"os"
	"time"

	"github.com/chainpoint/chainpoint-core/go-abci-service/abci"
	"github.com/chainpoint/chainpoint-core/go-abci-service/aggregator"
	"github.com/chainpoint/chainpoint-core/go-abci-service/merkletools"
	//"github.com/jasonlvhit/gocron"
	"github.com/tendermint/tendermint/abci/server"
	"github.com/tendermint/tendermint/abci/types"
	cmn "github.com/tendermint/tendermint/libs/common"
	dbm "github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/rpc/core/types"
)

var proofDB *dbm.GoLevelDB
var nodeStatus abci.NodeStatus
var netInfo *core_types.ResultNetInfo
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

	for {
		var err error
		if nodeStatus, err = abci.GetStatus(tmServer, tmPort); err != nil {
			continue
		} else {
			break
		}
	}

	// Begin scheduled methods
	go func(){
		calThread := gocron.NewScheduler()
		calThread.Every(1).Minutes().Do(loopCAL, tmServer, tmPort)
		<-calThread.Start()
	}()

	go func(){
		gocron.Every(1).Minutes().Do(aggregateAndAnchorBTC, tmServer, tmPort)
		<-gocron.Start()
	}()

	// Wait forever
	cmn.TrapSignal(func() {
		// Cleanup
		srv.Stop()
	})
	return
}

func aggregateAndAnchorBTC(tmServer string, tmPort string)  {
	//iAmLeader, leader := abci.ElectLeader(tmServer, tmPort)
	//TODO: ElectLeader should check sync status of elected peer
	//TODO: Grab all transactions since 
}

func loopCAL(tmServer string, tmPort string) error {
	fmt.Println("starting scheduled aggregation")
	rpc := abci.GetHTTPClient(tmServer, tmPort)
	defer rpc.Stop()
	var agg aggregator.Aggregation
	agg.Aggregate()
	if agg.AggRoot != "" {
		fmt.Printf("Root: %s\n", agg.AggRoot)
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
	return errors.New("No hashes to aggregate")
}




