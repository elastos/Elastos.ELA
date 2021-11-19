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

type CRCAppropriationTransaction struct {
	BaseTransaction
}

func (t *CRCAppropriationTransaction) CheckTxHeightVersion() error {
	txn := t.contextParameters.Transaction
	blockHeight := t.contextParameters.BlockHeight
	chainParams := t.contextParameters.Config

	if blockHeight < chainParams.CRCommitteeStartHeight {
		return errors.New(fmt.Sprintf("not support %s transaction "+
			"before CRCommitteeStartHeight", txn.TxType().Name()))
	}
	return nil
}

func (t *CRCAppropriationTransaction) SpecialCheck() (result elaerr.ELAError, end bool) {
	// Check if current session has appropriated.
	if !t.contextParameters.BlockChain.GetCRCommittee().IsAppropriationNeeded() {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("should have no appropriation transaction")), true
	}

	// Inputs need to only from CR assets address
	var totalInput common.Fixed64
	for _, output := range t.references {
		totalInput += output.Value
		if !output.ProgramHash.IsEqual(t.contextParameters.Config.CRAssetsAddress) {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("input does not from CR assets address")), true
		}
	}

	// Inputs amount need equal to outputs amount
	var totalOutput common.Fixed64
	for _, output := range t.Outputs() {
		totalOutput += output.Value
	}
	if totalInput != totalOutput {
		return elaerr.Simple(elaerr.ErrTxPayload, fmt.Errorf("inputs does not equal to outputs amount, "+
			"inputs:%s outputs:%s", totalInput, totalOutput)), true
	}

	// Check output amount to CRExpensesAddress:
	// (CRAssetsAddress + CRExpensesAddress)*CRCAppropriatePercentage/100 -
	// CRExpensesAddress + CRCCommitteeUsedAmount
	//
	// Outputs has check in CheckTransactionOutput function:
	// first one to CRCommitteeAddress, second one to CRAssetsAddress
	appropriationAmount := t.contextParameters.BlockChain.GetCRCommittee().AppropriationAmount
	if appropriationAmount != t.Outputs()[0].Value {
		return elaerr.Simple(elaerr.ErrTxPayload, fmt.Errorf("invalid appropriation amount %s, need to be %s",
			t.Outputs()[0].Value, appropriationAmount)), true
	}

	return nil, true
}
