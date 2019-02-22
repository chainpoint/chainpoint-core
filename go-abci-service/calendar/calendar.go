package calendar

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	core_types "github.com/tendermint/tendermint/rpc/core/types"

	"github.com/google/uuid"

	"github.com/chainpoint/chainpoint-core/go-abci-service/rabbitmq"

	"github.com/chainpoint/chainpoint-core/go-abci-service/abci"
	"github.com/streadway/amqp"

	"github.com/chainpoint/chainpoint-core/go-abci-service/util"

	"github.com/chainpoint/chainpoint-core/go-abci-service/aggregator"
	"github.com/chainpoint/chainpoint-core/go-abci-service/merkletools"
)

type BtcAgg struct {
	AggId     string         `json:"anchor_btc_agg_id"`
	AggRoot   string         `json:"anchor_btc_agg_root"`
	ProofData []BtcProofData `json:"proofData"`
}

type BtcProofData struct {
	CalId string  `json:"cal_id"`
	Proof []Proof `json:"proof"`
}

type CalAgg struct {
	CalRoot   string      `json:"cal_root"`
	ProofData []ProofData `json:"proofData"`
}

type CalState struct {
	CalId     string      `json:"cal_id"`
	Anchor    CalAnchor   `json:"anchor"`
	ProofData []ProofData `json:"proofData"`
}

type BtcMonMsg struct {
	BtcId         string `json:"btctx_id"`
	BtcHeadHeight int64  `json:"btchead_height"`
	BtcHeadRoot   string `json:"btchead_root"`
}

type CalAnchor struct {
	AnchorId string   `json:"anchor_id"`
	Uris     []string `json:"uris"`
}

type ProofData struct {
	AggId string  `json:"agg_id"`
	Proof []Proof `json:"proof"`
}

type Proof struct {
	Left  string `json:"l,omitempty"`
	Right string `json:"r,omitempty"`
	Op    string `json: "op"`
}

// GenerateCalendarTree creates the MerkleTree for the aggregation roots which will be committed to the calendar
func (treeDataObj *CalAgg) GenerateCalendarTree(aggs []aggregator.Aggregation) {
	var tree merkletools.MerkleTree
	for _, agg := range aggs {
		aggRootBytes := []byte(agg.AggRoot)
		tree.AddLeaf(aggRootBytes)
	}
	tree.MakeTree()
	treeDataObj.CalRoot = hex.EncodeToString(tree.GetMerkleRoot())
	treeDataObj.ProofData = make([]ProofData, len(aggs))
	for i, agg := range aggs {
		var proofData ProofData
		proofData.AggId = agg.AggId
		proof := tree.GetProof(i)
		proofData.Proof = make([]Proof, len(proof))
		for j, p := range proof {
			if p.Left {
				proofData.Proof[j] = Proof{Left: string(p.Value), Op: "sha-256"}
			} else {
				proofData.Proof[j] = Proof{Right: string(p.Value), Op: "sha-256"}
			}
		}
		treeDataObj.ProofData[i] = proofData
	}
}

// QueueCalStateMessage lets proof state service know about a cal anchoring via rabbitmq
func (treeDataObj *CalAgg) QueueCalStateMessage(rabbitmqConnectUri string, tx abci.TxTm) {
	var calState CalState
	base_uri := util.GetEnv("CHAINPOINT_CORE_BASE_URI", "tendermint.chainpoint.org")
	uri := fmt.Sprintf("%s/calendar/%x/data", base_uri, tx.Hash)
	anchor := CalAnchor{
		AnchorId: hex.EncodeToString(tx.Hash),
		Uris:     []string{uri},
	}
	calState.Anchor = anchor
	calState.ProofData = treeDataObj.ProofData
	calState.CalId = hex.EncodeToString(tx.Hash)
	calStateJson, _ := json.Marshal(calState)
	err := rabbitmq.Publish(rabbitmqConnectUri, "work.proofstate", "cal_batch", calStateJson)
	if err != nil {
		rabbitmq.LogError(err, "rmq dial failure, is rmq connected?")
	}
}

// AggregateAnchorTx takes in cal transactions and creates a merkleroot and proof path. Called by the anchor loop
func AggregateAnchorTx(txLeaves []core_types.ResultTx) BtcAgg {
	calBytes := make([][]byte, 0)
	calLeaves := make([]core_types.ResultTx, 0)
	for _, t := range txLeaves {
		decodedTx, err := abci.DecodeTx(t.Tx)
		if err != nil {
			util.LogError(err)
			continue
		}
		if string(decodedTx.TxType) == "CAL" {
			calBytes = append(calBytes, []byte(decodedTx.Data))
			calLeaves = append(calLeaves, t)
		}
	}
	tree := merkletools.MerkleTree{}
	tree.AddLeaves(calBytes)
	tree.MakeTree()
	var treeData BtcAgg
	uuid, _ := uuid.NewUUID()
	treeData.AggId = uuid.String()
	treeData.AggRoot = hex.EncodeToString(tree.GetMerkleRoot())
	treeData.ProofData = make([]BtcProofData, len(calLeaves))
	for i, tx := range calLeaves {
		var proofDataItem BtcProofData
		proofDataItem.CalId = hex.EncodeToString(tx.Hash)
		proofs := tree.GetProof(i)
		proofDataItem.Proof = make([]Proof, len(proofs))
		for j, p := range proofs {
			if p.Left {
				proofDataItem.Proof[j] = Proof{Left: string(p.Value), Op: "sha-256"}
			} else {
				proofDataItem.Proof[j] = Proof{Right: string(p.Value), Op: "sha-256"}
			}
		}
		treeData.ProofData[i] = proofDataItem
	}
	return treeData
}

// QueueBtcaStateDataMessage notifies proof and btc tx services of BTC-A anchoring
func (anchorDataObj *BtcAgg) QueueBtcaStateDataMessage(rabbitmqUri string, isLeader bool) error {
	treeDataJSON, err := json.Marshal(anchorDataObj)
	if util.LogError(err) != nil {
		return err
	}
	errBatch := rabbitmq.Publish(rabbitmqUri, "work.proofstate", "anchor_btc_agg_batch", treeDataJSON)
	if errBatch != nil {
		return errBatch
	}
	if isLeader {
		errBtcTx := rabbitmq.Publish(rabbitmqUri, "work.btctx", "", treeDataJSON)
		if errBtcTx != nil {
			return errBtcTx
		}
	}
	return nil
}

func processMessage(rabbitmqUri string, rpcUri abci.TendermintURI, msg amqp.Delivery) error {
	switch msg.Type {
	case "btctx":
		time.Sleep(30 * time.Second)
		abci.BroadcastTx(rpcUri, "BTC-T", string(msg.Body), 2, time.Now().Unix())
		msg.Ack(false)
		break
	case "btcmon":
		var btcMonObj BtcMonMsg
		json.Unmarshal(msg.Body, &btcMonObj)
		heightAndRoot := string(btcMonObj.BtcHeadHeight) + ":" + btcMonObj.BtcHeadRoot
		_, err := abci.BroadcastTx(rpcUri, "BTC-C", heightAndRoot, 2, time.Now().Unix())
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

func ReceiveCalRMQ(rabbitmqUri string, rpcUri abci.TendermintURI) error {
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
