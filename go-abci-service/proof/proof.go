package proof

import (
	"encoding/json"
	"github.com/chainpoint/chainpoint-core/go-abci-service/types"
	"github.com/google/uuid"
	"time"
)

type P map[string]interface{}

func Proof() P {
	return P{}
}

func (proof *P) AddChainpointHeader(hash string, proofId string) error {
	(*proof)["@context"] = "https://w3id.org/chainpoint/v4"
	(*proof)["type"] = "Chainpoint"
	(*proof)["hash"] = hash
	(*proof)["proof_id"] = proofId
	proofUUID, err := uuid.Parse(proofId)
	if err != nil {
		return err
	}
	unixTime, _ := proofUUID.Time().UnixTime()
	(*proof)["hash_received"] = time.Unix(unixTime, 0).Format(time.RFC3339)
	return nil
}

func ConvertGoOpsToJsonMap(ops []types.ProofLineItem) ([]P) {
	var opsJsonArray []P
	for _, op := range ops {
		var leftOrRight map[string]interface{}
		var operation map[string]interface{}
		if len(op.Left) > 0 {
			leftOrRight["l"] = op.Left
		} else {
			leftOrRight["r"] = op.Right
		}
		operation["op"] = op.Op
		opsJsonArray = append(opsJsonArray, leftOrRight, operation)
	}
	return opsJsonArray
}

func (proof *P) AddCalendarBranch(aggState types.AggState, calState string, network string) error {
	var calendarBranch map[string]interface{}
	calendarBranch["label"] = "cal_anchor_branch"
	aggStateOps := types.OpsState{}
	if err := json.Unmarshal([]byte(aggState.AggState), & aggStateOps); err != nil {
		return err
	}
	calStateOps := types.AnchorOpsState{}
	if err := json.Unmarshal([]byte(calState), &calStateOps); err != nil {
		return err
	}
	ops := append(aggStateOps.Ops, calStateOps.Ops...)
	opsJson := ConvertGoOpsToJsonMap(ops)

	var calendarAnchor map[string]interface{}
	calendarAnchor["type"] = network
	calendarAnchor["anchor_id"] = calStateOps.Anchor.AnchorID
	calendarAnchor["uris"] = calStateOps.Anchor.Uris

	var anchorOp map[string]interface{}
	anchorOp["anchors"] = []P{calendarAnchor}
	opsJson = append(opsJson, anchorOp)

	calendarBranch["ops"] = opsJson
	(*proof)["branches"] = []P{calendarBranch}
	return nil
}

func (proof *P) AddBtcBranch(btcAggState types.AnchorBtcAggState, btcTxState types.AnchorBtcTxState, btcHeadState types.AnchorBtcHeadState, network string) error {
	var btcBranch map[string]interface{}
	btcBranch["label"] = "btc_anchor_branch"
	aggState := types.OpsState{}
	if err := json.Unmarshal([]byte(btcAggState.AnchorBtcAggState), &aggState); err != nil {
		return err
	}
	txState := types.OpsState{}
	if err := json.Unmarshal([]byte(btcTxState.BtcTxState), &txState); err != nil {
		return err
	}
	headState := types.AnchorOpsState{}
	if err := json.Unmarshal([]byte(btcHeadState.BtcHeadState), &headState); err != nil {
		return err
	}
	ops := append(append(aggState.Ops, txState.Ops...), headState.Ops...)
	opsJson := ConvertGoOpsToJsonMap(ops)

	var btcAnchor map[string]interface{}
	btcAnchor["type"] = network
	btcAnchor["anchor_id"] = headState.Anchor.AnchorID
	btcAnchor["uris"] = headState.Anchor.Uris

	var anchorOp map[string]interface{}
	anchorOp["anchors"] = []P{btcAnchor}
	opsJson = append(opsJson, anchorOp)
	btcBranch["ops"] = opsJson

	(*proof)["branches"].([]P)[0]["branches"] = []P{btcBranch}
	return nil
}
