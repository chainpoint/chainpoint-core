package abci

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/chainpoint/chainpoint-core/go-abci-service/types"
	"github.com/chainpoint/chainpoint-core/go-abci-service/util"
)

//LoadNodesFromContract : load all past node staking events and update events
func (app *AnchorApplication) LoadNodesFromContract() error {
	//Consume all past node events from this contract and import them into the local postgres instance
	nodesStaked, err := app.ethClient.GetPastNodesStakedEvents()
	if util.LoggerError(app.logger, err) != nil {
		return err
	}
	for _, node := range nodesStaked {
		pubKeyHex := hex.EncodeToString(node.NodePublicKey[:])
		newNode := types.Node{
			EthAddr:         node.Sender.Hex(),
			PublicIP:        sql.NullString{String: util.BytesToIP(node.NodeIp[:]), Valid: true},
			PublicKey:       sql.NullString{String: pubKeyHex, Valid: true},
			AmountStaked:    sql.NullInt64{Int64: node.AmountStaked.Int64(), Valid: true},
			StakeExpiration: sql.NullInt64{Int64: node.Duration.Int64(), Valid: true},
			BlockNumber:     sql.NullInt64{Int64: int64(node.Raw.BlockNumber), Valid: true},
		}
		inserted, err := app.pgClient.NodeUpsert(newNode)
		if util.LoggerError(app.logger, err) != nil {
			return err
		}
		app.logger.Info(fmt.Sprintf("Inserted for %#v: %t\n", newNode, inserted))
	}

	//Consume all updated events and reconcile them with the previous states
	nodesStakedUpdated, err := app.ethClient.GetPastNodesStakeUpdatedEvents()
	if util.LoggerError(app.logger, err) != nil {
		return err
	}
	for _, node := range nodesStakedUpdated {
		pubKeyHex := hex.EncodeToString(node.PublicKey[:])
		newNode := types.Node{
			EthAddr:         node.Sender.Hex(),
			PublicIP:        sql.NullString{String: util.BytesToIP(node.NodeIp[:]), Valid: true},
			PublicKey:       sql.NullString{String: pubKeyHex, Valid: true},
			AmountStaked:    sql.NullInt64{Int64: node.AmountStaked.Int64(), Valid: true},
			StakeExpiration: sql.NullInt64{Int64: node.Duration.Int64(), Valid: true},
			BlockNumber:     sql.NullInt64{Int64: int64(node.Raw.BlockNumber), Valid: true},
		}
		inserted, err := app.pgClient.NodeUpsert(newNode)
		if util.LoggerError(app.logger, err) != nil {
			return err
		}
		fmt.Printf("Inserted Update for %#v: %t\n", newNode, inserted)
	}
	return nil
}

//WatchNodesFromContract : get all future node staking events and updates
func (app *AnchorApplication) WatchNodesFromContract() error {
	highestBlock, err := app.ethClient.HighestBlock()
	if util.LoggerError(app.logger, err) != nil {
		highestBlock = big.NewInt(0)
	}
	go app.ethClient.WatchNodeStakeEvents(app.pgClient.HandleNodeStaking, *highestBlock)
	go app.ethClient.WatchNodeStakeUpdatedEvents(app.pgClient.HandleNodeStakeUpdating, *highestBlock)
	return nil
}

//ValidateRepChainItemHash : Validate hash of chain item
func (app *AnchorApplication) ValidateRepChainItemHash(chainItem types.RepChainItem) (string, error) {
	buf := new(bytes.Buffer)
	bid := make([]byte, 4)
	bbh := make([]byte, 4)

	binary.BigEndian.PutUint32(bid, chainItem.ID)
	binary.BigEndian.PutUint32(bbh, chainItem.CalBlockHeight)

	buf.Write(bid)
	buf.Write(bbh)

	hashBytes, err := hex.DecodeString(chainItem.CalBlockHash)
	util.LoggerError(app.logger, err)
	buf.Write(hashBytes)

	prevRepItemHashBytes, err := hex.DecodeString(chainItem.PrevRepItemHash)
	util.LoggerError(app.logger, err)
	buf.Write(prevRepItemHashBytes)

	hashIDNodeNoHyphens := strings.Replace(chainItem.HashIDNode, "-", "", -1)
	hashIDNodeNoHyphensBytes, err := hex.DecodeString(hashIDNodeNoHyphens)
	util.LoggerError(app.logger, err)
	buf.Write(hashIDNodeNoHyphensBytes)

	hash := sha256.Sum256(buf.Bytes())
	hashStr := hex.EncodeToString(hash[:])
	if !strings.Contains(chainItem.RepItemHash, hashStr) {
		return "", errors.New(fmt.Sprintf("Hash mismatch between local record %s and repItem %s\n", hashStr, chainItem.RepItemHash))
	}
	return hashStr, nil
}

//ValidateRepChainItemSig : validates the signature from a node's reputation chain item
func (app *AnchorApplication) ValidateRepChainItemSig(node types.Node, chainItem types.RepChainItem) error {
	repItemHashBytes, err := hex.DecodeString(chainItem.RepItemHash)
	if util.LoggerError(app.logger, err) != nil {
		return err
	}
	sigBytes, err := hex.DecodeString(chainItem.Signature)
	if util.LoggerError(app.logger, err) != nil {
		return err
	}
	sigPublicKey, err := crypto.Ecrecover(repItemHashBytes, sigBytes)
	pubKeyHex := hex.EncodeToString(sigPublicKey)
	if !strings.Contains(node.PublicKey.String, pubKeyHex) {
		return errors.New(fmt.Sprintf("Signature mismatch between local record %s and repItem %s\n", node.PublicKey.String, pubKeyHex))
	}
	return nil
}
