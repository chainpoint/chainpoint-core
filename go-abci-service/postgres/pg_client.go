package postgres

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/chainpoint/chainpoint-core/go-abci-service/types"
	"strings"

	"github.com/chainpoint/chainpoint-core/go-abci-service/util"

	_ "github.com/lib/pq"
	"github.com/tendermint/tendermint/libs/log"
)

// Postgres : holds db connection info
type Postgres struct {
	DB     sql.DB
	Logger log.Logger
}

//NewPG : creates new postgres connection and tests it
func NewPG(user string, password string, host string, port string, dbName string, logger log.Logger) (*Postgres, error) {
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, password, host, port, dbName)
	db, err := sql.Open("postgres", connStr)
	if util.LoggerError(logger, err) != nil {
		return nil, err
	}
	err = db.Ping()
	if util.LoggerError(logger, err) != nil {
		return nil, err
	}
	return &Postgres{
		DB:     *db,
		Logger: logger,
	}, nil
}

func NewPGFromURI(connStr string, logger log.Logger) (*Postgres, error) {
	db, err := sql.Open("postgres", connStr)
	if util.LoggerError(logger, err) != nil {
		return nil, err
	}
	err = db.Ping()
	if util.LoggerError(logger, err) != nil {
		return nil, err
	}
	return &Postgres{
		DB:     *db,
		Logger: logger,
	}, nil
}

// GetProofIdsByAggIds : get proof ids from proof table, based on aggId
func (pg *Postgres) GetProofIdsByAggIds(aggIds []string) ([]string, error) {
	stmt := "SELECT proof_id FROM proofs WHERE agg_id::TEXT = ANY($1);"
	rows, err := pg.DB.Query(stmt, aggIds)
	if err != nil {
		return []string{}, err
	}
	proofIds := make([]string, 0)
	for rows.Next() {
		proofid := ""
		switch err := rows.Scan(&proofid); err {
		case sql.ErrNoRows:
			return []string{}, nil
		case nil:
			proofIds = append(proofIds, proofid)
			break;
		default:
			util.LoggerError(pg.Logger, err)
			return []string{}, err
		}
	}
	return proofIds, nil
}

//GetCalStateObjectsByProofIds : Get calstate objects, given an array of aggIds
func (pg *Postgres) GetCalStateObjectsByAggIds(aggIds []string) ([]types.CalStateObject, error) {
	stmt := "SELECT agg_id, cal_id, cal_state FROM cal_states WHERE agg_id::TEXT = ANY($1);"
	rows, err := pg.DB.Query(stmt, aggIds)
	if err != nil {
		return []types.CalStateObject{}, err
	}
	calStates := make([]types.CalStateObject, 0)
	for rows.Next() {
		var calState types.CalStateObject
		switch err := rows.Scan(&calState.AggID, &calState.CalId, &calState.CalState); err {
		case sql.ErrNoRows:
			return []types.CalStateObject{}, nil
		case nil:
			calStates = append(calStates, calState)
			break;
		default:
			util.LoggerError(pg.Logger, err)
			return []types.CalStateObject{}, err
		}
	}
	return calStates, nil
}

//GetAggStateObjectsByProofIds : Get aggstate objects, given an array of proofIds
func (pg *Postgres) GetAggStateObjectsByProofIds(proofIds []string) ([]types.AggState, error) {
	stmt := "SELECT proof_id, hash, agg_id, agg_state, agg_root FROM agg_states WHERE proof_id::TEXT = ANY($1);"
	rows, err := pg.DB.Query(stmt, proofIds)
	if err != nil {
		return []types.AggState{}, err
	}
	aggStates := make ([]types.AggState, 0)
	for rows.Next() {
		var aggState types.AggState
		switch err := rows.Scan(&aggState.ProofID, &aggState.Hash, &aggState.AggID, &aggState.AggState, &aggState.AggRoot); err {
		case sql.ErrNoRows:
			return []types.AggState{}, nil
		case nil:
			aggStates = append(aggStates, aggState)
			break;
		default:
			util.LoggerError(pg.Logger, err)
			return []types.AggState{}, err
		}
	}
	return aggStates, err
}

