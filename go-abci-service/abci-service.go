package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/knq/pemutil"

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
	ethereumURL := util.GetEnv("ETH_URI", fmt.Sprintf("https://ropsten.infura.io/v3/%s", ethInfuraApiKey))
	ethTokenContract := util.GetEnv("TokenContractAddr", "0xB439eBe79cAeaA92C8E8813cEF14411B80bB8ef0")
	ethRegistryContract := util.GetEnv("RegistryContractAddr", "0x2Cfa392F736C1f562C5aA3D62226a29b7D1517b6")
	ethPrivateKey := util.GetEnv("ETH_PRIVATE_KEY", "")
	tendermintRPC := types.TendermintURI{
		TMServer: util.GetEnv("TENDERMINT_HOST", "tendermint"),
		TMPort:   util.GetEnv("TENDERMINT_PORT", "26657"),
	}
	postgresUser := util.GetEnv(" POSTGRES_CONNECT_USER", "chainpoint")
	postgresPw := util.GetEnv("POSTGRES_CONNECT_PW", "chainpoint")
	postgresHost := util.GetEnv("POSTGRES_CONNECT_HOST", "postgres")
	postgresPort := util.GetEnv("POSTGRES_CONNECT_PORT", "5432")
	postgresDb := util.GetEnv("POSTGRES_CONNECT_DB", "chainpoint")

	allowLevel, _ := log.AllowLevel(strings.ToLower(util.GetEnv("LOG_LEVEL", "DEBUG")))
	tmLogger := log.NewFilter(log.NewTMLogger(log.NewSyncWriter(os.Stdout)), allowLevel)

	ethConfig := types.EthConfig{
		EthereumURL:          ethereumURL,
		EthPrivateKey:        ethPrivateKey,
		TokenContractAddr:    ethTokenContract,
		RegistryContractAddr: ethRegistryContract,
	}

	store, err := pemutil.LoadFile("/run/secrets/ECDSA_KEYPAIR")
	if err != nil {
		util.LogError(err)
	}
	ecPrivKey, ok := store.ECPrivateKey()
	if !ok {
		util.LogError(errors.New("ecdsa key load failed"))
	}

	// Create config object
	config := types.AnchorConfig{
		DBType:         "goleveldb",
		RabbitmqURI:    util.GetEnv("RABBITMQ_URI", "amqp://chainpoint:chainpoint@rabbitmq:5672/"),
		TendermintRPC:  tendermintRPC,
		PostgresURI:    fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", postgresUser, postgresPw, postgresHost, postgresPort, postgresDb),
		EthConfig:      ethConfig,
		ECPrivateKey:   *ecPrivKey,
		DoCal:          doCalLoop,
		DoAnchor:       doAnchorLoop,
		AnchorInterval: anchorInterval,
		Logger:         &tmLogger,
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
