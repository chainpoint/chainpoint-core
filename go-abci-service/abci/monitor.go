package abci

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	beacon "github.com/chainpoint/go-nist-beacon"

	"github.com/streadway/amqp"

	"github.com/chainpoint/chainpoint-core/go-abci-service/types"

	"github.com/chainpoint/chainpoint-core/go-abci-service/rabbitmq"
	"github.com/chainpoint/chainpoint-core/go-abci-service/util"
)

// ConsumeBtcTxMsg : Consumes a btctx RMQ message to initiate monitoring on all nodes
func (app *AnchorApplication) ConsumeBtcTxMsg(msgBytes []byte) error {
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
					Left: btcTxObj.BtcTxBody[:strings.Index(btcTxObj.BtcTxBody, btcTxObj.AnchorBtcAggRoot)],
				},
				types.ProofLineItem{
					Right: btcTxObj.BtcTxBody[strings.Index(btcTxObj.BtcTxBody, btcTxObj.AnchorBtcAggRoot)+len(btcTxObj.AnchorBtcAggRoot):],
				},
				types.ProofLineItem{
					Op: "sha-256-x2",
				},
			},
		},
	}
	dataJSON, err := json.Marshal(stateObj)
	if util.LogError(err) != nil {
		return err
	}
	err = rabbitmq.Publish(app.config.RabbitmqURI, "work.proofstate", "btctx", dataJSON)
	if err != nil {
		rabbitmq.LogError(err, "rmq dial failure, is rmq connected?")
		return err
	}
	txIDBytes, err := json.Marshal(types.TxID{TxID: btcTxObj.BtcTxID})
	err = rabbitmq.Publish(app.config.RabbitmqURI, "work.btcmon", "", txIDBytes)
	if err != nil {
		rabbitmq.LogError(err, "rmq dial failure, is rmq connected?")
		return err
	}
	return nil
}

func (app *AnchorApplication) processMessage(msg amqp.Delivery) error {
	switch msg.Type {
	case "btctx":
		time.Sleep(30 * time.Second)
		BroadcastTx(app.config.TendermintRPC, "BTC-M", string(msg.Body), 2, time.Now().Unix())
		msg.Ack(false)
		break
	case "btcmon":
		var hash []byte
		var btcMonObj types.BtcMonMsg
		json.Unmarshal(msg.Body, &btcMonObj)
		result, err := BroadcastTx(app.config.TendermintRPC, "BTC-C", btcMonObj.BtcHeadRoot, 2, time.Now().Unix())
		if util.LogError(err) != nil {
			if strings.Contains(err.Error(), "-32603") {
				app.logger.Error("Another core has already committed a BTCC tx")
				txResult, err := GetTxByInt(app.config.TendermintRPC, app.state.LatestBtccTxInt)
				if util.LogError(err) != nil && len(txResult.Txs) > 0 {
					hash = txResult.Txs[0].Hash
				}
			} else {
				return err
			}
		} else {
			hash = result.Hash
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
		uri := strings.ToLower(fmt.Sprintf("%s/calendar/%x/data", baseURI, hash))
		btccStateObj.BtcHeadState.Anchor = types.AnchorObj{
			AnchorID: strconv.FormatInt(btcMonObj.BtcHeadHeight, 10),
			Uris:     []string{uri},
		}
		stateObjBytes, err := json.Marshal(btccStateObj)

		err = rabbitmq.Publish(app.config.RabbitmqURI, "work.proofstate", "btcmon", stateObjBytes)
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
func (app *AnchorApplication) ReceiveCalRMQ() error {
	var session rabbitmq.Session
	var err error
	endConsume := false
	for {
		session, err = rabbitmq.ConnectAndConsume(app.config.RabbitmqURI, "work.cal")
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
					go app.processMessage(msg)
				}
			}
		}
	}
}

//SyncMonitor : turns off anchoring if we're not synced. Not cron scheduled since we need it to start immediately.
func (app *AnchorApplication) SyncMonitor() {
	for {
		status, err := GetStatus(app.config.TendermintRPC)
		if util.LogError(err) != nil {
			time.Sleep(5 * time.Second)
			continue
		}
		if status.SyncInfo.CatchingUp {
			app.state.AnchorEnabled = false
		} else {
			app.state.AnchorEnabled = true
		}
		time.Sleep(30 * time.Second)
	}
}

// NistBeaconMonitor : gets the latest NIST beacon record and elects a leader to write a NIST transaction
func (app *AnchorApplication) NistBeaconMonitor() {
	if leader, _ := ElectLeader(app.config.TendermintRPC); leader {
		nistRecord, err := beacon.LastRecord()
		if util.LogError(err) != nil {
			app.logger.Error("Unable to obtain new NIST beacon value")
			return
		}
		_, err = BroadcastTx(app.config.TendermintRPC, "NIST", nistRecord.ChainpointFormat(), 2, time.Now().Unix()) // elect a leader to send a NIST tx
		if util.LogError(err) != nil {
			app.logger.Debug(fmt.Sprintf("Gossiped new NIST beacon value of %s", nistRecord.ChainpointFormat()))
		}
	}
}
