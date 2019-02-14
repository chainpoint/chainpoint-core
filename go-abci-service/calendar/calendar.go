package calendar

import (
	"encoding/hex"
	"fmt"

	"github.com/chainpoint/chainpoint-core/go-abci-service/abci"

	"github.com/chainpoint/chainpoint-core/go-abci-service/util"

	"github.com/chainpoint/chainpoint-core/go-abci-service/aggregator"
	"github.com/chainpoint/chainpoint-core/go-abci-service/merkletools"
)

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

func (treeDataObj *CalAgg) QueueCalStateMessage(tx abci.TxTm) {
	var calState CalState
	base_uri := util.GetEnv("CHAINPOINT_CORE_BASE_URI", "tendermint.chainpoint.org")
	uri := fmt.Sprintf("%s/calendar/%x/hash", base_uri, tx.Hash())
	var anchor CalAnchor
	anchor.AnchorId = hex.EncodeToString(tx.Hash)
	anchor.Uris = []string{uri}
	//TODO: opsToBlockHash, sendToQueue
}
