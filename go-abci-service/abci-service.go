package main

import (
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
		DBType:         "goleveldb",
		RabbitmqURI:    util.GetEnv("RABBITMQ_URI", "amqp://chainpoint:chainpoint@rabbitmq:5672/"),
		TendermintRPC:  tendermintRPC,
		DoCal:          doCalLoop,
		DoAnchor:       doAnchorLoop,
		AnchorInterval: anchorInterval,
		Logger:         &tmLogger,
	}

	/*	pgClient, err := postgres.NewPG(POSTGRES_USER, POSTGRES_PW, POSTGRES_HOST, POSTGRES_PORT, POSTGRES_DB, tmLogger)
		util.LoggerError(tmLogger, err)
		ethClient, err := ethcontracts.NewClient(fmt.Sprintf("https://ropsten.infura.io/%s", ethInfuraApiKey),
			"0xC58f7d9a97bE0aC0084DBb2011Da67f36A0deD9F",
			"0x5AfdE9fFFf63FF1f883405615965422889B8dF29",
			tmLogger)
		util.LoggerError(tmLogger, err)
		nodesStaked, err := ethClient.GetPastNodesStakedEvents()
		util.LoggerError(tmLogger, err)
		for _, node := range nodesStaked {
			newNode := types.Node{
				EthAddr:         node.Sender.Hex(),
				PublicIP:        sql.NullString{String: util.BytesToIP(node.NodeIp[:]), Valid: true},
				AmountStaked:    sql.NullInt64{Int64: node.AmountStaked.Int64(), Valid: true},
				StakeExpiration: sql.NullInt64{Int64: node.Duration.Int64(), Valid: true},
				BlockNumber:     sql.NullInt64{Int64: int64(node.Raw.BlockNumber), Valid: true},
			}
			inserted, err := pgClient.NodeUpsert(newNode)
			util.LoggerError(tmLogger, err)
			fmt.Printf("Inserted for %#v: %t\n", newNode, inserted)
		}
		nodesStakedUpdated, err := ethClient.GetPastNodesStakeUpdatedEvents()
		util.LoggerError(tmLogger, err)
		for _, node := range nodesStakedUpdated {
			newNode := types.Node{
				EthAddr:         node.Sender.Hex(),
				PublicIP:        sql.NullString{String: util.BytesToIP(node.NodeIp[:]), Valid: true},
				AmountStaked:    sql.NullInt64{Int64: node.AmountStaked.Int64(), Valid: true},
				StakeExpiration: sql.NullInt64{Int64: node.Duration.Int64(), Valid: true},
				BlockNumber:     sql.NullInt64{Int64: int64(node.Raw.BlockNumber), Valid: true},
			}
			inserted, err := pgClient.NodeUpsert(newNode)
			util.LoggerError(tmLogger, err)
			fmt.Printf("Inserted Update for %#v: %t\n", newNode, inserted)
		}
		retrievedNode, err := pgClient.GetNodeByEthAddr("0xc6a7897cc8F2e3B294844A07165573C6194324aB")
		util.LoggerError(tmLogger, err)
		fmt.Printf("Retrieved for %#v\n", retrievedNode)*/

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
