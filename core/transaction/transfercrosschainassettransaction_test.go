// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/contract"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/core/types/outputpayload"
	"github.com/elastos/Elastos.ELA/core/types/payload"
)

func (s *txValidatorTestSuite) TestTransferCrossChainAssetTransaction() {
	{
		crosschain := &payload.TransferCrossChainAsset{}

		txn := functions.CreateTransaction(
			0,
			common2.TransferCrossChainAsset,
			0,
			crosschain,
			[]*common2.Attribute{},
			[]*common2.Input{},
			[]*common2.Output{},
			0,
			nil,
		)

		txn = CreateTransactionByType(txn, s.Chain)
		err, _ := txn.SpecialContextCheck()
		s.EqualError(err,
			"transaction validate error: payload content invalid:Invalid transaction payload content")

		txn.SetPayload(&payload.TransferCrossChainAsset{
			CrossChainAddresses: []string{"EJuQigHDPEMwhTJDsHNw4B95pXE2pLnMTX"},
			OutputIndexes:       []uint64{1},
			CrossChainAmounts:   []common.Fixed64{10 * 100000000},
		})
		txn.SetOutputs([]*common2.Output{
			{
				ProgramHash: *randomUint168(),
			},
		})
		err, _ = txn.SpecialContextCheck()
		s.EqualError(err,
			"transaction validate error: payload content invalid:Invalid transaction payload cross chain index")

		txn.SetPayload(&payload.TransferCrossChainAsset{
			CrossChainAddresses: []string{"EJuQigHDPEMwhTJDsHNw4B95pXE2pLnMTX"},
			OutputIndexes:       []uint64{0},
			CrossChainAmounts:   []common.Fixed64{10*100000000 - 10000},
		})
		err, _ = txn.SpecialContextCheck()
		s.EqualError(err,
			"transaction validate error: payload content invalid:Invalid transaction output address, without \"X\" at beginning")

		prefix := contract.PrefixCrossChain
		program := randomUint168().Bytes()
		program[0] = byte(prefix)
		p, _ := common.Uint168FromBytes(program)
		txn.SetOutputs([]*common2.Output{
			{
				ProgramHash: *p,
			},
		})

		err, _ = txn.SpecialContextCheck()
		s.EqualError(err,
			"transaction validate error: payload content invalid:Invalid transaction cross chain amount")

		txn.SetOutputs([]*common2.Output{
			{
				ProgramHash: *p,
				Value:       10 * 100000000,
			},
		})

		err, _ = txn.SpecialContextCheck()
		s.EqualError(err,
			"transaction validate error: payload content invalid:Invalid transaction fee")

		refer := make(map[*common2.Input]common2.Output)
		refer[&common2.Input{}] = common2.Output{
			Value: 10*100000000 + 10000,
		}
		txn.SetReferences(refer)
		err, _ = txn.SpecialContextCheck()
		s.NoError(err)

	}

	{
		crosschain := &payload.TransferCrossChainAsset{}

		txn := functions.CreateTransaction(
			9,
			common2.TransferCrossChainAsset,
			1,
			crosschain,
			[]*common2.Attribute{},
			[]*common2.Input{},
			[]*common2.Output{},
			0,
			nil,
		)

		txn = CreateTransactionByType(txn, s.Chain)
		err, _ := txn.SpecialContextCheck()
		s.EqualError(err,
			"transaction validate error: payload content invalid:invalid cross chain output count")

		txn.SetOutputs([]*common2.Output{
			{
				ProgramHash: *randomUint168(),
				Type:        common2.OTCrossChain,
			},
		})
		err, _ = txn.SpecialContextCheck()
		s.EqualError(err,
			"transaction validate error: payload content invalid:invalid transaction output address, without \"X\" at beginning")

		prefix := contract.PrefixCrossChain
		program := randomUint168().Bytes()
		program[0] = byte(prefix)
		p, _ := common.Uint168FromBytes(program)
		txn.SetOutputs([]*common2.Output{
			{
				ProgramHash: *p,
				Type:        common2.OTCrossChain,
			},
		})
		err, _ = txn.SpecialContextCheck()
		s.EqualError(err,
			"transaction validate error: payload content invalid:invalid cross chain output payload")

		txn.SetOutputs([]*common2.Output{
			{
				ProgramHash: *p,
				Type:        common2.OTCrossChain,
				Payload: &outputpayload.CrossChainOutput{
					TargetAddress: "EJuQigHDPEMwhTJDsHNw4B95pXE2pLnMTX",
				},
			},
		})
		err, _ = txn.SpecialContextCheck()
		s.EqualError(err,
			"transaction validate error: payload content invalid:invalid cross chain output amount")

		txn.SetOutputs([]*common2.Output{
			{
				ProgramHash: *p,
				Type:        common2.OTCrossChain,
				Payload: &outputpayload.CrossChainOutput{
					TargetAddress: "EJuQigHDPEMwhTJDsHNw4B95pXE2pLnMTX",
					TargetAmount:  10*100000000 - 10000,
				},
				Value: 10 * 100000000,
			},
		})
		err, _ = txn.SpecialContextCheck()
		s.NoError(err)
	}

}
