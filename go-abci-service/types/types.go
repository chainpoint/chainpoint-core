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
	TxType  string `json: "type"`
	Data    string `json: "hash"`
	Version int64  `json: "version"`
	Time    int64  `json: "time"`
}

// TxTm holds result of submitting a CAL transaction (needed in order to get Hash
type TxTm struct {
	Hash []byte
	Data []byte
}

type BtcAgg struct {
	AggId     string         `json:"anchor_btc_agg_id"`
	AggRoot   string         `json:"anchor_btc_agg_root"`
	ProofData []BtcProofData `json:"proofData"`
}

type BtcProofData struct {
	CalId string  `json:"cal_id"`
	Proof []Proof `json:"proof"`
}

type BtcTxMsg struct {
	AggId   string `json:"anchor_btc_agg_id"`
	AggRoot string `json:"anchor_btc_agg_root"`
	BtxId   string `json:"btctx_id"`
	BtxBody string `json:"btctx_body"`
}

type BtcTxProofState struct {
	AggId    string        `json:"anchor_btc_agg_id"`
	BtcId    string        `json:"btctx_id"`
	BtcState BtcTxOpsState `json:"btctx_state"`
}

type BtcTxOpsState struct {
	Ops []Proof `json:"ops"`
}

type TxId struct {
	TxID string `json:"tx_id"`
}

type CalAgg struct {
	CalRoot   string      `json:"cal_root"`
	ProofData []ProofData `json:"proofData"`
}

type CalState struct {
	CalId     string      `json:"cal_id"`
	Anchor    CalAnchor   `json:"anchor"`
	ProofData []ProofData `json:"proofData"`
}

type BtcMonMsg struct {
	BtcId         string `json:"btctx_id"`
	BtcHeadHeight int64  `json:"btchead_height"`
	BtcHeadRoot   string `json:"btchead_root"`
}

type CalAnchor struct {
	AnchorId string   `json:"anchor_id"`
	Uris     []string `json:"uris"`
}

type ProofData struct {
	AggId string  `json:"agg_id"`
	Proof []Proof `json:"proof"`
}

type Proof struct {
	Left  string `json:"l,omitempty"`
	Right string `json:"r,omitempty"`
	Op    string `json:"op,omitempty"`
}

// NodeStatus rpc endpoint results. Custom struct is needed for remote_ip encoding workaround
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

// NodeInfo rpc endpoint results. Custom struct is needed for remote_ip encoding workaround
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

// NetInfo rpc endpoint results. Custom struct is needed for remote_ip encoding workaround
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
