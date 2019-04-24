package postgres

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

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

//NodeUpsert : Inserts a new node row if it doesn't exist, other wise updates it on conflict
func (pg *Postgres) NodeUpsert(node types.Node) (bool, error) {
	stmt := "INSERT INTO node_state (eth_addr, public_ip, block_number, created_at, updated_at) " +
		"VALUES ($1, $2, $3, now(), now()) " +
		"ON CONFLICT (eth_addr) " +
		"DO UPDATE " +
		"SET " +
		"public_ip = $2, " +
		"block_number = $3 " +
		"WHERE $3 > node_state.block_number OR $2 <> node_state.public_ip;"
	res, err := pg.DB.Exec(stmt, node.EthAddr, node.PublicIP, node.BlockNumber)
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

//NodeDelete : deletes a row if the blockNumber of the input unstake event is higher
func (pg *Postgres) NodeDelete(node types.Node) (bool, error) {
	stmt := "DELETE FROM node_state WHERE eth_addr = $1 AND block_number > $2;"
	res, err := pg.DB.Exec(stmt, node.EthAddr, node.BlockNumber)
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

//TokenHashUpsert : insert new token hash when received by the network
func (pg *Postgres) TokenHashUpsert(data string) (bool, error) {
	payloadSlice := strings.Split(data, "|")
	if len(payloadSlice) != 2 {
		pg.Logger.Error(fmt.Sprintf("Error: Token payload %s is malformed", data))
		return false, errors.New("TOKEN tx is malformed")
	}
	nodeIP := payloadSlice[0]
	tokenHash := payloadSlice[1]
	stmt := "INSERT INTO active_token (node_ip, token_hash, created_at, updated_at) " +
		"VALUES ($1, $2, now(), now()) " +
		"ON CONFLICT (node_ip) " +
		"DO UPDATE " +
		"SET " +
		"node_ip = $1, " +
		"token_hash = $2;"
	res, err := pg.DB.Exec(stmt, nodeIP, tokenHash)
	if util.LoggerError(pg.Logger, err) != nil {
		pg.Logger.Error("Unable to execute upsert on active_token table")
		return false, err
	}
	affect, err := res.RowsAffected()
	if util.LoggerError(pg.Logger, err) != nil {
		pg.Logger.Error("Unable to obtain rows affected by postgres upsert")
		return false, err
	}
	if affect > 0 {
		return true, nil
	}
	return false, nil
}

//GetSeededRandomNodes : Get seeded random sequence of 3 nodes from the node_states table
func (pg *Postgres) GetSeededRandomNodes(seed []byte) ([]types.Node, error) {
	//Usage:
	//nodes, err := app.pgClient.GetSeededRandomNodes([]byte("3719ADA3EEE198F3A7A33616EA60ED6D72D94D31A2B2422FA12E2BCDDCABD4D4"))
	//fmt.Printf("Random nodes: %#v\n", nodes)
	seedFloat := util.GetSeededRandFloat([]byte(seed))
	seedStmt := fmt.Sprintf("SET seed TO %f;", seedFloat)
	_, err := pg.DB.Exec(seedStmt)
	if util.LoggerError(pg.Logger, err) != nil {
		return []types.Node{}, err
	}
	randomStmt := "SELECT eth_addr, public_ip,block_number FROM node_state ORDER BY random() LIMIT 3;"
	rows, err := pg.DB.Query(randomStmt)
	if util.LoggerError(pg.Logger, err) != nil {
		return []types.Node{}, err
	}
	defer rows.Close()
	nodes := make([]types.Node, 0)
	for rows.Next() {
		var node types.Node
		switch err := rows.Scan(&node.EthAddr, &node.PublicIP, &node.BlockNumber); err {
		case sql.ErrNoRows:
			return []types.Node{}, nil
		case nil:
			nodes = append(nodes, node)
			break
		default:
			util.LoggerError(pg.Logger, err)
			return []types.Node{}, err
		}
	}
	return nodes, nil
}

//GetRandomNodes : Get random sequence of 3 nodes from the node_states table
func (pg *Postgres) GetRandomNodes() ([]types.Node, error) {
	randomStmt := "SELECT eth_addr, public_ip,block_number FROM node_state ORDER BY random() LIMIT 3;"
	rows, err := pg.DB.Query(randomStmt)
	if util.LoggerError(pg.Logger, err) != nil {
		return []types.Node{}, err
	}
	defer rows.Close()
	nodes := make([]types.Node, 0)
	for rows.Next() {
		var node types.Node
		switch err := rows.Scan(&node.EthAddr, &node.PublicIP, &node.BlockNumber); err {
		case sql.ErrNoRows:
			return []types.Node{}, nil
		case nil:
			nodes = append(nodes, node)
			break
		default:
			util.LoggerError(pg.Logger, err)
			return []types.Node{}, err
		}
	}
	return nodes, nil
}

func (pg *Postgres) GetNodeCount() (int, error) {
	stmt := "SELECT count(*) FROM node_state;" //WHERE (node_state.public_ip <> NULL) AND (node_state.public_ip <> '');"
	row := pg.DB.QueryRow(stmt)
	var nodeCount int
	switch err := row.Scan(&nodeCount); err {
	case sql.ErrNoRows:
		return 0, nil
	case nil:
		return nodeCount, nil
	default:
		return 0, err
	}
}

// GetNodeByEthAddr : gets staked nodes by their ethereum address (in hex with 0x format)
func (pg *Postgres) GetNodeByEthAddr(ethAddr string) (types.Node, error) {
	stmt := "SELECT eth_addr, public_ip, block_number FROM node_state where eth_addr = $1"
	row := pg.DB.QueryRow(stmt, ethAddr)
	var node types.Node
	switch err := row.Scan(&node.EthAddr, &node.PublicIP, &node.BlockNumber); err {
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
	stmt := "SELECT eth_addr, public_ip, block_number FROM node_state where public_ip = $1"
	row := pg.DB.QueryRow(stmt, publicIP)
	var node types.Node
	switch err := row.Scan(&node.EthAddr, &node.PublicIP, &node.BlockNumber); err {
	case sql.ErrNoRows:
		return types.Node{}, nil
	case nil:
		return node, nil
	default:
		util.LoggerError(pg.Logger, err)
		return types.Node{}, err
	}
}

//HandleNodeStaking : receive watch event and upsert it into db
func (pg *Postgres) HandleNodeStaking(node ethcontracts.ChpRegistryNodeStaked) error {
	newNode := types.Node{
		EthAddr:     node.Sender.Hex(),
		PublicIP:    sql.NullString{String: util.Int2Ip(node.NodeIp).String(), Valid: true},
		BlockNumber: sql.NullInt64{Int64: int64(node.Raw.BlockNumber), Valid: true},
	}
	inserted, err := pg.NodeUpsert(newNode)
	if util.LoggerError(pg.Logger, err) != nil {
		return err
	}
	pg.Logger.Info(fmt.Sprintf("Inserted for %#v: %t\n", newNode, inserted))
	return nil
}

//HandleNodeStakeUpdating : receive update event and upsert it into db if it supercedes the existing row
func (pg *Postgres) HandleNodeStakeUpdating(node ethcontracts.ChpRegistryNodeStakeUpdated) error {
	newNode := types.Node{
		EthAddr:     node.Sender.Hex(),
		PublicIP:    sql.NullString{String: util.Int2Ip(node.NodeIp).String(), Valid: true},
		BlockNumber: sql.NullInt64{Int64: int64(node.Raw.BlockNumber), Valid: true},
	}
	inserted, err := pg.NodeUpsert(newNode)
	if util.LoggerError(pg.Logger, err) != nil {
		return err
	}
	pg.Logger.Info(fmt.Sprintf("Updated for %#v: %t\n", newNode, inserted))
	return nil
}

//HandleNodeUnstaking : receive unstake event and delete it if the event is more recent than the stake or update
func (pg *Postgres) HandleNodeUnstake(node ethcontracts.ChpRegistryNodeUnStaked) error {
	newNode := types.Node{
		EthAddr:     node.Sender.Hex(),
		PublicIP:    sql.NullString{String: util.Int2Ip(node.NodeIp).String(), Valid: true},
		BlockNumber: sql.NullInt64{Int64: int64(node.Raw.BlockNumber), Valid: true},
	}
	inserted, err := pg.NodeDelete(newNode)
	if util.LoggerError(pg.Logger, err) != nil {
		return err
	}
	pg.Logger.Info(fmt.Sprintf("Deleted for %#v: %t\n", newNode, inserted))
	return nil
}
