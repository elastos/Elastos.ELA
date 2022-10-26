// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	crstate "github.com/elastos/Elastos.ELA/cr/state"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type UnregisterCRTransaction struct {
	BaseTransaction
}

func (t *UnregisterCRTransaction) CheckTransactionPayload() error {
	switch t.Payload().(type) {
	case *payload.UnregisterCR:
		return nil
	}

	return errors.New("invalid payload type")
}

func (t *UnregisterCRTransaction) IsAllowedInPOWConsensus() bool {
	return false
}

func (t *UnregisterCRTransaction) HeightVersionCheck() error {
	blockHeight := t.parameters.BlockHeight
	chainParams := t.parameters.Config

	if blockHeight < chainParams.CRConfiguration.CRVotingStartHeight {
		return errors.New(fmt.Sprintf("not support %s transaction "+
			"before CRVotingStartHeight", t.TxType().Name()))
	}
	return nil
}

func (t *UnregisterCRTransaction) SpecialContextCheck() (elaerr.ELAError, bool) {
	info, ok := t.Payload().(*payload.UnregisterCR)
	if !ok {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid payload")), true
	}

	if !t.parameters.BlockChain.GetCRCommittee().IsInVotingPeriod(t.parameters.BlockHeight) {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("should create tx during voting period")), true
	}

	cr := t.parameters.BlockChain.GetCRCommittee().GetCandidate(info.CID)
	if cr == nil {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("unregister unknown CR")), true
	}
	if cr.State != crstate.Pending && cr.State != crstate.Active {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("unregister canceled or returned CR")), true
	}

	signedBuf := new(bytes.Buffer)
	err := info.SerializeUnsigned(signedBuf, payload.UnregisterCRVersion)
	if err != nil {
		return elaerr.Simple(elaerr.ErrTxPayload, err), true
	}

	err = blockchain.CheckCRTransactionSignature(info.Signature, cr.Info.Code, signedBuf.Bytes())
	if err != nil {
		return elaerr.Simple(elaerr.ErrTxPayload, err), true
	}

	return nil, false
}
