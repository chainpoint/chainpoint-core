package abci

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/knq/pemutil"

	types2 "github.com/tendermint/tendermint/abci/types"

	"github.com/chainpoint/chainpoint-core/go-abci-service/types"
	"github.com/chainpoint/chainpoint-core/go-abci-service/util"
	"github.com/tendermint/tendermint/libs/log"
)

func DeclareABCI() *AnchorApplication {
	doCalLoop, _ := strconv.ParseBool(util.GetEnv("AGGREGATE", "false"))
	doAnchorLoop, _ := strconv.ParseBool(util.GetEnv("ANCHOR", "false"))
	anchorInterval, _ := strconv.Atoi(util.GetEnv("ANCHOR_BLOCK_INTERVAL", "60"))
	ethInfuraAPIKey := util.GetEnv("ETH_INFURA_API_KEY", "")
	ethereumURL := util.GetEnv("ETH_URI", fmt.Sprintf("https://ropsten.infura.io/v3/%s", ethInfuraAPIKey))
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

	store, err := pemutil.LoadFile("/run/secrets/ECDSA_PKPEM")
	if err != nil {
		util.LogError(err)
	}
	ecPrivKey, ok := store.ECPrivateKey()
	if !ok {
		util.LogError(errors.New("ecdsa key load failed"))
	}

	// Create config object
	config := types.AnchorConfig{
		DBType:         "memdb",
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

	app := NewAnchorApplication(config)
	return app
}

func sendTx(app *AnchorApplication) {
	tx := types.Tx{TxType: "CAL", Data: "test", Version: 2, Time: time.Now().Unix()}
	txEncoded := []byte(util.EncodeTx(tx, &app.config.ECPrivateKey))
	app.DeliverTx(txEncoded)
}

func TestABCIDeclaration(t *testing.T) {
	app := DeclareABCI()

	if app.Db == nil {
		t.Errorf("App state db did not initialize")
	}
}

func TestABCIDeliverTx(t *testing.T) {
	app := DeclareABCI()
	sendTx(app)
	if app.state.LatestCalTxInt != app.state.TxInt || app.state.LatestCalTxInt == 0 {
		t.Errorf("Cal tx not properly delivered and saved in app state")
	}
}

func TestABCIInfo(t *testing.T) {
	app := DeclareABCI()
	sendTx(app)
	response := app.Info(types2.RequestInfo{})
	var state types.AnchorState
	t.Logf(response.Data)
	err := json.Unmarshal([]byte(response.Data), &state)
	if err != nil {
		t.Errorf("ABCIInfo call failed: %s", err)
	} else {
		t.Logf("Successful state response: %v", state)
	}
}

func TestABCICommit(t *testing.T) {
	app := DeclareABCI()
	app.Commit()
	if app.state.Height == 0 {
		t.Logf("App State: %v\n", app.state)
		t.Errorf("ABCI Commit call failed to update app state")
	}
}
