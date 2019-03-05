package blocks

import (
	"github.com/elastos/Elastos.ELA/core/types"
	"github.com/elastos/Elastos.ELA/version/verconf"
)

// Ensure blockV0 implement the BlockVersion interface.
var _ BlockVersion = (*blockV0)(nil)

// blockV0 represent the version 0 block.
type blockV0 struct {
	cfg *verconf.Config
}

func (b *blockV0) GetVersion() uint32 {
	return 0
}

func (b *blockV0) CheckConfirmedBlockOnFork(block *types.Block) error {
	return nil
}

func (b *blockV0) AddDposBlock(dposBlock *types.DposBlock) (bool, bool, error) {
	return b.cfg.Chain.ProcessBlock(dposBlock.Block, dposBlock.Confirm)
}

func NewBlockV0(cfg *verconf.Config) *blockV0 {
	return &blockV0{cfg: cfg}
}
