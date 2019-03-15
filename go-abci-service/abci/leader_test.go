package abci

import (
	"testing"

	"github.com/tendermint/tendermint/p2p"

	core_types "github.com/tendermint/tendermint/rpc/core/types"
)

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
			core_types.Peer{
				NodeInfo: p2p.DefaultNodeInfo{
					ID_: "a",
				},
				RemoteIP: "AAAAAAAAAAAAAP//I7zuug==",
			},
			core_types.Peer{
				NodeInfo: p2p.DefaultNodeInfo{
					ID_: "b",
				},
				RemoteIP: "AAAAAAAAAAAAAP//I7zuug==",
			},
			core_types.Peer{
				NodeInfo: p2p.DefaultNodeInfo{
					ID_: "c",
				},
				RemoteIP: "AAAAAAAAAAAAAP//I7zuug==",
			},
			core_types.Peer{
				NodeInfo: p2p.DefaultNodeInfo{
					ID_: "d",
				},
				RemoteIP: "AAAAAAAAAAAAAP//I7zuug==",
			},
		},
	}
	amILeader, LeaderID := determineLeader(status, netInfo.Peers, seed)
	// We should be leader
	if !amILeader || LeaderID != "b" {
		t.Errorf("Expected amILeader=true and LeaderID=b, got amILeader=%t and LeaderID=%s instead\n", amILeader, LeaderID)
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
			core_types.Peer{
				NodeInfo: p2p.DefaultNodeInfo{
					ID_: "a",
				},
				RemoteIP: "AAAAAAAAAAAAAP//I7zuug==",
			},
			core_types.Peer{
				NodeInfo: p2p.DefaultNodeInfo{
					ID_: "b",
				},
				RemoteIP: "AAAAAAAAAAAAAP//I7zuug==",
			},
			core_types.Peer{
				NodeInfo: p2p.DefaultNodeInfo{
					ID_: "c",
				},
				RemoteIP: "AAAAAAAAAAAAAP//I7zuug==",
			},
			core_types.Peer{
				NodeInfo: p2p.DefaultNodeInfo{
					ID_: "d",
				},
				RemoteIP: "AAAAAAAAAAAAAP//I7zuug==",
			},
		},
	}
	amILeader, LeaderID := determineLeader(status, netInfo.Peers, seed)
	// We should not be leader
	if amILeader || LeaderID != "b" {
		t.Errorf("Expected amILeader=false and LeaderID=b, got amILeader=%t and LeaderID=%s instead\n", amILeader, LeaderID)
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
	amILeader, LeaderID := determineLeader(status, netInfo.Peers, seed)
	// We're the only node so we should be leader
	if !amILeader || LeaderID != "c" {
		t.Errorf("Expected amILeader=false and LeaderID=c, got amILeader=%t and LeaderID=%s instead\n", amILeader, LeaderID)
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
			core_types.Peer{
				NodeInfo: p2p.DefaultNodeInfo{
					ID_: "a",
				},
				RemoteIP: "AAAAAAAAAAAAAP//I7zuug==",
			},
			core_types.Peer{
				NodeInfo: p2p.DefaultNodeInfo{
					ID_: "b",
				},
				RemoteIP: "AAAAAAAAAAAAAP//I7zuug==",
			},
			core_types.Peer{
				NodeInfo: p2p.DefaultNodeInfo{
					ID_: "c",
				},
				RemoteIP: "AAAAAAAAAAAAAP//I7zuug==",
			},
			core_types.Peer{
				NodeInfo: p2p.DefaultNodeInfo{
					ID_: "d",
				},
				RemoteIP: "AAAAAAAAAAAAAP//I7zuug==",
			},
		},
	}
	amILeader, LeaderID := determineLeader(status, netInfo.Peers, seed)
	// We're catching up so we shouldn't be leader
	if amILeader || LeaderID != "b" {
		t.Errorf("Expected amILeader=false and LeaderID=b, got amILeader=%t and LeaderID=%s instead\n", amILeader, LeaderID)
	}
}
