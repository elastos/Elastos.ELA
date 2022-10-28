// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/core/types/payload"
)

func (s *txValidatorTestSuite) TestVotesRealWithdrawTransaction() {

	tx1 := *randomUint256()
	txn := functions.CreateTransaction(
		0,
		common2.VotesRealWithdraw,
		0,
		&payload.VotesRealWithdrawPayload{
			VotesRealWithdraw: []payload.VotesRealWidhdraw{
				{
					ReturnVotesTXHash: tx1,
					StakeAddress:      *randomUint168(),
					Value:             10 * 100000000,
				},
			},
		},
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		nil,
	)

	txn = CreateTransactionByType(txn, s.Chain)
	err, _ := txn.SpecialContextCheck()
	s.EqualError(err,
		"transaction validate error: payload content invalid:invalid real votes withdraw transaction outputs count")

	receipt := *randomUint168()
	txn.SetOutputs([]*common2.Output{
		{
			ProgramHash: receipt,
		},
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err,
		"transaction validate error: payload content invalid:invalid return votes transaction hash")

	s.Chain.GetState().StateKeyFrame.VotesWithdrawableTxInfo[tx1] = common2.OutputInfo{}
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err,
		"transaction validate error: payload content invalid:invalid real votes withdraw output address")

	s.Chain.GetState().StateKeyFrame.VotesWithdrawableTxInfo[tx1] = common2.OutputInfo{
		Recipient: receipt,
	}
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err,
		"transaction validate error: payload content invalid:invalid real votes withdraw output amount:0, need to be:-0.00010000")

	s.Chain.GetState().StateKeyFrame.VotesWithdrawableTxInfo[tx1] = common2.OutputInfo{
		Recipient: receipt,
		Amount:    100*100000000 + 10000,
	}
	txn.SetOutputs([]*common2.Output{
		{
			ProgramHash: receipt,
			Value:       100 * 100000000,
		},
	})

	err, _ = txn.SpecialContextCheck()
	s.EqualError(err,
		"transaction validate error: payload content invalid:invalid real votes withdraw transaction fee:-100, need to be:0.00010000, txsCount:1")

	ref := make(map[*common2.Input]common2.Output)
	ref[&common2.Input{}] = common2.Output{
		Value: 100*100000000 + 10000,
	}
	txn.SetReferences(ref)
	err, _ = txn.SpecialContextCheck()
	s.NoError(err)

}
