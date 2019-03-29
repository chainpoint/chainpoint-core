package postgres

import (
	"database/sql"
	"fmt"

	"github.com/chainpoint/chainpoint-core/go-abci-service/ethcontracts"

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

//Inserts a new node row if it doesn't exist, other wise updates it on conflict
func (pg *Postgres) NodeUpsert(node types.Node) (bool, error) {
	stmt := "INSERT INTO staked_node (eth_addr, public_ip, amount_staked, stake_expiration, block_number, created_at, updated_at) " +
		"VALUES ($1, $2, $3, $4, $5, now(), now()) " +
		"ON CONFLICT (eth_addr) " +
		"DO UPDATE " +
		"SET " +
		"public_ip = $2, " +
		"amount_staked = $3, " +
		"stake_expiration = $4, " +
		"block_number = $5 " +
		"WHERE $5 > staked_node.block_number OR $2 <> staked_node.public_ip;"
	res, err := pg.DB.Exec(stmt, node.EthAddr, node.PublicIP, node.AmountStaked, node.StakeExpiration, node.BlockNumber)
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

// GetNodeByEthAddr : gets staked nodes by their ethereum address (in hex with 0x format)
func (pg *Postgres) GetNodeByEthAddr(ethAddr string) (types.Node, error) {
	stmt := "SELECT eth_addr, public_ip, amount_staked, stake_expiration, active_token_hash, active_token_timestamp, balance, block_number FROM staked_node where eth_addr = $1"
	row := pg.DB.QueryRow(stmt, ethAddr)
	var node types.Node
	switch err := row.Scan(&node.EthAddr, &node.PublicIP, &node.AmountStaked, &node.StakeExpiration, &node.ActiveTokenHash, &node.ActiveTokenTimestamp, &node.Balance, &node.BlockNumber); err {
	case sql.ErrNoRows:
		return types.Node{}, nil
	case nil:
		return node, nil
	default:
		util.LoggerError(pg.Logger, err)
		return types.Node{}, err
	}
}

//GetNodeByPublicIP : get staked nodes by their public IP string (should be unique)
func (pg *Postgres) GetNodeByPublicIP(publicIP string) (types.Node, error) {
	stmt := "SELECT eth_addr, public_ip, amount_staked, stake_expiration, active_token_hash, active_token_timestamp, balance, block_number FROM staked_node where public_ip = $1"
	row := pg.DB.QueryRow(stmt, publicIP)
	var node types.Node
	switch err := row.Scan(&node.EthAddr, &node.PublicIP, &node.AmountStaked, &node.StakeExpiration, &node.ActiveTokenHash, &node.ActiveTokenTimestamp, &node.Balance, &node.BlockNumber); err {
	case sql.ErrNoRows:
		return types.Node{}, nil
	case nil:
		return node, nil
	default:
		util.LoggerError(pg.Logger, err)
		return types.Node{}, err
	}
}

func (pg *Postgres) HandleNodeStaking(node ethcontracts.ChpRegistryNodeStaked) error {
	newNode := types.Node{
		EthAddr:         node.Sender.Hex(),
		PublicIP:        sql.NullString{String: util.BytesToIP(node.NodeIp[:]), Valid: true},
		AmountStaked:    sql.NullInt64{Int64: node.AmountStaked.Int64(), Valid: true},
		StakeExpiration: sql.NullInt64{Int64: node.Duration.Int64(), Valid: true},
		BlockNumber:     sql.NullInt64{Int64: int64(node.Raw.BlockNumber), Valid: true},
	}
	inserted, err := pg.NodeUpsert(newNode)
	if util.LoggerError(pg.Logger, err) != nil {
		return err
	}
	pg.Logger.Info(fmt.Sprintf("Inserted for %#v: %t\n", newNode, inserted))
	return nil
}

func (pg *Postgres) HandleNodeStakeUpdating(node ethcontracts.ChpRegistryNodeStakeUpdated) error {
	newNode := types.Node{
		EthAddr:         node.Sender.Hex(),
		PublicIP:        sql.NullString{String: util.BytesToIP(node.NodeIp[:]), Valid: true},
		AmountStaked:    sql.NullInt64{Int64: node.AmountStaked.Int64(), Valid: true},
		StakeExpiration: sql.NullInt64{Int64: node.Duration.Int64(), Valid: true},
		BlockNumber:     sql.NullInt64{Int64: int64(node.Raw.BlockNumber), Valid: true},
	}
	inserted, err := pg.NodeUpsert(newNode)
	if util.LoggerError(pg.Logger, err) != nil {
		return err
	}
	pg.Logger.Info(fmt.Sprintf("Updated for %#v: %t\n", newNode, inserted))
	return nil
}
