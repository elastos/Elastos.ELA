package transaction

import (
	. "github.com/elastos/Elastos.ELA.Utility/common"
)

// ILedgerStore provides func with store package.
type ILedgerStore interface {
	GetTransaction(hash Uint256) (*NodeTransaction, uint32, error)
	//GetQuantityIssued(AssetId Uint256) (Fixed64, error)
}
