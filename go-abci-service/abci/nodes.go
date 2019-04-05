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
	"sync"
	"time"

	"github.com/chainpoint/chainpoint-core/go-abci-service/ethcontracts"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/chainpoint/chainpoint-core/go-abci-service/types"
	"github.com/chainpoint/chainpoint-core/go-abci-service/util"
)

//MintRewardNodes : mint rewards for nodes
func (app *AnchorApplication) MintRewardNodes(sig []string) error {
	if leader, _ := app.ElectLeader(1); leader {
		sigBytes := make([][]byte, len(sig))
		for i, sigStr := range sig {
			decodedSig, err := hex.DecodeString(sigStr)
			if util.LoggerError(app.logger, err) != nil {
				continue
			}
			sigBytes[i] = decodedSig
		}
		rewardCandidates, rewardHash, err := app.GetNodeRewardCandidates()
		if util.LoggerError(app.logger, err) != nil {
			return err
		}
		err = app.ethClient.Mint(rewardCandidates, rewardHash, sigBytes)
		if util.LoggerError(app.logger, err) != nil {
			return err
		}
		app.RewardSignatures = make([]string, 0)
	}
	return nil
}

//CollectRewardNodes : collate and sign reward node list
func (app *AnchorApplication) CollectRewardNodes() error {
	if leader, _ := app.ElectLeader(5); leader {
		currentEthBlock, err := app.ethClient.HighestBlock()
		if util.LoggerError(app.logger, err) != nil {
			return err
		}
		if currentEthBlock.Int64()-app.state.LastMintedAtBlock < 5760 {
			return errors.New("Too soon for minting")
		}
		_, rewardHash, err := app.GetNodeRewardCandidates()
		if util.LoggerError(app.logger, err) != nil {
			return err
		}
		signature, err := ethcontracts.SignMsg(rewardHash, app.ethClient.EthPrivateKey)
		if util.LoggerError(app.logger, err) != nil {
			return err
		}
		res, err := app.rpc.BroadcastTx("SIGN", hex.EncodeToString(signature), 2, time.Now().Unix())
		if err != nil {
			return err
		}
		if res.Code == 0 {
			return nil
		}
		return errors.New("did not successfully broadcast SIGN-RC tx")
	}
	return nil
}

//GetNodeRewardCandidates : scans for and collates the reward candidates in the current epoch
func (app *AnchorApplication) GetNodeRewardCandidates() ([]common.Address, []byte, error) {
	txResult, err := app.rpc.client.TxSearch(fmt.Sprintf("NODERC=%d", app.state.LastMintedAtBlock), false, 1, 25)
	if util.LoggerError(app.logger, err) != nil {
		return []common.Address{}, []byte{}, err
	}
	nodeArray := make([]common.Address, 0)
	for _, tx := range txResult.Txs {
		decoded, err := util.DecodeTx(tx.Tx)
		if err != nil {
			continue
		}
		var nodes []types.NodeJson
		if err := json.Unmarshal([]byte(decoded.Data), &nodes); err != nil {
			return []common.Address{}, []byte{}, err
		}
		for _, nodeJson := range nodes {
			nodeArray = append(nodeArray, common.HexToAddress(nodeJson.EthAddr))
		}
	}
	addresses := uniquify(nodeArray)
	rewardHash := ethcontracts.AddressesToHash(addresses)
	return addresses, rewardHash, nil
}

//AuditNodes : Audit nodes for reputation chain and hash submission validity. Submits reward tx with info if successful
func (app *AnchorApplication) AuditNodes() error {
	if leader, _ := app.ElectLeader(1); leader {
		status, err := app.rpc.GetStatus()
		if util.LoggerError(app.logger, err) != nil {
			return err
		}
		blockHash := status.SyncInfo.LatestBlockHash.String()
		rewardCandidates := make([]types.NodeJson, 0)
		startTime := time.Now()
		var wg sync.WaitGroup
		var mux sync.Mutex
		for len(rewardCandidates) < 3 || time.Since(startTime) < (5*time.Minute) {
			nodes, err := app.pgClient.GetSeededRandomNodes([]byte(blockHash))
			if util.LoggerError(app.logger, err) != nil {
				return err
			}
			for _, nodeCandidate := range nodes {
				go func(node types.Node) {
					wg.Add(1)
					defer wg.Done()
					if err := app.AuditNode(node); err != nil {
						return
					}
					nodeJson := types.NodeJson{
						EthAddr:  node.EthAddr,
						PublicIP: node.PublicIP.String,
					}
					mux.Lock()
					rewardCandidates = append(rewardCandidates, nodeJson)
					mux.Unlock()
				}(nodeCandidate)
			}
			wg.Wait()
		}
		if len(rewardCandidates) > 3 {
			rewardCandidates = rewardCandidates[0:3]
		}
		if len(rewardCandidates) == 0 {
			return errors.New("Unspecified reward candidate collation failure")
		}
		rcJson, err := json.Marshal(rewardCandidates)
		if err != nil {
			return err
		}
		res, err := app.rpc.BroadcastTx("NODE-RC", string(rcJson), 2, time.Now().Unix())
		if err != nil {
			return err
		}
		if res.Code == 0 {
			return nil
		}
		return errors.New("Problem validating and submitting nodes for rewards!")
	}
	app.logger.Info("Not leader")
	return nil
}

