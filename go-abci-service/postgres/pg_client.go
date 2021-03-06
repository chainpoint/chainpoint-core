package postgres

import (
	"database/sql"
	"fmt"
	"github.com/chainpoint/chainpoint-core/go-abci-service/types"
	"github.com/lib/pq"
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
	//pg.Logger.Info(util.GetCurrentFuncName(1))
	stmt := "SELECT proof_id FROM agg_states WHERE agg_id::TEXT = ANY($1);"
	rows, err := pg.DB.Query(stmt, pq.Array(aggIds))
	if err != nil {
		return []string{}, err
	}
	defer rows.Close()
	proofIds := make([]string, 0)
	for rows.Next() {
		var proofid string
		switch err := rows.Scan(&proofid); err {
		case sql.ErrNoRows:
			return []string{}, nil
		case nil:
			proofIds = append(proofIds, proofid)
			break
		default:
			util.LoggerError(pg.Logger, err)
			return []string{}, err
		}
	}
	return proofIds, nil
}

// GetProofsByProofIds : get proofs from proof table, based on id
func (pg *Postgres) GetProofsByProofIds(proofIds []string) (map[string]types.ProofState, error) {
	//pg.Logger.Info(util.GetCurrentFuncName(1))
	stmt := "SELECT proof_id, proof FROM proofs WHERE proof_id::TEXT = ANY($1);"
	rows, err := pg.DB.Query(stmt, pq.Array(proofIds))
	if err != nil {
		return map[string]types.ProofState{}, err
	}
	defer rows.Close()
	proofs := make(map[string]types.ProofState)
	for rows.Next() {
		var proof types.ProofState
		switch err := rows.Scan(&proof.ProofID, &proof.Proof); err {
		case sql.ErrNoRows:
			return map[string]types.ProofState{}, nil
		case nil:
			proofs[proof.ProofID] = proof
			break
		default:
			util.LoggerError(pg.Logger, err)
			return map[string]types.ProofState{}, err
		}
	}
	return proofs, nil
}

// GetProofIdsByBtcTxId : get proof ids from proof table, based on btctxId
func (pg *Postgres) GetProofIdsByBtcTxId(btcTxId string) ([]string, error) {
	//pg.Logger.Info(util.GetCurrentFuncName(1))
	stmt := `SELECT a.proof_id FROM agg_states a
    INNER JOIN cal_states c ON c.agg_id = a.agg_id
    INNER JOIN anchor_btc_agg_states aa ON aa.cal_id = c.cal_id
    INNER JOIN btctx_states tx ON tx.anchor_btc_agg_id = aa.anchor_btc_agg_id
    WHERE tx.btctx_id = $1`
	rows, err := pg.DB.Query(stmt, btcTxId)
	if err != nil {
		return []string{}, err
	}
	defer rows.Close()
	proofIds := make([]string, 0)
	for rows.Next() {
		var proofid string
		switch err := rows.Scan(&proofid); err {
		case sql.ErrNoRows:
			return []string{}, nil
		case nil:
			proofIds = append(proofIds, proofid)
			break
		default:
			util.LoggerError(pg.Logger, err)
			return []string{}, err
		}
	}
	return proofIds, nil
}

//GetCalStateObjectsByProofIds : Get calstate objects, given an array of aggIds
func (pg *Postgres) GetCalStateObjectsByAggIds(aggIds []string) ([]types.CalStateObject, error) {
	//pg.Logger.Info(util.GetCurrentFuncName(1))
	stmt := "SELECT agg_id, cal_id, cal_state FROM cal_states WHERE agg_id::TEXT = ANY($1);"
	rows, err := pg.DB.Query(stmt, pq.Array(aggIds))
	if err != nil {
		return []types.CalStateObject{}, err
	}
	defer rows.Close()
	calStates := make([]types.CalStateObject, 0)
	for rows.Next() {
		var calState types.CalStateObject
		switch err := rows.Scan(&calState.AggID, &calState.CalId, &calState.CalState); err {
		case sql.ErrNoRows:
			return []types.CalStateObject{}, nil
		case nil:
			calStates = append(calStates, calState)
			break
		default:
			util.LoggerError(pg.Logger, err)
			return []types.CalStateObject{}, err
		}
	}
	return calStates, nil
}

//GetAggStateObjectsByProofIds : Get aggstate objects, given an array of proofIds
func (pg *Postgres) GetAggStateObjectsByProofIds(proofIds []string) ([]types.AggState, error) {
	//pg.Logger.Info(util.GetCurrentFuncName(1))
	stmt := "SELECT proof_id, hash, agg_id, agg_state, agg_root FROM agg_states WHERE proof_id::TEXT = ANY($1);"
	rows, err := pg.DB.Query(stmt, pq.Array(proofIds))
	if err != nil {
		return []types.AggState{}, err
	}
	defer rows.Close()
	aggStates := make([]types.AggState, 0)
	for rows.Next() {
		var aggState types.AggState
		switch err := rows.Scan(&aggState.ProofID, &aggState.Hash, &aggState.AggID, &aggState.AggState, &aggState.AggRoot); err {
		case sql.ErrNoRows:
			return []types.AggState{}, nil
		case nil:
			aggStates = append(aggStates, aggState)
			break
		default:
			util.LoggerError(pg.Logger, err)
			return []types.AggState{}, err
		}
	}
	return aggStates, err
}

//GetAnchorBTCAggStateObjectsByCalIds: Get anchor state objects, given an array of calIds
func (pg *Postgres) GetAnchorBTCAggStateObjectsByCalIds(calIds []string) ([]types.AnchorBtcAggState, error) {
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
func (pg *Postgres) GetBTCTxStateObjectByAnchorBTCAggId(aggId string) (types.AnchorBtcTxState, error) {
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
func (pg *Postgres) GetBTCHeadStateObjectByBTCTxId(btcTxId string) (types.AnchorBtcHeadState, error) {
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
func (pg *Postgres) BulkInsertProofs(proofs []types.ProofState) error {
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
func (pg *Postgres) BulkInsertAggState(aggStates []types.AggState) error {
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
func (pg *Postgres) BulkInsertCalState(calStates []types.CalStateObject) error {
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
func (pg *Postgres) BulkInsertBtcAggState(aggStates []types.AnchorBtcAggState) error {
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
func (pg *Postgres) BulkInsertBtcTxState(txStates []types.AnchorBtcTxState) error {
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
func (pg *Postgres) BulkInsertBtcHeadState(headStates []types.AnchorBtcHeadState) error {
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

// PruneProofStateTables : prunes proof tables
func (pg *Postgres) PruneProofStateTables() error {
	tables := []string{"proofs", "agg_states", "cal_states", "anchor_btc_agg_states", "btctx_states", "btchead_states"}
	var err error
	for _, tabl := range tables {
		go func(table string) {
			pruneStmt := fmt.Sprintf("DELETE FROM %s WHERE created_at < NOW() - INTERVAL '24 HOURS'", table)
			_, err := pg.DB.Exec(pruneStmt)
			util.LoggerError(pg.Logger, err)
		}(tabl)
	}
	return err
}
