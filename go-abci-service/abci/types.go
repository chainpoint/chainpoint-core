package abci

import (
	dbm "github.com/tendermint/tendermint/libs/db"
)

type State struct {
	db      		   dbm.DB
	txInt   		   int64  `json:"txInt"`
	Size    		   int64  `json:"size"`
	Height  		   int64  `json:"height"`
	AppHash 		   []byte `json:"app_hash"`
	LatestBtcaTx       []byte `json:"latest_btca"`
	LatestBtcaHeight   int64  `json:"latest_btca_height"`
	LatestBtccTx       []byte `json:"latest_btcc"`
	LatestBtccHeight   int64  `json:"latest_btcc_height"`
}

type Tx struct {
	TxType  []byte `json: "type"`
	Data    []byte `json: "hash"`
	Version int64  `json: "version"`
	Time    int64  `json: "time"`
}