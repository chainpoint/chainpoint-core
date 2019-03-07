package abci

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/streadway/amqp"

	"github.com/chainpoint/chainpoint-core/go-abci-service/types"

	"github.com/chainpoint/chainpoint-core/go-abci-service/rabbitmq"
	"github.com/chainpoint/chainpoint-core/go-abci-service/util"
)

// ConsumeBtcTxMsg : Consumes a btctx RMQ message to initiate monitoring on all nodes
func ConsumeBtcTxMsg(rabbitmqURI string, msgBytes []byte) error {
	var btcTxObj types.BtcTxMsg
	if err := json.Unmarshal(msgBytes, &btcTxObj); err != nil {
		return util.LogError(err)
	}
	stateObj := types.BtcTxProofState{
		AnchorBtcAggID: btcTxObj.AnchorBtcAggID,
		BtcTxID:        btcTxObj.BtcTxID,
		BtcTxState: types.BtcTxOpsState{
			Ops: []types.ProofLineItem{
				types.ProofLineItem{
					Left:  btcTxObj.BtcTxBody[:strings.Index(btcTxObj.BtcTxBody, btcTxObj.AnchorBtcAggRoot)],
					Right: btcTxObj.BtcTxBody[strings.Index(btcTxObj.BtcTxBody, btcTxObj.AnchorBtcAggRoot)+len(btcTxObj.AnchorBtcAggRoot):],
					Op:    "sha-256-x2",
				},
			},
		},
	}
	dataJSON, err := json.Marshal(stateObj)
	if util.LogError(err) != nil {
		return err
	}
	err = rabbitmq.Publish(rabbitmqURI, "work.proofstate", "btctx", dataJSON)
	if err != nil {
		rabbitmq.LogError(err, "rmq dial failure, is rmq connected?")
		return err
	}
	txIDBytes, err := json.Marshal(types.TxID{TxID: btcTxObj.BtcTxID})
	err = rabbitmq.Publish(rabbitmqURI, "work.btcmon", "", txIDBytes)
	if err != nil {
		rabbitmq.LogError(err, "rmq dial failure, is rmq connected?")
		return err
	}
	return nil
}

func processMessage(rabbitmqURI string, rpcURI types.TendermintURI, msg amqp.Delivery) error {
	switch msg.Type {
	case "btctx":
		time.Sleep(30 * time.Second)
		BroadcastTx(rpcURI, "BTC-M", string(msg.Body), 2, time.Now().Unix())
		msg.Ack(false)
		break
	case "btcmon":
		var btcMonObj types.BtcMonMsg
		json.Unmarshal(msg.Body, &btcMonObj)
		result, err := BroadcastTx(rpcURI, "BTC-C", btcMonObj.BtcHeadRoot, 2, time.Now().Unix())
		if util.LogError(err) != nil {
			if strings.Contains(err.Error(), "-32603") {
				fmt.Println("Another core has already committed a BTCC tx")
			} else {
				return err
			}
		}
		var btccStateObj types.BtccStateObj
		btccStateObj.BtcTxID = btcMonObj.BtcTxID
		btccStateObj.BtcHeadHeight = btcMonObj.BtcHeadHeight
		btccStateObj.BtcHeadState.Ops = make([]types.ProofLineItem, 0)
		for _, p := range btcMonObj.Path {
			if p.Left != "" {
				btccStateObj.BtcHeadState.Ops = append(btccStateObj.BtcHeadState.Ops, types.ProofLineItem{Left: string(p.Left)})
			}
			if p.Right != "" {
				btccStateObj.BtcHeadState.Ops = append(btccStateObj.BtcHeadState.Ops, types.ProofLineItem{Right: string(p.Right)})
			}
			btccStateObj.BtcHeadState.Ops = append(btccStateObj.BtcHeadState.Ops, types.ProofLineItem{Op: "sha-256-x2"})
		}
		baseURI := util.GetEnv("CHAINPOINT_CORE_BASE_URI", "https://tendermint.chainpoint.org")
		uri := fmt.Sprintf("%s/calendar/%x/data", baseURI, result.Hash)
		btccStateObj.BtcHeadState.Anchor = types.AnchorObj{
			AnchorID: hex.EncodeToString(result.Hash),
			Uris:     []string{uri},
		}
		stateObjBytes, err := json.Marshal(btccStateObj)

		err = rabbitmq.Publish(rabbitmqURI, "work.proofstate", "btcmon", stateObjBytes)
		if err != nil {
			rabbitmq.LogError(err, "rmq dial failure, is rmq connected?")
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

// ReceiveCalRMQ : Continually consume the calendar work queue and
// process any resulting messages from the tx and monitor services
func ReceiveCalRMQ(rabbitmqURI string, rpcURI types.TendermintURI) error {
	var session rabbitmq.Session
	var err error
	endConsume := false
	for {
		session, err = rabbitmq.ConnectAndConsume(rabbitmqURI, "work.cal")
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
					go processMessage(rabbitmqURI, rpcURI, msg)
				}
			}
		}
	}
}
