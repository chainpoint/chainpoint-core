package calendar

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/chainpoint/chainpoint-core/go-abci-service/types"
	core_types "github.com/tendermint/tendermint/rpc/core/types"

	"github.com/chainpoint/chainpoint-core/go-abci-service/rabbitmq"

	"github.com/chainpoint/chainpoint-core/go-abci-service/util"

	"github.com/chainpoint/chainpoint-core/go-abci-service/merkletools"
)

type Calendar struct {
	RabbitmqURI string
	Logger      log.Logger
}

// NewCalendar returns a new Calendar Object with a built-in logger. Useful for testing
func NewCalendar(rmqUri string) *Calendar {
	allowLevel, _ := log.AllowLevel(strings.ToLower("ERROR"))
	tmLogger := log.NewFilter(log.NewTMLogger(log.NewSyncWriter(os.Stdout)), allowLevel)
	calendar := Calendar{
		RabbitmqURI: rmqUri,
		Logger:      tmLogger,
	}
	return &calendar
}

// GenerateCalendarTree creates the MerkleTree for the aggregation roots which will be committed to the calendar
func (calendar *Calendar) GenerateCalendarTree(aggs []types.Aggregation) types.CalAgg {
	var treeDataObj types.CalAgg
	var tree merkletools.MerkleTree
	for _, agg := range aggs {
		aggRootBytes, err := hex.DecodeString(agg.AggRoot)
		if util.LogError(err) != nil {
			continue
		}
		tree.AddLeaf(aggRootBytes)
	}
	tree.MakeTree()
	treeDataObj.CalRoot = hex.EncodeToString(tree.GetMerkleRoot())
	treeDataObj.ProofData = make([]types.CalProofData, len(aggs))
	for i, agg := range aggs {
		var proofData types.CalProofData
		proofData.AggID = agg.AggID
		proof := tree.GetProof(i)
		proofData.Proof = make([]types.ProofLineItem, 0)
		for _, p := range proof {
			if p.Left {
				proofData.Proof = append(proofData.Proof, types.ProofLineItem{Left: hex.EncodeToString(p.Value)})
			} else {
				proofData.Proof = append(proofData.Proof, types.ProofLineItem{Right: hex.EncodeToString(p.Value)})
			}
			proofData.Proof = append(proofData.Proof, types.ProofLineItem{Op: "sha-256"})
		}
		treeDataObj.ProofData[i] = proofData
	}
	calendar.Logger.Info(fmt.Sprintf("AggTree Input: %#v\nCalTree Output: %#v\n", aggs, treeDataObj))
	return treeDataObj
}

// QueueCalStateMessage lets proof state service know about a cal anchoring via rabbitmq
func (calendar *Calendar) QueueCalStateMessage(tx types.TxTm, treeDataObj types.CalAgg) {
	var calState types.CalState
	baseURI := util.GetEnv("CHAINPOINT_CORE_BASE_URI", "https://tendermint.chainpoint.org")
	uri := fmt.Sprintf("%s/calendar/%x/data", baseURI, tx.Hash)
	anchor := types.AnchorObj{
		AnchorID: hex.EncodeToString(tx.Hash),
		Uris:     []string{uri},
	}
	calState.Anchor = anchor
	calState.ProofData = treeDataObj.ProofData
	calState.CalID = hex.EncodeToString(tx.Hash)
	calStateJSON, _ := json.Marshal(calState)
	err := rabbitmq.Publish(calendar.RabbitmqURI, "work.proofstate", "cal_batch", calStateJSON)
	if err != nil {
		rabbitmq.LogError(err, "rmq dial failure, is rmq connected?")
	}
}

// AggregateAnchorTx takes in cal transactions and creates a merkleroot and proof path. Called by the anchor loop
func (calendar *Calendar) AggregateAnchorTx(txLeaves []core_types.ResultTx) types.BtcAgg {
	if len(txLeaves) == 0 {
		calendar.Logger.Error("No txLeaves to aggregate, exiting")
		return types.BtcAgg{}
	}
	calBytes := make([][]byte, 0)
	calLeaves := make([]core_types.ResultTx, 0)
	for _, t := range txLeaves {
		decodedTx, err := util.DecodeTx(t.Tx)
		if err != nil {
			util.LogError(err)
			continue
		}
		if string(decodedTx.TxType) == "CAL" {
			decodedBytes, err := hex.DecodeString(decodedTx.Data)
			if util.LogError(err) != nil {
				continue
			}
			calBytes = append(calBytes, decodedBytes)
			calLeaves = append(calLeaves, t)
		}
	}
	tree := merkletools.MerkleTree{}
	tree.AddLeaves(calBytes)
	tree.MakeTree()
	var treeData types.BtcAgg
	uuid, err := util.UUIDFromHash(tree.Root[0:16])
	util.LogError(err)
	treeData.AnchorBtcAggID = uuid.String()
	treeData.AnchorBtcAggRoot = hex.EncodeToString(tree.GetMerkleRoot())
	treeData.ProofData = make([]types.BtcProofData, len(calLeaves))
	for i, tx := range calLeaves {
		var proofDataItem types.BtcProofData
		proofDataItem.CalID = hex.EncodeToString(tx.Hash)
		proofs := tree.GetProof(i)
		proofDataItem.Proof = make([]types.ProofLineItem, 0)
		for _, p := range proofs {
			if p.Left {
				proofDataItem.Proof = append(proofDataItem.Proof, types.ProofLineItem{Left: hex.EncodeToString(p.Value)})
			} else {
				proofDataItem.Proof = append(proofDataItem.Proof, types.ProofLineItem{Right: hex.EncodeToString(p.Value)})
			}
			proofDataItem.Proof = append(proofDataItem.Proof, types.ProofLineItem{Op: "sha-256"})
		}
		treeData.ProofData[i] = proofDataItem
	}
	calendar.Logger.Info(fmt.Sprintf("AggTree Input: %#v\nAggTree Output: %#v\n", txLeaves, treeData))
	return treeData
}

// QueueBtcaStateDataMessage notifies proof and btc tx services of BTC-A anchoring
func (calendar *Calendar) QueueBtcaStateDataMessage(anchorDataObj types.BtcAgg) error {
	treeDataJSON, err := json.Marshal(anchorDataObj)
	if util.LogError(err) != nil {
		return err
	}
	errBatch := rabbitmq.Publish(calendar.RabbitmqURI, "work.proofstate", "anchor_btc_agg_batch", treeDataJSON)
	if errBatch != nil {
		return errBatch
	}
	return nil
}

// QueueBtcTxStateDataMessage
func (calendar *Calendar) QueueBtcTxStateDataMessage(anchorDataObj types.BtcAgg) error {
	treeDataJSON, err := json.Marshal(anchorDataObj)
	if util.LogError(err) != nil {
		return err
	}
	errBtcTx := rabbitmq.Publish(calendar.RabbitmqURI, "work.btctx", "", treeDataJSON)
	if errBtcTx != nil {
		return errBtcTx
	}
	return nil
}
