// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//
package transaction

import (
	"errors"

	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type IllegalBlockTransaction struct {
	BaseTransaction
}

func (a *IllegalBlockTransaction) SpecialCheck() (elaerr.ELAError, bool) {
	p, ok := a.Payload().(*payload.DPOSIllegalBlocks)
	if !ok {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid payload")), true
	}

	if a.contextParameters.BlockChain.GetState().SpecialTxExists(a) {
		return elaerr.Simple(elaerr.ErrTxDuplicate, errors.New("tx already exists")), true
	}

	return elaerr.Simple(elaerr.ErrTxDuplicate, blockchain.CheckDPOSIllegalBlocks(p)), true
}
