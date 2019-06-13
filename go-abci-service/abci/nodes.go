package abci

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"net"
	"net/http"
	"sort"
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

//LoadJWK : load public keys derived from JWTs from redis
func (app *AnchorApplication) LoadJWK() error {
	var cursor uint64
	var idKeys []string
	for {
		var keys []string
		var err error
		keys, cursor, err = app.redisClient.Scan(cursor, "CoreID:*", 10).Result()
		if err != nil {
			return err
		}
		idKeys = append(idKeys, keys...)
		if cursor == 0 {
			break
		}
	}
	if len(idKeys) == 0 {
		return util.LoggerError(app.logger, errors.New("no JWT keys found in redis"))
	}
	for _, k := range idKeys {
		var coreID string
		idStr := strings.Split(k, ":")
		if len(idStr) == 2 {
			coreID = idStr[1]
		} else {
			continue
		}
		b64Str, err := app.redisClient.Get(k).Result()
		if util.LoggerError(app.logger, err) != nil {
			continue
		}
		pubKeyBytes, err := base64.StdEncoding.DecodeString(b64Str)
		if util.LoggerError(app.logger, err) != nil {
			continue
		}
		x, y := elliptic.Unmarshal(elliptic.P256(), pubKeyBytes)
		pubKey := ecdsa.PublicKey{
			Curve: elliptic.P256(),
			X:     x,
			Y:     y,
		}
		app.logger.Info(fmt.Sprintf("Setting JWK for Core %s: %s", coreID, b64Str))
		app.CoreKeys[coreID] = pubKey
	}
	return nil
}

//SaveJWK : save the JWT value retrieved
func (app *AnchorApplication) SaveJWK(tx types.Tx) error {
	var jwkType types.Jwk
	json.Unmarshal([]byte(tx.Data), &jwkType)
	key := fmt.Sprintf("CorePublicKey:%s", jwkType.Kid)
	if jwkType.Kid == app.JWK.Kid {
		app.logger.Info("JWK keysync tx committed")
		app.JWKSent = true
	}
	jsonJwk, err := json.Marshal(jwkType)
	if util.LoggerError(app.logger, err) != nil {
		return err
	}
	pubKey, err := util.DecodePubKey(tx)
	if util.LoggerError(app.logger, err) == nil {
		app.CoreKeys[tx.CoreID] = *pubKey
		pubKeyBytes := elliptic.Marshal(pubKey.Curve, pubKey.X, pubKey.Y)
		util.LoggerError(app.logger, app.redisClient.Set("CoreID:"+tx.CoreID, base64.StdEncoding.EncodeToString(pubKeyBytes), 0).Err())
	}
	value, err := app.redisClient.Get(key).Result()
	if err == redis.Nil || value != string(jsonJwk) {
		err = app.redisClient.Set(key, value, 0).Err()
		if util.LoggerError(app.logger, err) != nil {
			return err
		}
		app.logger.Info(fmt.Sprintf("Set JWK cache for kid %s", jwkType.Kid))
	}
	return nil
}

//MintNodeReward : mint rewards for nodes
func (app *AnchorApplication) MintNodeReward(sig []string, rewardCandidates []common.Address, rewardHash []byte) error {
	leader, ids := app.ElectValidator(1)
	if len(ids) == 1 {
		app.state.LastMintCoreID = ids[0]
	}
	if leader {
		app.logger.Info("Mint: Elected Leader for Minting")
		if len(sig) > 6 {
			sig = sig[0:6]
		}
		app.logger.Info(fmt.Sprintf("Mint Signatures: %v\nReward Candidates: %v\nReward Hash: %x\n", sig, rewardCandidates, rewardHash))
		sigBytes := make([][]byte, len(sig))
		for i, sigStr := range sig {
			decodedSig, err := hex.DecodeString(sigStr)
			if util.LoggerError(app.logger, err) != nil {
				app.logger.Info("Mint Error: mint hex decoding failed")
				continue
			}
			sigBytes[i] = decodedSig
		}
		err := app.ethClient.MintNodes(rewardCandidates, rewardHash, sigBytes)
		if util.LoggerError(app.logger, err) != nil {
			app.logger.Info("Mint Error: invoking smart contract failed")
			return err
		}
		app.logger.Info("Mint process complete")
	}
	return nil
}

//StartNodeMintProcess : wraps signing/minting process and handles state updates
func (app *AnchorApplication) StartNodeMintProcess() error {
	app.SetNodeMintPendingState(true) //needed since we can't do a blocking lock in commit
	err := app.SignNodeRewards()
	app.SetNodeMintPendingState(false)
	if util.LoggerError(app.logger, err) != nil {
		return err
	}
	return nil
}

//SetNodeMintPendingState : create a deferable method to set mint state
func (app *AnchorApplication) SetNodeMintPendingState(val bool) {
	app.state.NodeMintPending = val
	app.NodeRewardSignatures = make([]string, 0)
}

