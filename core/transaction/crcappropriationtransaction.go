// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"errors"
	"fmt"
	"math"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type CRCAppropriationTransaction struct {
	BaseTransaction
}

func (t *CRCAppropriationTransaction) CheckTransactionOutput() error {
	blockHeight := t.parameters.BlockHeight
	chainParams := t.parameters.Config
	if len(t.Outputs()) > math.MaxUint16 {
		return errors.New("output count should not be greater than 65535(MaxUint16)")
	}

	if len(t.Outputs()) != 2 {
		return errors.New("new CRCAppropriation tx must have two output")
	}
	if !t.Outputs()[0].ProgramHash.IsEqual(chainParams.CRExpensesAddress) {
		return errors.New("new CRCAppropriation tx must have the first" +
			"output to CR expenses address")
	}
	if !t.Outputs()[1].ProgramHash.IsEqual(chainParams.CRAssetsAddress) {
		return errors.New("new CRCAppropriation tx must have the second" +
			"output to CR assets address")
	}

	// check if output address is valid
	specialOutputCount := 0
	for _, output := range t.Outputs() {
		if output.AssetID != config.ELAAssetID {
			return errors.New("asset ID in output is invalid")
		}

		// output value must >= 0
		if output.Value < common.Fixed64(0) {
			return errors.New("invalid transaction UTXO output")
		}

		if err := checkOutputProgramHash(blockHeight, output.ProgramHash); err != nil {
			return err
		}

		if t.Version() >= common2.TxVersion09 {
			if output.Type != common2.OTNone {
				specialOutputCount++
			}
			if err := checkOutputPayload(output); err != nil {
				return err
			}
		}
	}

	if t.parameters.BlockChain.GetHeight() >= chainParams.PublicDPOSHeight && specialOutputCount > 1 {
		return errors.New("special output count should less equal than 1")
	}

	return nil
}

func (t *CRCAppropriationTransaction) CheckAttributeProgram() error {
	if len(t.Programs()) != 0 {
		return errors.New("txs should have no programs")
	}
	if len(t.Attributes()) != 0 {
		return errors.New("txs should have no attributes")
	}
	return nil
}

func (t *CRCAppropriationTransaction) CheckTransactionPayload() error {
	switch t.Payload().(type) {
	case *payload.CRCAppropriation:
		return nil
	}

	return errors.New("invalid payload type")
}

func (t *CRCAppropriationTransaction) IsAllowedInPOWConsensus() bool {
	return true
}

func (t *CRCAppropriationTransaction) HeightVersionCheck() error {
	blockHeight := t.parameters.BlockHeight
	chainParams := t.parameters.Config

	if blockHeight < chainParams.CRCommitteeStartHeight {
		return errors.New(fmt.Sprintf("not support %s transaction "+
			"before CRCommitteeStartHeight", t.TxType().Name()))
	}
	return nil
}

func (t *CRCAppropriationTransaction) SpecialContextCheck() (result elaerr.ELAError, end bool) {
	// Check if current session has appropriated.
	if !t.parameters.BlockChain.GetCRCommittee().IsAppropriationNeeded() {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("should have no appropriation transaction")), true
	}

	// Inputs need to only from CR assets address
	var totalInput common.Fixed64
	for _, output := range t.references {
		totalInput += output.Value
		if !output.ProgramHash.IsEqual(t.parameters.Config.CRAssetsAddress) {
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
	appropriationAmount := t.parameters.BlockChain.GetCRCommittee().AppropriationAmount
	if appropriationAmount != t.Outputs()[0].Value {
		return elaerr.Simple(elaerr.ErrTxPayload, fmt.Errorf("invalid appropriation amount %s, need to be %s",
			t.Outputs()[0].Value, appropriationAmount)), true
	}

	return nil, true
}
