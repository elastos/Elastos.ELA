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
	"github.com/elastos/Elastos.ELA/core/types/outputpayload"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type ReturnSideChainDepositCoinTransaction struct {
	BaseTransaction
}

func (t *ReturnSideChainDepositCoinTransaction) CheckTransactionOutput() error {
	blockHeight := t.parameters.BlockHeight
	if len(t.Outputs()) > math.MaxUint16 {
		return errors.New("output count should not be greater than 65535(MaxUint16)")
	}

	if len(t.Outputs()) < 1 {
		return errors.New("transaction has no outputs")
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
			if err := checkReturnSideChainDepositOutputPayload(output); err != nil {
				return err
			}
		}
	}

	return nil
}

func checkReturnSideChainDepositOutputPayload(output *common2.Output) error {
	switch output.Type {
	case common2.OTNone:
	case common2.OTReturnSideChainDepositCoin:
	default:
		return errors.New("transaction type dose not match the output payload type")
	}

	return output.Payload.Validate()
}

func (t *ReturnSideChainDepositCoinTransaction) CheckTransactionPayload() error {
	switch t.Payload().(type) {
	case *payload.ReturnSideChainDepositCoin:
		return nil
	}

	return errors.New("invalid payload type")
}

func (t *ReturnSideChainDepositCoinTransaction) IsAllowedInPOWConsensus() bool {
	return false
}

func (t *ReturnSideChainDepositCoinTransaction) HeightVersionCheck() error {
	blockHeight := t.parameters.BlockHeight
	chainParams := t.parameters.Config

	if blockHeight < chainParams.ReturnCrossChainCoinStartHeight {
		return errors.New(fmt.Sprintf("not support %s transaction "+
			"before ReturnCrossChainCoinStartHeight", t.TxType().Name()))
	}
	return nil
}

func (t *ReturnSideChainDepositCoinTransaction) SpecialContextCheck() (result elaerr.ELAError, end bool) {
	_, ok := t.Payload().(*payload.ReturnSideChainDepositCoin)
	if !ok {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid payload")), true
	}

	// check outputs
	fee := t.parameters.Config.ReturnDepositCoinFee
	for _, o := range t.Outputs() {
		if o.Type != common2.OTReturnSideChainDepositCoin {
			continue
		}
		py, ok := o.Payload.(*outputpayload.ReturnSideChainDeposit)
		if !ok {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid ReturnSideChainDeposit output payload")), true
		}

		tx, _, err := t.parameters.BlockChain.GetDB().GetTransaction(py.DepositTransactionHash)
		if err != nil {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid deposit tx:"+py.DepositTransactionHash.String())), true
		}
		refTx, _, err := t.parameters.BlockChain.GetDB().GetTransaction(tx.Inputs()[0].Previous.TxID)
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
