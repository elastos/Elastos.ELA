package blocks

import (
	"github.com/elastos/Elastos.ELA/version/verconf"
)

// Ensure blockV2 implement the BlockVersion interface.
var _ BlockVersion = (*blockV2)(nil)

// blockV2 represent the current block version.
type blockV2 struct {
	*blockV1
}

func (b *blockV2) GetVersion() uint32 {
	return 2
}

func NewBlockV2(cfg *verconf.Config) *blockV2 {
	return &blockV2{NewBlockV1(cfg)}
}
