// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package unit

import (
	"testing"

	"github.com/elastos/Elastos.ELA/blockchain/indexers"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/common/log"
	"github.com/elastos/Elastos.ELA/core/contract/program"
	"github.com/elastos/Elastos.ELA/core/transaction"
	"github.com/elastos/Elastos.ELA/core/types"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/database"
	_ "github.com/elastos/Elastos.ELA/database/ffldb"
	"github.com/elastos/Elastos.ELA/utils/test"

	"github.com/stretchr/testify/assert"
)

var (
	unspentIndexReferTx1    = common.Uint256{1}
	unspentIndexReferTx2    = common.Uint256{2}
	unspentIndexReferTx3    = common.Uint256{3}
	unspentIndexReferIndex1 = uint16(1)
	unspentIndexReferIndex2 = uint16(2)
	unspentIndexReferIndex3 = uint16(3)

	unspentIndexCoinbase interfaces.Transaction
	testUnspentIndexTx1  interfaces.Transaction
	testUnspentIndexTx2  interfaces.Transaction
	testUnspentIndexTx3  interfaces.Transaction
	unspentIndexBlock    *types.Block

	testUnspentIndexTx4   interfaces.Transaction
	testUnspentIndexTx5   interfaces.Transaction
	unspentIndexCoinbase2 interfaces.Transaction
	unspentIndexBlock2    *types.Block

	testUnspentIndex *indexers.UnspentIndex
	unspentIndexDB   database.DB
)

func init() {
	functions.GetTransactionByTxType = transaction.GetTransaction
	functions.GetTransactionByBytes = transaction.GetTransactionByBytes
	functions.CreateTransaction = transaction.CreateTransaction
	functions.GetTransactionParameters = transaction.GetTransactionparameters
	config.DefaultParams = *config.GetDefaultParams()

	unspentIndexCoinbase = functions.CreateTransaction(
		0,
		common2.CoinBase,
		0,
		new(payload.CoinBase),
		[]*common2.Attribute{},
		nil,
		[]*common2.Output{
			{
				Value: 10,
			},
			{
				Value: 20,
			},
		},
		0,
		[]*program.Program{},
	)

	testUnspentIndexTx1 = functions.CreateTransaction(
		0,
		common2.TransferAsset,
		0,
		new(payload.TransferAsset),
		[]*common2.Attribute{},
		[]*common2.Input{
			{
				Previous: common2.OutPoint{
					TxID:  unspentIndexReferTx1,
					Index: unspentIndexReferIndex1,
				},
			},
		},
		[]*common2.Output{
			{
				Value: 30,
			},
			{
				Value: 40,
			},
		},
		0,
		[]*program.Program{},
	)

	testUnspentIndexTx2 = functions.CreateTransaction(
		0,
		common2.TransferAsset,
		0,
		new(payload.TransferAsset),
		[]*common2.Attribute{},
		[]*common2.Input{
			{
				Previous: common2.OutPoint{
					TxID:  unspentIndexReferTx2,
					Index: unspentIndexReferIndex2,
				},
			},
		},
		[]*common2.Output{
			{
				Value: 50,
			},
			{
				Value: 60,
			},
		},
		0,
		[]*program.Program{},
	)

	testUnspentIndexTx3 = functions.CreateTransaction(
		0,
		common2.TransferAsset,
		payload.CRInfoDIDVersion,
		new(payload.TransferAsset),
		[]*common2.Attribute{},
		[]*common2.Input{
			{
				Previous: common2.OutPoint{
					TxID:  unspentIndexReferTx3,
					Index: unspentIndexReferIndex3,
				},
			},
		},
		[]*common2.Output{
			{
				Value: 0,
			},
			{
				Value: 50,
			},
		},
		0,
		[]*program.Program{},
	)

	unspentIndexBlock = &types.Block{
		Header: common2.Header{},
		Transactions: []interfaces.Transaction{
			unspentIndexCoinbase,
			testUnspentIndexTx1,
			testUnspentIndexTx2,
			testUnspentIndexTx3,
		},
	}

	testUnspentIndexTx4 = functions.CreateTransaction(
		0,
		common2.TransferAsset,
		payload.CRInfoDIDVersion,
		new(payload.TransferAsset),
		[]*common2.Attribute{},
		[]*common2.Input{
			{
				Previous: common2.OutPoint{
					TxID:  testUnspentIndexTx3.Hash(),
					Index: 0,
				},
			},
			{
				Previous: common2.OutPoint{
					TxID:  testUnspentIndexTx3.Hash(),
					Index: 1,
				},
			},
		},
		[]*common2.Output{
			{
				Value: 40,
			},
		},
		0,
		[]*program.Program{},
	)

	testUnspentIndexTx5 = functions.CreateTransaction(
		0,
		common2.TransferAsset,
		0,
		new(payload.TransferAsset),
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)

	unspentIndexCoinbase2 = functions.CreateTransaction(
		0,
		common2.CoinBase,
		0,
		new(payload.TransferAsset),
		[]*common2.Attribute{},
		nil,
		[]*common2.Output{
			{
				Value: 30,
			},
			{
				Value: 40,
			},
		},
		0,
		[]*program.Program{},
	)

	unspentIndexBlock2 = &types.Block{
		Header: common2.Header{},
		Transactions: []interfaces.Transaction{
			unspentIndexCoinbase2,
			testUnspentIndexTx4,
			testUnspentIndexTx5,
		},
	}
}

