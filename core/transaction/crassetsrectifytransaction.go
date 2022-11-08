// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"errors"
	"fmt"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type CRAssetsRectifyTransaction struct {
	BaseTransaction
}

func (t *CRAssetsRectifyTransaction) CheckAttributeProgram() error {
	if len(t.Programs()) != 0 {
		return errors.New("txs should have no programs")
	}
	if len(t.Attributes()) != 0 {
		return errors.New("txs should have no attributes")
	}
	return nil
}

func (t *CRAssetsRectifyTransaction) CheckTransactionPayload() error {
	switch t.Payload().(type) {
	case *payload.CRAssetsRectify:
		return nil
	}

	return errors.New("invalid payload type")
}

func (t *CRAssetsRectifyTransaction) IsAllowedInPOWConsensus() bool {
	return true
}

func (t *CRAssetsRectifyTransaction) HeightVersionCheck() error {
	blockHeight := t.parameters.BlockHeight
	chainParams := t.parameters.Config

	if blockHeight < chainParams.CRConfiguration.CRAssetsRectifyTransactionHeight {
		return errors.New(fmt.Sprintf("not support %s transaction "+
			"before CRCProposalWithdrawPayloadV1Height", t.TxType().Name()))
	}
	return nil
}

func (t *CRAssetsRectifyTransaction) SpecialContextCheck() (result elaerr.ELAError, end bool) {
	// Inputs count should be less than or equal to MaxCRAssetsAddressUTXOCount
	if len(t.Inputs()) > int(t.parameters.Config.CRConfiguration.MaxCRAssetsAddressUTXOCount) {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("inputs count should be less than or "+
			"equal to MaxCRAssetsAddressUTXOCount")), true
	}

	// Inputs count should be greater than or equal to MinCRAssetsAddressUTXOCount
	if len(t.Inputs()) < int(t.parameters.Config.CRConfiguration.MinCRAssetsAddressUTXOCount) {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("inputs count should be greater than or "+
			"equal to MinCRAssetsAddressUTXOCount")), true
	}

	// Inputs need to only from CR assets address
	var totalInput common.Fixed64
	for _, output := range t.references {
		totalInput += output.Value
		if !output.ProgramHash.IsEqual(*t.parameters.Config.CRConfiguration.CRAssetsProgramHash) {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("input does not from CRAssetsProgramHash")), true
		}
	}

	// Outputs count should be only one
	if len(t.Outputs()) != 1 {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("outputs count should be only one")), true
	}

	// common2.Output should translate to CR assets address only
	if !t.Outputs()[0].ProgramHash.IsEqual(*t.parameters.Config.CRConfiguration.CRAssetsProgramHash) {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("output does not to CRAssetsProgramHash")), true
	}

	// Inputs amount need equal to outputs amount
	totalOutput := t.Outputs()[0].Value
	if totalInput != totalOutput+t.parameters.Config.CRConfiguration.RectifyTxFee {
		return elaerr.Simple(elaerr.ErrTxPayload, fmt.Errorf("inputs minus outputs does not match with %d sela fee , "+
			"inputs:%s outputs:%s", t.parameters.Config.CRConfiguration.RectifyTxFee, totalInput, totalOutput)), true
	}

	return nil, false
}
