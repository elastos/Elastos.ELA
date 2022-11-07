// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/core/contract/program"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/core/types/payload"
)

func (s *txValidatorTestSuite) TestCheckCRCProposalRealWithdrawTransaction() {
	// Set CR expenses address.
	var err error
	ceAddr := config.CRCExpensesAddressUint168
	ceExpensesAddress := ceAddr
	s.Chain.GetParams().CRConfiguration.CRExpensesAddressUint168 = ceAddr

	// Set WithdrawableTxInfo
	withdrawTransactionHash1 := *randomUint256()
	recipient1 := *randomUint168()
	withdrawTransactionHash2 := *randomUint256()
	recipient2 := *randomUint168()
	wtHashes := make(map[common.Uint256]common2.OutputInfo, 0)
	wtHashes[withdrawTransactionHash1] = common2.OutputInfo{
		Recipient: recipient1,
		Amount:    10 * 1e8,
	}
	wtHashes[withdrawTransactionHash2] = common2.OutputInfo{
		Recipient: recipient2,
		Amount:    9 * 1e8,
	}
	s.Chain.GetCRCommittee().GetProposalManager().WithdrawableTxInfo = wtHashes

	// Create reference.
	reference := make(map[*common2.Input]common2.Output)
	input := &common2.Input{
		Previous: common2.OutPoint{
			TxID:  *randomUint256(),
			Index: 0,
		},
	}
	refOutput := common2.Output{
		Value:       20 * 1e8,
		ProgramHash: *ceExpensesAddress,
	}
	reference[input] = refOutput

	// create outputs
	output1 := &common2.Output{
		Value:       10*1e8 - 10000,
		ProgramHash: recipient1,
	}
	output2 := &common2.Output{
		Value:       9*1e8 - 10000,
		ProgramHash: recipient2,
	}
	output3 := &common2.Output{
		Value:       1 * 1e8,
		ProgramHash: *ceExpensesAddress,
	}
	output1Err := &common2.Output{
		Value:       10 * 1e8,
		ProgramHash: recipient1,
	}
	output2Err := &common2.Output{
		Value:       9*1e8 - 10000,
		ProgramHash: recipient1,
	}
	output3Err := &common2.Output{
		Value:       1 * 1e8,
		ProgramHash: recipient1,
	}

	// check transaction
	txn := s.getCRCProposalRealWithdrawTx(input,
		[]common.Uint256{},
		[]*common2.Output{output1, output2})
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetReferences(reference)
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:invalid real withdraw transaction hashes count")

	txn = s.getCRCProposalRealWithdrawTx(input,
		[]common.Uint256{withdrawTransactionHash1, withdrawTransactionHash2},
		[]*common2.Output{output1Err, output2, output3})
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetReferences(reference)
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:invalid real withdraw output amount:10, need to be:9.99990000")

	txn = s.getCRCProposalRealWithdrawTx(input,
		[]common.Uint256{withdrawTransactionHash1, withdrawTransactionHash2},
		[]*common2.Output{output1, output2Err, output3})
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetReferences(reference)
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:invalid real withdraw output address")

	txn = s.getCRCProposalRealWithdrawTx(input,
		[]common.Uint256{withdrawTransactionHash1, withdrawTransactionHash2},
		[]*common2.Output{output1, output1, output3Err})
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetReferences(reference)
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:last output is invalid")

	txn = s.getCRCProposalRealWithdrawTx(input,
		[]common.Uint256{withdrawTransactionHash1, withdrawTransactionHash1},
		[]*common2.Output{output1, output1, output3})
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetReferences(reference)
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:duplicated real withdraw transactions hash")

	txn = s.getCRCProposalRealWithdrawTx(input,
		[]common.Uint256{withdrawTransactionHash1, withdrawTransactionHash2},
		[]*common2.Output{output1, output2, output3})
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetReferences(reference)
	err, _ = txn.SpecialContextCheck()
	s.NoError(err)
}

func (s *txValidatorTestSuite) getCRCProposalRealWithdrawTx(input *common2.Input,
	hashes []common.Uint256, outputs []*common2.Output) interfaces.Transaction {

	txn := functions.CreateTransaction(
		common2.TxVersion09,
		common2.CRCProposalRealWithdraw,
		0,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)

	cPayload := &payload.CRCProposalRealWithdraw{WithdrawTransactionHashes: hashes}
	txn.SetPayload(cPayload)
	txn.SetInputs([]*common2.Input{input})
	txn.SetOutputs(outputs)
	return txn
}