func TestUnspentIndexInit(t *testing.T) {
	log.NewDefault(test.NodeLogPath, 0, 0, 0)

	var err error
	unspentIndexDB, err = LoadBlockDB(test.DataPath)
	assert.NoError(t, err)
	testUnspentIndex = indexers.NewUnspentIndex(unspentIndexDB, config.GetDefaultParams())
	assert.NotEqual(t, nil, testUnspentIndex)
	assert.Equal(t, indexers.UnspentIndexKey, testUnspentIndex.Key())
	assert.Equal(t, indexers.UnspentIndexName, testUnspentIndex.Name())
	_ = unspentIndexDB.Update(func(dbTx database.Tx) error {
		err := testUnspentIndex.Create(dbTx)
		assert.NoError(t, err)

		// initialize test unspent
		err = indexers.DBPutUnspentIndexEntry(dbTx, &unspentIndexReferTx1, []uint16{unspentIndexReferIndex1})
		assert.NoError(t, err)
		err = indexers.DBPutUnspentIndexEntry(dbTx, &unspentIndexReferTx2, []uint16{unspentIndexReferIndex2})
		assert.NoError(t, err)
		err = indexers.DBPutUnspentIndexEntry(dbTx, &unspentIndexReferTx3, []uint16{unspentIndexReferIndex3})
		assert.NoError(t, err)
		//testUnspentIndex.txCache.setTxn(1, &types.BaseTransaction{
		//	LockTime: 10,
		//})
		//testUnspentIndex.txCache.setTxn(1, &types.BaseTransaction{
		//	LockTime: 20,
		//})

		// check the initialization
		indexes, err := indexers.DBFetchUnspentIndexEntry(dbTx, &unspentIndexReferTx1)
		assert.NoError(t, err)
		assert.Equal(t, []uint16{unspentIndexReferIndex1}, indexes)
		indexes, err = indexers.DBFetchUnspentIndexEntry(dbTx, &unspentIndexReferTx2)
		assert.NoError(t, err)
		assert.Equal(t, []uint16{unspentIndexReferIndex2}, indexes)
		return nil
	})
}

