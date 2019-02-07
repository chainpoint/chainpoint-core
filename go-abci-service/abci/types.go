package abci

import (
	dbm "github.com/tendermint/tendermint/libs/db"
)

type State struct {
	db      dbm.DB
	txInt   int64  `json:"txInd"`
	Size    int64  `json:"size"`
	Height  int64  `json:"height"`
	AppHash []byte `json:"app_hash"`
}