// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package mempool

import (
	"bytes"
	"math/rand"
	"testing"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/contract/program"
	transaction2 "github.com/elastos/Elastos.ELA/core/transaction"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/elanet/pact"
	"github.com/stretchr/testify/assert"
)

func init() {
	functions.GetTransactionByTxType = transaction2.GetTransaction
	functions.GetTransactionByBytes = transaction2.GetTransactionByBytes
	functions.CreateTransaction = transaction2.CreateTransaction
	functions.GetTransactionParameters = transaction2.GetTransactionparameters
}

func TestTxFeeOrderedList_AddTx(t *testing.T) {
	firedPopBack := false
	onPopBack := func(common.Uint256) {
		firedPopBack = true
	}

	protoTx := functions.CreateTransaction(
		0,
		common2.TransferAsset,
		0,
		&payload.TransferAsset{},
		[]*common2.Attribute{{
			Usage: common2.Nonce,
			Data:  randomNonceData(),
		}},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)
	protoTx.SetFee(100)
	protoTxSize := protoTx.GetSize()

	orderedList := newTxFeeOrderedList(onPopBack, uint64(protoTxSize*10))
	for i := 0; i < 10; i++ {
		tx := protoTx
		protoTx.SetFee(100)
		tx.SetFee(tx.Fee() - common.Fixed64(rand.Int63n(100)))
		tx.SetAttributes([]*common2.Attribute{
			{
				Usage: common2.Nonce,
				Data:  randomNonceData(),
			},
		})

		assert.NoError(t, orderedList.AddTx(tx))
		assert.False(t, firedPopBack)
	}
	assert.True(t, isListDescendingOrder(orderedList))
	assert.Equal(t, 10, orderedList.GetSize())

	protoTx.SetFee(0)
	assert.True(t, orderedList.OverSize(uint64(protoTx.GetSize())))
	err := orderedList.AddTx(protoTx)
	assert.True(t, err == addingTxExcluded)
	assert.False(t, firedPopBack)
	assert.True(t, isListDescendingOrder(orderedList))

	tx := protoTx
	tx.SetFee(1000)
	tx.SetAttributes([]*common2.Attribute{
		{
			Usage: common2.Nonce,
			Data:  randomNonceData(),
		},
	})
	assert.NoError(t, orderedList.AddTx(tx))
	assert.True(t, firedPopBack)
	assert.True(t, isListDescendingOrder(orderedList))

	tx = protoTx
	tx.SetFee(50) // set to the center
	tx.SetAttributes([]*common2.Attribute{
		{
			Usage: common2.Nonce,
			Data:  randomNonceData(),
		},
	})
	firedPopBack = false
	assert.NoError(t, orderedList.AddTx(tx))
	assert.True(t, firedPopBack)
	assert.True(t, isListDescendingOrder(orderedList))
	assert.Equal(t, 10, orderedList.GetSize())
}

func TestTxFeeOrderedList_RemoveTx(t *testing.T) {
	orderedList := newTxFeeOrderedList(func(common.Uint256) {},
		pact.MaxTxPoolSize)

	hashMap := make(map[common.Uint256]float64)
	var txSize int
	for i := 0; i < 100; i++ {
		protoTx := functions.CreateTransaction(
			0,
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
		txSize = protoTx.GetSize()
		tx := protoTx
		tx.SetAttributes([]*common2.Attribute{
			{
				Usage: common2.Nonce,
				Data:  randomNonceData(),
			},
		})
		tx.SetFee(common.Fixed64(rand.Int63n(1000) + 1))
		assert.NoError(t, orderedList.AddTx(tx))
		hashMap[tx.Hash()] = float64(tx.Fee()) / float64(txSize)

		assert.Equal(t, i+1, orderedList.GetSize())
	}

	i := 1
	for k, v := range hashMap {
		orderedList.RemoveTx(k, uint64(txSize), v)
		assert.Equal(t, 100-i, orderedList.GetSize())
		assert.True(t, isListDescendingOrder(orderedList))
		i++
	}
	assert.Equal(t, uint64(0), orderedList.totalSize)
}

func TestTxFeeOrderedList_Deserialize(t *testing.T) {
	list := txFeeOrderedList{
		totalSize: rand.Uint64(),
	}
	for i := 0; i < 5; i++ {
		list.list = append(list.list, *randomTxItem())
	}

	buf := bytes.Buffer{}
	assert.NoError(t, list.Serialize(&buf))

	var list2 txFeeOrderedList
	assert.NoError(t, list2.Deserialize(&buf))

	for i := 0; i < 5; i++ {
		assert.True(t, txItemsEqual(&list.list[i], &list2.list[i]))
	}
}

func TestTxItem_Deserialize(t *testing.T) {
	item := randomTxItem()
	buf := bytes.Buffer{}
	assert.NoError(t, item.Serialize(&buf))

	var item2 txItem
	assert.NoError(t, item2.Deserialize(&buf))

	assert.True(t, txItemsEqual(item, &item2))
}

func randomTxItem() *txItem {
	return &txItem{
		Hash:    *randomHash(),
		FeeRate: rand.Float64(),
		Size:    rand.Uint32(),
	}
}

func txItemsEqual(item, item2 *txItem) bool {
	return item.Hash.IsEqual(item2.Hash) && item.FeeRate == item2.FeeRate &&
		item.Size == item2.Size
}

func isListDescendingOrder(l *txFeeOrderedList) bool {
	for i := 0; i < len(l.list)-1; i++ {
		if l.list[i].FeeRate < l.list[i+1].FeeRate {
			return false
		}
	}
	return true
}

func randomNonceData() []byte {
	var data [20]byte
	rand.Read(data[:])
	return data[:]
}