//CollectRewardNodes : collate and sign reward node list
func (app *AnchorApplication) SignNodeRewards() error {
	var candidates []common.Address
	var rewardHash []byte

	//Lock the minting process
	if leader, leaders := app.ElectValidator(7); leader {
		app.logger.Info(fmt.Sprintf("Elected Leaders for Mint Signing: %v", leaders))
		currentEthBlock, err := app.ethClient.HighestBlock()
		if util.LoggerError(app.logger, err) != nil {
			app.logger.Error("Mint Error: problem retrieving highest block")
			return err
		}
		if currentEthBlock.Int64()-app.state.LastNodeMintedAtBlock < MINT_EPOCH {
			app.logger.Info("Mint: Too soon for minting")
			return errors.New("Too soon for minting")
		}
		candidates, rewardHash, err = app.GetNodeRewardCandidates()
		if util.LoggerError(app.logger, err) != nil {
			app.logger.Info("Mint Error: Error retrieving node reward candidates")
			return err
		}
		app.logger.Info(fmt.Sprintf("Mint: raw SHA3 hash: %x", rewardHash))
		rewardHash = signHash(rewardHash)
		app.logger.Info(fmt.Sprintf("Mint: with prefix: %x", rewardHash))
		signature, err := ethcontracts.SignMsg(rewardHash, app.ethClient.EthPrivateKey)
		signature[64] += 27
		if util.LoggerError(app.logger, err) != nil {
			app.logger.Info("Mint Error: Problem with signing message for minting")
			return err
		}
		_, err = app.rpc.BroadcastTx("NODE-SIGN", hex.EncodeToString(signature), 2, time.Now().Unix(), app.ID, &app.config.ECPrivateKey)
		if err != nil {
			app.logger.Info("Mint Error: Error issuing SIGN tx")
			return err
		}
	}
	// wait for 6 SIGN tx
	deadline := time.Now().Add(4 * time.Minute)
	for len(app.NodeRewardSignatures) < 6 && !time.Now().After(deadline) {
		time.Sleep(10 * time.Second)
	}
	// Mint if 6+ SIGN txs are received
	if len(app.NodeRewardSignatures) >= 6 {
		app.logger.Info("Mint: Enough SIGN TXs received, calling mint")
		err := app.MintNodeReward(app.NodeRewardSignatures, candidates, rewardHash)
		if util.LoggerError(app.logger, err) != nil {
			return err
		}
	} else {
		app.logger.Info("Mint: Not enough SIGN TXs")
		return errors.New("Mint: Not enough SIGN TXs")
	}
	return nil
}

//GetNodeRewardCandidates : scans for and collates the reward candidates in the current epoch
func (app *AnchorApplication) GetNodeRewardCandidates() ([]common.Address, []byte, error) {
	txResult, err := app.rpc.client.TxSearch(fmt.Sprintf("NODERC=%d", app.state.LastNodeMintedAtBlock), false, 1, 25)
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
			app.logger.Info(fmt.Sprintf("Mint: Decoded NODE-RC: %#v", nodeJSON))
			nodeArray = append(nodeArray, common.HexToAddress(nodeJSON.EthAddr))
		}
	}
	if len(nodeArray) == 0 {
		return []common.Address{}, []byte{}, errors.New("No NODE-RC tx from the last epoch have been found")
	}
	addresses := util.UniquifyAddresses(nodeArray)
	sort.Slice(addresses[:], func(i, j int) bool {
		return addresses[i].Hex() > addresses[j].Hex()
	})
	app.logger.Info(fmt.Sprintf("Mint: input node addresses: %#v", addresses))
	rewardHash := ethcontracts.AddressesToHash(addresses)
	return addresses, rewardHash, nil
}

//AuditNodes : Audit nodes for reputation chain and hash submission validity. Submits reward tx with info if successful
func (app *AnchorApplication) AuditNodes() error {
	leader, ids := app.ElectValidator(1)
	if len(ids) == 1 {
		app.state.LastAuditCoreID = ids[0]
	}
	if leader {
		rewardCandidates := make([]types.NodeJSON, 0)
		deadline := time.Now().Add(9 * time.Minute)
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
					app.logger.Info(fmt.Sprintf("node audit IP %s", node.PublicIP.String))
					if err := app.AuditNode(node); err != nil {
						app.logger.Debug(fmt.Sprintf("node audit of node IP %s unsuccessful: %s", node.PublicIP.String, err.Error()))
						return
					}
					app.logger.Info(fmt.Sprintf("Node IP %s passed node audit", node.PublicIP.String))
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
		if time.Now().After(deadline) {
			app.logger.Info("node audit collation deadline passed")
		}
		if len(rewardCandidates) > 3 {
			rewardCandidates = rewardCandidates[0:3]
		}
		if len(rewardCandidates) == 0 {
			err := errors.New("Unspecified reward candidate collation failure for node audit")
			util.LoggerError(app.logger, err)
			return err
		}
		if app.state.NodeMintPending {
			app.logger.Info("Minting in progress, not auditing")
			return nil
		}
		rcJSON, err := json.Marshal(rewardCandidates)
		if util.LoggerError(app.logger, err) != nil {
			return err
		}
		res, err := app.rpc.BroadcastTx("NODE-RC", string(rcJSON), 2, time.Now().Unix(), app.ID, &app.config.ECPrivateKey)
		if util.LoggerError(app.logger, err) != nil {
			return err
		}
		if res.Code == 0 {
			app.logger.Info("node audit success, NODE-RC tx issued")
			return nil
		}
		err = util.LoggerError(app.logger, errors.New("problem validating and submitting nodes for rewards after node audit process"))
		return err
	}
	app.logger.Info("Not leader for node audits")
	return nil
}

