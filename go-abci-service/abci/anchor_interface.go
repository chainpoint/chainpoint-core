package abci

import "github.com/chainpoint/chainpoint-core/go-abci-service/types"

type AnchorEngine interface {

	AnchorToChain(startTxRange int64, endTxRange int64) (error)

	CheckAnchor(btcmsg types.BtcTxMsg) (error)

	BeginTxMonitor(msgBytes []byte) (error)

	ConfirmTxMsg(btcMonObj types.BtcMonMsg) (error)

	BlockSyncMonitor()

	FailedAnchorMonitor()

	MonitorConfirmedTx()

}
