package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/chainpoint/chainpoint-core/go-abci-service/abci"
	"github.com/chainpoint/chainpoint-core/go-abci-service/aggregator"
	"github.com/chainpoint/chainpoint-core/go-abci-service/calendar"
	"github.com/chainpoint/chainpoint-core/go-abci-service/merkletools"
	"github.com/chainpoint/chainpoint-core/go-abci-service/util"

	//"github.com/jasonlvhit/gocron"
	"github.com/tendermint/tendermint/abci/server"
	"github.com/tendermint/tendermint/abci/types"
	cmn "github.com/tendermint/tendermint/libs/common"
	dbm "github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/libs/log"
	core_types "github.com/tendermint/tendermint/rpc/core/types"
	cron "gopkg.in/robfig/cron.v3"
)

var proofDB *dbm.GoLevelDB
var nodeStatus abci.NodeStatus
var netInfo *core_types.ResultNetInfo
var currentCalTree merkletools.MerkleTree
var tendermintRPC abci.TendermintURI
var rabbitmqUri string

func main() {
	tmServer := util.GetEnv("TENDERMINT_HOST", "tendermint")
	tmPort := util.GetEnv("TENDERMINT_PORT", "26657")
	rabbitmqUri = util.GetEnv("RABBITMQ_URI", "amqp://chainpoint:chainpoint@rabbitmq:5672/")
	doCalLoop, _ := strconv.ParseBool(util.GetEnv("AGGREGATE", "false"))
	doAnchorLoop, _ := strconv.ParseBool(util.GetEnv("ANCHOR", "false"))

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

	scheduler := cron.New(cron.WithLocation(time.UTC))

	// Begin scheduled methods
	if doCalLoop {
		scheduler.AddFunc("0/1 0-23 * * *", func() { loopCAL() })
	}

	if doAnchorLoop {
		scheduler.AddFunc("0/3 0-23 * * *", func() { loopAnchor() })
		go calendar.ReceiveCalRMQ(rabbitmqUri, tendermintRPC)
	}

	scheduler.Start()

	// Wait forever
	cmn.TrapSignal(func() {
		// Cleanup
		srv.Stop()
	})
	return
}

/* Scans all CAL transactions since last anchor epoch and writes the merkle root to the Calendar and to bitcoin */
func loopAnchor() error {
	iAmLeader, _ := abci.ElectLeader(tendermintRPC)
	fmt.Println("starting scheduled anchor")
	rpc := abci.GetHTTPClient(tendermintRPC)
	defer rpc.Stop()
	state, err := abci.GetAbciInfo(tendermintRPC)
	if util.LogError(err) != nil {
		return err
	}
	/* Get CAL transactions between the latest BTCA tx and the current latest tx */
	txLeaves, err := abci.GetTxRange(tendermintRPC, state.LatestBtcaTxInt, state.LatestCalTxInt)
	if util.LogError(err) != nil {
		return err
	}
	treeData := calendar.AggregateAnchorTx(txLeaves)
	fmt.Printf("treeData for current anchor: %v\n", treeData)
	if treeData.AggRoot != "" {
		if iAmLeader {
			result, err := abci.BroadcastTx(tendermintRPC, "BTC-A", treeData.AggRoot, 2, time.Now().Unix())
			if util.LogError(err) != nil {
				return err
			}
			fmt.Printf("Anchor result: %v\n", result)
		}
		treeData.QueueBtcaStateDataMessage(rabbitmqUri, iAmLeader)
		return nil
	}
	return errors.New("no transactions to aggregate")
}

/* Aggregate submitted hashes into a calendar transaction */
func loopCAL() error {
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
		result, err := abci.BroadcastTx(tendermintRPC, "CAL", agg.AggRoot, 2, time.Now().Unix())
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
