package abci

import (
	"encoding/json"
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
	allowLevel, _ := log.AllowLevel(strings.ToLower(util.GetEnv("LOG_LEVEL", "DEBUG")))
	tmLogger := log.NewFilter(log.NewTMLogger(log.NewSyncWriter(os.Stdout)), allowLevel)

	// Create config object
	config := types.AnchorConfig{
		DBType:         "memdb",
		RabbitmqURI:    util.GetEnv("RABBITMQ_URI", "amqp://chainpoint:chainpoint@rabbitmq:5672/"),
		TendermintRPC:  tendermintRPC,
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
