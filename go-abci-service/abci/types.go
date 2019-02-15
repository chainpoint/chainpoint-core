package abci

import (
	"time"

	dbm "github.com/tendermint/tendermint/libs/db"
)

type State struct {
	db               dbm.DB
	txInt            int64  `json:"txInt"`
	Size             int64  `json:"size"`
	Height           int64  `json:"height"`
	AppHash          []byte `json:"app_hash"`
	LatestBtcaTx     []byte `json:"latest_btca"`
	LatestBtcaTxInt  int64  `json:"latest_btca_int"`
	LatestBtcaHeight int64  `json:"latest_btca_height"`
	LatestBtccTx     []byte `json:"latest_btcc"`
	LatestBtccTxInt  int64  `json:"latest_btcc_int"`
	LatestBtccHeight int64  `json:"latest_btcc_height"`
}

type Tx struct {
	TxType  []byte `json: "type"`
	Data    []byte `json: "hash"`
	Version int64  `json: "version"`
	Time    int64  `json: "time"`
}

type TxTm struct {
	Hash []byte
	Data []byte
}

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
