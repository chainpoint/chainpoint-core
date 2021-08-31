package level

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/chainpoint/chainpoint-core/go-abci-service/types"
	"github.com/chainpoint/chainpoint-core/go-abci-service/util"
	"github.com/lib/pq"
	"strings"
)

// GetProofIdsByAggIds : get proof ids from agg table, based on aggId
func (cache *Cache) GetProofIdsByAggIds(aggIds []string) ([]string, error) {
	aggResults := []string{}
	for _, id := range aggIds {
		results, err := cache.Get("agg_state:" + id)
		if err != nil {
			return []string{}, err
		}
		for _, res := range results {
			aggState := types.AggState{}
			json.Unmarshal([]byte(res), &aggState)
			aggResults = append(aggResults, aggState.ProofID)
		}
	}
	return aggResults, nil
}

// GetProofsByProofIds : get proofs from proof table, based on id
func (cache *Cache) GetProofsByProofIds(proofIds []string) (map[string]types.ProofState, error) {
	proofs := make(map[string]types.ProofState)
	for _, id := range proofIds {
		results, err := cache.Get("proof:" + id)
		if err != nil {
			return map[string]types.ProofState{}, err
		}
		for _, res := range results {
			proof := types.ProofState{}
			json.Unmarshal([]byte(res), &proof)
			proofs[proof.ProofID] = proof
		}
	}
	return proofs, nil
}

// GetProofIdsByBtcTxId : get proof ids from proof table, based on btctxId
func (cache *Cache) GetProofIdsByBtcTxId(btcTxId string) ([]string, error) {
	btcTxStateStr, err := cache.GetOne("btctxstate:" + btcTxId)
	if err != nil {
		return []string{}, nil
	}
	btcTxState := types.AnchorBtcTxState{}
	json.Unmarshal([]byte(btcTxStateStr), &btcTxState)
	anchoraggs, err := cache.Get("anchorbtcaggstate:"+btcTxState.AnchorBtcAggId)
	if err != nil {
		return []string{}, nil
	}
	proofIds := []string{}
	for _, agg := range anchoraggs {
		anchorAggState := types.AnchorBtcAggState{}
		if err := json.Unmarshal([]byte(agg), &anchorAggState); err != nil {
			continue
		}
		calstates, err := cache.Get("calstate:" + anchorAggState.CalId)
		if err != nil {
			continue
		}
		for _, cs := range calstates {
			cso := types.CalStateObject{}
			if err := json.Unmarshal([]byte(cs), &cso); err != nil {
				continue
			}
			aggstates, err := cache.Get("aggstate:" + cso.AggID)
			if err != nil {
				continue
			}
			for _, aggs := range aggstates {
				aggState := types.AggState{}
				if err := json.Unmarshal([]byte(aggs), &aggState); err != nil {
					continue
				}
				proofIds = append(proofIds, aggState.ProofID)
			}
		}
	}
	return proofIds, nil
}

//GetCalStateObjectsByProofIds : Get calstate objects, given an array of aggIds
func  (cache *Cache) GetCalStateObjectsByAggIds(aggIds []string) ([]types.CalStateObject, error) {
	results := []types.CalStateObject{}
	for _, agg := range aggIds {
		cals, err := cache.Get("cal_state_by_agg_id:" + agg)
		if err != nil {
			return []types.CalStateObject{}, err
		}
		for _, cal := range cals {
			result := types.CalStateObject{}
			if err := json.Unmarshal([]byte(cal), &result); err != nil {
				continue
			}
			results = append(results, result)
		}
	}
	return results, nil
}

//GetAggStateObjectsByProofIds : Get aggstate objects, given an array of proofIds
func (cache *Cache) GetAggStateObjectsByProofIds(proofIds []string) ([]types.AggState, error) {
	results := []types.AggState{}
	for _, id := range proofIds {
		aggs, err := cache.Get("agg_state_by_proof_id:" + id)
		if err != nil {
			return []types.AggState{}, err
		}
		for _, agg := range aggs {
			result := types.AggState{}
			if err := json.Unmarshal([]byte(agg), &result); err != nil {
				continue
			}
			results = append(results, result)
		}
	}
	return results, nil
}

//GetAnchorBTCAggStateObjectsByCalIds: Get anchor state objects, given an array of calIds
func (cache *Cache) GetAnchorBTCAggStateObjectsByCalIds(calIds []string) ([]types.AnchorBtcAggState, error) {
	//pg.Logger.Info(util.GetCurrentFuncName(1))
	stmt := "SELECT cal_id, anchor_btc_agg_id, anchor_btc_agg_state FROM anchor_btc_agg_states WHERE cal_id::TEXT = ANY($1);"
	rows, err := pg.DB.Query(stmt, pq.Array(calIds))
	if err != nil {
		return []types.AnchorBtcAggState{}, err
	}
	defer rows.Close()
	aggStates := make([]types.AnchorBtcAggState, 0)
	for rows.Next() {
		var aggState types.AnchorBtcAggState
		switch err := rows.Scan(&aggState.CalId, &aggState.AnchorBtcAggId, &aggState.AnchorBtcAggState); err {
		case sql.ErrNoRows:
			return []types.AnchorBtcAggState{}, nil
		case nil:
			aggStates = append(aggStates, aggState)
			break
		default:
			util.LoggerError(pg.Logger, err)
			return []types.AnchorBtcAggState{}, err
		}
	}
	return aggStates, err
}

