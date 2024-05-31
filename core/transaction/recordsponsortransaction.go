// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"bytes"
	"errors"

	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type RecordSponsorTransaction struct {
	BaseTransaction
}

func (t *RecordSponsorTransaction) CheckTransactionInput() error {

	if len(t.Inputs()) != 0 {
		return errors.New("no cost transactions must has no input")
	}

	return nil
}

func (t *RecordSponsorTransaction) CheckTransactionOutput() error {

	if len(t.Outputs()) != 0 {
		return errors.New("no need to have output in sponsor transaction")
	}

	return nil
}

func (t *RecordSponsorTransaction) CheckAttributeProgram() error {

	if len(t.Programs()) != 0 || len(t.Attributes()) != 0 {
		return errors.New("no need to have attribute or program in sponsor transaction")
	}

	return nil
}

func (t *RecordSponsorTransaction) CheckTransactionPayload() error {
	switch t.Payload().(type) {
	case *payload.RecordSponsor:
		return nil
	}

	return errors.New("invalid payload type")
}

func (t *RecordSponsorTransaction) IsAllowedInPOWConsensus() bool {
	return true
}

func (t *RecordSponsorTransaction) SpecialContextCheck() (elaerr.ELAError, bool) {
	payloadRecordSponsor, ok := t.Payload().(*payload.RecordSponsor)
	if !ok {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("record sponsor transaction has invalid payload")), true
	}

	// check sponsor is in current or last arbitrators
	current, last := blockchain.DefaultLedger.Arbitrators.GetCurrentAndLastArbitrators()
	exist := false
	for _, currentArbiter := range current {
		if bytes.Equal(currentArbiter.NodePublicKey, payloadRecordSponsor.Sponsor) {
			exist = true
			break
		}
	}
	for _, lastArbiter := range last {
		if bytes.Equal(lastArbiter.NodePublicKey, payloadRecordSponsor.Sponsor) {
			exist = true
			break
		}
	}
	if !exist {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("sponsor is not in current or last arbitrators")), true
	}

	return nil, false
}
