// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"bytes"
	"errors"
	"math"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/core/contract"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/outputpayload"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type TransferCrossChainAssetTransaction struct {
	BaseTransaction
}

func (t *TransferCrossChainAssetTransaction) CheckTransactionOutput() error {
	txn := t.sanityParameters.Transaction
	blockHeight := t.sanityParameters.BlockHeight
	if len(txn.Outputs()) > math.MaxUint16 {
		return errors.New("output count should not be greater than 65535(MaxUint16)")
	}

	if len(txn.Outputs()) < 1 {
		return errors.New("transaction has no outputs")
	}

	// check if output address is valid
	specialOutputCount := 0
	for _, output := range txn.Outputs() {
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

		if txn.Version() >= common2.TxVersion09 {
			if output.Type != common2.OTNone {
				specialOutputCount++
			}
			if err := checkTransferCrossChainAssetOutputPayload(output); err != nil {
				return err
			}
		}
	}

	return nil
}

func checkTransferCrossChainAssetOutputPayload(output *common2.Output) error {
	// common2.OTCrossChain information can only be placed in TransferCrossChainAsset transaction.
	switch output.Type {
	case common2.OTNone:
	case common2.OTCrossChain:
	default:
		return errors.New("transaction type dose not match the output payload type")
	}

	return output.Payload.Validate()
}

func (t *TransferCrossChainAssetTransaction) CheckTransactionPayload() error {
	switch t.Payload().(type) {
	case *payload.TransferCrossChainAsset:
		return nil
	}

	return errors.New("invalid payload type")
}

func (t *TransferCrossChainAssetTransaction) IsAllowedInPOWConsensus() bool {
	return false
}

func (t *TransferCrossChainAssetTransaction) HeightVersionCheck() error {
	txn := t.contextParameters.Transaction
	blockHeight := t.contextParameters.BlockHeight
	chainParams := t.contextParameters.Config

	if blockHeight <= chainParams.NewCrossChainStartHeight {
		if txn.PayloadVersion() != payload.TransferCrossChainVersion {
			return errors.New("not support " +
				"TransferCrossChainAsset payload version V1 before NewCrossChainStartHeight")
		}
		return nil
	} else {
		if txn.PayloadVersion() != payload.TransferCrossChainVersionV1 {
			return errors.New("not support " +
				"TransferCrossChainAsset payload version V0 after NewCrossChainStartHeight")
		}
	}
	return nil
}

func (t *TransferCrossChainAssetTransaction) SpecialContextCheck() (elaerr.ELAError, bool) {
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
