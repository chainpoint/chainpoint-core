package abci

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/streadway/amqp"

	"github.com/chainpoint/chainpoint-core/go-abci-service/types"

	"github.com/chainpoint/chainpoint-core/go-abci-service/rabbitmq"
	"github.com/chainpoint/chainpoint-core/go-abci-service/util"
)

func ConsumeBtcTxMsg(rabbitmqUri string, msgBytes []byte) error {
	var btcTxObj types.BtcTxMsg
	if err := json.Unmarshal(msgBytes, &btcTxObj); err != nil {
		return util.LogError(err)
	}
	stateObj := types.BtcTxProofState{
		AggId: btcTxObj.AggId,
		BtcId: btcTxObj.BtxId,
		BtcState: types.BtcTxOpsState{
			Ops: []types.Proof{
				types.Proof{
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
	txIdBytes, err := json.Marshal(types.TxId{TxID: btcTxObj.BtxId})
	err = rabbitmq.Publish(rabbitmqUri, "work.btcmon", "", txIdBytes)
	if err != nil {
		rabbitmq.LogError(err, "rmq dial failure, is rmq connected?")
		return err
	}
	return nil
}

func processMessage(rabbitmqUri string, rpcUri types.TendermintURI, msg amqp.Delivery) error {
	switch msg.Type {
	case "btctx":
		time.Sleep(30 * time.Second)
		BroadcastTx(rpcUri, "BTC-M", string(msg.Body), 2, time.Now().Unix())
		msg.Ack(false)
		break
	case "btcmon":
		var btcMonObj types.BtcMonMsg
		json.Unmarshal(msg.Body, &btcMonObj)
		heightAndRoot := string(btcMonObj.BtcHeadHeight) + ":" + btcMonObj.BtcHeadRoot
		_, err := BroadcastTx(rpcUri, "BTC-C", heightAndRoot, 2, time.Now().Unix())
		if util.LogError(err) != nil {
			return err
		}
		msg.Ack(false)
		break
	case "reward":
		break
	default:
		msg.Ack(false)
	}
	return nil
}

func ReceiveCalRMQ(rabbitmqUri string, rpcUri types.TendermintURI) error {
	var session rabbitmq.Session
	var err error
	endConsume := false
	for {
		session, err = rabbitmq.ConnectAndConsume(rabbitmqUri, "work.cal")
		if err != nil {
			rabbitmq.LogError(err, "failed to dial for work.cal queue")
			time.Sleep(5 * time.Second)
			continue
		}
		for {
			select {
			case err = <-session.Notify:
				if endConsume {
					return err
				}
				time.Sleep(5 * time.Second)
				break //reconnect
			case msg := <-session.Msgs:
				if len(msg.Body) > 0 {
					go processMessage(rabbitmqUri, rpcUri, msg)
				}
			}
		}
	}
}