//AuditNode : Used by the audit leader and all confirming cores to validate node performance
func (app *AnchorApplication) AuditNode(node types.Node) error {
	err := ValidateNodeRecentReputation(node)
	if util.LoggerError(app.logger, err) != nil {
		app.logger.Info("unable to validate node audit reputation")
		return err
	}
	nodeResp, err := SendNodeHash(node)
	if util.LoggerError(app.logger, err) != nil && len(nodeResp.Hashes) == 0 {
		app.logger.Info("node audit hash post failed")
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

//PollNodesFromContract : load all past node staking events and update events
func (app *AnchorApplication) PollNodesFromContract() {
	highestBlock := big.NewInt(0)
	first := true
	for {
		app.logger.Info(fmt.Sprintf("Polling for Registry events after block %d", highestBlock.Int64()))
		if first {
			first = false
		} else {
			time.Sleep(30 * time.Second)
		}

		//Consume all past node events from this contract and import them into the local postgres instance
		nodesStaked, err := app.ethClient.GetPastNodesStakedEvents(*highestBlock)
		if util.LoggerError(app.logger, err) != nil {
			app.logger.Info("error in finding past staked nodes")
			continue
		}
		for _, node := range nodesStaked {
			newNode := types.Node{
				EthAddr:     node.Sender.Hex(),
				PublicIP:    sql.NullString{String: util.Int2Ip(node.NodeIp).String(), Valid: true},
				BlockNumber: sql.NullInt64{Int64: int64(node.Raw.BlockNumber), Valid: true},
			}
			inserted, err := app.pgClient.NodeUpsert(newNode)
			if util.LoggerError(app.logger, err) != nil {
				continue
			}
			app.logger.Info(fmt.Sprintf("Inserted for %#v: %t", newNode, inserted))
		}

		//Consume all updated events and reconcile them with the previous states
		nodesStakedUpdated, err := app.ethClient.GetPastNodesStakeUpdatedEvents(*highestBlock)
		if util.LoggerError(app.logger, err) != nil {
			continue
		}
		for _, node := range nodesStakedUpdated {
			newNode := types.Node{
				EthAddr:     node.Sender.Hex(),
				PublicIP:    sql.NullString{String: util.Int2Ip(node.NodeIp).String(), Valid: true},
				BlockNumber: sql.NullInt64{Int64: int64(node.Raw.BlockNumber), Valid: true},
			}
			inserted, err := app.pgClient.NodeUpsert(newNode)
			if util.LoggerError(app.logger, err) != nil {
				continue
			}
			app.logger.Info(fmt.Sprintf("Updated for %#v: %t", newNode, inserted))
		}

		//Consume unstake events and delete nodes where the blockNumber of this event is higher than the last stake or update
		nodesUnstaked, err := app.ethClient.GetPastNodesUnstakeEvents(*highestBlock)
		if util.LoggerError(app.logger, err) != nil {
			continue
		}
		for _, node := range nodesUnstaked {
			newNode := types.Node{
				EthAddr:     node.Sender.Hex(),
				PublicIP:    sql.NullString{String: util.Int2Ip(node.NodeIp).String(), Valid: true},
				BlockNumber: sql.NullInt64{Int64: int64(node.Raw.BlockNumber), Valid: true},
			}
			deleted, err := app.pgClient.NodeDelete(newNode)
			if util.LoggerError(app.logger, err) != nil {
				continue
			}
			app.logger.Info(fmt.Sprintf("Deleted for %#v: %t", newNode, deleted))
		}

		highestBlock, err = app.ethClient.HighestBlock()
		if util.LoggerError(app.logger, err) != nil {
			continue
		}
	}
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
		//fmt.Printf("node proof Response: %#v\n", nodeResponse)
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
	return types.NodeHashResponse{}, errors.New(fmt.Sprintf("cannot parse node IP: %s", node.PublicIP.String))
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
		//fmt.Printf("node proof: %#v\n", nodeProof)
		if len(nodeProof) > 0 && len(nodeProof[0].AnchorsComplete) > 0 {
			for _, anchor := range nodeProof[0].AnchorsComplete {
				if anchor == "cal" || anchor == "tcal" {
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
