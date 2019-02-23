package abci

import (
	"encoding/json"
	"strings"

	"github.com/chainpoint/chainpoint-core/go-abci-service/rabbitmq"
	"github.com/chainpoint/chainpoint-core/go-abci-service/util"
)

func ConsumeBtcTxMsg(rabbitmqUri string, msgBytes []byte) error {
	var btcTxObj BtcTxMsg
	if err := json.Unmarshal(msgBytes, &btcTxObj); err != nil {
		return util.LogError(err)
	}
	stateObj := BtcTxProofState{
		AggId: btcTxObj.AggId,
		BtcId: btcTxObj.BtxId,
		BtcState: BtcTxOpsState{
			Ops: []Proof{
				Proof{
					Left:  btcTxObj.BtxBody[:strings.Index(btcTxObj.BtxBody, btcTxObj.AggRoot)],
					Right: btcTxObj.BtxBody[strings.Index(btcTxObj.BtxBody, btcTxObj.AggRoot)+len(btcTxObj.AggRoot):],
					Op:    "sha-256-x2",
				},
			},
		},
	}
	dataJSON, err := json.Marshal(stateObj)
	if util.LogError(err) != nil {
		return err
	}
	err = rabbitmq.Publish(rabbitmqUri, "work.proofstate", "btctx", dataJSON)
	if err != nil {
		rabbitmq.LogError(err, "rmq dial failure, is rmq connected?")
		return err
	}
	txIdBytes, err := json.Marshal(TxId{TxID: btcTxObj.BtxId})
	err = rabbitmq.Publish(rabbitmqUri, "work.btcmon", "", txIdBytes)
	if err != nil {
		rabbitmq.LogError(err, "rmq dial failure, is rmq connected?")
		return err
	}
	return nil
}
