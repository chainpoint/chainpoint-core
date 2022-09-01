package proof

import (
	"encoding/json"
	"github.com/chainpoint/chainpoint-core/types"
	"github.com/chainpoint/chainpoint-core/util"
	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
	"time"
)

type P map[string]interface{}

func Proof() P {
	proof := make(P)
	return proof
}

func (proof *P) AddChainpointHeader(context string, jsonType string, hash string, proofId string) error {
	(*proof)["@context"] = context
	(*proof)["type"] = jsonType
	(*proof)["hash"] = hash
	(*proof)["proof_id"] = proofId
	var unixTime int64
	if util.IsUUID(proofId) {
		proofUUID, err := uuid.Parse(proofId)
		if err != nil {
			return err
		}
		unixTime, _ = proofUUID.Time().UnixTime()
	} else if util.IsULID(proofId) {
		proofULID, err := ulid.Parse(proofId)
		if err != nil {
			return err
		}
		unixTime = ulid.Time(proofULID.Time()).Unix()
	}
	(*proof)["hash_received"] = time.Unix(unixTime, 0).Format(time.RFC3339)
	return nil
}

func ConvertGoOpsToJsonMap(ops []types.ProofLineItem) []P {
	opsJsonArray := make([]P, 0)
	for _, op := range ops {
		leftOrRight := make(map[string]interface{})
		operation := make(map[string]interface{})
		if len(op.Left) > 0 {
			leftOrRight["l"] = op.Left
		}
		if len(op.Right) > 0 {
			leftOrRight["r"] = op.Right
		}
		if len(leftOrRight) > 0 {
			opsJsonArray = append(opsJsonArray, leftOrRight)
		}
		if len(op.Op) > 0 {
			operation["op"] = op.Op
			opsJsonArray = append(opsJsonArray, operation)
		}
	}
	return opsJsonArray
}

func (proof *P) AddCalendarBranch(aggState types.AggState, calState string, proofType string) error {
	calendarBranch := make(map[string]interface{})
	calendarBranch["label"] = "cal_anchor_branch"
	aggStateOps := types.OpsState{}
	if err := json.Unmarshal([]byte(aggState.AggState), &aggStateOps); err != nil {
		return err
	}
	calStateOps := types.AnchorOpsState{}
	if err := json.Unmarshal([]byte(calState), &calStateOps); err != nil {
		return err
	}
	ops := append(aggStateOps.Ops, calStateOps.Ops...)
	opsJson := ConvertGoOpsToJsonMap(ops)

	calendarAnchor := make(map[string]interface{})
	calendarAnchor["type"] = proofType
	calendarAnchor["anchor_id"] = calStateOps.Anchor.AnchorID
	calendarAnchor["uris"] = calStateOps.Anchor.Uris

	anchorOp := make(map[string]interface{})
	anchorOp["anchors"] = []P{calendarAnchor}
	opsJson = append(opsJson, anchorOp)

	calendarBranch["ops"] = opsJson
	(*proof)["branches"] = []P{calendarBranch}
	return nil
}

func (proof *P) AddChainBranch(btcAggState types.AnchorBtcAggState, btcTxState types.AnchorBtcTxState, btcHeadState types.AnchorBtcHeadState, proofType string) error {
	btcBranch := make(map[string]interface{})
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

	btcAnchor := make(map[string]interface{})
	btcAnchor["type"] = proofType
	btcAnchor["anchor_id"] = headState.Anchor.AnchorID
	btcAnchor["uris"] = headState.Anchor.Uris

	anchorOp := make(map[string]interface{})
	anchorOp["anchors"] = []P{btcAnchor}
	opsJson = append(opsJson, anchorOp)
	btcBranch["ops"] = opsJson

	if _, branchExists := (*proof)["branches"]; branchExists && len((*proof)["branches"].([]P)) > 0 {
		(*proof)["branches"].([]P)[0]["branches"] = []P{btcBranch}
	} else {
		(*proof)["branches"] = []P{btcBranch}
	}
	return nil
}

func (proof *P) SetProofType(network string, chainType string) string {
	if network == "testnet" {
		return "t" + chainType
	}
	return chainType
}