//GetAnchorBTCAggStateObjectsByCalIds: Get anchor state objects, given an array of calIds
func (pg *Postgres) GetAnchorBTCAggStateObjectsByCalIds(calIds []string) ([]types.AnchorBtcAggState, error) {
	stmt := "SELECT cal_id, anchor_btc_agg_id, anchor_btc_agg_state FROM anchor_btc_agg_states WHERE cal_id::TEXT = ANY($1);"
	rows, err := pg.DB.Query(stmt, calIds)
	if err != nil {
		return []types.AnchorBtcAggState{}, err
	}
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
func (pg *Postgres) GetBTCTxStateObjectByAnchorBTCAggId(aggIds []string) ([]types.AnchorBtcTxState, error) {
	stmt := "SELECT cal_id, anchor_btc_agg_id, anchor_btc_agg_state FROM btctx_states WHERE agg_id::TEXT = ANY($1);"
	rows, err := pg.DB.Query(stmt, aggIds)
	if err != nil {
		return []types.AnchorBtcTxState{}, err
	}
	aggStates := make([]types.AnchorBtcTxState, 0)
	for rows.Next() {
		var aggState types.AnchorBtcTxState
		switch err := rows.Scan(&aggState.AnchorBtcAggId, &aggState.BtcTxId, &aggState.BtcTxState); err {
		case sql.ErrNoRows:
			return []types.AnchorBtcTxState{}, nil
		case nil:
			aggStates = append(aggStates, aggState)
			break
		default:
			util.LoggerError(pg.Logger, err)
			return []types.AnchorBtcTxState{}, err
		}
	}
	return aggStates, err
}

//GetBTCHeadStateObjectByBTCTxId: Get btc header state objects, given an array of btcTxIds
func (pg *Postgres) GetBTCHeadStateObjectByBTCTxId(btcTxIds []string) ([]types.AnchorBtcHeadState, error) {
	stmt := "SELECT btctx_id, btchead_height, btchead_state FROM btchead_states WHERE btctx_id = ANY($1);"
	rows, err := pg.DB.Query(stmt, btcTxIds)
	if err != nil {
		return []types.AnchorBtcHeadState{}, err
	}
	aggStates := make([]types.AnchorBtcHeadState, 0)
	for rows.Next() {
		var aggState types.AnchorBtcHeadState
		switch err := rows.Scan(&aggState.BtcTxId, &aggState.BtcHeadHeight, &aggState.BtcHeadState); err {
		case sql.ErrNoRows:
			return []types.AnchorBtcHeadState{}, nil
		case nil:
			aggStates = append(aggStates, aggState)
			break;
		default:
			util.LoggerError(pg.Logger, err)
			return []types.AnchorBtcHeadState{}, err
		}
	}
	return aggStates, err
}

//BulkInsertProofs : Use pg driver and loop to create bulk proof insert statement
func (pg *Postgres) BulkInsertProofs(proofs []types.ProofState) error {
	insert := "INSERT INTO proofs (proof_id, proof, created_at, updated_at) VALUES "
	values := []string{}
	valuesArgs := make([]interface{}, 0)
	i := 0
	for _, p := range proofs {
		jsonStr, err := json.Marshal(p.Proof)
		if util.LoggerError(pg.Logger, err) != nil {
			continue
		}
		values = append(values, fmt.Sprintf("($%d, $%d, clock_timestamp(), clock_timestamp())", i * 2 + 1, i * 2 + 2))
		valuesArgs = append(valuesArgs, p.ProofID)
		valuesArgs = append(valuesArgs, jsonStr)
		i++
	}
	stmt := insert + strings.Join(values, ", ") + " ON CONFLICT (proof_id) DO UPDATE SET proof = EXCLUDED.proof"
	_, err := pg.DB.Exec(stmt, valuesArgs)
	return err
}

// BulkInsertAggState : inserts aggregator state into postgres
func (pg *Postgres) BulkInsertAggState (aggStates []types.AggState) error {
	insert := "INSERT INTO agg_states (proof_id, hash, agg_id, agg_state, agg_root, created_at, updated_at) VALUES "
	values := []string{}
	valuesArgs := make([]interface{}, 0)
	i := 0
	for _, a := range aggStates{
		values = append(values, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, clock_timestamp(), clock_timestamp())", i * 5 + 1, i * 5 + 2, i * 5 + 3, i * 5 + 4, i * 5 + 5))
		valuesArgs = append(valuesArgs, a.ProofID)
		valuesArgs = append(valuesArgs, a.Hash)
		valuesArgs = append(valuesArgs, a.AggID)
		valuesArgs = append(valuesArgs, a.AggState)
		valuesArgs = append(valuesArgs, a.AggRoot)
		i++
	}
	stmt := insert + strings.Join(values, ", ") + " ON CONFLICT (proof_id) DO NOTHING"
	_, err := pg.DB.Exec(stmt, valuesArgs)
	return err
}

// BulkInsertAggState : inserts aggregator state into postgres
func (pg *Postgres) BulkInsertCalState (calStates []types.CalStateObject) error {
	insert := "INSERT INTO cal_states (agg_id, cal_id, cal_state, created_at, updated_at) VALUES "
	values := []string{}
	valuesArgs := make([]interface{}, 0)
	i := 0
	for _, c := range calStates{
		values = append(values, fmt.Sprintf("($%d, $%d, $%d, clock_timestamp(), clock_timestamp())", i * 3 + 1, i * 3 + 2, i * 3 + 3))
		valuesArgs = append(valuesArgs, c.AggID)
		valuesArgs = append(valuesArgs, c.CalId)
		valuesArgs = append(valuesArgs, c.CalState)
		i++
	}
	stmt := insert + strings.Join(values, ", ") + " ON CONFLICT (agg_id) DO NOTHING"
	_, err := pg.DB.Exec(stmt, valuesArgs)
	return err
}
