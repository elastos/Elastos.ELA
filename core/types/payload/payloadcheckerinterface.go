// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package payload

import (
	"github.com/elastos/Elastos.ELA/common"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type PayloadChecker interface {
	ContextCheck(p *CheckParameters) elaerr.ELAError

	// todo ... add SanityCheck
}

type BasePayloadChecker interface {
	CheckTxHeightVersion(p *CheckParameters) error

	IsTxHashDuplicate(txHash common.Uint256) bool

	// todo add description
	SpecialCheck(p *CheckParameters) (error elaerr.ELAError, end bool)

	// todo ... more check
}
