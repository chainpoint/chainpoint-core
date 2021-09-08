package calendar

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/chainpoint/chainpoint-core/go-abci-service/types"
	core_types "github.com/tendermint/tendermint/rpc/core/types"

	"github.com/chainpoint/chainpoint-core/go-abci-service/util"

	"github.com/chainpoint/chainpoint-core/go-abci-service/merkletools"
)

// GenerateCalendarTree creates the MerkleTree for the aggregation roots which will be committed to the calendar
func GenerateCalendarTree(aggs []types.Aggregation) types.CalAgg {
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
	return treeDataObj
}

// CreateCalStateMessage lets proof state service know about a cal anchoring via rabbitmq
func CreateCalStateMessage(tx types.TxTm, treeDataObj types.CalAgg) []types.CalStateObject {
	calStates := make([]types.CalStateObject, 0)
	baseURI := util.GetEnv("CHAINPOINT_CORE_BASE_URI", "https://tendermint.chainpoint.org")
	uri := fmt.Sprintf("%s/calendar/%x/data", baseURI, tx.Hash)
	anchor := types.AnchorObj{
		AnchorID: hex.EncodeToString(tx.Hash),
		Uris:     []string{uri},
	}
	calID := hex.EncodeToString(tx.Hash)
	for _, ops := range treeDataObj.ProofData {
		anchorOps := types.AnchorOpsState{
			Anchor: anchor,
			Ops:    ops.Proof,
		}
		anchorBytes, err := json.Marshal(anchorOps)
		if err != nil {
			continue
		}
		calState := types.CalStateObject{
			AggID:    ops.AggID,
			CalId:    calID,
			CalState: string(anchorBytes),
		}
		calStates = append(calStates, calState)
	}
	return calStates
}

// AggregateAnchorTx takes in cal transactions and creates a merkleroot and proof path. Called by the anchor loop
func AggregateAnchorTx(txLeaves []core_types.ResultTx) types.BtcAgg {
	if len(txLeaves) == 0 {
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
	//calendar.Logger.Info(fmt.Sprintf("AggTree Input: %#v\nAggTree Output: %#v\n", txLeaves, treeData))
	return treeData
}

// PrepareBtcaStateData notifies proof and btc tx services of BTC-A anchoring
func PrepareBtcaStateData(anchorDataObj types.BtcAgg) []types.AnchorBtcAggState {
	anchorObjects := []types.AnchorBtcAggState{}
	for _, proofDataItem := range anchorDataObj.ProofData {
		opsState := types.OpsState{Ops: proofDataItem.Proof}
		opsBytes, err := json.Marshal(opsState)
		if err != nil {
			continue
		}
		anchorObj := types.AnchorBtcAggState{
			CalId:             proofDataItem.CalID,
			AnchorBtcAggId:    anchorDataObj.AnchorBtcAggID,
			AnchorBtcAggState: string(opsBytes),
		}
		anchorObjects = append(anchorObjects, anchorObj)
	}
	return anchorObjects
}

// GenerateHeadStateObject: Assembles anchor operations into object ready for insertion into postgres
func GenerateHeadStateObject(uri string, hash []byte, btcMonObj types.BtcMonMsg) types.AnchorBtcHeadState {
	anchorOps := types.AnchorOpsState{}
	anchorOps.Ops = make([]types.ProofLineItem, 0)
	for _, p := range btcMonObj.Path {
		if p.Left != "" {
			anchorOps.Ops = append(anchorOps.Ops, types.ProofLineItem{Left: string(p.Left)})
		}
		if p.Right != "" {
			anchorOps.Ops = append(anchorOps.Ops, types.ProofLineItem{Right: string(p.Right)})
		}
		anchorOps.Ops = append(anchorOps.Ops, types.ProofLineItem{Op: "sha-256-x2"})
	}
	proofUri := strings.ReplaceAll(uri, "http://", "")
	uriParts := strings.Split(proofUri, ":")
	proofUri = uriParts[0]
	anchorOps.Anchor = types.AnchorObj{
		AnchorID: strconv.FormatInt(btcMonObj.BtcHeadHeight, 10),
		Uris:     []string{strings.ToLower(fmt.Sprintf("%s/calendar/%x/data", proofUri, hash))},
	}
	headState, _ := json.Marshal(anchorOps)
	headStateObj := types.AnchorBtcHeadState{
		BtcTxId:       btcMonObj.BtcTxID,
		BtcHeadHeight: btcMonObj.BtcHeadHeight,
		BtcHeadState:  string(headState),
	}
	return headStateObj
}

//GenerateAnchorBtcTxState : gather btc anchor info into form ready for postgres
func GenerateAnchorBtcTxState(btcTxObj types.BtcTxMsg) (stateObj types.AnchorBtcTxState) {
	btcTxState := types.OpsState{
		Ops: []types.ProofLineItem{
			{
				Left: btcTxObj.BtcTxBody[:strings.Index(btcTxObj.BtcTxBody, btcTxObj.AnchorBtcAggRoot)],
			},
			{
				Right: btcTxObj.BtcTxBody[strings.Index(btcTxObj.BtcTxBody, btcTxObj.AnchorBtcAggRoot)+len(btcTxObj.AnchorBtcAggRoot):],
			},
			{
				Op: "sha-256-x2",
			},
		},
	}
	btcTxStateJSON, _ := json.Marshal(btcTxState)
	stateObj = types.AnchorBtcTxState{
		AnchorBtcAggId: btcTxObj.AnchorBtcAggID,
		BtcTxId:        btcTxObj.BtcTxID,
		BtcTxState:     string(btcTxStateJSON),
	}
	return stateObj
}
