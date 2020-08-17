package abci

import (
	"fmt"
	"github.com/chainpoint/chainpoint-core/go-abci-service/validation"
	"sort"

	"github.com/tendermint/tendermint/types"

	"github.com/chainpoint/chainpoint-core/go-abci-service/util"
	"github.com/tendermint/tendermint/p2p"
	core_types "github.com/tendermint/tendermint/rpc/core/types"
)

// ElectPeerAsLeader deterministically elects a network leader by creating an array of peers and using a blockhash-seeded random int as an index
func (app *AnchorApplication) ElectPeerAsLeader(numLeaders int, blacklistedIDs []string) (isLeader bool, leaderID []string) {
	status, err := app.rpc.GetStatus()
	if app.LogError(err) != nil {
		return false, []string{}
	}
	netInfo, err := app.rpc.GetNetInfo()
	if app.LogError(err) != nil {
		return false, []string{}
	}
	blockHash := status.SyncInfo.LatestBlockHash.String()
	app.logger.Info(fmt.Sprintf("Blockhash Seed: %s", blockHash))
	return determineLeader(numLeaders, blacklistedIDs, status, netInfo, blockHash)
}

// ElectValidatorAsLeader : elect a slice of validators as a leader and return whether we're the leader
func (app *AnchorApplication) ElectValidatorAsLeader(numLeaders int, blacklistedIDs []string) (isLeader bool, leaderID []string) {
	status, err := app.rpc.GetStatus()
	if app.LogError(err) != nil {
		return false, []string{}
	}
	blockHash := status.SyncInfo.LatestBlockHash.String()
	app.logger.Info(fmt.Sprintf("Blockhash Seed: %s", blockHash))
	return determineValidatorLeader(numLeaders, blacklistedIDs, status, app.Validators, blockHash, app.config.FilePV.GetAddress().String())
}

