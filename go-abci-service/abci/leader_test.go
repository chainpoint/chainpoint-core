package abci

import (
	"testing"

	"github.com/tendermint/tendermint/p2p"

	core_types "github.com/tendermint/tendermint/rpc/core/types"
)

func TestValidatorElection(t *testing.T) {
	/*	seed := "3719ADA3EEE198F3A7A33616EA60ED6D72D94D31A2B2422FA12E2BCDDCABD4D4"
		status := core_types.ResultStatus{
			NodeInfo: p2p.DefaultNodeInfo{
				ID_: "b",
			},
			SyncInfo: core_types.SyncInfo{
				CatchingUp: false,
			},
			ValidatorInfo: core_types.ValidatorInfo{},
		}*/
}

func TestLeaderElectionLeader(t *testing.T) {
	seed := "3719ADA3EEE198F3A7A33616EA60ED6D72D94D31A2B2422FA12E2BCDDCABD4D4"
	status := core_types.ResultStatus{
		NodeInfo: p2p.DefaultNodeInfo{
			ID_: "b",
		},
		SyncInfo: core_types.SyncInfo{
			CatchingUp: false,
		},
		ValidatorInfo: core_types.ValidatorInfo{},
	}
	netInfo := core_types.ResultNetInfo{
		Peers: []core_types.Peer{
			{
				NodeInfo: p2p.DefaultNodeInfo{
					ID_: "a",
				},
				RemoteIP: "127.0.0.1",
			},
			{
				NodeInfo: p2p.DefaultNodeInfo{
					ID_: "b",
				},
				RemoteIP: "127.0.0.1",
			},
			{
				NodeInfo: p2p.DefaultNodeInfo{
					ID_: "c",
				},
				RemoteIP: "127.0.0.1",
			},
			{
				NodeInfo: p2p.DefaultNodeInfo{
					ID_: "d",
				},
				RemoteIP: "127.0.0.1",
			},
		},
	}
	amILeader, LeaderIDs := determineLeader(1, status, netInfo, seed)
	// We should be leader
	if !amILeader || LeaderIDs[0] != "b" {
		t.Errorf("Expected amILeader=true and LeaderID=b, got amILeader=%t and LeaderID=%s instead\n", amILeader, LeaderIDs[0])
	}
}

func TestLeaderElectionNotLeader(t *testing.T) {
	seed := "3719ADA3EEE198F3A7A33616EA60ED6D72D94D31A2B2422FA12E2BCDDCABD4D4"
	status := core_types.ResultStatus{
		NodeInfo: p2p.DefaultNodeInfo{
			ID_: "c",
		},
		SyncInfo: core_types.SyncInfo{
			CatchingUp: false,
		},
		ValidatorInfo: core_types.ValidatorInfo{},
	}
	netInfo := core_types.ResultNetInfo{
		Peers: []core_types.Peer{
			{
				NodeInfo: p2p.DefaultNodeInfo{
					ID_: "a",
				},
				RemoteIP: "127.0.0.1",
			},
			{
				NodeInfo: p2p.DefaultNodeInfo{
					ID_: "b",
				},
				RemoteIP: "127.0.0.1",
			},
			{
				NodeInfo: p2p.DefaultNodeInfo{
					ID_: "c",
				},
				RemoteIP: "127.0.0.1",
			},
			{
				NodeInfo: p2p.DefaultNodeInfo{
					ID_: "d",
				},
				RemoteIP: "127.0.0.1",
			},
		},
	}
	amILeader, LeaderIDs := determineLeader(1, status, netInfo, seed)
	// We should not be leader
	if amILeader || LeaderIDs[0] != "b" {
		t.Errorf("Expected amILeader=false and LeaderID=b, got amILeader=%t and LeaderID=%s instead\n", amILeader, LeaderIDs[0])
	}
}

func TestLeaderElectionSingleCore(t *testing.T) {
	seed := "3719ADA3EEE198F3A7A33616EA60ED6D72D94D31A2B2422FA12E2BCDDCABD4D4"
	status := core_types.ResultStatus{
		NodeInfo: p2p.DefaultNodeInfo{
			ID_: "c",
		},
		SyncInfo: core_types.SyncInfo{
			CatchingUp: false,
		},
		ValidatorInfo: core_types.ValidatorInfo{},
	}
	netInfo := core_types.ResultNetInfo{
		Peers: []core_types.Peer{},
	}
	amILeader, LeaderIDs := determineLeader(1, status, netInfo, seed)
	// We're the only node so we should be leader
	if !amILeader || LeaderIDs[0] != "c" {
		t.Errorf("Expected amILeader=false and LeaderID=c, got amILeader=%t and LeaderID=%s instead\n", amILeader, LeaderIDs[0])
	}
}

func TestLeaderElectionCatchingUp(t *testing.T) {
	seed := "3719ADA3EEE198F3A7A33616EA60ED6D72D94D31A2B2422FA12E2BCDDCABD4D4"
	status := core_types.ResultStatus{
		NodeInfo: p2p.DefaultNodeInfo{
			ID_: "b",
		},
		SyncInfo: core_types.SyncInfo{
			CatchingUp: true,
		},
		ValidatorInfo: core_types.ValidatorInfo{},
	}
	netInfo := core_types.ResultNetInfo{
		Peers: []core_types.Peer{
			{
				NodeInfo: p2p.DefaultNodeInfo{
					ID_: "a",
				},
				RemoteIP: "127.0.0.1",
			},
			{
				NodeInfo: p2p.DefaultNodeInfo{
					ID_: "b",
				},
				RemoteIP: "127.0.0.1",
			},
			{
				NodeInfo: p2p.DefaultNodeInfo{
					ID_: "c",
				},
				RemoteIP: "127.0.0.1",
			},
			{
				NodeInfo: p2p.DefaultNodeInfo{
					ID_: "d",
				},
				RemoteIP: "127.0.0.1",
			},
		},
	}
	amILeader, LeaderIDs := determineLeader(1, status, netInfo, seed)
	// We're catching up so we shouldn't be leader
	if amILeader || LeaderIDs[0] != "b" {
		t.Errorf("Expected amILeader=false and LeaderID=b, got amILeader=%t and LeaderID=%s instead\n", amILeader, LeaderIDs[0])
	}
}
