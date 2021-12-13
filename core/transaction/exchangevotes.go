// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"errors"
	"fmt"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/core/contract"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type ExchangeVotesTransaction struct {
	BaseTransaction
}

func (t *ExchangeVotesTransaction) HeightVersionCheck() error {
	blockHeight := t.parameters.BlockHeight
	chainParams := t.parameters.Config

	if blockHeight < chainParams.DposV2StartHeight {
		return errors.New(fmt.Sprintf("not support %s transaction "+
			"before DposV2StartHeight", t.TxType().Name()))
	}
	return nil
}

func (t *ExchangeVotesTransaction) CheckTransactionOutput() error {
	if len(t.Outputs()) > 2 {
		return errors.New("output count should not be greater than 2")
	}

	if len(t.Outputs()) < 1 {
		return errors.New("transaction has no outputs")
	}

	// check if output address is valid
	for _, output := range t.Outputs() {
		if output.AssetID != config.ELAAssetID {
			return errors.New("asset ID in output is invalid")
		}

		// output value must >= 0
		if output.Value < common.Fixed64(0) {
			return errors.New("invalid transaction UTXO output")
		}
	}

	if contract.GetPrefixType(t.Outputs()[0].ProgramHash) != contract.PrefixDposV2 {
		return errors.New("first output address need to be DPoSV2")
	}

	if len(t.Outputs()) == 2 {
		if contract.GetPrefixType(t.Outputs()[1].ProgramHash) != contract.PrefixStandard &&
			contract.GetPrefixType(t.Outputs()[1].ProgramHash) != contract.PrefixMultiSig {
			return errors.New("second output address need to be Standard or MultiSig")
		}
	}

	return nil
}

func (t *ExchangeVotesTransaction) CheckTransactionPayload() error {
	switch t.Payload().(type) {
	case *payload.ExchangeVotes:
		return nil
	}

	return errors.New("invalid payload type")
}

func (t *ExchangeVotesTransaction) CheckAttributeProgram() error {
	// Check attributes
	for _, attr := range t.Attributes() {
		if !common2.IsValidAttributeType(attr.Usage) {
			return fmt.Errorf("invalid attribute usage %v", attr.Usage)
		}
	}

	// Check programs
	if len(t.Programs()) != 1 {
		return errors.New("transaction should have only one program")
	}
	if t.Programs()[0].Code == nil {
		return fmt.Errorf("invalid program code nil")
	}
	if t.Programs()[0].Parameter == nil {
		return fmt.Errorf("invalid program parameter nil")
	}

	return nil
}

func (t *ExchangeVotesTransaction) IsAllowedInPOWConsensus() bool {

	return true
}

func (t *ExchangeVotesTransaction) SpecialContextCheck() (result elaerr.ELAError, end bool) {

	// 1.first output address need to be the stake address from code
	// 2.inputs should from one address only
	// 3.stake output value need to be equal to payload amount
	// outputs count has been checked in sanity check
	outputProgramHash := t.Outputs()[0].ProgramHash
	code := t.Programs()[0].Code
	ct, err := contract.CreateStakeContractByCode(code)
	if err != nil {
		return elaerr.Simple(elaerr.ErrTxInvalidOutput, err), true
	}
	stakeProgramHash := ct.ToProgramHash()
	if !stakeProgramHash.IsEqual(outputProgramHash) {
		return elaerr.Simple(elaerr.ErrTxInvalidOutput, errors.New("code not match output program hsh")), true
	}

	inputsAddr := make(map[common.Uint168]struct{})
	for _, o := range t.references {
		inputsAddr[o.ProgramHash] = struct{}{}
	}
	if len(inputsAddr) != 1 {
		return elaerr.Simple(elaerr.ErrTxInvalidOutput,
			errors.New("has different input address")), true
	}

	if t.Outputs()[0].Value != t.Payload().(*payload.ExchangeVotes).ExchangeValue {
		return elaerr.Simple(elaerr.ErrTxInvalidOutput,
			errors.New("payload value is not equal to output value")), true
	}

	return nil, false
}
