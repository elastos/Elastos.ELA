// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package mempool

import (
	"math/rand"
	"testing"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/contract/program"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/core/types/payload"
)

const (
	txCount = 40000
)

func BenchmarkTxFeeOrderedList_AddTx(b *testing.B) {
	protoTx := functions.CreateTransaction(
		common2.TxVersion09,
		common2.TransferAsset,
		0,
		&payload.TransferAsset{},
		[]*common2.Attribute{
			{
				Usage: common2.Nonce,
				Data:  randomNonceData(),
			},
		},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)

	txSize := protoTx.GetSize()
	orderedList := newTxFeeOrderedList(func(common.Uint256) {},
		uint64(txSize*txCount))

	for i := 0; i < txCount; i++ {
		tx := protoTx
		tx.SetAttributes([]*common2.Attribute{
			{
				Usage: common2.Nonce,
				Data:  randomNonceData(),
			},
		})
		tx.SetFee(common.Fixed64(rand.Int63n(1000)))
		orderedList.AddTx(tx)
	}
}

func BenchmarkTxFeeOrderedList_RemoveTx(b *testing.B) {

	protoTx := functions.CreateTransaction(
		common2.TxVersion09,
		common2.TransferAsset,
		0,
		&payload.TransferAsset{},
		[]*common2.Attribute{
			{
				Usage: common2.Nonce,
				Data:  randomNonceData(),
			},
		},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)

	txSize := protoTx.GetSize()
	orderedList := newTxFeeOrderedList(func(common.Uint256) {},
		uint64(txSize*txCount))

	hashMap := make(map[common.Uint256]float64)
	for i := 0; i < txCount; i++ {
		tx := protoTx
		tx.SetAttributes([]*common2.Attribute{
			{
				Usage: common2.Nonce,
				Data:  randomNonceData(),
			},
		})
		tx.SetFee(common.Fixed64(rand.Int63n(1000)))
		orderedList.AddTx(tx)

		hashMap[tx.Hash()] = float64(tx.Fee()) / float64(txSize)
	}

	b.ResetTimer()
	for k, v := range hashMap {
		orderedList.RemoveTx(k, uint64(txSize), v)
	}
	b.StopTimer()
}

func BenchmarkTxFeeOrderedList_EliminateTx(b *testing.B) {

	protoTx := functions.CreateTransaction(
		common2.TxVersion09,
		common2.TransferAsset,
		0,
		&payload.TransferAsset{},
		[]*common2.Attribute{
			{
				Usage: common2.Nonce,
				Data:  randomNonceData(),
			},
		},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)

	txSize := protoTx.GetSize()
	// size set 10000 means about 40000-30000 times eliminating action
	orderedList := newTxFeeOrderedList(func(common.Uint256) {},
		uint64(txSize*10000))

	for i := 0; i < txCount; i++ {
		tx := protoTx
		tx.SetAttributes([]*common2.Attribute{
			{
				Usage: common2.Nonce,
				Data:  randomNonceData(),
			},
		})
		tx.SetFee(common.Fixed64(rand.Int63n(1000)))
		orderedList.AddTx(tx)
	}
}
