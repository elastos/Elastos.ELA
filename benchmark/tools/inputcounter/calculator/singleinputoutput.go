// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package calculator

import (
	"github.com/elastos/Elastos.ELA/account"
	"github.com/elastos/Elastos.ELA/benchmark/common/tx"
	"github.com/elastos/Elastos.ELA/common"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
)

type singleInputOutput struct {
	protoTx interfaces.Transaction
}

func (s *singleInputOutput) initialSize() uint64 {
	return 0
}

func (s *singleInputOutput) increase() uint64 {
	return uint64(s.protoTx.GetSize())
}

func newSingleInputOutput() (*singleInputOutput, error) {
	txn, err := createSingleInputOutputTx()
	return &singleInputOutput{
		protoTx: txn,
	}, err
}

func createSingleInputOutputTx() (interfaces.Transaction, error) {
	acc, err := account.NewAccount()
	if err != nil {
		return nil, err
	}

	generator := tx.NewGenerator(common2.TransferAsset, acc)
	txn := generator.Generate()

	utxo := common2.UTXO{
		TxID:  common.Uint256{},
		Index: 0,
		Value: 100000000,
	}
	assigner := tx.NewAssigner(tx.NoChanges, acc, &utxo)
	err = assigner.SignAndChange(txn)
	if err != nil {
		return nil, err
	}

	return txn, nil
}
