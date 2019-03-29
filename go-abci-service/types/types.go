package types

import (
	"database/sql"

	"github.com/tendermint/tendermint/libs/log"
)

// TendermintURI holds connection info for RPC
type TendermintURI struct {
	TMServer string
	TMPort   string
}

//AnchorConfig represents values to configure all connections within the ABCI anchor app
type AnchorConfig struct {
	DBType               string
	RabbitmqURI          string
	TendermintRPC        TendermintURI
	PostgresURI          string
	EthereumURL          string
	TokenContractAddr    string
	RegistryContractAddr string
	DoCal                bool
	DoAnchor             bool
	AnchorInterval       int
	Logger               *log.Logger
}

// AnchorState holds Tendermint/ABCI application state. Persisted by ABCI app
type AnchorState struct {
	TxInt            int64  `json:"tx_int"`
	Height           int64  `json:"height"`
	ChainSynced      bool   `json:"chain_synced"`
	AppHash          []byte `json:"app_hash"`
	BeginCalTxInt    int64  `json:"begin_cal_int"`
	EndCalTxInt      int64  `json:"end_cal_int"`
	LatestCalTxInt   int64  `json:"latest_cal_int"`
	LatestBtcaTx     []byte `json:"latest_btca"`
	LatestBtcaTxInt  int64  `json:"latest_btca_int"`
	LatestBtcaHeight int64  `json:"latest_btca_height"`
	LatestBtcTx      string `json:"latest_btc"`
	LatestBtcmTxInt  int64  `json:"latest_btcm_int"`
	LatestBtcmHeight int64  `json:"latest_btcm_height"`
	LatestBtccTx     []byte `json:"latest_btcc"`
	LatestBtccTxInt  int64  `json:"latest_btcc_int"`
	LatestBtccHeight int64  `json:"latest_btcc_height"`
	LatestNistRecord string `json:"latest_nist_record"`
}

// Tx holds custom transaction data and metadata for the Chainpoint Calendar
type Tx struct {
	TxType  string `json:"type"`
	Data    string `json:"data"`
	Version int64  `json:"version"`
	Time    int64  `json:"time"`
}

// BtcA struct will be included in the BTC-A tx data field
type BtcA struct {
	AnchorBtcAggRoot string `json:"anchor_btc_agg_root"`
	BtcTxID          string `json:"btctx_id"`
}

// TxTm holds result of submitting a CAL transaction (needed in order to get Hash)
type TxTm struct {
	Hash []byte
	Data []byte
}

// Aggregation : An object containing all the relevant data for an aggregation event
type Aggregation struct {
	AggID     string      `json:"agg_id"`
	AggRoot   string      `json:"agg_root"`
	ProofData []ProofData `json:"proofData"`
}

// HashItem : An object contains the Core ID and value for a hash
type HashItem struct {
	HashID string `json:"hash_id"`
	Hash   string `json:"hash"`
}

// ProofData : The proof data for a given hash within an aggregation
type ProofData struct {
	HashID string          `json:"hash_id"`
	Hash   string          `json:"hash"`
	Proof  []ProofLineItem `json:"proof"`
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

// BtcTxOpsState : An RMQ message generated as part of the monitoring proof object
type BtcTxOpsState struct {
	Ops []ProofLineItem `json:"ops"`
}

// BtccStateObj :  An RMQ message object issued to generate proofs after BTCC confirmation
type BtccStateObj struct {
	BtcTxID       string       `json:"btctx_id"`
	BtcHeadHeight int64        `json:"btchead_height"`
	BtcHeadState  BtccOpsState `json:"btchead_state"`
}

// BtccOpsState : Part of the RMQ message for btc anchoring post-confirmation
type BtccOpsState struct {
	Ops    []ProofLineItem `json:"ops"`
	Anchor AnchorObj       `json:"anchor"`
}

// TxID : RMQ message dispatched to initiate monitoring
type TxID struct {
	TxID string `json:"tx_id"`
}

// CalAgg : An RMQ message representing an intermediate aggregation object to be fed into the Cal anchor tree
type CalAgg struct {
	CalRoot   string         `json:"cal_root"`
	ProofData []CalProofData `json:"proofData"`
}

// CalState : An RMQ message confirming a CAL anchor, sent to the proof service to generate/store the proof
type CalState struct {
	CalID     string         `json:"cal_id"`
	Anchor    AnchorObj      `json:"anchor"`
	ProofData []CalProofData `json:"proofData"`
}

// BtcMonMsg : An RMQ message sent by the monitoring service to confirm a BTC transaction has occurred
type BtcMonMsg struct {
	BtcTxID       string    `json:"btctx_id"`
	BtcHeadHeight int64     `json:"btchead_height"`
	BtcHeadRoot   string    `json:"btchead_root"`
	Path          []JSProof `json:"path"`
}

// AnchorObj : Utilized by the proof spec to represent an anchoring proof step
type AnchorObj struct {
	AnchorID string   `json:"anchor_id"`
	Uris     []string `json:"uris"`
}

// CalProofData : Represents a step in a cal proof
type CalProofData struct {
	AggID string          `json:"agg_id"`
	Proof []ProofLineItem `json:"proof"`
}

// ProofLineItem : A step in a Chainpoint proof
type ProofLineItem struct {
	Left  string `json:"l,omitempty"`
	Right string `json:"r,omitempty"`
	Op    string `json:"op,omitempty"`
}

// JSProof : Used to unmarshall the Javascript MerkleTools proofs. The library generates a different proof structure than the go version.
type JSProof struct {
	Left  string `json:"left,omitempty"`
	Right string `json:"right,omitempty"`
}

// Node : Used to represent Node info to and from postgres
type Node struct {
	EthAddr              string
	PublicIP             sql.NullString
	AmountStaked         sql.NullInt64
	StakeExpiration      sql.NullInt64
	ActiveTokenHash      sql.NullString
	ActiveTokenTimestamp sql.NullInt64
	Balance              sql.NullInt64
	BlockNumber          sql.NullInt64
}
