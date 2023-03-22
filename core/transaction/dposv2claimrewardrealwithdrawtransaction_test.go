// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"math"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/core/types/payload"
)

func (s *txValidatorTestSuite) TestDposV2ClaimRewardRealWithdrawTransaction() {

	txid := *randomUint256()
	claimPayload := &payload.DposV2ClaimRewardRealWithdraw{
		WithdrawTransactionHashes: []common.Uint256{
			txid,
		},
	}

	txn := functions.CreateTransaction(
		0,
		common2.DposV2ClaimRewardRealWithdraw,
		payload.DposV2ClaimRewardVersionV0,
		claimPayload,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		nil,
	)

	txn = CreateTransactionByType(txn, s.Chain)
	err, _ := txn.SpecialContextCheck()
	s.EqualError(err,
		"transaction validate error: payload content invalid:invalid real withdraw transaction hashes count")

	receipt := *randomUint168()
	output1 := &common2.Output{
		Value:       20*1e8 - 10000,
		ProgramHash: receipt,
	}

	txn.SetOutputs([]*common2.Output{output1})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err,
		"transaction validate error: payload content invalid:invalid withdraw transaction hash")

	s.Chain.GetState().WithdrawableTxInfo[txid] = common2.OutputInfo{
		Recipient: *randomUint168(),
	}
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err,
		"transaction validate error: payload content invalid:invalid real withdraw output address")

	s.Chain.GetState().WithdrawableTxInfo[txid] = common2.OutputInfo{
		Recipient: receipt,
	}
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err,
		"transaction validate error: payload content invalid:invalid real withdraw output amount:19.99990000, need to be:-0.00010000")

	s.Chain.GetState().WithdrawableTxInfo[txid] = common2.OutputInfo{
		Recipient: receipt,
		Amount:    20 * 1e8,
	}
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err,
		"transaction validate error: payload content invalid:invalid real withdraw transaction fee:-19.99990000, need to be:0.00010000, txsCount:1")

	inputs := []*common2.Input{
		{
			Previous: common2.OutPoint{
				TxID:  common.EmptyHash,
				Index: 1,
			},
			Sequence: math.MaxUint32,
		},
	}
	outputs := []*common2.Output{
		{
			AssetID: core.ELAAssetID,
			Value:   common.Fixed64(20 * 1e8),
		},
	}

	references := make(map[*common2.Input]common2.Output)
	references[inputs[0]] = *outputs[0]
	txn.SetReferences(references)

	err, _ = txn.SpecialContextCheck()
	s.NoError(err)

}
