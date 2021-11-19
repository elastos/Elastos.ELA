// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"errors"
	"fmt"

	"github.com/elastos/Elastos.ELA/common"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type ReturnDepositCoinTransaction struct {
	BaseTransaction
}

func (t *ReturnDepositCoinTransaction) IsAllowedInPOWConsensus() bool {
	return false
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

	var availableAmount common.Fixed64
	for _, program := range t.Programs() {
		p := t.contextParameters.BlockChain.GetState().GetProducer(program.Code[1 : len(program.Code)-1])
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
