// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"errors"
	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	elaerr "github.com/elastos/Elastos.ELA/errors"
	"math"
)

type CRCProposalResultTransaction struct {
	BaseTransaction
}

func (t *CRCProposalResultTransaction) RegisterFunctions() {
	t.DefaultChecker.CheckTransactionSize = t.checkTransactionSize
	t.DefaultChecker.CheckTransactionInput = t.checkTransactionInput
	t.DefaultChecker.CheckTransactionOutput = t.checkTransactionOutput
	t.DefaultChecker.CheckTransactionPayload = t.CheckTransactionPayload
	t.DefaultChecker.HeightVersionCheck = t.heightVersionCheck
	t.DefaultChecker.IsAllowedInPOWConsensus = t.IsAllowedInPOWConsensus
	t.DefaultChecker.SpecialContextCheck = t.SpecialContextCheck
	t.DefaultChecker.CheckAttributeProgram = t.checkAttributeProgram
}

func (t *CRCProposalResultTransaction) CheckTransactionInput() error {
	if len(t.parameters.Transaction.Inputs()) != 0 {
		return errors.New("no cost transactions must has no input")
	}
	return nil
}

func (t *CRCProposalResultTransaction) CheckTransactionOutput() error {

	txn := t.parameters.Transaction
	if len(txn.Outputs()) > math.MaxUint16 {
		return errors.New("output count should not be greater than 65535(MaxUint16)")
	}
	if len(txn.Outputs()) != 0 {
		return errors.New("no cost transactions should have no output")
	}

	return nil
}

func (t *CRCProposalResultTransaction) CheckAttributeProgram() error {
	if len(t.Programs()) != 0 || len(t.Attributes()) != 0 {
		return errors.New("zero cost tx should have no attributes and programs")
	}
	return nil
}

func (t *CRCProposalResultTransaction) CheckTransactionPayload() error {
	switch t.Payload().(type) {
	case *payload.RecordProposalResult:
		return nil
	}

	return errors.New("invalid payload type")
}

func (t *CRCProposalResultTransaction) IsAllowedInPOWConsensus() bool {
	return true
}

func (t *CRCProposalResultTransaction) SpecialContextCheck() (elaerr.ELAError, bool) {
	if !blockchain.DefaultLedger.Committee.IsProposalResultNeeded() {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("should not have proposal result transaction")), true
	}
	p, ok := t.Payload().(*payload.RecordProposalResult)
	if !ok {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid proposal result payload")), true
	}
	results := blockchain.DefaultLedger.Committee.GetCustomIDResults()
	targetResults := make(map[common.Uint256]payload.ProposalResult, 0)
	for _, r := range results {
		targetResults[r.ProposalHash] = r
	}
	if len(p.ProposalResults) != len(targetResults) {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid proposal results count")), true
	}
	for _, r := range p.ProposalResults {
		ret, ok := targetResults[r.ProposalHash]
		if !ok || ret.Result != r.Result || ret.ProposalType != r.ProposalType {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid proposal results")), true
		}
	}
	return nil, true
}
