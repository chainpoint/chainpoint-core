package types

import (
	"crypto/ecdsa"
	"database/sql"
	"math/big"
	"time"

	"github.com/chainpoint/chainpoint-core/go-abci-service/lightning"

	"github.com/chainpoint/tendermint/privval"

	"github.com/chainpoint/tendermint/libs/log"
	"github.com/chainpoint/tendermint/p2p"

	cfg "github.com/chainpoint/tendermint/config"
)

// TendermintConfig holds connection info for RPC
type TendermintConfig struct {
	TMServer string
	TMPort   string
	Config   *cfg.Config
	Logger   log.Logger
	FilePV   privval.FilePV
	NodeKey  *p2p.NodeKey
}

//AnchorConfig represents values to configure all connections within the ABCI anchor app
type AnchorConfig struct {
	DBType           string
	BitcoinNetwork   string
	RabbitmqURI      string
	TendermintConfig TendermintConfig
	LightningConfig  lightning.LnClient
	PostgresURI      string
	RedisURI         string
	APIURI           string
	EthConfig        EthConfig
	ECPrivateKey     ecdsa.PrivateKey
	DoNodeManagement bool
	DoNodeAudit      bool
	DoPrivateNetwork bool
	PrivateNodeIPs   []string
	PrivateCoreIPs   []string
	DoCal            bool
	DoAnchor         bool
	AnchorInterval   int
	Logger           *log.Logger
	FilePV           privval.FilePV
	AnchorTimeout    int
	AnchorReward     int
	StakePerVal      int64
}

//EthConfig holds contract addresses and eth node URI
type EthConfig struct {
	EthereumURL          string
	EthPrivateKey        string
	TokenContractAddr    string
	RegistryContractAddr string
}

// AnchorState holds Tendermint/ABCI application state. Persisted by ABCI app
type AnchorState struct {
	TxInt            int64                      `json:"tx_int"`
	Height           int64                      `json:"height"`
	AmValidator      bool                       `json:"validator"`
	AppHash          []byte                     `json:"app_hash"`
	BeginCalTxInt    int64                      `json:"begin_cal_int"`
	EndCalTxInt      int64                      `json:"end_cal_int"`
	LatestCalTxInt   int64                      `json:"latest_cal_int"`
	LatestBtcaTx     []byte                     `json:"latest_btca"`
	LatestBtcaTxInt  int64                      `json:"latest_btca_int"`
	LatestBtcaHeight int64                      `json:"latest_btca_height"`
	LatestBtcTx      string                     `json:"latest_btc"`
	LatestBtcAggRoot string                     `json:"latest_btc_root"`
	LatestBtccTx     []byte                     `json:"latest_btcc"`
	LatestBtccTxInt  int64                      `json:"latest_btcc_int"`
	LatestBtccHeight int64                      `json:"latest_btcc_height"`
	LatestErrRoot	 string                     `json:"latest_btce"`
	LastAnchorCoreID string                     `json:"last_anchor_core_id"`
	LastAuditCoreID  string                     `json:"last_audit_core_id"`
	TxValidation     map[string]TxValidation    `json:"tx_validation"`
	CoreKeys         map[string]ecdsa.PublicKey `json:"-"`
	LnUris           map[string]LnIdentity      `json:"lightning_identities"`
	ChainSynced      bool
	JWKStaked		 bool
	LnStakePrice     int64						`json:"total_stake_price"`
	LnStakePerVal	 int64						`json:"validator_stake_price"`
	LatestNistRecord string
}

type LnIdentity struct {
	Peer            string `json:"peer"`
	RequiredChanAmt int64  `json:"required_satoshis"`
}

// Tx holds custom transaction data and metadata for the Chainpoint Calendar
type Tx struct {
	TxType  string `json:"type"`
	Data    string `json:"data"`
	Version int64  `json:"version"`
	Time    int64  `json:"time"`
	CoreID  string `json:"core_id"`
	Meta    string `json:"meta,omitempty"`
	Sig     string `json:"sig,omitempty"`
}

// Uses simple token bucket method
type RateLimit struct {
	AllowedRate int64
	PerBlocks   int64
	LastCheck   int64
	Bucket      float32
}

// Holds state for validating Transactions
type TxValidation struct {
	LastJWKTxHeight 		 int64
	JWKAllowedRate  		 RateLimit

	LastCalTxHeight 		 int64
	CalAllowedRate  		 RateLimit

	LastBtcaTxHeight 	 	 int64 // for anchoring Cores
	ConfirmedAnchors 		 int64
	BtcaAllowedRate  		 RateLimit

	LastBtccTxHeight 		 int64 // for Cores submitting confirmations, not anchoring Cores
	BtccAllowedRate  		 RateLimit

	LastNISTTxHeight int64 // last "good", non-stale nist record
	NISTAllowedRate  RateLimit
}

// EcdsaSignature : Allows for unmarshalling an ecdsa signature
type EcdsaSignature struct {
	R, S *big.Int
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
	ProofID string `json:"proof_id"`
	Hash    string `json:"hash"`
}

// ProofData : The proof data for a given hash within an aggregation
type ProofData struct {
	ProofID string          `json:"proof_id"`
	Hash    string          `json:"hash"`
	Proof   []ProofLineItem `json:"proof"`
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
	BtcTxHeight      int64  `json:"btctx_height"`
}

// BtcTxMsg : An RMQ message object from btc-tx to btc-mon service
type BtcMsgObj struct {
	BtcTxID   string `json:"tx_id"`
	BtcTxBody string `json:"tx_body"`
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
	TxID        string `json:"tx_id"`
	BlockHeight int64  `json:"block_height"`
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

// Core : Used to represent Core info to and from postgres
type Core struct {
	EthAddr     string
	CoreId      sql.NullString
	PublicIP    sql.NullString
	BlockNumber sql.NullInt64
}

//Jwk : holds key info for validating node requests
type Jwk struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Crv string `json:"crv"`
	X   string `json:"x"`
	Y   string `json:"y"`
}

//CoreAPIStatus : status from Core's api service. Includes pubkey
type CoreAPIStatus struct {
	Version             string    `json:"version"`
	Time                time.Time `json:"time"`
	BaseURI             string    `json:"base_uri"`
	Jwk                 Jwk       `json:"jwk"`
	Network             string    `json:"network"`
	IdentityPubkey      string    `json:"identity_pubkey"`
	LightningAddress    string    `json:"lightning_address"`
	PublicKey           string    `json:"public_key"`
	Uris                []string  `json:"uris"`
	Alias               string    `json:"alias"`
	HashPriceSatoshis   int       `json:"hash_price_satoshis"`
	TotalStakePrice     int       `json:"total_stake_price"`
	ValidatorStakePrice int       `json:"validator_stake_price"`
	ActiveChannelsCount int       `json:"active_channels_count"`
	NodeInfo            struct {
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
	} `json:"node_info"`
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
}
