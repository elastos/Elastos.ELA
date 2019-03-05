package blocks

import (
	"github.com/elastos/Elastos.ELA/core/types"
)

type BlockVersion interface {
	GetVersion() uint32
	AddDposBlock(block *types.DposBlock) (bool, bool, error)
	CheckConfirmedBlockOnFork(block *types.Block) error
}
