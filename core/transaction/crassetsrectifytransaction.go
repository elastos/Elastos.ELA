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

type CRAssetsRectifyTransaction struct {
	BaseTransaction
}

func (t *CRAssetsRectifyTransaction) RegisterFunctions() {
	t.DefaultChecker.CheckTransactionSize = t.checkTransactionSize
	t.DefaultChecker.CheckTransactionInput = t.checkTransactionInput
	t.DefaultChecker.CheckTransactionOutput = t.checkTransactionOutput
	t.DefaultChecker.CheckTransactionPayload = t.CheckTransactionPayload
	t.DefaultChecker.CheckAttributeProgram = t.CheckAttributeProgram
	t.DefaultChecker.HeightVersionCheck = t.HeightVersionCheck
	t.DefaultChecker.IsAllowedInPOWConsensus = t.IsAllowedInPOWConsensus
	t.DefaultChecker.SpecialContextCheck = t.SpecialContextCheck
}

func (t *CRAssetsRectifyTransaction) CheckAttributeProgram(params *TransactionParameters) error {
	if len(t.Programs()) != 0 {
		return errors.New("txs should have no programs")
	}
	if len(t.Attributes()) != 0 {
		return errors.New("txs should have no attributes")
	}
	return nil
}

func (t *CRAssetsRectifyTransaction) CheckTransactionPayload(params *TransactionParameters) error {
	switch t.Payload().(type) {
	case *payload.CRAssetsRectify:
		return nil
	}

	return errors.New("invalid payload type")
}

func (t *CRAssetsRectifyTransaction) IsAllowedInPOWConsensus(params *TransactionParameters, references map[*common2.Input]common2.Output) bool {
	return true
}

func (t *CRAssetsRectifyTransaction) HeightVersionCheck(params *TransactionParameters) error {
	txn := params.Transaction
	blockHeight := params.BlockHeight
	chainParams := params.Config

	if blockHeight < chainParams.CRAssetsRectifyTransactionHeight {
		return errors.New(fmt.Sprintf("not support %s transaction "+
			"before CRCProposalWithdrawPayloadV1Height", txn.TxType().Name()))
	}
	return nil
}

func (t *CRAssetsRectifyTransaction) SpecialContextCheck(params *TransactionParameters, references map[*common2.Input]common2.Output) (result elaerr.ELAError, end bool) {
	// Inputs count should be less than or equal to MaxCRAssetsAddressUTXOCount
	if len(t.Inputs()) > int(params.Config.MaxCRAssetsAddressUTXOCount) {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("inputs count should be less than or "+
			"equal to MaxCRAssetsAddressUTXOCount")), true
	}

	// Inputs count should be greater than or equal to MinCRAssetsAddressUTXOCount
	if len(t.Inputs()) < int(params.Config.MinCRAssetsAddressUTXOCount) {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("inputs count should be greater than or "+
			"equal to MinCRAssetsAddressUTXOCount")), true
	}

	// Inputs need to only from CR assets address
	var totalInput common.Fixed64
	for _, output := range t.references {
		totalInput += output.Value
		if !output.ProgramHash.IsEqual(params.Config.CRAssetsAddress) {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("input does not from CRAssetsAddress")), true
		}
	}

	// Outputs count should be only one
	if len(t.Outputs()) != 1 {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("outputs count should be only one")), true
	}

	// common2.Output should translate to CR assets address only
	if !t.Outputs()[0].ProgramHash.IsEqual(params.Config.CRAssetsAddress) {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("output does not to CRAssetsAddress")), true
	}

	// Inputs amount need equal to outputs amount
	totalOutput := t.Outputs()[0].Value
	if totalInput != totalOutput+params.Config.RectifyTxFee {
		return elaerr.Simple(elaerr.ErrTxPayload, fmt.Errorf("inputs minus outputs does not match with %d sela fee , "+
			"inputs:%s outputs:%s", params.Config.RectifyTxFee, totalInput, totalOutput)), true
	}

	return nil, false
}
