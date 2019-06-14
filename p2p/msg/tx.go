package msg

import (
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/elanet/pact"
	"github.com/elastos/Elastos.ELA/p2p"
)

// txCacheSize indicates the limit size of tx cache.
const txCacheSize = 2

// Ensure Tx implement p2p.Message interface.
var _ p2p.Message = (*Tx)(nil)

var (
	toTx = func(block common.Serializable) p2p.Message {
		return &Tx{block}
	}

	txCache = NewCache(txCacheSize, toTx)
)

type Tx struct {
	common.Serializable
}

func NewTx(tx common.Serializable) *Tx {
	return txCache.Get(tx).(*Tx)
}

func (msg *Tx) CMD() string {
	return p2p.CmdTx
}

func (msg *Tx) MaxLength() uint32 {
	return pact.MaxBlockSize
}
