package calendar

import (
	"encoding/hex"
	"encoding/json"
	"fmt"

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

func (treeDataObj *CalAgg) GenerateCalendarTree(aggs []aggregator.Aggregation) {
	var tree merkletools.MerkleTree
	for _, agg := range aggs {
		aggRootBytes := []byte(agg.AggRoot)
		tree.AddLeaf(aggRootBytes)
	}
	tree.MakeTree()
	treeDataObj.CalRoot = fmt.Sprintf("%x", tree.GetMerkleRoot())
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
	destSession, err := rabbitmq.Dial(rabbitmqConnectUri, "work.proofstate")
	if err != nil {
		rabbitmq.LogError(err, "failed to dial for cal queue")
	}
	defer destSession.Conn.Close()
	defer destSession.Ch.Close()
	err = destSession.Ch.Publish(
		"",
		destSession.Queue.Name,
		false,
		false,
		amqp.Publishing{
			Type:         "cal_batch",
			Body:         calStateJson,
			DeliveryMode: 2, //persistent
			ContentType:  "application/json",
		})
	if err != nil {
		rabbitmq.LogError(err, "rmq dial failure, is rmq connected?")
	}
}

func AggregateAndAnchorBTC(txLeaves []core_types.ResultTx) BtcAgg {
	calBytes := make([][]byte, len(txLeaves))
	for i, t := range txLeaves {
		calBytes[i] = t.Tx
	}
	tree := merkletools.MerkleTree{}
	tree.AddLeaves(calBytes)
	tree.MakeTree()
	var treeData BtcAgg
	uuid, _ := uuid.NewUUID()
	treeData.AggId = uuid.String()
	treeData.AggRoot = hex.EncodeToString(tree.GetMerkleRoot())
	treeData.ProofData = make([]BtcProofData, len(txLeaves))
	for i, tx := range txLeaves {
		var proofDataItem BtcProofData
		proofDataItem.CalId = hex.EncodeToString(tx.Hash)
		proofs := tree.GetProof(i)
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
