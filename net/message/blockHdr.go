package message

import (
	"Elastos.ELA/common"
	"Elastos.ELA/core/ledger"
	. "Elastos.ELA/net/protocol"
)

func SendMsgSyncHeaders(node Noder, startHash common.Uint256) {
	var emptyHash common.Uint256
	blocator := ledger.DefaultLedger.Blockchain.BlockLocatorFromHash(&startHash)
	SendMsgSyncBlockHeaders(node, blocator, emptyHash)
}