// ElectChainContributedAsLeaderNaive : elects a node that's contributed to the chain without checking if its been active recently
func (app *AnchorApplication) ElectChainContributorAsLeaderNaive(numLeaders int, blacklistedIDs []string) (isLeader bool, leaderID []string) {
	status, err := app.rpc.GetStatus()
	if app.LogError(err) != nil {
		return false, []string{}
	}
	keys := make([]string, 0, len(app.state.CoreKeys))
	for k := range app.state.CoreKeys {
		filtered := false
		for _, id := range blacklistedIDs {
			if k == id {
				filtered = true
			}
		}
		if !filtered {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	coreListLength := len(keys)
	index := util.GetSeededRandInt([]byte(status.SyncInfo.LatestBlockHash.String()), coreListLength)
	app.logger.Info(fmt.Sprintf("Leader is %d th out of %v", index, keys))
	if err := util.RotateLeft(keys[:], index); err != nil { //get a wrapped-around slice of numLeader leaders
		util.LogError(err)
		return false, []string{}
	}
	if numLeaders <= coreListLength {
		keys = keys[0:numLeaders]
	} else {
		keys = keys[0:1]
	}
	iAmLeader := false
	for _, leader := range keys {
		if leader == app.ID && !status.SyncInfo.CatchingUp {
			iAmLeader = true
		}
	}
	return iAmLeader, keys
}

// ElectChainContributedAsLeader : elects a node that's contributed to the chain while checking if it's submitted a NIST value recently
func (app *AnchorApplication) ElectChainContributorAsLeader(numLeaders int, blacklistedIDs []string) (isLeader bool, leaderID []string) {
	status, err := app.rpc.GetStatus()
	if app.LogError(err) != nil {
		return false, []string{}
	}
	keys := make([]string, 0, len(app.state.CoreKeys))
	cores := validation.GetLastDrandSubmitters(128, app.state)
	for k := range cores {
		filtered := false
		for _, id := range blacklistedIDs {
			if k == id {
				filtered = true
			}
		}
		if !filtered {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	coreListLength := len(keys)
	if coreListLength == 0 {
		app.logger.Error("coreListLength is 0")
		return false, []string{}
	}
	index := util.GetSeededRandInt([]byte(status.SyncInfo.LatestBlockHash.String()), coreListLength)
	app.logger.Info(fmt.Sprintf("Leader is %d th out of %v", index, keys))
	if err := util.RotateLeft(keys[:], index); err != nil { //get a wrapped-around slice of numLeader leaders
		util.LogError(err)
		return false, []string{}
	}
	if numLeaders <= coreListLength {
		keys = keys[0:numLeaders]
	} else {
		keys = keys[0:1]
	}
	iAmLeader := false
	for _, leader := range keys {
		if leader == app.ID && !status.SyncInfo.CatchingUp {
			iAmLeader = true
		}
	}
	return iAmLeader, keys
}

func determineValidatorLeader(numLeaders int, blacklistedIDs []string, status core_types.ResultStatus, validators []*types.Validator, seed string, address string) (isLeader bool, leaderIDs []string) {
	leaders := make([]types.Validator, 0)
	validatorList := GetSortedValidatorList(validators)
	filteredArray := make([]types.Validator, 0)
	if len(validatorList) == 0 {
		return false, []string{}
	}
	for _, val := range validatorList {
		filtered := false
		for _, id := range blacklistedIDs {
			if val.Address.String() == id {
				filtered = true
			}
		}
		if !filtered {
			filteredArray = append(filteredArray, val)
		}
	}
	validatorLength := len(filteredArray)
	index := util.GetSeededRandInt([]byte(seed), validatorLength)    //seed the first time
	if err := util.RotateLeft(filteredArray[:], index); err != nil { //get a wrapped-around slice of numLeader leaders
		util.LogError(err)
		return false, []string{}
	}
	if numLeaders <= validatorLength {
		leaders = filteredArray[0:numLeaders]
	} else {
		leaders = filteredArray[0:1]
	}
	leaderStrings := make([]string, 0)
	iAmLeader := false
	for _, leader := range leaders {
		leaderID := leader.Address.String()
		leaderStrings = append(leaderStrings, leaderID)
		if leaderID == address && !status.SyncInfo.CatchingUp {
			iAmLeader = true
		}
	}
	return iAmLeader, leaderStrings
}

// GetSortedValidatorList : collate and deterministically sort validator list
func GetSortedValidatorList(validators []*types.Validator) []types.Validator {
	validatorList := make([]types.Validator, 0)
	for _, val := range validators {
		validatorList = append(validatorList, *val)
	}
	sort.Slice(validatorList[:], func(i, j int) bool {
		return validatorList[i].Address.String() > validatorList[j].Address.String()
	})
	return validatorList
}

// GetPeers : get list of all peers
func (app *AnchorApplication) GetPeers() []core_types.Peer {
	var status core_types.ResultStatus
	var netInfo core_types.ResultNetInfo
	var err error
	var err2 error

	status, err = app.rpc.GetStatus()
	netInfo, err2 = app.rpc.GetNetInfo()

	if app.LogError(err) != nil || util.LogError(err2) != nil {
		return []core_types.Peer{}
	}

	peers := GetSortedPeerList(status, netInfo)
	return peers
}

//GetSortedPeerList : returns sorted list of peers including self
func GetSortedPeerList(status core_types.ResultStatus, netInfo core_types.ResultNetInfo) []core_types.Peer {
	peers := netInfo.Peers
	nodeArray := make([]core_types.Peer, 0)
	for i := 0; i < len(peers); i++ {
		peers[i].RemoteIP = util.DetermineIP(peers[i])
		nodeArray = append(nodeArray, peers[i])
	}
	selfPeer := core_types.Peer{
		NodeInfo:         status.NodeInfo,
		IsOutbound:       false,
		ConnectionStatus: p2p.ConnectionStatus{},
		RemoteIP:         "127.0.0.1",
	}
	nodeArray = append(nodeArray, selfPeer)
	sort.Slice(nodeArray[:], func(i, j int) bool {
		return nodeArray[i].NodeInfo.ID() > nodeArray[j].NodeInfo.ID()
	})
	return nodeArray
}

// determineLeader accepts current node status and a peer array, then finds a leader based on the latest blockhash
func determineLeader(numLeaders int, blacklistedIDs []string, status core_types.ResultStatus, netInfo core_types.ResultNetInfo, seed string) (isLeader bool, leaderIDs []string) {
	currentNodeID := status.NodeInfo.ID()
	if len(netInfo.Peers) > 0 {
		nodeArray := GetSortedPeerList(status, netInfo)
		filteredArray := make([]core_types.Peer, 0)
		for _, peer := range nodeArray {
			filtered := false
			for _, id := range blacklistedIDs {
				if string(peer.NodeInfo.ID()) == id {
					filtered = true
				}
			}
			if !filtered {
				filteredArray = append(filteredArray, peer)
			}
		}
		index := util.GetSeededRandInt([]byte(seed), len(filteredArray)) //seed the first time
		if err := util.RotateLeft(filteredArray[:], index); err != nil { //get a wrapped-around slice of numLeader leaders
			util.LogError(err)
			return false, []string{}
		}
		leaders := make([]core_types.Peer, 0)
		if numLeaders <= len(filteredArray) {
			leaders = filteredArray[0:numLeaders]
		} else {
			leaders = filteredArray[0:1]
		}
		leaderStrings := make([]string, 0)
		iAmLeader := false
		for _, leader := range leaders {
			leaderStrings = append(leaderStrings, string(leader.NodeInfo.ID()))
			if leader.NodeInfo.ID() == currentNodeID && !status.SyncInfo.CatchingUp {
				iAmLeader = true
			}
		}
		return iAmLeader, leaderStrings
	}
	return true, []string{string(currentNodeID)}
}

// AmValidator : determines if this node is a validator, without needing to load an ID from elsewhere
func (app *AnchorApplication) AmValidator() (amValidator bool, err error) {
	status, err := app.rpc.GetStatus()
	if app.LogError(err) != nil {
		return false, err
	}
	for _, validator := range app.Validators {
		if validator.Address.String() == status.ValidatorInfo.Address.String() {
			return true, nil
		}
	}
	return false, nil
}

//IsValidator : determines if a node is a validator by checking an external ID
func (app *AnchorApplication) IsValidator(ID string) (amValidator bool, err error) {
	for _, validator := range app.Validators {
		if validator.Address.String() == ID {
			return true, nil
		}
	}
	return false, nil
}
