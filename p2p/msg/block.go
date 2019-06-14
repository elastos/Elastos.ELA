package msg

import (
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/elanet/pact"
	"github.com/elastos/Elastos.ELA/p2p"
)

// blockCacheSize indicates the limit size of block cache.
const blockCacheSize = 2

// Ensure Block implement p2p.Message interface.
var _ p2p.Message = (*Block)(nil)

var (
	toBlock = func(block common.Serializable) p2p.Message {
		return &Block{block}
	}

	blockCache = NewCache(blockCacheSize, toBlock)
)

type Block struct {
	common.Serializable
}

func NewBlock(block common.Serializable) *Block {
	return blockCache.Get(block).(*Block)
}

func (msg *Block) CMD() string {
	return p2p.CmdBlock
}

func (msg *Block) MaxLength() uint32 {
	return pact.MaxBlockSize
}
