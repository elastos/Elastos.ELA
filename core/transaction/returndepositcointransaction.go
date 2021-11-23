// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"errors"
	"fmt"
	"github.com/elastos/Elastos.ELA/core/types/payload"

	"github.com/elastos/Elastos.ELA/common"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type ReturnDepositCoinTransaction struct {
	BaseTransaction
}

func (t *ReturnDepositCoinTransaction) RegisterFunctions() {
	t.DefaultChecker.CheckTransactionSize = t.checkTransactionSize
	t.DefaultChecker.CheckTransactionInput = t.checkTransactionInput
	t.DefaultChecker.CheckTransactionOutput = t.checkTransactionOutput
	t.DefaultChecker.CheckTransactionPayload = t.CheckTransactionPayload
	t.DefaultChecker.HeightVersionCheck = t.heightVersionCheck
	t.DefaultChecker.IsAllowedInPOWConsensus = t.IsAllowedInPOWConsensus
	t.DefaultChecker.SpecialContextCheck = t.SpecialContextCheck
	t.DefaultChecker.CheckAttributeProgram = t.CheckAttributeProgram
}

func (t *ReturnDepositCoinTransaction) CheckAttributeProgram(params *TransactionParameters) error {

	if params.BlockHeight >= params.Config.CRVotingStartHeight {
		if len(t.Programs()) != 1 {
			return errors.New("return deposit coin transactions should have one and only one program")
		}
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

func (t *ReturnDepositCoinTransaction) CheckTransactionPayload(params *TransactionParameters) error {
	switch t.Payload().(type) {
	case *payload.ReturnDepositCoin:
		return nil
	}

	return errors.New("invalid payload type")
}

func (t *ReturnDepositCoinTransaction) IsAllowedInPOWConsensus(params *TransactionParameters, references map[*common2.Input]common2.Output) bool {
	return false
}

func (t *ReturnDepositCoinTransaction) SpecialContextCheck(params *TransactionParameters, references map[*common2.Input]common2.Output) (elaerr.ELAError, bool) {
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

	var availableAmount common.Fixed64
	for _, program := range t.Programs() {
		p := params.BlockChain.GetState().GetProducer(program.Code[1 : len(program.Code)-1])
		if p == nil {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("signer must be producer")), true
		}
		availableAmount += p.AvailableAmount()
	}

	if inputValue-changeValue > availableAmount ||
		outputValue >= availableAmount {
		return elaerr.Simple(elaerr.ErrTxPayload, fmt.Errorf("overspend deposit")), true
	}

	return nil, false
}