//AuditNode : Used by the audit leader and all confirming cores to validate node performance
func (app *AnchorApplication) AuditNode(node types.Node) error {
	err := ValidateNodeRecentReputation(node)
	if util.LoggerError(app.logger, err) != nil {
		return err
	}
	nodeResp, err := SendNodeHash(node)
	if util.LoggerError(app.logger, err) != nil && len(nodeResp.Hashes) == 0 {
		return err
	}
	time.Sleep(180 * time.Second)
	err = RetrieveNodeCalProof(node, nodeResp)
	if err != nil {
		return err
	}
	return nil
}

//LoadNodesFromContract : load all past node staking events and update events
func (app *AnchorApplication) LoadNodesFromContract() error {
	//Consume all past node events from this contract and import them into the local postgres instance
	nodesStaked, err := app.ethClient.GetPastNodesStakedEvents()
	if util.LoggerError(app.logger, err) != nil {
		return err
	}
	for _, node := range nodesStaked {
		newNode := types.Node{
			EthAddr:     node.Sender.Hex(),
			PublicIP:    sql.NullString{String: util.BytesToIP(node.NodeIp[:]), Valid: true},
			BlockNumber: sql.NullInt64{Int64: int64(node.Raw.BlockNumber), Valid: true},
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
		newNode := types.Node{
			EthAddr:     node.Sender.Hex(),
			PublicIP:    sql.NullString{String: util.BytesToIP(node.NodeIp[:]), Valid: true},
			BlockNumber: sql.NullInt64{Int64: int64(node.Raw.BlockNumber), Valid: true},
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
func ValidateNodeRecentReputation(node types.Node) error {
	repChain, err := GetNodeRecentReputation(node)
	if err != nil {
		return err
	}
	err = ValidateRepChain(node, repChain)
	return err
}

//SendNodeHash : Post a hash to a node
func SendNodeHash(node types.Node) (types.NodeHashResponse, error) {
	if net.ParseIP(node.PublicIP.String) != nil {
		HashURI := fmt.Sprintf("http://%s/hashes", node.PublicIP.String)
		nodeHash := types.NodeHash{
			Hashes: []string{"c3ab8ff13720e8ad97dd39466b3c8974e592c2fa383d4a3960714caef0c4f2"},
		}
		hashJson, err := json.Marshal(nodeHash)
		if err != nil {
			return types.NodeHashResponse{}, err
		}
		req, err := http.NewRequest("POST", HashURI, bytes.NewReader(hashJson))
		if err != nil {
			return types.NodeHashResponse{}, err
		}
		client := http.Client{}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		var nodeResponse types.NodeHashResponse
		resp, err := client.Do(req)
		if err != nil {
			return types.NodeHashResponse{}, err
		}
		if resp.StatusCode == http.StatusOK {
			contents, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return types.NodeHashResponse{}, err
			}
			err = json.Unmarshal(contents, &nodeResponse)
			if err != nil {
				return types.NodeHashResponse{}, err
			}
			return nodeResponse, nil
		}
	}
	return types.NodeHashResponse{}, errors.New("cannot parse node IP")
}

//RetrieveNodeCalProof : get back a node cal proof to validate Node health
func RetrieveNodeCalProof(node types.Node, hashID types.NodeHashResponse) error {
	if net.ParseIP(node.PublicIP.String) != nil && len(hashID.Hashes) > 0 {
		calProofURI := fmt.Sprintf("http://%s/proofs/%s", node.PublicIP.String, hashID.Hashes[0].HashIDNode)
		resp, err := http.Get(calProofURI)
		if err != nil {
			return err
		}
		contents, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		var nodeProof types.NodeProofResponse
		if err := json.Unmarshal(contents, &nodeProof); err != nil {
			return err
		}
		if len(nodeProof) > 0 && len(nodeProof[0].AnchorsComplete) > 0 {
			for _, anchor := range nodeProof[0].AnchorsComplete {
				if anchor == "cal" {
					return nil
				}
			}
		}
		return errors.New("no Cal anchor")
	}
	return errors.New("cannot parse node IP")
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

func uniquify(s []common.Address) []common.Address {
	seen := make(map[common.Address]struct{}, len(s))
	j := 0
	for _, v := range s {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		s[j] = v
		j++
	}
	return s[:j]
}
