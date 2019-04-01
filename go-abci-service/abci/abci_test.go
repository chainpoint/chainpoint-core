package abci

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	types2 "github.com/tendermint/tendermint/abci/types"

	"github.com/chainpoint/chainpoint-core/go-abci-service/types"
	"github.com/chainpoint/chainpoint-core/go-abci-service/util"
	"github.com/tendermint/tendermint/libs/log"
)

func DeclareABCI() *AnchorApplication {
	doCalLoop := false
	doAnchorLoop := false
	anchorInterval, _ := strconv.Atoi(util.GetEnv("ANCHOR_BLOCK_INTERVAL", "60"))
	tendermintRPC := types.TendermintURI{
		TMServer: util.GetEnv("TENDERMINT_HOST", "tendermint"),
		TMPort:   util.GetEnv("TENDERMINT_PORT", "26657"),
	}
	ethInfuraApiKey := util.GetEnv("ETH_INFURA_API_KEY", "")
	ethTokenContract := util.GetEnv("TokenContractAddr", "0xC58f7d9a97bE0aC0084DBb2011Da67f36A0deD9F")
	ethRegistryContract := util.GetEnv("RegistryContractAddr", "0x5AfdE9fFFf63FF1f883405615965422889B8dF29")
	POSTGRES_USER := util.GetEnv(" POSTGRES_CONNECT_USER", "chainpoint")
	POSTGRES_PW := util.GetEnv("POSTGRES_CONNECT_PW", "chainpoint")
	POSTGRES_HOST := util.GetEnv("POSTGRES_CONNECT_HOST", "postgres")
	POSTGRES_PORT := util.GetEnv("POSTGRES_CONNECT_PORT", "5432")
	POSTGRES_DB := util.GetEnv("POSTGRES_CONNECT_DB", "chainpoint")
	allowLevel, _ := log.AllowLevel(strings.ToLower(util.GetEnv("LOG_LEVEL", "DEBUG")))
	tmLogger := log.NewFilter(log.NewTMLogger(log.NewSyncWriter(os.Stdout)), allowLevel)

	// Create config object
	config := types.AnchorConfig{
		DBType:               "memdb",
		RabbitmqURI:          util.GetEnv("RABBITMQ_URI", "amqp://chainpoint:chainpoint@rabbitmq:5672/"),
		TendermintRPC:        tendermintRPC,
		PostgresURI:          fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", POSTGRES_USER, POSTGRES_PW, POSTGRES_HOST, POSTGRES_PORT, POSTGRES_DB),
		EthereumURL:          fmt.Sprintf("https://ropsten.infura.io/%s", ethInfuraApiKey),
		TokenContractAddr:    ethTokenContract,
		RegistryContractAddr: ethRegistryContract,
		DoCal:                doCalLoop,
		DoAnchor:             doAnchorLoop,
		AnchorInterval:       anchorInterval,
		Logger:               &tmLogger,
	}

	app := NewAnchorApplication(config)
	return app
}

func sendTx(app *AnchorApplication) {
	tx := types.Tx{TxType: "CAL", Data: "test", Version: 2, Time: time.Now().Unix()}
	txEncoded := []byte(util.EncodeTx(tx))
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
