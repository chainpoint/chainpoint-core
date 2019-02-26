package calendar

import (
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/chainpoint/chainpoint-core/go-abci-service/types"
	"github.com/google/uuid"
	core_types "github.com/tendermint/tendermint/rpc/core/types"

	"github.com/chainpoint/chainpoint-core/go-abci-service/rabbitmq"

	"github.com/chainpoint/chainpoint-core/go-abci-service/util"

	"github.com/chainpoint/chainpoint-core/go-abci-service/aggregator"
	"github.com/chainpoint/chainpoint-core/go-abci-service/merkletools"
)

// GenerateCalendarTree creates the MerkleTree for the aggregation roots which will be committed to the calendar
func GenerateCalendarTree(aggs []aggregator.Aggregation) types.CalAgg {
	var treeDataObj types.CalAgg
	var tree merkletools.MerkleTree
	for _, agg := range aggs {
		aggRootBytes := []byte(agg.AggRoot)
		tree.AddLeaf(aggRootBytes)
	}
	tree.MakeTree()
	treeDataObj.CalRoot = hex.EncodeToString(tree.GetMerkleRoot())
	treeDataObj.ProofData = make([]types.ProofData, len(aggs))
	for i, agg := range aggs {
		var proofData types.ProofData
		proofData.AggId = agg.AggId
		proof := tree.GetProof(i)
		proofData.Proof = make([]types.Proof, len(proof))
		for j, p := range proof {
			if p.Left {
				proofData.Proof[j] = types.Proof{Left: string(p.Value), Op: "sha-256"}
			} else {
				proofData.Proof[j] = types.Proof{Right: string(p.Value), Op: "sha-256"}
			}
		}
		treeDataObj.ProofData[i] = proofData
	}
	return treeDataObj
}

// QueueCalStateMessage lets proof state service know about a cal anchoring via rabbitmq
func QueueCalStateMessage(rabbitmqConnectUri string, tx types.TxTm, treeDataObj types.CalAgg) {
	var calState types.CalState
	base_uri := util.GetEnv("CHAINPOINT_CORE_BASE_URI", "tendermint.chainpoint.org")
	uri := fmt.Sprintf("%s/calendar/%x/data", base_uri, tx.Hash)
	anchor := types.CalAnchor{
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
func AggregateAnchorTx(txLeaves []core_types.ResultTx) types.BtcAgg {
	calBytes := make([][]byte, 0)
	calLeaves := make([]core_types.ResultTx, 0)
	for _, t := range txLeaves {
		decodedTx, err := util.DecodeTx(t.Tx)
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
	var treeData types.BtcAgg
	uuid, _ := uuid.NewUUID()
	treeData.AggId = uuid.String()
	treeData.AggRoot = hex.EncodeToString(tree.GetMerkleRoot())
	treeData.ProofData = make([]types.BtcProofData, len(calLeaves))
	for i, tx := range calLeaves {
		var proofDataItem types.BtcProofData
		proofDataItem.CalId = hex.EncodeToString(tx.Hash)
		proofs := tree.GetProof(i)
		proofDataItem.Proof = make([]types.Proof, len(proofs))
		for j, p := range proofs {
			if p.Left {
				proofDataItem.Proof[j] = types.Proof{Left: string(p.Value), Op: "sha-256"}
			} else {
				proofDataItem.Proof[j] = types.Proof{Right: string(p.Value), Op: "sha-256"}
			}
		}
		treeData.ProofData[i] = proofDataItem
	}
	return treeData
}

// QueueBtcaStateDataMessage notifies proof and btc tx services of BTC-A anchoring
func QueueBtcaStateDataMessage(rabbitmqUri string, isLeader bool, anchorDataObj types.BtcAgg) error {
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
