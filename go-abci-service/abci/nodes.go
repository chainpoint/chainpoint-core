package abci

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"net"
	"net/http"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

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

//ValidateNodeRepChain : download and verify reputation chain items from a node
func (app *AnchorApplication) ValidateNodeRecentReputation(node types.Node) error {
	repChain, err := GetNodeRecentReputation(node)
	if util.LoggerError(app.logger, err) != nil {
		return err
	}
	err = ValidateRepChain(node, repChain)
	util.LoggerError(app.logger, err)
	return err
}

//SendNodeHash : Post a hash to a node
func SendNodeHash(node types.Node) (bool, error) {
	if net.ParseIP(node.PublicIP.String) != nil {
		HashURI := fmt.Sprintf("http://%s/hashes", node.PublicIP.String)
		nodeHash := types.NodeHash{
			Hashes: []string{"c3ab8ff13720e8ad97dd39466b3c8974e592c2fa383d4a3960714caef0c4f2"},
		}
		hashJson, err := json.Marshal(nodeHash)
		if err != nil {
			return false, err
		}
		req, err := http.NewRequest("POST", HashURI, bytes.NewReader(hashJson))
		if err != nil {
			return false, err
		}
		client := http.Client{}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			return false, err
		}
		if resp.StatusCode == http.StatusOK {
			return true, nil
		}
	}
	return false, errors.New("cannot parse node IP")
}

//ValidateRepChain : validates a reputation chain array struct. Returns nil if valid
func ValidateRepChain(node types.Node, repChain types.RepChain) error {
	for _, repItem := range repChain {
		hash, errHash := ValidateRepChainItemHash(repItem)
		valid, errSig := ValidateRepChainItemSig(node, repItem)
		if errHash != nil {
			return errHash
		}
		if errSig != nil {
			return errSig
		}
		if !valid || hash == "" {
			return errors.New("Rep Item did not validate")
		}
	}
	return nil
}

//GetNodeRecentReputation : get reputation chain item array from node_ip/reputation/recent
func GetNodeRecentReputation(node types.Node) (types.RepChain, error) {
	if net.ParseIP(node.PublicIP.String) != nil {
		RecentRepURI := fmt.Sprintf("http://%s/reputation/recent", node.PublicIP.String)
		resp, err := http.Get(RecentRepURI)
		var repChain types.RepChain
		if err != nil {
			return types.RepChain{}, err
		}
		contents, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return types.RepChain{}, err
		}
		err = json.Unmarshal(contents, &repChain)
		if err != nil {
			return types.RepChain{}, err
		}
		return repChain, nil
	}
	return types.RepChain{}, errors.New("cannot parse node IP")
}

//ValidateRepChainItemHash : Validate hash of chain item
func ValidateRepChainItemHash(chainItem types.RepChainItem) (string, error) {
	buf := new(bytes.Buffer)
	bid := make([]byte, 4)
	bbh := make([]byte, 4)

	binary.BigEndian.PutUint32(bid, chainItem.ID)
	binary.BigEndian.PutUint32(bbh, chainItem.CalBlockHeight)

	buf.Write(bid)
	buf.Write(bbh)

	hashBytes, err := hex.DecodeString(chainItem.CalBlockHash)
	if err != nil {
		return "", err
	}
	buf.Write(hashBytes)

	prevRepItemHashBytes, err := hex.DecodeString(chainItem.PrevRepItemHash)
	if err != nil {
		return "", err
	}
	buf.Write(prevRepItemHashBytes)

	hashIDNodeNoHyphens := strings.Replace(chainItem.HashIDNode, "-", "", -1)
	hashIDNodeNoHyphensBytes, err := hex.DecodeString(hashIDNodeNoHyphens)
	if err != nil {
		return "", err
	}
	buf.Write(hashIDNodeNoHyphensBytes)

	hash := sha256.Sum256(buf.Bytes())
	hashStr := hex.EncodeToString(hash[:])
	if !strings.Contains(chainItem.RepItemHash, hashStr) {
		return "", errors.New(fmt.Sprintf("Hash mismatch between local record %s and repItem %s\n", hashStr, chainItem.RepItemHash))
	}
	return hashStr, nil
}

//ValidateRepChainItemSig : validates the signature from a node's reputation chain item
func ValidateRepChainItemSig(node types.Node, chainItem types.RepChainItem) (bool, error) {
	repItemHashBytes, err := hex.DecodeString(chainItem.RepItemHash)
	if err != nil {
		return true, err
	}
	verified, err := verifySig(node.EthAddr, chainItem.Signature, repItemHashBytes)
	if !verified {
		return false, err
	}
	return true, nil
}

func verifySig(from, sigHex string, msg []byte) (bool, error) {
	fromAddr := common.HexToAddress(from)
	sig, err := hexutil.Decode(sigHex)
	if err != nil {
		return false, err
	}
	if sig[64] != 27 && sig[64] != 28 {
		return false, nil
	}
	sig[64] -= 27
	pubKey, err := crypto.SigToPub(signHash(msg), sig)
	if err != nil {
		return false, err
	}
	recoveredAddr := crypto.PubkeyToAddress(*pubKey)
	return fromAddr == recoveredAddr, nil
}

func signHash(data []byte) []byte {
	msg := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(data), data)
	return crypto.Keccak256([]byte(msg))
}
