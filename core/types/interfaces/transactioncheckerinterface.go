// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package interfaces

import (
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type PayloadChecker interface {
	ContextCheck(p Parameters) (map[*common2.Input]common2.Output, elaerr.ELAError)

	SanityCheck(p Parameters) elaerr.ELAError
}

type BasePayloadChecker interface {

	HeightVersionCheck() error

	IsAllowedInPOWConsensus() bool

	// todo add description
	SpecialContextCheck() (error elaerr.ELAError, end bool)
}
