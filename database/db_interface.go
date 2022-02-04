package database

import "github.com/chainpoint/chainpoint-core/types"

type ChainpointDatabase interface {
	GetProofIdsByAggIds(aggIds []string) ([]string, error)
	GetProofsByProofIds(proofIds []string) (map[string]types.ProofState, error)
	GetProofIdsByBtcTxId(btcTxId string) ([]string, error)
	GetCalStateObjectsByAggIds(aggIds []string) ([]types.CalStateObject, error)
	GetAggStateObjectsByProofIds(proofIds []string) ([]types.AggState, error)
	GetAnchorBTCAggStateObjectsByCalIds(calIds []string) ([]types.AnchorBtcAggState, error)
	GetBTCTxStateObjectByAnchorBTCAggId(aggId string) (types.AnchorBtcTxState, error)
	GetBTCTxStateObjectByBtcHeadState(btctx string) (types.AnchorBtcTxState, error)
	BulkInsertProofs(proofs []types.ProofState) error
	BulkInsertAggState(aggStates []types.AggState) error
	BulkInsertCalState(calStates []types.CalStateObject) error
	BulkInsertBtcAggState(aggStates []types.AnchorBtcAggState) error
	BulkInsertBtcTxState(txStates []types.AnchorBtcTxState) error
	PruneOldState()
}
