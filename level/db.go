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

// GetProofIdsByAggIds : get proof ids from agg table, based on aggId
func (cache *Cache) GetProofIdsByAggIds(aggIds []string) ([]string, error) {
	aggResults := []string{}
	for _, id := range aggIds {
		results, err := cache.Get("aggstate:" + id)
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
		result, err := cache.GetOne("proof:" + id)
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
func (cache *Cache) GetProofIdsByBtcTxId(btcTxId string) ([]string, error) {
	btcTxStateStr, err := cache.GetOne("btctxstate:" + btcTxId)
	if err != nil {
		return []string{}, nil
	}
	btcTxState := types.AnchorBtcTxState{}
	json.Unmarshal([]byte(btcTxStateStr), &btcTxState)
	cache.Logger.Info(fmt.Sprint("Getting anchorbtcaggstates for tx %s", btcTxId))
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
		cache.Logger.Info(fmt.Sprint("Getting calStates for AnchorBtcAggState %s (calId %s)", anchorAggState.AnchorBtcAggId, anchorAggState.CalId))
		calstates, err := cache.Get("calstate:" + anchorAggState.CalId)
		if err != nil {
			continue
		}
		for _, cs := range calstates {
			cso := types.CalStateObject{}
			if err := json.Unmarshal([]byte(cs), &cso); err != nil {
				continue
			}
			cache.Logger.Info(fmt.Sprint("Getting aggStates for calState %s (aggId %s)", cso.CalId, cso.AggID))
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
		cals, err := cache.Get("calstate_by_agg:" + agg)
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
		aggs, err := cache.Get("aggstate_by_proof:" + id)
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
	results := []types.AnchorBtcAggState{}
	for _, id := range calIds {
		anchoraggs, err := cache.Get("anchorbtcaggstate_by_cal:" + id)
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

//GetBTCTxStateObjectByAnchorBTCAggId: Get btc state objects, given an array of agg ids
func (cache *Cache) GetBTCTxStateObjectByAnchorBTCAggId(aggId string) (types.AnchorBtcTxState, error) {
	btcTxStateStr , err := cache.GetOne("btctxstate_by_agg:" + aggId)
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
func (cache *Cache) BulkInsertProofs(proofs []types.ProofState) error {
	for _, proof := range proofs {
/*		proofExists, err := cache.GetOne("proof:"+proof.ProofID)
		if err != nil {
			return err
		}
		if strings.Contains(proofExists, "btc_anchor_branch") {
			continue
		}*/
		p, err := json.Marshal(proof)
		if err != nil {
			return err
		}
		err = cache.Set("proof:"+proof.ProofID, string(p))
		if err != nil {
			return err
		}
		cache.Set("proofCreated:" + proof.ProofID, strconv.FormatInt(time.Now().Unix(), 10))
	}
	return nil
}

// BulkInsertAggState : inserts aggregator state into postgres
func (cache *Cache) BulkInsertAggState(aggStates []types.AggState) error {
	for _, agg := range aggStates {
		a, err := json.Marshal(agg)
		if err != nil {
			continue
		}
		err = cache.Add("aggstate:" + agg.AggID, string(a))
		err2 := cache.Add("aggstate_by_proof:" + agg.ProofID, string(a))
		cache.Set("aggstateCreated:" + agg.ProofID, strconv.FormatInt(time.Now().Unix(), 10))
		if err != nil || err2 != nil {
			return errors.New("aggstate insert failed")
		}
	}
	return nil
}

// BulkInsertCalState : inserts aggregator state into postgres
func (cache *Cache) BulkInsertCalState(calStates []types.CalStateObject) error {
	for _, cal := range calStates {
		c, err := json.Marshal(cal)
		if err != nil {
			continue
		}
		err = cache.Add("calstate:" + cal.CalId, string(c))
		err2 := cache.Add("calstate_by_agg:" + cal.AggID, string(c))
		cache.Set("calstateCreated:" + cal.AggID, strconv.FormatInt(time.Now().Unix(), 10))
		if err != nil || err2 != nil {
			return errors.New("calstate insert failed")
		}
	}
	return nil
}

// BulkInsertBtcAggState : inserts aggregator state into postgres
func (cache *Cache) BulkInsertBtcAggState(aggStates []types.AnchorBtcAggState) error {
	for _, agg := range aggStates {
		a, err := json.Marshal(agg)
		if err != nil {
			continue
		}
		err = cache.Add("anchorbtcaggstate:"+ agg.AnchorBtcAggId, string(a))
		err2 := cache.Add("anchorbtcaggstate_by_cal:" + agg.CalId, string(a))
		cache.Set("anchorbtcaggstateCreated:" + agg.CalId, strconv.FormatInt(time.Now().Unix(), 10))
		if err != nil || err2 != nil {
			return errors.New("anchorbtcaggstate insert failed")
		}
	}
	return nil
}

// BulkInsertBtcTxState : inserts aggregator state into postgres
func (cache *Cache) BulkInsertBtcTxState(txStates []types.AnchorBtcTxState) error {
	for _, state := range txStates {
		s, err := json.Marshal(state)
		if err != nil {
			continue
		}
		err = cache.Set("btctxstate:"+ state.BtcTxId, string(s))
		err2 := cache.Set("btctxstate_by_agg:" + state.AnchorBtcAggId, string(s))
		cache.Set("btctxstateCreated:" + state.AnchorBtcAggId, strconv.FormatInt(time.Now().Unix(), 10))
		if err != nil || err2 != nil {
			return errors.New("anchorbtcaggstate insert failed")
		}
	}
	return nil
}

func (cache *Cache) PruneOldState() {
	btctxstateIt, _ := db.IteratePrefix(cache.LevelDb, []byte("btctxstateCreated:"))
	anchoraggstateIt, _ := db.IteratePrefix(cache.LevelDb, []byte("anchorbtcaggstateCreated:"))
	calstateIt, _ :=db.IteratePrefix(cache.LevelDb, []byte("calstateCreated:"))
	aggstateIt, _ :=db.IteratePrefix(cache.LevelDb, []byte("aggstateCreated:"))
	proofstateIt, _ :=db.IteratePrefix(cache.LevelDb, []byte("proofCreated:"))
	for ; btctxstateIt.Valid(); btctxstateIt.Next() {
		value := btctxstateIt.Value()
		t, err := strconv.ParseInt(string(value), 10, 64)
		if err != nil {
			continue
		}
		tm := time.Unix(t, 0)
		if time.Now().After(tm.Add(12 * time.Hour)) {
			key := string(btctxstateIt.Key())
			id := strings.Split(key, ":")[1]
			state, _ := cache.GetBTCTxStateObjectByAnchorBTCAggId(id)
			cache.Del(key, "")
			cache.Del("btctxstate_by_agg:" + id, "")
			cache.Del("btctxstate:" + state.BtcTxId, "")
			cache.Logger.Info("db pruned", "btcTxState", state.BtcTxId)
		}
	}
	for ; anchoraggstateIt.Valid(); anchoraggstateIt.Next() {
		value := anchoraggstateIt.Value()
		t, err := strconv.ParseInt(string(value), 10, 64)
		if err != nil {
			continue
		}
		tm := time.Unix(t, 0)
		if time.Now().After(tm.Add(12 * time.Hour)) {
			key := string(anchoraggstateIt.Key())
			id := strings.Split(key, ":")[1]
			states, _ := cache.GetAnchorBTCAggStateObjectsByCalIds([]string{id})
			cache.Del(key, "")
			cache.Del("anchorbtcaggstate_by_cal" + id, "")
			for _, s := range states {
				cache.Del("anchorbtcaggstate:"+s.AnchorBtcAggId, "")
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
		if time.Now().After(tm.Add(12 * time.Hour)) {
			key := string(calstateIt.Key())
			id := strings.Split(key, ":")[1]
			states, _ := cache.GetCalStateObjectsByAggIds([]string{id})
			cache.Del(key, "")
			cache.Del("calstate_by_agg:" + id, "")
			for _, s := range states {
				cache.Del("calstate:"+s.CalId, "")
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
		if time.Now().After(tm.Add(12 * time.Hour)) {
			key := string(aggstateIt.Key())
			id := strings.Split(key, ":")[1]
			states, _ := cache.GetAggStateObjectsByProofIds([]string{id})
			cache.Del(key, "")
			cache.Del("aggstate_by_proof:" + id, "")
			for _, s := range states {
				cache.Del("aggstate:"+s.AggID, "")
			}
		}
	}
	for ; proofstateIt.Valid(); proofstateIt.Next() {
		value := proofstateIt.Value()
		t, err := strconv.ParseInt(string(value), 10, 64)
		//cache.Logger.Info("Iterating", "key", proofstateIt.Key(), "proofstateIt", string(value))
		if err != nil {
			continue
		}
		tm := time.Unix(t, 0)
		if time.Now().After(tm.Add(24 * time.Hour)) {
			key := string(proofstateIt.Key())
			id := strings.Split(key, ":")[1]
			cache.Del(key, "")
			cache.Del("proof:" + id, "")
			cache.Logger.Info("db pruned", "proof", id)
		}
	}
}

