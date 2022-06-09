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

type UnstakeRealWithdrawTransaction struct {
	BaseTransaction
}

func (t *UnstakeRealWithdrawTransaction) CheckAttributeProgram() error {
	if len(t.Programs()) != 0 {
		return errors.New("txs should have no programs")
	}
	if len(t.Attributes()) != 0 {
		return errors.New("txs should have no attributes")
	}
	return nil
}

func (t *UnstakeRealWithdrawTransaction) CheckTransactionPayload() error {
	switch t.Payload().(type) {
	case *payload.UnstakeRealWithdrawPayload:
		return nil
	}

	return errors.New("invalid payload type")
}

func (t *UnstakeRealWithdrawTransaction) IsAllowedInPOWConsensus() bool {
	return true
}

func (t *UnstakeRealWithdrawTransaction) HeightVersionCheck() error {
	blockHeight := t.parameters.BlockHeight
	chainParams := t.parameters.Config

	if blockHeight < chainParams.DPoSV2StartHeight {
		return errors.New(fmt.Sprintf("not support %s transaction "+
			"before DPoSV2StartHeight", t.TxType().Name()))
	}
	return nil
}

func (t *UnstakeRealWithdrawTransaction) SpecialContextCheck() (result elaerr.ELAError, end bool) {
	unstakeRealWithdraw, ok := t.Payload().(*payload.UnstakeRealWithdrawPayload)
	if !ok {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid payload")), true
	}
	txsCount := len(unstakeRealWithdraw.UnstakeRealWithdraw)
	// check UnstakeRealWithdraw count and output count
	if txsCount != len(t.Outputs()) && txsCount != len(t.Outputs())-1 {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid real unstake transaction outputs count")), true
	}

	// check other outputs, need to match with UnstakeRealWithdraw
	txs := t.parameters.BlockChain.GetState().GetVotesWithdrawableTxInfo()
	txsMap := make(map[common.Uint256]struct{})
	for i, realUnstake := range unstakeRealWithdraw.UnstakeRealWithdraw {
		txInfo, ok := txs[realUnstake.UnstakeTXHash]
		if !ok {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid unstake transaction hash")), true
		}
		output := t.Outputs()[i]
		if !output.ProgramHash.IsEqual(txInfo.Recipient) {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid real unstake output address")), true
		}
		if output.Value != txInfo.Amount-t.parameters.Config.RealWithdrawSingleFee {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New(fmt.Sprintf("invalid real unstake output "+
				"amount:%s, need to be:%s",
				output.Value, txInfo.Amount-t.parameters.Config.RealWithdrawSingleFee))), true
		}
		if _, ok := txsMap[realUnstake.UnstakeTXHash]; ok {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("duplicated real unstake transactions hash")), true
		}
		txsMap[realUnstake.UnstakeTXHash] = struct{}{}
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
	if inputAmount-outputAmount != t.parameters.Config.RealWithdrawSingleFee*common.Fixed64(txsCount) {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New(fmt.Sprintf("invalid real unstaketransaction"+
			" fee:%s, need to be:%s, txsCount:%d", inputAmount-outputAmount,
			t.parameters.Config.RealWithdrawSingleFee*common.Fixed64(txsCount), txsCount))), true
	}

	return nil, false
}
