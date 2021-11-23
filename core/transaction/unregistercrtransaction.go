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
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	crstate "github.com/elastos/Elastos.ELA/cr/state"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type UnregisterCRTransaction struct {
	BaseTransaction
}

func (t *UnregisterCRTransaction) RegisterFunctions() {
	t.DefaultChecker.CheckTransactionSize = t.checkTransactionSize
	t.DefaultChecker.CheckTransactionInput = t.checkTransactionInput
	t.DefaultChecker.CheckTransactionOutput = t.checkTransactionOutput
	t.DefaultChecker.CheckTransactionPayload = t.CheckTransactionPayload
	t.DefaultChecker.HeightVersionCheck = t.HeightVersionCheck
	t.DefaultChecker.IsAllowedInPOWConsensus = t.IsAllowedInPOWConsensus
	t.DefaultChecker.SpecialContextCheck = t.SpecialContextCheck
	t.DefaultChecker.CheckAttributeProgram = t.checkAttributeProgram
}

func (t *UnregisterCRTransaction) CheckTransactionPayload(params *TransactionParameters) error {
	switch t.Payload().(type) {
	case *payload.UnregisterCR:
		return nil
	}

	return errors.New("invalid payload type")
}

func (t *UnregisterCRTransaction) IsAllowedInPOWConsensus(params *TransactionParameters, references map[*common2.Input]common2.Output) bool {
	return false
}

func (t *UnregisterCRTransaction) HeightVersionCheck(params *TransactionParameters) error {
	txn := params.Transaction
	blockHeight := params.BlockHeight
	chainParams := params.Config

	if blockHeight < chainParams.CRVotingStartHeight {
		return errors.New(fmt.Sprintf("not support %s transaction "+
			"before CRVotingStartHeight", txn.TxType().Name()))
	}
	return nil
}

func (t *UnregisterCRTransaction) SpecialContextCheck(params *TransactionParameters, references map[*common2.Input]common2.Output) (elaerr.ELAError, bool) {
	info, ok := t.Payload().(*payload.UnregisterCR)
	if !ok {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid payload")), true
	}

	if !params.BlockChain.GetCRCommittee().IsInVotingPeriod(params.BlockHeight) {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("should create tx during voting period")), true
	}

	cr := params.BlockChain.GetCRCommittee().GetCandidate(info.CID)
	if cr == nil {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("unregister unknown CR")), true
	}
	if cr.State() != crstate.Pending && cr.State() != crstate.Active {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("unregister canceled or returned CR")), true
	}

	signedBuf := new(bytes.Buffer)
	err := info.SerializeUnsigned(signedBuf, payload.UnregisterCRVersion)
	if err != nil {
		return elaerr.Simple(elaerr.ErrTxPayload, err), true
	}

	err = blockchain.CheckCRTransactionSignature(info.Signature, cr.Info().Code, signedBuf.Bytes())
	if err != nil {
		return elaerr.Simple(elaerr.ErrTxPayload, err), true
	}

	return nil, false
}
