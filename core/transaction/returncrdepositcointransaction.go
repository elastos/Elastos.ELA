// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"errors"
	"fmt"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/contract"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type ReturnCRDepositCoinTransaction struct {
	BaseTransaction
}

func (t *ReturnCRDepositCoinTransaction) RegisterFunctions() {
	t.DefaultChecker.CheckTransactionSize = t.checkTransactionSize
	t.DefaultChecker.CheckTransactionInput = t.checkTransactionInput
	t.DefaultChecker.CheckTransactionOutput = t.checkTransactionOutput
	t.DefaultChecker.CheckTransactionPayload = t.CheckTransactionPayload
	t.DefaultChecker.HeightVersionCheck = t.HeightVersionCheck
	t.DefaultChecker.IsAllowedInPOWConsensus = t.IsAllowedInPOWConsensus
	t.DefaultChecker.SpecialContextCheck = t.SpecialContextCheck
	t.DefaultChecker.CheckAttributeProgram = t.CheckAttributeProgram
}

func (t *ReturnCRDepositCoinTransaction) CheckAttributeProgram() error {

	if len(t.Programs()) != 1 {
		return errors.New("return CR deposit coin transactions should have one and only one program")
	}

	// Check attributes
	for _, attr := range t.Attributes() {
		if !common2.IsValidAttributeType(attr.Usage) {
			return fmt.Errorf("invalid attribute usage %v", attr.Usage)
		}
	}

	// Check programs
	if len(t.Programs()) == 0 {
		return fmt.Errorf("no programs found in transaction")
	}
	for _, program := range t.Programs() {
		if program.Code == nil {
			return fmt.Errorf("invalid program code nil")
		}
		if program.Parameter == nil {
			return fmt.Errorf("invalid program parameter nil")
		}
	}

	return nil
}

func (t *ReturnCRDepositCoinTransaction) CheckTransactionPayload() error {
	if t.Payload() != nil {
		return errors.New("invalid payload nee to be nil")
	}
	return nil
}

func (t *ReturnCRDepositCoinTransaction) IsAllowedInPOWConsensus() bool {
	return false
}

func (t *ReturnCRDepositCoinTransaction) HeightVersionCheck() error {
	txn := t.parameters.Transaction
	blockHeight := t.parameters.BlockHeight
	chainParams := t.parameters.Config

	if blockHeight < chainParams.CRVotingStartHeight {
		return errors.New(fmt.Sprintf("not support %s transaction "+
			"before CRVotingStartHeight", txn.TxType().Name()))
	}
	return nil
}

func (t *ReturnCRDepositCoinTransaction) SpecialContextCheck() (elaerr.ELAError, bool) {

	var inputValue common.Fixed64
	fromAddrMap := make(map[common.Uint168]struct{})
	for _, output := range t.references {
		inputValue += output.Value
		fromAddrMap[output.ProgramHash] = struct{}{}
	}

	if len(fromAddrMap) != 1 {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("UTXO should from same deposit address")), true
	}

	var programHash common.Uint168
	for k := range fromAddrMap {
		programHash = k
	}

	var changeValue common.Fixed64
	var outputValue common.Fixed64
	for _, output := range t.Outputs() {
		if output.ProgramHash.IsEqual(programHash) {
			changeValue += output.Value
		} else {
			outputValue += output.Value
		}
	}

	var availableValue common.Fixed64
	for _, program := range t.Programs() {
		// Get candidate from code.
		ct, err := contract.CreateCRIDContractByCode(program.Code)
		if err != nil {
			return elaerr.Simple(elaerr.ErrTxPayload, err), true
		}
		cid := ct.ToProgramHash()
		if !t.parameters.BlockChain.GetCRCommittee().Exist(*cid) {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("signer must be candidate or member")), true
		}

		availableValue += t.parameters.BlockChain.GetCRCommittee().GetAvailableDepositAmount(*cid)
	}

	// Check output amount.
	if inputValue-changeValue > availableValue ||
		outputValue >= availableValue {
		return elaerr.Simple(elaerr.ErrTxPayload, fmt.Errorf("candidate overspend deposit")), true
	}

	return nil, false
}
