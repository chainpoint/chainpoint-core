package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/chainpoint/chainpoint-core/go-abci-service/aggregator"
	"github.com/chainpoint/chainpoint-core/go-abci-service/calendar"
	"github.com/chainpoint/chainpoint-core/go-abci-service/util"
	"github.com/jasonlvhit/gocron"

	"github.com/chainpoint/chainpoint-core/go-abci-service/abci"
	"github.com/chainpoint/chainpoint-core/go-abci-service/merkletools"

	//"github.com/jasonlvhit/gocron"
	"github.com/tendermint/tendermint/abci/server"
	"github.com/tendermint/tendermint/abci/types"
	cmn "github.com/tendermint/tendermint/libs/common"
	dbm "github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/libs/log"
	core_types "github.com/tendermint/tendermint/rpc/core/types"
)

var proofDB *dbm.GoLevelDB
var nodeStatus abci.NodeStatus
var netInfo *core_types.ResultNetInfo
var currentCalTree merkletools.MerkleTree
var tendermintRPC abci.TendermintURI

func main() {
	tmServer := util.GetEnv("TENDERMINT_HOST", "tendermint")
	tmPort := util.GetEnv("TENDERMINT_PORT", "26657")
	rabbitmqUri := util.GetEnv("RABBITMQ_URI", "amqp://chainpoint:chainpoint@rabbitmq:5672/")

	tendermintRPC = abci.TendermintURI{
		TMServer: tmServer,
		TMPort:   tmPort,
	}

	allowLevel, _ := log.AllowLevel("debug")
	logger := log.NewFilter(log.NewTMLogger(log.NewSyncWriter(os.Stdout)), allowLevel)

	/* Instantiate ABCI application */
	var app types.Application
	app = abci.NewAnchorApplication()

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

	// Begin scheduled methods
	go func() {
		calThread := gocron.NewScheduler()
		calThread.Every(1).Minutes().Do(loopCAL, tendermintRPC, rabbitmqUri)
		<-calThread.Start()
	}()

	go func() {
		gocron.Every(2).Minutes().Do(loopAnchor, tendermintRPC, rabbitmqUri)
		<-gocron.Start()
	}()

	// Wait forever
	cmn.TrapSignal(func() {
		// Cleanup
		srv.Stop()
	})
	return
}

/* Scans all CAL transactions since last anchor epoch and writes the merkle root to the Calendar and to bitcoin */
func loopAnchor(tendermintRPC abci.TendermintURI, rabbitmqUri string) error {
	iAmLeader, _ := abci.ElectLeader(tendermintRPC)
	if !iAmLeader {
		return nil //bail if we aren't the leader
	}
	fmt.Println("starting scheduled anchor")
	rpc := abci.GetHTTPClient(tendermintRPC)
	defer rpc.Stop()
	state, err := abci.GetAbciInfo(tendermintRPC)
	if util.LogError(err) != nil {
		return err
	}
	/* Get CAL transactions between the latest BTCA tx and the current latest tx */
	txLeaves, err := abci.GetTxRange(tendermintRPC, state.LatestBtcaTxInt+1, state.TxInt)
	fmt.Printf("leaves for current anchor: %#v\n", txLeaves)
	if util.LogError(err) != nil {
		return err
	}
	treeData := calendar.AggregateAndAnchorBTC(txLeaves)
	btca := abci.Tx{TxType: []byte("BTC-A"), Data: []byte(treeData.AggRoot), Version: 2, Time: time.Now().Unix()}
	txJSON, _ := json.Marshal(btca)
	params := base64.StdEncoding.EncodeToString(txJSON)
	result, err := rpc.BroadcastTxSync([]byte(params))
	if util.LogError(err) != nil {
		return err
	}
	fmt.Printf("Anchor result: %v\n", result)
	if result.Code == 0 {
		var tx abci.TxTm
		tx.Hash = result.Hash.Bytes()
		tx.Data = result.Data.Bytes()
		//calendar.queueBtcAStateDataMessage(rabbitmqUri, tx)
		return nil
	}
	//TODO: ElectLeader should check sync status of elected peer
	//TODO: Grab all transactions since
	return nil
}

/* Aggregate submitted hashes into a calendar transaction */
func loopCAL(tendermintRPC abci.TendermintURI, rabbitmqUri string) error {
	fmt.Println("starting scheduled aggregation")
	rpc := abci.GetHTTPClient(tendermintRPC)
	defer rpc.Stop()
	var agg aggregator.Aggregation
	agg.Aggregate(rabbitmqUri)
	var calendar calendar.CalAgg
	// Because there is a 1 : 1 calendar/aggregation interval, there is only one  root here
	calendar.GenerateCalendarTree([]aggregator.Aggregation{agg})
	if agg.AggRoot != "" {
		fmt.Printf("Root: %s\n", agg.AggRoot)
		tx := abci.Tx{TxType: []byte("CAL"), Data: []byte(agg.AggRoot), Version: 2, Time: time.Now().Unix()}
		txJSON, _ := json.Marshal(tx)
		params := base64.StdEncoding.EncodeToString(txJSON)
		result, err := rpc.BroadcastTxSync([]byte(params))
		if util.LogError(err) != nil {
			return err
		}
		fmt.Printf("CAL result: %v\n", result)
		if result.Code == 0 {
			var tx abci.TxTm
			tx.Hash = result.Hash.Bytes()
			tx.Data = result.Data.Bytes()
			calendar.QueueCalStateMessage(rabbitmqUri, tx)
			return nil
		}
	}
	return errors.New("No hashes to aggregate")
}