func TestUnspentIndex_ConnectBlock(t *testing.T) {
	_ = unspentIndexDB.Update(func(dbTx database.Tx) error {
		err := testUnspentIndex.ConnectBlock(dbTx, unspentIndexBlock)
		assert.NoError(t, err)
		err = testUnspentIndex.ConnectBlock(dbTx, unspentIndexBlock2)
		assert.NoError(t, err)

		// the unspent txn should be cached
		assert.Equal(t, &indexers.TxInfo{
			Txn:         unspentIndexCoinbase,
			BlockHeight: unspentIndexBlock.Height,
		}, testUnspentIndex.TxCache.GetTxn(unspentIndexCoinbase.Hash()))
		assert.Equal(t, &indexers.TxInfo{
			Txn:         testUnspentIndexTx1,
			BlockHeight: unspentIndexBlock.Height,
		}, testUnspentIndex.TxCache.GetTxn(testUnspentIndexTx1.Hash()))
		assert.Equal(t, &indexers.TxInfo{
			Txn:         testUnspentIndexTx2,
			BlockHeight: unspentIndexBlock.Height,
		}, testUnspentIndex.TxCache.GetTxn(testUnspentIndexTx2.Hash()))

		// the spent txn should be removed
		assert.Equal(t, (*indexers.TxInfo)(nil), testUnspentIndex.TxCache.GetTxn(unspentIndexReferTx1))
		assert.Equal(t, (*indexers.TxInfo)(nil), testUnspentIndex.TxCache.GetTxn(unspentIndexReferTx2))

		// input items should be removed from db
		indexes, err := indexers.DBFetchUnspentIndexEntry(dbTx, &unspentIndexReferTx1)
		assert.NoError(t, err)
		assert.Equal(t, []uint16(nil), indexes)
		indexes, err = indexers.DBFetchUnspentIndexEntry(dbTx, &unspentIndexReferTx2)
		assert.NoError(t, err)
		assert.Equal(t, []uint16(nil), indexes)

		// output items should be added in db
		coinbaseHash := unspentIndexCoinbase.Hash()
		indexes, err = indexers.DBFetchUnspentIndexEntry(dbTx, &coinbaseHash)
		assert.NoError(t, err)
		assert.Equal(t, []uint16{0, 1}, indexes)
		txHash1 := testUnspentIndexTx1.Hash()
		indexes, err = indexers.DBFetchUnspentIndexEntry(dbTx, &txHash1)
		assert.NoError(t, err)
		assert.Equal(t, []uint16{0, 1}, indexes)
		txHash2 := testUnspentIndexTx1.Hash()
		indexes, err = indexers.DBFetchUnspentIndexEntry(dbTx, &txHash2)
		assert.NoError(t, err)
		assert.Equal(t, []uint16{0, 1}, indexes)

		return nil
	})
}

func TestUnspentIndex_DisconnectBlock(t *testing.T) {
	_ = unspentIndexDB.Update(func(dbTx database.Tx) error {
		err := testUnspentIndex.DisconnectBlock(dbTx, unspentIndexBlock2)
		assert.NoError(t, err)

		err = testUnspentIndex.DisconnectBlock(dbTx, unspentIndexBlock)
		assert.NoError(t, err)

		// the spent txn should be removed
		assert.Equal(t, (*indexers.TxInfo)(nil), testUnspentIndex.TxCache.GetTxn(unspentIndexReferTx1))
		assert.Equal(t, (*indexers.TxInfo)(nil), testUnspentIndex.TxCache.GetTxn(unspentIndexReferTx2))

		// the refer txn should be cached
		//_, ok1 := testUnspentIndex.txns[unspentIndexReferTx1]
		//assert.True(t, ok1)
		//_, ok2 := testUnspentIndex.txns[unspentIndexReferTx2]
		//assert.True(t, ok2)

		// input items should be added in db
		indexes, err := indexers.DBFetchUnspentIndexEntry(dbTx, &unspentIndexReferTx1)
		assert.NoError(t, err)
		assert.Equal(t, []uint16{unspentIndexReferIndex1}, indexes)
		indexes, err = indexers.DBFetchUnspentIndexEntry(dbTx, &unspentIndexReferTx2)
		assert.NoError(t, err)
		assert.Equal(t, []uint16{unspentIndexReferIndex2}, indexes)

		// output items should be removed from db
		coinbaseHash := unspentIndexCoinbase.Hash()
		indexes, err = indexers.DBFetchUnspentIndexEntry(dbTx, &coinbaseHash)
		assert.NoError(t, err)
		assert.Equal(t, []uint16(nil), indexes)
		txHash1 := testUnspentIndexTx1.Hash()
		indexes, err = indexers.DBFetchUnspentIndexEntry(dbTx, &txHash1)
		assert.NoError(t, err)
		assert.Equal(t, []uint16(nil), indexes)
		txHash2 := testUnspentIndexTx1.Hash()
		indexes, err = indexers.DBFetchUnspentIndexEntry(dbTx, &txHash2)
		assert.NoError(t, err)
		assert.Equal(t, []uint16(nil), indexes)

		return nil
	})
}

func TestUnspentIndexEnd(t *testing.T) {
	_ = unspentIndexDB.Update(func(dbTx database.Tx) error {
		meta := dbTx.Metadata()
		err := meta.DeleteBucket(indexers.UnspentIndexKey)
		assert.NoError(t, err)
		return nil
	})
	unspentIndexDB.Close()
}
