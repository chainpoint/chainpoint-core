package anchor

import (
	"github.com/chainpoint/chainpoint-core/proof"
	"github.com/chainpoint/chainpoint-core/types"
)

type AnchorEngine interface {
	AnchorToChain(startTxRange int64, endTxRange int64) error

	CheckAnchor(btcmsg types.BtcTxMsg) error

	BeginTxMonitor(msgBytes []byte) error

	ConfirmAnchor(btcMonObj types.BtcMonMsg) error

	ConstructProof(btca types.BtcTxMsg) (proof.P, error)

	AnchorReward(CoreID string) error

	BlockSyncMonitor()

	MonitorFailedAnchor()

	MonitorConfirmedTx()
}
