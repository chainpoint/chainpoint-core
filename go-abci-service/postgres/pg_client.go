package postgres

import (
	"database/sql"
	"fmt"
	"github.com/chainpoint/chainpoint-core/go-abci-service/types"

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

//GetCalStateObjectsByProofIds : Get calstate objects, given an array of aggIds
func (pg *Postgres) GetCalStateObjectsByAggIds(aggIds []string) (types.CalStateObject, error) {
	stmt := "SELECT agg_id, cal_id, cal_state FROM cal_states WHERE agg_id::TEXT = ANY($1);"
	row := pg.DB.QueryRow(stmt, aggIds)
	var calState types.CalStateObject
	switch err := row.Scan(&calState.AggID, &calState.CalId, &calState.CalState); err {
	case sql.ErrNoRows:
		return types.CalStateObject{}, nil
	case nil:
		return calState, nil
	default:
		util.LoggerError(pg.Logger, err)
		return calState, err
	}
}

//GetAggStateObjectsByProofIds : Get aggstate objects, given an array of proofIds
func (pg *Postgres) GetAggStateObjectsByProofIds(proofIds []string) (types.AggState, error) {
	stmt := "SELECT proof_id, hash, agg_id, agg_state, agg_root FROM agg_states WHERE proof_id::TEXT = ANY($1);"
	row := pg.DB.QueryRow(stmt, proofIds)
	var aggState types.AggState
	switch err := row.Scan(&aggState.ProofID, &aggState.Hash, &aggState.AggID, &aggState.AggState, &aggState.AggRoot); err {
	case sql.ErrNoRows:
		return types.AggState{}, nil
	case nil:
		return aggState, nil
	default:
		util.LoggerError(pg.Logger, err)
		return aggState, err
	}
}

//GetAnchorBTCAggStateObjectsByCalIds: Get anchor state objects, given an array of calIds
func (pg *Postgres) GetAnchorBTCAggStateObjectsByCalIds(calIds []string) (types.AnchorBtcAggState, error) {
	stmt := "SELECT cal_id, anchor_btc_agg_id, anchor_btc_agg_state FROM anchor_btc_agg_states WHERE cal_id::TEXT = ANY($1);"
	row := pg.DB.QueryRow(stmt, calIds)
	var aggState types.AnchorBtcAggState
	switch err := row.Scan(&aggState.CalId, &aggState.AnchorBtcAggId, &aggState.AnchorBtcAggState); err {
	case sql.ErrNoRows:
		return types.AnchorBtcAggState{}, nil
	case nil:
		return aggState, nil
	default:
		util.LoggerError(pg.Logger, err)
		return aggState, err
	}
}

//GetBTCTxStateObjectByAnchorBTCAggId: Get btc state objects, given an array of agg ids
func (pg *Postgres) GetBTCTxStateObjectByAnchorBTCAggId(aggIds []string) (types.AnchorBtcTxState, error) {
	stmt := "SELECT cal_id, anchor_btc_agg_id, anchor_btc_agg_state FROM btctx_states WHERE agg_id::TEXT = ANY($1);"
	row := pg.DB.QueryRow(stmt, aggIds)
	var aggState types.AnchorBtcTxState
	switch err := row.Scan(&aggState.AnchorBtcAggId, &aggState.BtcTxId, &aggState.BtcTxState); err {
	case sql.ErrNoRows:
		return types.AnchorBtcTxState{}, nil
	case nil:
		return aggState, nil
	default:
		util.LoggerError(pg.Logger, err)
		return aggState, err
	}
}

//GetBTCTxStateObjectByAnchorBTCAggId: Get btc header state objects, given an array of btcTxIds
func (pg *Postgres) GetBTCHeadStateObjectByBTCTxId(btcTxIds []string) (types.AnchorBtcHeadState, error) {
	stmt := "SELECT btctx_id, btchead_height, btchead_state FROM btchead_states WHERE btctx_id = ANY($1);"
	row := pg.DB.QueryRow(stmt, btcTxIds)
	var aggState types.AnchorBtcHeadState
	switch err := row.Scan(&aggState.BtcTxId, &aggState.BtcHeadHeight, &aggState.BtcHeadState); err {
	case sql.ErrNoRows:
		return types.AnchorBtcHeadState{}, nil
	case nil:
		return aggState, nil
	default:
		util.LoggerError(pg.Logger, err)
		return aggState, err
	}
}
