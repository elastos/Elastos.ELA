// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"errors"

	"github.com/elastos/Elastos.ELA/core/types/payload"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type UpdateVersionTransaction struct {
	BaseTransaction
}

func (t *UpdateVersionTransaction) CheckTransactionInput() error {
	if len(t.sanityParameters.Transaction.Inputs()) != 0 {
		return errors.New("no cost transactions must has no input")
	}
	return nil
}

func (t *UpdateVersionTransaction) IsAllowedInPOWConsensus() bool {
	return false
}

func (t *UpdateVersionTransaction) SpecialContextCheck() (elaerr.ELAError, bool) {
	payload, ok := t.Payload().(*payload.UpdateVersion)
	if !ok {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid payload")), true
	}

	if payload.EndHeight <= payload.StartHeight ||
		payload.StartHeight < t.contextParameters.BlockChain.GetHeight() {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid update version height")), true
	}

	return elaerr.Simple(elaerr.ErrTxPayload, checkCRCArbitratorsSignatures(t.Programs()[0])), true
}
