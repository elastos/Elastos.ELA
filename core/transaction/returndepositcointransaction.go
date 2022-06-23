// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"errors"
	"fmt"

	"github.com/elastos/Elastos.ELA/common"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type ReturnDepositCoinTransaction struct {
	BaseTransaction
}

func (t *ReturnDepositCoinTransaction) CheckAttributeProgram() error {

	if t.parameters.BlockHeight >= t.parameters.Config.CRVotingStartHeight {
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

func (t *ReturnDepositCoinTransaction) CheckTransactionPayload() error {
	switch t.Payload().(type) {
	case *payload.ReturnDepositCoin:
		return nil
	}

	return errors.New("invalid payload type")
}

func (t *ReturnDepositCoinTransaction) IsAllowedInPOWConsensus() bool {
	return true
}

func (t *ReturnDepositCoinTransaction) SpecialContextCheck() (elaerr.ELAError, bool) {
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
	state := t.parameters.BlockChain.GetState()
	var availableAmount common.Fixed64
	for _, program := range t.Programs() {
		p := state.GetProducer(program.Code[1 : len(program.Code)-1])
		if p == nil {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("signer must be producer")), true
		}
		if t.parameters.BlockHeight >= state.DPoSV2ActiveHeight && p.Info().StakeUntil != 0 {
			availableAmount += p.GetDPoSV2AvailableAmount(t.parameters.BlockHeight)
		} else {
			availableAmount += p.AvailableAmount()
		}
	}

	if inputValue-changeValue > availableAmount ||
		outputValue >= availableAmount {
		return elaerr.Simple(elaerr.ErrTxPayload, fmt.Errorf("overspend deposit")), true
	}

	return nil, false
}
