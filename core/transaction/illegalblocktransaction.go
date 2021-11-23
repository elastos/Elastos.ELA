// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//
package transaction

import (
	"errors"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"math"

	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type IllegalBlockTransaction struct {
	BaseTransaction
}

func (t *IllegalBlockTransaction) RegisterFunctions() {
	t.DefaultChecker.CheckTransactionSize = t.checkTransactionSize
	t.DefaultChecker.CheckTransactionInput = t.CheckTransactionInput
	t.DefaultChecker.CheckTransactionOutput = t.CheckTransactionOutput
	t.DefaultChecker.CheckTransactionPayload = t.CheckTransactionPayload
	t.DefaultChecker.HeightVersionCheck = t.heightVersionCheck
	t.DefaultChecker.IsAllowedInPOWConsensus = t.IsAllowedInPOWConsensus
	t.DefaultChecker.SpecialContextCheck = t.SpecialContextCheck
	t.DefaultChecker.CheckAttributeProgram = t.CheckAttributeProgram
}

func (t *IllegalBlockTransaction) CheckTransactionInput(params *TransactionParameters) error {
	if len(params.Transaction.Inputs()) != 0 {
		return errors.New("no cost transactions must has no input")
	}
	return nil
}

func (t *IllegalBlockTransaction) CheckTransactionOutput(params *TransactionParameters) error {

	txn := params.Transaction
	if len(txn.Outputs()) > math.MaxUint16 {
		return errors.New("output count should not be greater than 65535(MaxUint16)")
	}
	if len(txn.Outputs()) != 0 {
		return errors.New("no cost transactions should have no output")
	}

	return nil
}

func (t *IllegalBlockTransaction) CheckAttributeProgram(params *TransactionParameters) error {
	if len(t.Programs()) != 0 {
		return errors.New("illegal block transactions should have one and only one program")
	}
	if len(t.Attributes()) != 0 {
		return errors.New("illegal block transactions should have no programs")
	}
	return nil
}

func (t *IllegalBlockTransaction) CheckTransactionPayload(params *TransactionParameters) error {
	switch t.Payload().(type) {
	case *payload.DPOSIllegalBlocks:
		return nil
	}

	return errors.New("invalid payload type")
}

func (t *IllegalBlockTransaction) IsAllowedInPOWConsensus(params *TransactionParameters, references map[*common2.Input]common2.Output) bool {
	return true
}

func (t *IllegalBlockTransaction) SpecialContextCheck(params *TransactionParameters, references map[*common2.Input]common2.Output) (elaerr.ELAError, bool) {
	p, ok := t.Payload().(*payload.DPOSIllegalBlocks)
	if !ok {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid payload")), true
	}

	if t.parameters.BlockChain.GetState().SpecialTxExists(t) {
		return elaerr.Simple(elaerr.ErrTxDuplicate, errors.New("tx already exists")), true
	}

	if err := blockchain.CheckDPOSIllegalBlocks(p); err != nil {
		return elaerr.Simple(elaerr.ErrTxPayload, err), true
	}

	return nil, true
}
