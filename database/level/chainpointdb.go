package level

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/chainpoint/chainpoint-core/types"
	db "github.com/tendermint/tm-db"
	"strconv"
	"strings"
	"time"
)

type Chainpoint_DB struct {
	db *KVStore
}

func NewDB(db *KVStore) *Chainpoint_DB {
	return &Chainpoint_DB{db}
}

// GetProofIdsByAggIds : get proof ids from agg table, based on aggId
func (chp *Chainpoint_DB) GetProofIdsByAggIds(aggIds []string) ([]string, error) {
	aggResults := []string{}
	for _, id := range aggIds {
		results, err := chp.db.GetArray("aggstate:" + id)
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
func (chp *Chainpoint_DB) GetProofsByProofIds(proofIds []string) (map[string]types.ProofState, error) {
	proofs := make(map[string]types.ProofState)
	for _, id := range proofIds {
		result, err := chp.db.Get("proof:" + id)
		if err != nil {
			return map[string]types.ProofState{}, err
		}
		proof := types.ProofState{}
		json.Unmarshal([]byte(result), &proof)
		proofs[proof.ProofID] = proof
	}
	return proofs, nil
}

// GetProofIdsByBtcTxId : get proof ids from proof table, based on btctxId
func (chp *Chainpoint_DB) GetProofIdsByBtcTxId(btcTxId string) ([]string, error) {
	btcTxStateStr, err := chp.db.Get("btctxstate:" + btcTxId)
	if err != nil {
		return []string{}, nil
	}
	btcTxState := types.AnchorBtcTxState{}
	json.Unmarshal([]byte(btcTxStateStr), &btcTxState)
	chp.db.Logger.Info(fmt.Sprintf("Getting anchorbtcaggstates for tx %s", btcTxId))
	anchoraggs, err := chp.db.GetArray("anchorbtcaggstate:" + btcTxState.AnchorBtcAggId)
	if err != nil {
		return []string{}, nil
	}
	proofIds := []string{}
	for _, agg := range anchoraggs {
		anchorAggState := types.AnchorBtcAggState{}
		if err := json.Unmarshal([]byte(agg), &anchorAggState); err != nil {
			continue
		}
		chp.db.Logger.Info(fmt.Sprintf("Getting calStates %s for AnchorBtcAggState %s", anchorAggState.CalId, anchorAggState.AnchorBtcAggId))
		calstates, err := chp.db.GetArray("calstate:" + anchorAggState.CalId)
		if err != nil {
			continue
		}
		for _, cs := range calstates {
			cso := types.CalStateObject{}
			if err := json.Unmarshal([]byte(cs), &cso); err != nil {
				continue
			}
			chp.db.Logger.Info(fmt.Sprintf("Getting aggState %s for calState %s", cso.AggID, cso.CalId))
			aggstates, err := chp.db.GetArray("aggstate:" + cso.AggID)
			if err != nil {
				continue
			}
			for _, aggs := range aggstates {
				aggState := types.AggState{}
				if err := json.Unmarshal([]byte(aggs), &aggState); err != nil {
					continue
				}
				chp.db.Logger.Info(fmt.Sprintf("Getting proofID %s for aggState %s", aggState.ProofID, cso.AggID))
				proofIds = append(proofIds, aggState.ProofID)
			}
		}
	}
	return proofIds, nil
}

//GetCalStateObjectsByProofIds : GetArray calstate objects, given an array of aggIds
func (chp *Chainpoint_DB) GetCalStateObjectsByAggIds(aggIds []string) ([]types.CalStateObject, error) {
	results := []types.CalStateObject{}
	for _, agg := range aggIds {
		cals, err := chp.db.GetArray("calstate_by_agg:" + agg)
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

//GetAggStateObjectsByProofIds : GetArray aggstate objects, given an array of proofIds
func (chp *Chainpoint_DB) GetAggStateObjectsByProofIds(proofIds []string) ([]types.AggState, error) {
	results := []types.AggState{}
	for _, id := range proofIds {
		aggs, err := chp.db.GetArray("aggstate_by_proof:" + id)
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

//GetAnchorBTCAggStateObjectsByCalIds: GetArray anchor state objects, given an array of calIds
func (chp *Chainpoint_DB) GetAnchorBTCAggStateObjectsByCalIds(calIds []string) ([]types.AnchorBtcAggState, error) {
	results := []types.AnchorBtcAggState{}
	for _, id := range calIds {
		anchoraggs, err := chp.db.GetArray("anchorbtcaggstate_by_cal:" + id)
		if err != nil {
			return []types.AnchorBtcAggState{}, nil
		}
		for _, agg := range anchoraggs {
			result := types.AnchorBtcAggState{}
			if err := json.Unmarshal([]byte(agg), &result); err != nil {
				continue
			}
			results = append(results, result)
		}
	}
	return results, nil
}

//GetBTCTxStateObjectByAnchorBTCAggId: GetArray btc state objects, given an array of agg ids
func (chp *Chainpoint_DB) GetBTCTxStateObjectByAnchorBTCAggId(aggId string) (types.AnchorBtcTxState, error) {
	btcTxStateStr, err := chp.db.Get("btctxstate_by_agg:" + aggId)
	if err != nil {
		return types.AnchorBtcTxState{}, err
	}
	btcTxState := types.AnchorBtcTxState{}
	if err := json.Unmarshal([]byte(btcTxStateStr), &btcTxState); err != nil {
		return types.AnchorBtcTxState{}, err
	}
	return btcTxState, nil
}

//GetBTCTxStateObjectByAnchorBTCHeadState GetArray btc state objects, given an array of agg ids
func (chp *Chainpoint_DB) GetBTCTxStateObjectByBtcHeadState(btctx string) (types.AnchorBtcTxState, error) {
	btcTxStateStr, err := chp.db.Get("btctxstate:" + btctx)
	if err != nil {
		return types.AnchorBtcTxState{}, err
	}
	btcTxState := types.AnchorBtcTxState{}
	if err := json.Unmarshal([]byte(btcTxStateStr), &btcTxState); err != nil {
		return types.AnchorBtcTxState{}, err
	}
	return btcTxState, nil
}

//BulkInsertProofs : Use pg driver and loop to create bulk proof insert statement
func (chp *Chainpoint_DB) BulkInsertProofs(proofs []types.ProofState) error {
	for _, proof := range proofs {
		proofExists, err := chp.db.Get("proof:" + proof.ProofID)
		if err != nil {
			return err
		}
		if strings.Contains(proofExists, "btc_anchor_branch") {
			chp.db.Logger.Info(fmt.Sprintf("ProofID %s duplicated", proof.ProofID))
			continue
		}
		p, err := json.Marshal(proof)
		if err != nil {
			return err
		}
		err = chp.db.Set("proof:"+proof.ProofID, string(p))
		if err != nil {
			return err
		}
		chp.db.Set("proofCreated:"+proof.ProofID, strconv.FormatInt(time.Now().Unix(), 10))
	}
	return nil
}

// BulkInsertAggState : inserts aggregator state into postgres
func (chp *Chainpoint_DB) BulkInsertAggState(aggStates []types.AggState) error {
	for _, agg := range aggStates {
		a, err := json.Marshal(agg)
		if err != nil {
			continue
		}
		err = chp.db.Append("aggstate:"+agg.AggID, string(a))
		err2 := chp.db.Append("aggstate_by_proof:"+agg.ProofID, string(a))
		chp.db.Set("aggstateCreated:"+agg.ProofID, strconv.FormatInt(time.Now().Unix(), 10))
		if err != nil || err2 != nil {
			return errors.New("aggstate insert failed")
		}
	}
	return nil
}

// BulkInsertCalState : inserts aggregator state into postgres
func (chp *Chainpoint_DB) BulkInsertCalState(calStates []types.CalStateObject) error {
	for _, cal := range calStates {
		c, err := json.Marshal(cal)
		if err != nil {
			continue
		}
		err = chp.db.Append("calstate:"+cal.CalId, string(c))
		err2 := chp.db.Append("calstate_by_agg:"+cal.AggID, string(c))
		chp.db.Set("calstateCreated:"+cal.AggID, strconv.FormatInt(time.Now().Unix(), 10))
		if err != nil || err2 != nil {
			return errors.New("calstate insert failed")
		}
	}
	return nil
}

// BulkInsertBtcAggState : inserts aggregator state into postgres
func (chp *Chainpoint_DB) BulkInsertBtcAggState(aggStates []types.AnchorBtcAggState) error {
	for _, agg := range aggStates {
		a, err := json.Marshal(agg)
		if err != nil {
			continue
		}
		err = chp.db.Append("anchorbtcaggstate:"+agg.AnchorBtcAggId, string(a))
		err2 := chp.db.Append("anchorbtcaggstate_by_cal:"+agg.CalId, string(a))
		chp.db.Set("anchorbtcaggstateCreated:"+agg.CalId, strconv.FormatInt(time.Now().Unix(), 10))
		if err != nil || err2 != nil {
			return errors.New("anchorbtcaggstate insert failed")
		}
	}
	return nil
}

// BulkInsertBtcTxState : inserts aggregator state into postgres
func (chp *Chainpoint_DB) BulkInsertBtcTxState(txStates []types.AnchorBtcTxState) error {
	for _, state := range txStates {
		s, err := json.Marshal(state)
		if err != nil {
			continue
		}
		err = chp.db.Set("btctxstate:"+state.BtcTxId, string(s))
		err2 := chp.db.Set("btctxstate_by_agg:"+state.AnchorBtcAggId, string(s))
		chp.db.Set("btctxstateCreated:"+state.AnchorBtcAggId, strconv.FormatInt(time.Now().Unix(), 10))
		if err != nil || err2 != nil {
			return errors.New("anchorbtcaggstate insert failed")
		}
	}
	return nil
}

func (chp *Chainpoint_DB) PruneOldState() {
	btctxstateIt, _ := db.IteratePrefix(chp.db.LevelDb, []byte("btctxstateCreated:"))
	anchoraggstateIt, _ := db.IteratePrefix(chp.db.LevelDb, []byte("anchorbtcaggstateCreated:"))
	calstateIt, _ := db.IteratePrefix(chp.db.LevelDb, []byte("calstateCreated:"))
	aggstateIt, _ := db.IteratePrefix(chp.db.LevelDb, []byte("aggstateCreated:"))
	proofstateIt, _ := db.IteratePrefix(chp.db.LevelDb, []byte("proofCreated:"))
	for ; btctxstateIt.Valid(); btctxstateIt.Next() {
		value := btctxstateIt.Value()
		t, err := strconv.ParseInt(string(value), 10, 64)
		if err != nil {
			continue
		}
		tm := time.Unix(t, 0)
		if time.Now().After(tm.Add(24 * time.Hour)) {
			key := string(btctxstateIt.Key())
			id := strings.Split(key, ":")[1]
			state, _ := chp.GetBTCTxStateObjectByAnchorBTCAggId(id)
			chp.db.Del(key, "")
			chp.db.Del("btctxstate_by_agg:"+id, "")
			chp.db.Del("btctxstate:"+state.BtcTxId, "")
			chp.db.Logger.Info("db pruned", "btcTxState", state.BtcTxId)
		}
	}
	for ; anchoraggstateIt.Valid(); anchoraggstateIt.Next() {
		value := anchoraggstateIt.Value()
		t, err := strconv.ParseInt(string(value), 10, 64)
		if err != nil {
			continue
		}
		tm := time.Unix(t, 0)
		if time.Now().After(tm.Add(24 * time.Hour)) {
			key := string(anchoraggstateIt.Key())
			id := strings.Split(key, ":")[1]
			states, _ := chp.GetAnchorBTCAggStateObjectsByCalIds([]string{id})
			chp.db.Del(key, "")
			chp.db.Del("anchorbtcaggstate_by_cal"+id, "")
			for _, s := range states {
				chp.db.Del("anchorbtcaggstate:"+s.AnchorBtcAggId, "")
			}
		}
	}
	for ; calstateIt.Valid(); calstateIt.Next() {
		value := calstateIt.Value()
		t, err := strconv.ParseInt(string(value), 10, 64)
		if err != nil {
			continue
		}
		tm := time.Unix(t, 0)
		if time.Now().After(tm.Add(24 * time.Hour)) {
			key := string(calstateIt.Key())
			id := strings.Split(key, ":")[1]
			states, _ := chp.GetCalStateObjectsByAggIds([]string{id})
			chp.db.Del(key, "")
			chp.db.Del("calstate_by_agg:"+id, "")
			for _, s := range states {
				chp.db.Del("calstate:"+s.CalId, "")
			}
		}
	}
	for ; aggstateIt.Valid(); aggstateIt.Next() {
		value := aggstateIt.Value()
		t, err := strconv.ParseInt(string(value), 10, 64)
		if err != nil {
			continue
		}
		tm := time.Unix(t, 0)
		if time.Now().After(tm.Add(24 * time.Hour)) {
			key := string(aggstateIt.Key())
			id := strings.Split(key, ":")[1]
			states, _ := chp.GetAggStateObjectsByProofIds([]string{id})
			chp.db.Del(key, "")
			chp.db.Del("aggstate_by_proof:"+id, "")
			for _, s := range states {
				chp.db.Del("aggstate:"+s.AggID, "")
			}
		}
	}
	for ; proofstateIt.Valid(); proofstateIt.Next() {
		value := proofstateIt.Value()
		t, err := strconv.ParseInt(string(value), 10, 64)
		//chp.db.Logger.Info("Iterating", "key", proofstateIt.Key(), "proofstateIt", string(value))
		if err != nil {
			continue
		}
		tm := time.Unix(t, 0)
		if time.Now().After(tm.Add(24 * time.Hour)) {
			key := string(proofstateIt.Key())
			id := strings.Split(key, ":")[1]
			chp.db.Del(key, "")
			chp.db.Del("proof:"+id, "")
			chp.db.Logger.Info("db pruned", "proof", id)
		}
	}
}
