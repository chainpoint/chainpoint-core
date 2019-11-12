package postgres

import (
	"database/sql"
	"fmt"

	"github.com/chainpoint/chainpoint-core/go-abci-service/types"

	"github.com/chainpoint/chainpoint-core/go-abci-service/util"

	"github.com/chainpoint/tendermint/libs/log"
	_ "github.com/lib/pq"
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

//CoreUpsert : Inserts a new core row if it doesn't exist, other wise updates it on conflict
func (pg *Postgres) CoreUpsert(core types.Core) (bool, error) {
	stmt := "INSERT INTO staked_cores (eth_addr, public_ip, core_id, block_number, created_at, updated_at) " +
		"VALUES ($1, $2, $3, $4, now(), now()) " +
		"ON CONFLICT (eth_addr) " +
		"DO UPDATE " +
		"SET " +
		"public_ip = $2, " +
		"core_id = $3, " +
		"block_number = $4 " +
		"WHERE $4 > staked_cores.block_number OR $2 <> staked_cores.public_ip;"
	res, err := pg.DB.Exec(stmt, core.EthAddr, core.PublicIP, core.CoreId, core.BlockNumber)
	if util.LoggerError(pg.Logger, err) != nil {
		return false, err
	}
	affect, err := res.RowsAffected()
	if util.LoggerError(pg.Logger, err) != nil {
		return false, err
	}
	if affect > 0 {
		return true, nil
	}
	return false, nil
}

//CoreDelete : deletes a row if the blockNumber of the input unstake event is higher
func (pg *Postgres) CoreDelete(core types.Core) (bool, error) {
	stmt := "DELETE FROM staked_cores WHERE eth_addr = $1 AND block_number > $2;"
	res, err := pg.DB.Exec(stmt, core.EthAddr, core.BlockNumber)
	if util.LoggerError(pg.Logger, err) != nil {
		return false, err
	}
	affect, err := res.RowsAffected()
	if util.LoggerError(pg.Logger, err) != nil {
		return false, err
	}
	if affect > 0 {
		return true, nil
	}
	return false, nil
}

//GetCoreCount: retrieve number of known cores
func (pg *Postgres) GetCoreCount() (int, error) {
	stmt := "SELECT count(*) FROM staked_cores;" //WHERE (staked_nodes.public_ip <> NULL) AND (staked_nodes.public_ip <> '');"
	row := pg.DB.QueryRow(stmt)
	var coreCount int
	switch err := row.Scan(&coreCount); err {
	case sql.ErrNoRows:
		return 0, nil
	case nil:
		return coreCount, nil
	default:
		return 0, err
	}
}

//GetCoreByID : get staked nodes by their public IP string (should be unique)
func (pg *Postgres) GetCoreByID(coreId string) (types.Core, error) {
	stmt := "SELECT eth_addr, public_ip, core_id, block_number FROM staked_cores where core_id = $1"
	row := pg.DB.QueryRow(stmt, coreId)
	var core types.Core
	switch err := row.Scan(&core.EthAddr, &core.PublicIP, &core.CoreId, &core.BlockNumber); err {
	case sql.ErrNoRows:
		return types.Core{}, nil
	case nil:
		return core, nil
	default:
		util.LoggerError(pg.Logger, err)
		return types.Core{}, err
	}
}
