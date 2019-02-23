package abci

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	core_types "github.com/tendermint/tendermint/rpc/core/types"

	"github.com/chainpoint/chainpoint-core/go-abci-service/merkletools"
	"github.com/chainpoint/chainpoint-core/go-abci-service/rabbitmq"
	"github.com/chainpoint/chainpoint-core/go-abci-service/util"
)

// Anchor scans all CAL transactions since last anchor epoch and writes the merkle root to the Calendar and to bitcoin
func (app *AnchorApplication) Anchor(startTxRange int64, endTxRange int64) error {
	fmt.Println("starting scheduled anchor")
	iAmLeader, _ := ElectLeader(app.tendermintURI)
	/* Get CAL transactions between the latest BTCA tx and the current latest tx */
	txLeaves, err := GetTxRange(app.tendermintURI, startTxRange, endTxRange)
	if util.LogError(err) != nil {
		return err
	}
	treeData := AggregateAnchorTx(txLeaves)
	fmt.Printf("treeData for current anchor: %v\n", treeData)
	if treeData.AggRoot != "" {
		if iAmLeader {
			result, err := BroadcastTx(app.tendermintURI, "BTC-A", treeData.AggRoot, 2, time.Now().Unix())
			if util.LogError(err) != nil {
				return err
			}
			fmt.Printf("Anchor result: %v\n", result)
		}
		treeData.QueueBtcaStateDataMessage(app.rabbitmqUri, iAmLeader)
		return nil
	}
	return errors.New("no transactions to aggregate")
}

// AggregateAnchorTx takes in cal transactions and creates a merkleroot and proof path. Called by the anchor loop
func AggregateAnchorTx(txLeaves []core_types.ResultTx) BtcAgg {
	calBytes := make([][]byte, 0)
	calLeaves := make([]core_types.ResultTx, 0)
	for _, t := range txLeaves {
		decodedTx, err := DecodeTx(t.Tx)
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
