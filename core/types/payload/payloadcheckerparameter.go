// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package payload

import (
	"github.com/elastos/Elastos.ELA/common"
	pg "github.com/elastos/Elastos.ELA/core/contract/program"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
)

type CheckParameters struct {
	// transaction
	Version        common2.TransactionVersion
	TxType         common2.TxType
	PayloadVersion byte
	Attributes     []*common2.Attribute
	Inputs         []*common2.Input
	Outputs        []*common2.Output
	LockTime       uint32
	Programs       []*pg.Program
	TxHash         common.Uint256

	// others
	BlockHeight            uint32
	CRCommitteeStartHeight uint32
	ConsensusAlgorithm     byte
	DestroyELAAddress      common.Uint168
	CRAssetsAddress        common.Uint168
	FoundationAddress      common.Uint168
}
