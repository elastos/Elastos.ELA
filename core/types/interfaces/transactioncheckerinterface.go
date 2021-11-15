// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package interfaces

import (
	"github.com/elastos/Elastos.ELA/common"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type PayloadChecker interface {
	ContextCheck(p Parameters) (map[*common2.Input]common2.Output, elaerr.ELAError)

	// todo ... add SanityCheck
}

type BasePayloadChecker interface {
	CheckTxHeightVersion() error

	IsTxHashDuplicate(txHash common.Uint256) bool

	GetTxReference() (map[*common2.Input]common2.Output, error)

	CheckPOWConsensusTransaction(references map[*common2.Input]common2.Output) error

	// todo add description
	SpecialCheck() (error elaerr.ELAError, end bool)

	// todo ... more check
}
