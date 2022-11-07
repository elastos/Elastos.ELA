// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"github.com/elastos/Elastos.ELA/core/contract/program"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/functions"
)

func (s *txValidatorTestSuite) TestCRAssetsRectifyTransaction() {
	txn := functions.CreateTransaction(
		common2.TxVersion09,
		common2.CRAssetsRectify,
		0,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)
	txn = CreateTransactionByType(txn, s.Chain)
	err, _ := txn.SpecialContextCheck()
	s.EqualError(err,
		"transaction validate error: payload content invalid:inputs count should be greater than or equal to MinCRAssetsAddressUTXOCount")

	{
		addr := s.Chain.GetParams().CRConfiguration.CRExpensesAddressUint168
		s.Chain.GetParams().CRConfiguration.MinCRAssetsAddressUTXOCount = 1
		reference := make(map[*common2.Input]common2.Output)
		input := &common2.Input{
			Previous: common2.OutPoint{
				TxID:  *randomUint256(),
				Index: 0,
			},
		}
		refOutput := common2.Output{
			Value:       20 * 1e8,
			ProgramHash: *addr,
		}
		reference[input] = refOutput

		txn.SetInputs([]*common2.Input{input})
		txn.SetOutputs([]*common2.Output{&refOutput})
		txn.SetReferences(reference)

		err, _ = txn.SpecialContextCheck()
		s.EqualError(err,
			"transaction validate error: payload content invalid:input does not from CRAssetsAddressUint168")
	}

	{
		addr := s.Chain.GetParams().CRConfiguration.CRAssetsAddressUint168
		dAddr := s.Chain.GetParams().DestroyELAAddressUint168
		s.Chain.GetParams().CRConfiguration.MinCRAssetsAddressUTXOCount = 1
		reference := make(map[*common2.Input]common2.Output)
		input := &common2.Input{
			Previous: common2.OutPoint{
				TxID:  *randomUint256(),
				Index: 0,
			},
		}
		refOutput := common2.Output{
			Value:       20 * 1e8,
			ProgramHash: *addr,
		}
		reference[input] = refOutput
		output1 := common2.Output{
			Value:       20 * 1e8,
			ProgramHash: *dAddr,
		}

		txn.SetInputs([]*common2.Input{input})
		txn.SetOutputs([]*common2.Output{&output1})
		txn.SetReferences(reference)

		err, _ = txn.SpecialContextCheck()
		s.EqualError(err,
			"transaction validate error: payload content invalid:output does not to CRAssetsAddressUint168")
	}

	{
		s.Chain.GetParams().CRConfiguration.MinCRAssetsAddressUTXOCount = 1
		s.Chain.GetParams().CRConfiguration.MaxCRAssetsAddressUTXOCount = 1
		reference := make(map[*common2.Input]common2.Output)
		input := &common2.Input{
			Previous: common2.OutPoint{
				TxID:  *randomUint256(),
				Index: 0,
			},
		}
		addr := s.Chain.GetParams().CRConfiguration.CRExpensesAddressUint168
		refOutput := common2.Output{
			Value:       20 * 1e8,
			ProgramHash: *addr,
		}
		reference[input] = refOutput

		txn.SetInputs([]*common2.Input{input, input})
		txn.SetOutputs([]*common2.Output{&refOutput})
		txn.SetReferences(reference)

		err, _ = txn.SpecialContextCheck()
		s.EqualError(err,
			"transaction validate error: payload content invalid:inputs count should be less than or equal to MaxCRAssetsAddressUTXOCount")
	}

	{
		addr := s.Chain.GetParams().CRConfiguration.CRAssetsAddressUint168
		s.Chain.GetParams().CRConfiguration.MinCRAssetsAddressUTXOCount = 1
		reference := make(map[*common2.Input]common2.Output)
		input := &common2.Input{
			Previous: common2.OutPoint{
				TxID:  *randomUint256(),
				Index: 0,
			},
		}
		refOutput := common2.Output{
			Value:       20 * 1e8,
			ProgramHash: *addr,
		}
		reference[input] = refOutput

		txn.SetInputs([]*common2.Input{input})
		txn.SetOutputs([]*common2.Output{&refOutput})
		txn.SetReferences(reference)

		err, _ = txn.SpecialContextCheck()
		s.EqualError(err,
			"transaction validate error: payload content invalid:inputs minus outputs does not match with 10000 sela fee , inputs:20 outputs:20")
	}

	{
		addr := s.Chain.GetParams().CRConfiguration.CRAssetsAddressUint168
		s.Chain.GetParams().CRConfiguration.MinCRAssetsAddressUTXOCount = 1
		reference := make(map[*common2.Input]common2.Output)
		input := &common2.Input{
			Previous: common2.OutPoint{
				TxID:  *randomUint256(),
				Index: 0,
			},
		}
		refOutput := common2.Output{
			Value:       20 * 1e8,
			ProgramHash: *addr,
		}
		reference[input] = refOutput

		// create outputs
		output1 := &common2.Output{
			Value:       20*1e8 - 10000,
			ProgramHash: *addr,
		}

		txn.SetInputs([]*common2.Input{input})
		txn.SetOutputs([]*common2.Output{output1, output1})
		txn.SetReferences(reference)

		err, _ = txn.SpecialContextCheck()
		s.EqualError(err, "transaction validate error: payload content invalid:outputs count should be only one")
	}

	{
		addr := s.Chain.GetParams().CRConfiguration.CRAssetsAddressUint168
		s.Chain.GetParams().CRConfiguration.MinCRAssetsAddressUTXOCount = 1
		reference := make(map[*common2.Input]common2.Output)
		input := &common2.Input{
			Previous: common2.OutPoint{
				TxID:  *randomUint256(),
				Index: 0,
			},
		}
		refOutput := common2.Output{
			Value:       20 * 1e8,
			ProgramHash: *addr,
		}
		reference[input] = refOutput

		// create outputs
		output1 := &common2.Output{
			Value:       20*1e8 - 10000,
			ProgramHash: *addr,
		}

		txn.SetInputs([]*common2.Input{input})
		txn.SetOutputs([]*common2.Output{output1})
		txn.SetReferences(reference)

		err, _ = txn.SpecialContextCheck()
		s.NoError(err)
	}

}
