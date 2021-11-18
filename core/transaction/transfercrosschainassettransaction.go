// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"bytes"
	"errors"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/contract"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/outputpayload"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type TransferCrossChainAssetTransaction struct {
	BaseTransaction
}

func (t *TransferCrossChainAssetTransaction) SpecialCheck() (elaerr.ELAError, bool) {
	var err error
	if t.PayloadVersion() > payload.TransferCrossChainVersionV1 {
		err = errors.New("invalid payload version")
	} else if t.PayloadVersion() == payload.TransferCrossChainVersionV1 {
		err = t.checkTransferCrossChainAssetTransactionV1()
	} else {
		err = t.checkTransferCrossChainAssetTransactionV0()
	}

	if err != nil {
		return elaerr.Simple(elaerr.ErrTxPayload, err), true
	}

	return nil, false
}

func (t *TransferCrossChainAssetTransaction) checkTransferCrossChainAssetTransactionV1() error {
	if t.Version() < common2.TxVersion09 {
		return errors.New("invalid transaction version")
	}

	var crossChainOutputCount uint32
	for _, output := range t.Outputs() {
		switch output.Type {
		case common2.OTNone:
		case common2.OTCrossChain:
			if t.contextParameters.BlockHeight >= t.contextParameters.Config.ProhibitTransferToDIDHeight {
				address, err := output.ProgramHash.ToAddress()
				if err != nil {
					return err
				}
				if address == t.contextParameters.Config.DIDSideChainAddress {
					return errors.New("no more DIDSideChain tx ")

				}
			}
			if bytes.Compare(output.ProgramHash[0:1], []byte{byte(contract.PrefixCrossChain)}) != 0 {
				return errors.New("invalid transaction output address, without \"X\" at beginning")
			}

			p, ok := output.Payload.(*outputpayload.CrossChainOutput)
			if !ok {
				return errors.New("invalid cross chain output payload")
			}

			if output.Value < t.contextParameters.Config.MinCrossChainTxFee+p.TargetAmount {
				return errors.New("invalid cross chain output amount")
			}

			crossChainOutputCount++
		default:
			return errors.New("invalid output type in cross chain transaction")
		}
	}
	if crossChainOutputCount == 0 {
		return errors.New("invalid cross chain output count")
	}

	return nil
}

func (t *TransferCrossChainAssetTransaction) checkTransferCrossChainAssetTransactionV0() error {
	payloadObj, ok := t.Payload().(*payload.TransferCrossChainAsset)
	if !ok {
		return errors.New("Invalid transfer cross chain asset payload type")
	}
	if len(payloadObj.CrossChainAddresses) == 0 ||
		len(payloadObj.CrossChainAddresses) > len(t.Outputs()) ||
		len(payloadObj.CrossChainAddresses) != len(payloadObj.CrossChainAmounts) ||
		len(payloadObj.CrossChainAmounts) != len(payloadObj.OutputIndexes) {
		return errors.New("Invalid transaction payload content")
	}

	//check cross chain output index in payload
	outputIndexMap := make(map[uint64]struct{})
	for _, outputIndex := range payloadObj.OutputIndexes {
		if _, exist := outputIndexMap[outputIndex]; exist || int(outputIndex) >= len(t.Outputs()) {
			return errors.New("Invalid transaction payload cross chain index")
		}
		outputIndexMap[outputIndex] = struct{}{}
	}

	//check address in outputs and payload
	csAddresses := make(map[string]struct{}, 0)
	for i := 0; i < len(payloadObj.CrossChainAddresses); i++ {
		if _, ok := csAddresses[payloadObj.CrossChainAddresses[i]]; ok {
			return errors.New("duplicated cross chain address in payload")
		}
		csAddresses[payloadObj.CrossChainAddresses[i]] = struct{}{}
		if bytes.Compare(t.Outputs()[payloadObj.OutputIndexes[i]].ProgramHash[0:1], []byte{byte(contract.PrefixCrossChain)}) != 0 {
			return errors.New("Invalid transaction output address, without \"X\" at beginning")
		}
		if payloadObj.CrossChainAddresses[i] == "" {
			return errors.New("Invalid transaction cross chain address ")
		}
	}

	//check cross chain amount in payload
	for i := 0; i < len(payloadObj.CrossChainAmounts); i++ {
		if payloadObj.CrossChainAmounts[i] < 0 || payloadObj.CrossChainAmounts[i] >
			t.Outputs()[payloadObj.OutputIndexes[i]].Value-t.contextParameters.Config.MinCrossChainTxFee {
			return errors.New("Invalid transaction cross chain amount")
		}
	}

	//check transaction fee
	var totalInput common.Fixed64
	for _, output := range t.references {
		totalInput += output.Value
	}

	var totalOutput common.Fixed64
	for _, output := range t.Outputs() {
		totalOutput += output.Value
	}

	if totalInput-totalOutput < t.contextParameters.Config.MinCrossChainTxFee {
		return errors.New("Invalid transaction fee")
	}
	return nil
}
