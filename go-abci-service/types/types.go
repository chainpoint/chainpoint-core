package types

import (
	"time"

	dbm "github.com/tendermint/tendermint/libs/db"
)

// TendermintURI holds connection info for RPC
type TendermintURI struct {
	TMServer string
	TMPort   string
}

// State holds Tendermint/ABCI application state. Persisted by ABCI app
type State struct {
	Db               dbm.DB
	TxInt            int64  `json:"tx_int"`
	Size             int64  `json:"size"`
	Height           int64  `json:"height"`
	AppHash          []byte `json:"app_hash"`
	PrevCalTxInt     int64  `json:"prev_cal_int"`
	LatestCalTxInt   int64  `json:"latest_cal_int"`
	LatestBtcaTx     []byte `json:"latest_btca"`
	LatestBtcaTxInt  int64  `json:"latest_btca_int"`
	LatestBtcaHeight int64  `json:"latest_btca_height"`
	LatestBtccTx     []byte `json:"latest_btcc"`
	LatestBtccTxInt  int64  `json:"latest_btcc_int"`
	LatestBtccHeight int64  `json:"latest_btcc_height"`
}

// Tx holds custom transaction data and metadata for the Chainpoint Calendar
type Tx struct {
	TxType  string `json:"type"`
	Data    string `json:"hash"`
	Version int64  `json:"version"`
	Time    int64  `json:"time"`
}

// TxTm holds result of submitting a CAL transaction (needed in order to get Hash)
type TxTm struct {
	Hash []byte
	Data []byte
}

// BtcAgg : An object containing BTC anchoring aggregation data
type BtcAgg struct {
	AnchorBtcAggID   string         `json:"anchor_btc_agg_id"`
	AnchorBtcAggRoot string         `json:"anchor_btc_agg_root"`
	ProofData        []BtcProofData `json:"proofData"`
}

// BtcProofData : An individual proof object within a Btc aggregation set
type BtcProofData struct {
	CalID string          `json:"cal_id"`
	Proof []ProofLineItem `json:"proof"`
}

// BtcTxMsg : An RMQ message object
type BtcTxMsg struct {
	AnchorBtcAggID   string `json:"anchor_btc_agg_id"`
	AnchorBtcAggRoot string `json:"anchor_btc_agg_root"`
	BtcTxID          string `json:"btctx_id"`
	BtcTxBody        string `json:"btctx_body"`
}

// BtcTxProofState : An RMQ message object bound for proofstate service
type BtcTxProofState struct {
	AnchorBtcAggID string        `json:"anchor_btc_agg_id"`
	BtcTxID        string        `json:"btctx_id"`
	BtcTxState     BtcTxOpsState `json:"btctx_state"`
}

// BtcTxOpsState : TODO: Describe this
type BtcTxOpsState struct {
	Ops []ProofLineItem `json:"ops"`
}

// BtccStateObj : TODO: Describe this
type BtccStateObj struct {
	BtcTxID       string       `json:"btctx_id"`
	BtcHeadHeight int64        `json:"btchead_height"`
	BtcHeadState  BtccOpsState `json:"btchead_state"`
}

// BtccOpsState : TODO: Describe this
type BtccOpsState struct {
	Ops    []ProofLineItem `json:"ops"`
	Anchor AnchorObj       `json:"anchor"`
}

// TxID : TODO: Describe this
type TxID struct {
	TxID string `json:"tx_id"`
}

// CalAgg : TODO: Describe this
type CalAgg struct {
	CalRoot   string      `json:"cal_root"`
	ProofData []ProofData `json:"proofData"`
}

// CalState : TODO: Describe this
type CalState struct {
	CalID     string      `json:"cal_id"`
	Anchor    AnchorObj   `json:"anchor"`
	ProofData []ProofData `json:"proofData"`
}

// BtcMonMsg : TODO: Describe this
type BtcMonMsg struct {
	BtcTxID         string    `json:"btctx_id"`
	BtcHeadHeight int64     `json:"btchead_height"`
	BtcHeadRoot   string    `json:"btchead_root"`
	Path          []JSProof `json:"path"`
}

// AnchorObj : TODO: Describe this
type AnchorObj struct {
	AnchorID string   `json:"anchor_id"`
	Uris     []string `json:"uris"`
}

// ProofData : TODO: Describe this
type ProofData struct {
	AggID string          `json:"agg_id"`
	Proof []ProofLineItem `json:"proof"`
}

// ProofLineItem : TODO: Describe this
type ProofLineItem struct {
	Left  string `json:"l,omitempty"`
	Right string `json:"r,omitempty"`
	Op    string `json:"op,omitempty"`
}

// JSProof : TODO: Describe this
type JSProof struct {
	Left  string `json:"left,omitempty"`
	Right string `json:"right,omitempty"`
}

// NodeStatus : rpc endpoint results. Custom struct is needed for remote_ip encoding workaround
type NodeStatus struct {
	Jsonrpc string `json:"jsonrpc"`
	ID      string `json:"id"`
	Result  struct {
		NodeInfo NodeInfo `json:"node_info"`
		SyncInfo struct {
			LatestBlockHash   string    `json:"latest_block_hash"`
			LatestAppHash     string    `json:"latest_app_hash"`
			LatestBlockHeight string    `json:"latest_block_height"`
			LatestBlockTime   time.Time `json:"latest_block_time"`
			CatchingUp        bool      `json:"catching_up"`
		} `json:"sync_info"`
		ValidatorInfo struct {
			Address string `json:"address"`
			PubKey  struct {
				Type  string `json:"type"`
				Value string `json:"value"`
			} `json:"pub_key"`
			VotingPower string `json:"voting_power"`
		} `json:"validator_info"`
	} `json:"result"`
}

// NodeInfo  : rpc endpoint results. Custom struct is needed for remote_ip encoding workaround
type NodeInfo struct {
	ProtocolVersion struct {
		P2P   string `json:"p2p"`
		Block string `json:"block"`
		App   string `json:"app"`
	} `json:"protocol_version"`
	ID         string `json:"id"`
	ListenAddr string `json:"listen_addr"`
	Network    string `json:"network"`
	Version    string `json:"version"`
	Channels   string `json:"channels"`
	Moniker    string `json:"moniker"`
	Other      struct {
		TxIndex    string `json:"tx_index"`
		RPCAddress string `json:"rpc_address"`
	} `json:"other"`
}

// NetInfo  : rpc endpoint results. Custom struct is needed for remote_ip encoding workaround
type NetInfo struct {
	Jsonrpc string `json:"jsonrpc"`
	ID      string `json:"id"`
	Result  struct {
		Listening bool     `json:"listening"`
		Listeners []string `json:"listeners"`
		NPeers    string   `json:"n_peers"`
		Peers     []struct {
			NodeInfo NodeInfo `json:"node_info"`
			RemoteIP string   `json:"remote_ip"`
		} `json:"peers"`
	} `json:"result"`
}