//GetBTCTxStateObjectByAnchorBTCAggId: Get btc state objects, given an array of agg ids
func (cache *Cache) GetBTCTxStateObjectByAnchorBTCAggId(aggId string) (types.AnchorBtcTxState, error) {
	//pg.Logger.Info(util.GetCurrentFuncName(1))
	stmt := "SELECT anchor_btc_agg_id, btctx_id, btctx_state FROM btctx_states WHERE anchor_btc_agg_id::TEXT = $1;"
	rows, err := pg.DB.Query(stmt, aggId)
	if err != nil {
		return types.AnchorBtcTxState{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var aggState types.AnchorBtcTxState
		switch err := rows.Scan(&aggState.AnchorBtcAggId, &aggState.BtcTxId, &aggState.BtcTxState); err {
		case sql.ErrNoRows:
			return types.AnchorBtcTxState{}, nil
		case nil:
			return aggState, nil
		default:
			util.LoggerError(pg.Logger, err)
			return types.AnchorBtcTxState{}, err
		}
	}
	return types.AnchorBtcTxState{}, err
}

//GetBTCHeadStateObjectByBTCTxId: Get btc header state objects, given an array of btcTxIds
func (cache *Cache) GetBTCHeadStateObjectByBTCTxId(btcTxId string) (types.AnchorBtcHeadState, error) {
	//pg.Logger.Info(util.GetCurrentFuncName(1))
	stmt := "SELECT btctx_id, btchead_height, btchead_state FROM btchead_states WHERE btctx_id = $1;"
	rows, err := pg.DB.Query(stmt, btcTxId)
	if err != nil {
		return types.AnchorBtcHeadState{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var aggState types.AnchorBtcHeadState
		switch err := rows.Scan(&aggState.BtcTxId, &aggState.BtcHeadHeight, &aggState.BtcHeadState); err {
		case sql.ErrNoRows:
			return types.AnchorBtcHeadState{}, nil
		case nil:
			return aggState, nil
		default:
			util.LoggerError(pg.Logger, err)
			return types.AnchorBtcHeadState{}, err
		}
	}
	return types.AnchorBtcHeadState{}, err
}

//BulkInsertProofs : Use pg driver and loop to create bulk proof insert statement
func (cache *Cache) BulkInsertProofs(proofs []types.ProofState) error {
	//pg.Logger.Info(util.GetCurrentFuncName(1))
	insert := "INSERT INTO proofs (proof_id, proof, created_at, updated_at) VALUES "
	values := []string{}
	valuesArgs := make([]interface{}, 0)
	i := 0
	for _, p := range proofs {
		values = append(values, fmt.Sprintf("($%d, $%d, clock_timestamp(), clock_timestamp())", i*2+1, i*2+2))
		valuesArgs = append(valuesArgs, p.ProofID)
		valuesArgs = append(valuesArgs, p.Proof)
		i++
	}
	stmt := insert + strings.Join(values, ", ") + " ON CONFLICT (proof_id) DO UPDATE SET proof = EXCLUDED.proof"
	pg.Logger.Info(fmt.Sprintf("INSERT INTO PROOFS: %s", stmt))
	_, err := pg.DB.Exec(stmt, valuesArgs...)
	return err
}

// BulkInsertAggState : inserts aggregator state into postgres
func (cache *Cache) BulkInsertAggState(aggStates []types.AggState) error {
	//pg.Logger.Info(util.GetCurrentFuncName(1))
	insert := "INSERT INTO agg_states (proof_id, hash, agg_id, agg_state, agg_root, created_at, updated_at) VALUES "
	values := []string{}
	valuesArgs := make([]interface{}, 0)
	i := 0
	for _, a := range aggStates {
		values = append(values, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, clock_timestamp(), clock_timestamp())", i*5+1, i*5+2, i*5+3, i*5+4, i*5+5))
		valuesArgs = append(valuesArgs, a.ProofID)
		valuesArgs = append(valuesArgs, a.Hash)
		valuesArgs = append(valuesArgs, a.AggID)
		valuesArgs = append(valuesArgs, a.AggState)
		valuesArgs = append(valuesArgs, a.AggRoot)
		i++
	}
	stmt := insert + strings.Join(values, ", ") + " ON CONFLICT (proof_id) DO NOTHING"
	_, err := pg.DB.Exec(stmt, valuesArgs...)
	return err
}

// BulkInsertCalState : inserts aggregator state into postgres
func (cache *Cache) BulkInsertCalState(calStates []types.CalStateObject) error {
	//pg.Logger.Info(util.GetCurrentFuncName(1))
	insert := "INSERT INTO cal_states (agg_id, cal_id, cal_state, created_at, updated_at) VALUES "
	values := []string{}
	valuesArgs := make([]interface{}, 0)
	i := 0
	for _, c := range calStates {
		values = append(values, fmt.Sprintf("($%d, $%d, $%d, clock_timestamp(), clock_timestamp())", i*3+1, i*3+2, i*3+3))
		valuesArgs = append(valuesArgs, c.AggID)
		valuesArgs = append(valuesArgs, c.CalId)
		valuesArgs = append(valuesArgs, c.CalState)
		i++
	}
	stmt := insert + strings.Join(values, ", ") + " ON CONFLICT (agg_id) DO NOTHING"
	_, err := pg.DB.Exec(stmt, valuesArgs...)
	return err
}

// BulkInsertBtcAggState : inserts aggregator state into postgres
func (cache *Cache) BulkInsertBtcAggState(aggStates []types.AnchorBtcAggState) error {
	//pg.Logger.Info(util.GetCurrentFuncName(1))
	insert := "INSERT INTO anchor_btc_agg_states (cal_id, anchor_btc_agg_id, anchor_btc_agg_state, created_at, updated_at) VALUES "
	values := []string{}
	valuesArgs := make([]interface{}, 0)
	i := 0
	for _, a := range aggStates {
		values = append(values, fmt.Sprintf("($%d, $%d, $%d, clock_timestamp(), clock_timestamp())", i*3+1, i*3+2, i*3+3))
		valuesArgs = append(valuesArgs, a.CalId)
		valuesArgs = append(valuesArgs, a.AnchorBtcAggId)
		valuesArgs = append(valuesArgs, a.AnchorBtcAggState)
		i++
	}
	stmt := insert + strings.Join(values, ", ") + " ON CONFLICT (cal_id) DO NOTHING"
	_, err := pg.DB.Exec(stmt, valuesArgs...)
	return err
}

// BulkInsertBtcTxState : inserts aggregator state into postgres
func (cache *Cache) BulkInsertBtcTxState(txStates []types.AnchorBtcTxState) error {
	//pg.Logger.Info(util.GetCurrentFuncName(1))
	insert := "INSERT INTO btctx_states (anchor_btc_agg_id, btctx_id, btctx_state, created_at, updated_at) VALUES "
	values := []string{}
	valuesArgs := make([]interface{}, 0)
	i := 0
	for _, t := range txStates {
		values = append(values, fmt.Sprintf("($%d, $%d, $%d, clock_timestamp(), clock_timestamp())", i*3+1, i*3+2, i*3+3))
		valuesArgs = append(valuesArgs, t.AnchorBtcAggId)
		valuesArgs = append(valuesArgs, t.BtcTxId)
		valuesArgs = append(valuesArgs, t.BtcTxState)
		i++
	}
	stmt := insert + strings.Join(values, ", ") + " ON CONFLICT (anchor_btc_agg_id) DO UPDATE SET btctx_id = EXCLUDED.btctx_id, btctx_state = EXCLUDED.btctx_state"
	_, err := pg.DB.Exec(stmt, valuesArgs...)
	pg.Logger.Info(fmt.Sprintf("INSERT INTO BTCTX: %s", stmt))
	return err
}

// BulkInsertBtcHeadState : inserts head state into postgres
func (cache *Cache) BulkInsertBtcHeadState(headStates []types.AnchorBtcHeadState) error {
	//pg.Logger.Info(util.GetCurrentFuncName(1))
	insert := "INSERT INTO btchead_states (btctx_id, btchead_height, btchead_state, created_at, updated_at) VALUES "
	values := []string{}
	valuesArgs := make([]interface{}, 0)
	i := 0
	for _, h := range headStates {
		values = append(values, fmt.Sprintf("($%d, $%d, $%d, clock_timestamp(), clock_timestamp())", i*3+1, i*3+2, i*3+3))
		valuesArgs = append(valuesArgs, h.BtcTxId)
		valuesArgs = append(valuesArgs, h.BtcHeadHeight)
		valuesArgs = append(valuesArgs, h.BtcHeadState)
		i++
	}
	stmt := insert + strings.Join(values, ", ") + " ON CONFLICT (btctx_id) DO UPDATE SET btchead_height = EXCLUDED.btchead_height, btchead_state = EXCLUDED.btchead_state"
	_, err := pg.DB.Exec(stmt, valuesArgs...)
	return err
}
