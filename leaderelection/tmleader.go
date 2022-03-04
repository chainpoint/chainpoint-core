package leaderelection

import (
	"errors"
	types2 "github.com/chainpoint/chainpoint-core/types"
	"github.com/chainpoint/chainpoint-core/validation"
	"sort"

	"github.com/tendermint/tendermint/types"
	seededelection "github.com/chainpoint/leader-election"
	"github.com/chainpoint/chainpoint-core/util"
	"github.com/tendermint/tendermint/p2p"
	core_types "github.com/tendermint/tendermint/rpc/core/types"
)

// ElectPeerAsLeader deterministically elects a network leader by creating an array of peers and using a blockhash-seeded random int as an index
func ElectPeerAsLeader(numLeaders int, blacklistedIDs []string, state types2.AnchorState) (isLeader bool, leaderID []string) {
	if state.ID == "" {
		return false, []string{}
	}
	blockHash := state.TMState.SyncInfo.LatestBlockHash.String()
	return determineLeader(numLeaders, blacklistedIDs, state.TMState, state.TMNetInfo, blockHash)
}

// ElectValidatorAsLeader : elect a slice of validators as a leader and return whether we're the leader
func ElectValidatorAsLeader(numLeaders int, blacklistedIDs []string, state types2.AnchorState, config types2.AnchorConfig) (isLeader bool, leaderID []string) {
	if state.ID == "" {
		return false, []string{}
	}
	blockHash := state.TMState.SyncInfo.LatestBlockHash.String()
	return determineValidatorLeader(numLeaders, blacklistedIDs, state.TMState, state.Validators, blockHash, config.FilePV.GetAddress().String())
}

// ElectChainContributedAsLeaderNaive : elects a node that's contributed to the chain without checking if its been active recently
func ElectChainContributorAsLeaderNaive(numLeaders int, blacklistedIDs []string, state types2.AnchorState) (isLeader bool, leaderID []string) {
	if state.ID == "" {
		return false, []string{}
	}
	status := state.TMState
	keys := make([]string, 0, len(state.CoreKeys))
	for k := range state.CoreKeys {
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
	if len(keys) == 0 {
		return false, []string{}
	}
	keys = seededelection.ElectLeaders(keys, numLeaders, status.SyncInfo.LatestBlockHash.String()).([]string)
	if keys == nil {
		return false, []string{}
	}
	iAmLeader := false
	for _, leader := range keys {
		if leader == state.ID && !status.SyncInfo.CatchingUp {
			iAmLeader = true
		}
	}
	return iAmLeader, keys
}

// ElectChainContributedAsLeader : elects a node that's contributed to the chain while checking if it's submitted a NIST value recently
func ElectChainContributorAsLeader(numLeaders int, blacklistedIDs []string, state types2.AnchorState) (isLeader bool, leaderID []string) {
	if state.ID == "" {
		return false, []string{}
	}
	status := state.TMState
	keys := make([]string, 0, len(state.CoreKeys))
	cores := validation.GetLastNSubmitters(128, state)
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
	if len(keys) == 0 {
		return false, []string{}
	}
	keys = seededelection.ElectLeaders(keys, numLeaders, status.SyncInfo.LatestBlockHash.String()).([]string)
	if keys == nil {
		return false, []string{}
	}
	iAmLeader := false
	for _, leader := range keys {
		if leader == state.ID && !status.SyncInfo.CatchingUp {
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
	if len(filteredArray) == 0 {
		return false, []string{}
	}
	leaders = seededelection.ElectLeaders(filteredArray, numLeaders, seed).([]types.Validator)
	if leaders == nil {
		return false, []string{}
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
func GetPeers(state types2.AnchorState, tmState core_types.ResultStatus, tmNetInfo core_types.ResultNetInfo) []core_types.Peer {
	if state.ID == "" {
		return []core_types.Peer{}
	}

	peers := GetSortedPeerList(tmState, tmNetInfo)
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
		leaders := seededelection.ElectLeaders(filteredArray, numLeaders, status.SyncInfo.LatestBlockHash.String()).([]core_types.Peer)
		if leaders == nil {
			return false, []string{}
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
func AmValidator(state types2.AnchorState) (amValidator bool, err error) {
	if state.ID == "" {
		return false, errors.New("status unintialized")
	}
	status := state.TMState
	for _, validator := range state.Validators {
		if validator.Address.String() == status.ValidatorInfo.Address.String() {
			return true, nil
		}
	}
	return false, nil
}

//IsValidator : determines if a node is a validator by checking an external ID
func IsValidator(state types2.AnchorState, ID string) (amValidator bool, err error) {
	for _, validator := range state.Validators {
		if validator.Address.String() == ID {
			return true, nil
		}
	}
	return false, nil
}
