// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"errors"
	"github.com/elastos/Elastos.ELA/common"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/outputpayload"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type ReturnSideChainDepositCoinTransaction struct {
	BaseTransaction
}

func (t *ReturnSideChainDepositCoinTransaction) SpecialCheck() (result elaerr.ELAError, end bool) {
	_, ok := t.Payload().(*payload.ReturnSideChainDepositCoin)
	if !ok {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid payload")), true
	}

	// check outputs
	fee := t.contextParameters.Config.ReturnDepositCoinFee
	for _, o := range t.Outputs() {
		if o.Type != common2.OTReturnSideChainDepositCoin {
			continue
		}
		py, ok := o.Payload.(*outputpayload.ReturnSideChainDeposit)
		if !ok {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid ReturnSideChainDeposit output payload")), true
		}

		tx, _, err := t.contextParameters.BlockChain.GetDB().GetTransaction(py.DepositTransactionHash)
		if err != nil {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid deposit tx:" + py.DepositTransactionHash.String())), true
		}
		refTx, _, err := t.contextParameters.BlockChain.GetDB().GetTransaction(tx.Inputs()[0].Previous.TxID)
		if err != nil {
			return elaerr.Simple(elaerr.ErrTxPayload, err), true
		}

		// need to return the deposit coin to first input address
		refOutput := refTx.Outputs()[tx.Inputs()[0].Previous.Index]
		if o.ProgramHash != refOutput.ProgramHash {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid output address")), true
		}

		// side chain deposit address
		crossChainHash, err := common.Uint168FromAddress(py.GenesisBlockAddress)
		if err != nil {
			return elaerr.Simple(elaerr.ErrTxPayload, err), true
		}
		var crossChainAmount common.Fixed64
		switch tx.PayloadVersion() {
		case payload.TransferCrossChainVersion:
			p, ok := tx.Payload().(*payload.TransferCrossChainAsset)
			if !ok {
				log.Error("Invalid payload type need TransferCrossChainAsset")
				continue
			}

			for _, idx := range p.OutputIndexes {
				// output to current side chain
				if !crossChainHash.IsEqual(tx.Outputs()[idx].ProgramHash) {
					continue
				}
				crossChainAmount += tx.Outputs()[idx].Value
			}
		case payload.TransferCrossChainVersionV1:
			_, ok := tx.Payload().(*payload.TransferCrossChainAsset)
			if !ok {
				log.Error("Invalid payload type need TransferCrossChainAsset")
				continue
			}
			for _, o := range tx.Outputs() {
				if o.Type != common2.OTCrossChain {
					continue
				}
				// output to current side chain
				if !crossChainHash.IsEqual(o.ProgramHash) {
					continue
				}
				_, ok := o.Payload.(*outputpayload.CrossChainOutput)
				if !ok {
					continue
				}
				crossChainAmount += o.Value
			}
		}

		if o.Value+fee != crossChainAmount {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid output amount")), true
		}
	}

	return nil, false
}

