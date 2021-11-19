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

type CRCProposalRealWithdrawTransaction struct {
	BaseTransaction
}

func (t *CRCProposalRealWithdrawTransaction) IsAllowedInPOWConsensus() bool {
	return true
}

func (t *CRCProposalRealWithdrawTransaction) HeightVersionCheck() error {
	txn := t.contextParameters.Transaction
	blockHeight := t.contextParameters.BlockHeight
	chainParams := t.contextParameters.Config

	if blockHeight < chainParams.CRAssetsRectifyTransactionHeight {
		return errors.New(fmt.Sprintf("not support %s transaction "+
			"before CRCProposalWithdrawPayloadV1Height", txn.TxType().Name()))
	}
	return nil
}

func (t *CRCProposalRealWithdrawTransaction) SpecialCheck() (result elaerr.ELAError, end bool) {
	crcRealWithdraw, ok := t.Payload().(*payload.CRCProposalRealWithdraw)
	if !ok {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid payload")), true
	}
	txsCount := len(crcRealWithdraw.WithdrawTransactionHashes)
	// check WithdrawTransactionHashes count and output count
	if txsCount != len(t.Outputs()) && txsCount != len(t.Outputs())-1 {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid real withdraw transaction hashes count")), true
	}

	// if need change, the last output is only allowed to CRExpensesAddress.
	if txsCount != len(t.Outputs()) {
		toProgramHash := t.Outputs()[len(t.Outputs())-1].ProgramHash
		if !toProgramHash.IsEqual(t.contextParameters.Config.CRExpensesAddress) {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New(fmt.Sprintf("last output is invalid"))), true
		}
	}

	// check other outputs, need to match with WithdrawTransactionHashes
	txs := t.contextParameters.BlockChain.GetCRCommittee().GetRealWithdrawTransactions()
	txsMap := make(map[common.Uint256]struct{})
	for i, hash := range crcRealWithdraw.WithdrawTransactionHashes {
		txInfo, ok := txs[hash]
		if !ok {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid withdraw transaction hash")), true
		}
		output := t.Outputs()[i]
		if !output.ProgramHash.IsEqual(txInfo.Recipient) {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid real withdraw output address")), true
		}
		if output.Value != txInfo.Amount-t.contextParameters.Config.RealWithdrawSingleFee {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New(fmt.Sprintf("invalid real withdraw output "+
				"amount:%s, need to be:%s",
				output.Value, txInfo.Amount-t.contextParameters.Config.RealWithdrawSingleFee))), true
		}
		if _, ok := txsMap[hash]; ok {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("duplicated real withdraw transactions hash")), true
		}
		txsMap[hash] = struct{}{}
	}

	// check transaction fee
	var inputAmount common.Fixed64
	for _, v := range t.references {
		inputAmount += v.Value
	}
	var outputAmount common.Fixed64
	for _, o := range t.Outputs() {
		outputAmount += o.Value
	}
	if inputAmount-outputAmount != t.contextParameters.Config.RealWithdrawSingleFee*common.Fixed64(txsCount) {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New(fmt.Sprintf("invalid real withdraw transaction"+
			" fee:%s, need to be:%s, txsCount:%d", inputAmount-outputAmount,
			t.contextParameters.Config.RealWithdrawSingleFee*common.Fixed64(txsCount), txsCount))), true
	}

	return nil, false
}
