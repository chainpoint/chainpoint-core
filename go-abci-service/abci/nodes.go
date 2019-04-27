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

	"github.com/go-redis/redis"

	"github.com/chainpoint/chainpoint-core/go-abci-service/ethcontracts"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/chainpoint/chainpoint-core/go-abci-service/types"
	"github.com/chainpoint/chainpoint-core/go-abci-service/util"
)

//SaveJWK : save the JWT value retrieved
func (app *AnchorApplication) SaveJWK(jwk types.Jwk) error {
	key := fmt.Sprintf("CorePublicKey:%s", jwk.Kid)
	jsonJwk, err := json.Marshal(jwk)
	if util.LoggerError(app.logger, err) != nil {
		return err
	}
	value, err := app.redisClient.Get(key).Result()
	if err == redis.Nil || value != string(jsonJwk) {
		err = app.redisClient.Set(key, value, 0).Err()
		if util.LoggerError(app.logger, err) != nil {
			return err
		}
		app.logger.Info(fmt.Sprintf("Set JWK cache for kid %s", jwk.Kid))
	}
	return nil
}

//MintRewardNodes : mint rewards for nodes
func (app *AnchorApplication) MintRewardNodes(sig []string) error {
	if leader, _ := app.ElectLeader(1); leader {
		sigBytes := make([][]byte, len(sig))
		for i, sigStr := range sig {
			decodedSig, err := hex.DecodeString(sigStr)
			if util.LoggerError(app.logger, err) != nil {
				app.logger.Info("mint hex decoding failed")
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
		res, err := app.rpc.BroadcastTx("SIGN", hex.EncodeToString(signature), 2, time.Now().Unix(), app.ID)
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
		var nodes []types.NodeJSON
		if err := json.Unmarshal([]byte(decoded.Data), &nodes); err != nil {
			return []common.Address{}, []byte{}, err
		}
		for _, nodeJSON := range nodes {
			nodeArray = append(nodeArray, common.HexToAddress(nodeJSON.EthAddr))
		}
	}
	addresses := uniquify(nodeArray)
	rewardHash := ethcontracts.AddressesToHash(addresses)
	return addresses, rewardHash, nil
}

//AuditNodes : Audit nodes for reputation chain and hash submission validity. Submits reward tx with info if successful
func (app *AnchorApplication) AuditNodes() error {
	if leader, _ := app.ElectLeader(1); leader {
		rewardCandidates := make([]types.NodeJSON, 0)
		deadline := time.Now().Add(1 * time.Minute)
		var wg sync.WaitGroup
		var mux sync.Mutex
		for len(rewardCandidates) < 3 && !time.Now().After(deadline) {
			nodes, err := app.pgClient.GetRandomNodes()
			if util.LoggerError(app.logger, err) != nil {
				return err
			}
			for _, nodeCandidate := range nodes {
				wg.Add(1)
				go func(node types.Node) {
					defer wg.Done()
					app.logger.Info(fmt.Sprintf("Auditing Node IP %s", node.PublicIP.String))
					if err := app.AuditNode(node); err != nil {
						app.logger.Debug(fmt.Sprintf("Audit of node IP %s unsuccessful: %s", node.PublicIP.String, err.Error()))
						return
					}
					nodeJSON := types.NodeJSON{
						EthAddr:  node.EthAddr,
						PublicIP: node.PublicIP.String,
					}
					mux.Lock()
					rewardCandidates = append(rewardCandidates, nodeJSON)
					mux.Unlock()
				}(nodeCandidate)
			}
			wg.Wait()
		}
		if len(rewardCandidates) > 3 {
			rewardCandidates = rewardCandidates[0:3]
		}
		if len(rewardCandidates) == 0 {
			err := errors.New("Unspecified reward candidate collation failure")
			util.LoggerError(app.logger, err)
			return err
		}
		rcJSON, err := json.Marshal(rewardCandidates)
		if err != nil {
			return err
		}
		res, err := app.rpc.BroadcastTx("NODE-RC", string(rcJSON), 2, time.Now().Unix(), app.ID)
		if err != nil {
			return err
		}
		if res.Code == 0 {
			return nil
		}
		return errors.New("problem validating and submitting nodes for rewards")
	}
	app.logger.Info("Not leader")
	return nil
}

//AuditNode : Used by the audit leader and all confirming cores to validate node performance
func (app *AnchorApplication) AuditNode(node types.Node) error {
	err := ValidateNodeRecentReputation(node)
	if util.LoggerError(app.logger, err) != nil {
		app.logger.Info("unable to validate node reputation")
		return err
	}
	nodeResp, err := SendNodeHash(node)
	if util.LoggerError(app.logger, err) != nil && len(nodeResp.Hashes) == 0 {
		app.logger.Info("node hash post failed")
		return err
	}
	time.Sleep(180 * time.Second)
	err = RetrieveNodeCalProof(node, nodeResp)
	if err != nil {
		app.logger.Info("retrieving cal proof from node audit failed")
		return err
	}
	return nil
}

//LoadNodesFromContract : load all past node staking events and update events
func (app *AnchorApplication) LoadNodesFromContract() error {
	//Consume all past node events from this contract and import them into the local postgres instance
	nodesStaked, err := app.ethClient.GetPastNodesStakedEvents()
	if util.LoggerError(app.logger, err) != nil {
		app.logger.Info("error in finding past staked nodes")
		return err
	}
	app.logger.Info(fmt.Sprintf("nodesStaked: %#v", nodesStaked))
	for _, node := range nodesStaked {
		newNode := types.Node{
			EthAddr:     node.Sender.Hex(),
			PublicIP:    sql.NullString{String: util.Int2Ip(node.NodeIp).String(), Valid: true},
			BlockNumber: sql.NullInt64{Int64: int64(node.Raw.BlockNumber), Valid: true},
		}
		inserted, err := app.pgClient.NodeUpsert(newNode)
		if util.LoggerError(app.logger, err) != nil {
			return err
		}
		app.logger.Info(fmt.Sprintf("Inserted for %#v: %t", newNode, inserted))
	}

	//Consume all updated events and reconcile them with the previous states
	nodesStakedUpdated, err := app.ethClient.GetPastNodesStakeUpdatedEvents()
	if util.LoggerError(app.logger, err) != nil {
		return err
	}
	for _, node := range nodesStakedUpdated {
		newNode := types.Node{
			EthAddr:     node.Sender.Hex(),
			PublicIP:    sql.NullString{String: util.Int2Ip(node.NodeIp).String(), Valid: true},
			BlockNumber: sql.NullInt64{Int64: int64(node.Raw.BlockNumber), Valid: true},
		}
		inserted, err := app.pgClient.NodeUpsert(newNode)
		if util.LoggerError(app.logger, err) != nil {
			return err
		}
		fmt.Printf("Inserted Update for %#v: %t", newNode, inserted)
	}

	//Consume unstake events and delete nodes where the blockNumber of this event is higher than the last stake or update
	nodesUnstaked, err := app.ethClient.GetPastNodesUnstakeEvents()
	if util.LoggerError(app.logger, err) != nil {
		return err
	}
	for _, node := range nodesUnstaked {
		newNode := types.Node{
			EthAddr:     node.Sender.Hex(),
			PublicIP:    sql.NullString{String: util.Int2Ip(node.NodeIp).String(), Valid: true},
			BlockNumber: sql.NullInt64{Int64: int64(node.Raw.BlockNumber), Valid: true},
		}
		deleted, err := app.pgClient.NodeDelete(newNode)
		if util.LoggerError(app.logger, err) != nil {
			return err
		}
		fmt.Printf("Deleted node for %#v: %t", newNode, deleted)
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
	go app.ethClient.WatchNodeUnstakeEvents(app.pgClient.HandleNodeUnstake, *highestBlock)
	return nil
}

//ValidateNodeRecentReputation : download and verify reputation chain items from a node
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
		hashJSON, err := json.Marshal(nodeHash)
		if err != nil {
			return types.NodeHashResponse{}, err
		}
		req, err := http.NewRequest("POST", HashURI, bytes.NewReader(hashJSON))
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
		fmt.Println("Cannot decode calblockhash into bytes")
		return "", err
	}
	buf.Write(hashBytes)

	prevRepItemHashBytes, err := hex.DecodeString(chainItem.PrevRepItemHash)
	if err != nil {
		fmt.Println("Cannot decode prevRepItemHash into bytes")
		return "", err
	}
	buf.Write(prevRepItemHashBytes)

	hashIDNodeNoHyphens := strings.Replace(chainItem.HashIDNode, "-", "", -1)
	hashIDNodeNoHyphensBytes, err := hex.DecodeString(hashIDNodeNoHyphens)
	if err != nil {
		fmt.Println("Cannot decode hashIDNodeNoHyphens into bytes")
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
		fmt.Println("can't decode RepItemHash hex string")
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
		util.LogError(err)
		fmt.Printf("Can't decode signature from node %s\n", from)
		return false, err
	}
	if sig[64] != 27 && sig[64] != 28 {
		return false, nil
	}
	sig[64] -= 27
	pubKey, err := crypto.SigToPub(signHash(msg), sig)
	if err != nil {
		util.LogError(err)
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
