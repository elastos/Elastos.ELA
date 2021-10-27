// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package indexers

import (
	"errors"
	"fmt"
	"github.com/elastos/Elastos.ELA/core/types/transactions"
	"testing"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/log"
	"github.com/elastos/Elastos.ELA/core/types"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/outputpayload"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/database"
	"github.com/elastos/Elastos.ELA/utils/test"

	"github.com/stretchr/testify/assert"
)

var (
	referHeight        = uint32(100)
	referRecipient1, _ = common.Uint168FromAddress("EJMzC16Eorq9CuFCGtyMrq4Jmgw9jYCHQR")
	referRecipient2, _ = common.Uint168FromAddress("EKnWs1jyNdhzH65UST8qMo8ZpQrTpXGnLH")

	// refer tx hash: 160da301e49617c037ae9b630919af52b8ac458202cd64558af7e0dcc753e307
	testUtxoIndexReferTx = &transactions.BaseTransaction{
		Version:        common2.TxVersion09,
		TxType:         common2.TransferAsset,
		PayloadVersion: 0,
		Payload:        &payload.TransferAsset{},
		Attributes:     []*common2.Attribute{},
		Inputs: []*common2.Input{
			{
				Previous: common2.OutPoint{
					Index: 0,
					TxID:  common.EmptyHash,
				},
				Sequence: 0,
			},
		},
		Outputs: []*common2.Output{
			{
				Value:       0,
				Type:        common2.OTNone,
				Payload:     &outputpayload.DefaultOutput{},
				ProgramHash: *referRecipient1,
			},
			{
				Value:       100,
				Type:        common2.OTNone,
				Payload:     &outputpayload.DefaultOutput{},
				ProgramHash: *referRecipient1,
			},
			{
				Value:       200,
				Type:        common2.OTNone,
				Payload:     &outputpayload.DefaultOutput{},
				ProgramHash: *referRecipient2,
			},
			{
				Value:       300,
				Type:        common2.OTNone,
				Payload:     &outputpayload.DefaultOutput{},
				ProgramHash: *referRecipient2,
			},
		},
		LockTime: 5,
	}
	recipient1, _    = common.Uint168FromAddress("EQr9qjiXGF2y7YMtDCHtHNewZynakbDzF7")
	testUtxoIndexTx1 = &transactions.BaseTransaction{
		TxType:  common2.CoinBase,
		Payload: new(payload.CoinBase),
		Inputs:  nil,
		Outputs: []*common2.Output{
			{
				Value:       10,
				ProgramHash: *recipient1,
			},
			{
				Value:       20,
				ProgramHash: *recipient1,
			},
		},
	}
	recipient2, _    = common.Uint168FromAddress("EWQfnxDhXQ4vHjncuAG5si2zpbKR79CjLp")
	testUtxoIndexTx2 = &transactions.BaseTransaction{
		TxType:         common2.TransferAsset,
		PayloadVersion: 0,
		Payload:        &payload.TransferAsset{},
		Inputs: []*common2.Input{
			{
				Previous: common2.OutPoint{
					Index: 0,
					TxID:  testUtxoIndexReferTx.Hash(),
				},
				Sequence: 0,
			},
			{
				Previous: common2.OutPoint{
					Index: 1,
					TxID:  testUtxoIndexReferTx.Hash(),
				},
				Sequence: 0,
			},
			{
				Previous: common2.OutPoint{
					Index: 2,
					TxID:  testUtxoIndexReferTx.Hash(),
				},
				Sequence: 0,
			},
		},
		Outputs: []*common2.Output{
			{
				Value:       30,
				ProgramHash: *recipient1,
			},
			{
				Value:       40,
				ProgramHash: *recipient2,
			},
			{
				Value:       0,
				ProgramHash: *recipient2,
			},
		},
	}
	testUtxoIndexBlock = &types.Block{
		Header: types.Header{
			Height: 200,
		},
		Transactions: []*transactions.BaseTransaction{
			testUtxoIndexTx1,
			testUtxoIndexTx2,
		},
	}

	testUtxoIndex *UtxoIndex
	utxoIndexDB   database.DB
	testTxStore   *TestTxStore
)

type TestTxStore struct {
	transactions map[common.Uint256]*transactions.BaseTransaction
	heights      map[common.Uint256]uint32
}

func (s *TestTxStore) FetchTx(txID common.Uint256) (*transactions.BaseTransaction,
	uint32, error) {
	txn, exist := s.transactions[txID]
	if exist {
		return txn, s.heights[txID], nil
	}
	return nil, 0, errors.New("leveldb: not found")
}

func (s *TestTxStore) SetTx(txn *transactions.BaseTransaction, height uint32) {
	s.transactions[txn.Hash()] = txn
	s.heights[txn.Hash()] = height
}

func (s *TestTxStore) RemoveTx(txID common.Uint256) {
	delete(s.transactions, txID)
}

func NewTestTxStore() *TestTxStore {
	var db TestTxStore
	db.transactions = make(map[common.Uint256]*transactions.BaseTransaction)
	db.heights = make(map[common.Uint256]uint32)
	return &db
}

func TestUTXOIndexInit(t *testing.T) {
	log.NewDefault(test.NodeLogPath, 0, 0, 0)

	var err error
	utxoIndexDB, err = LoadBlockDB(test.DataPath)
	assert.NoError(t, err)
	testTxStore = NewTestTxStore()
	testTxStore.SetTx(testUtxoIndexReferTx, referHeight)
	fmt.Println("refer tx hash:", testUtxoIndexReferTx.Hash().String())

	testUtxoIndex = NewUtxoIndex(utxoIndexDB, testTxStore)
	assert.NotEqual(t, nil, testUtxoIndex)
	assert.Equal(t, []byte("utxobyhashidx"), testUtxoIndex.Key())
	assert.Equal(t, "utxo index", testUtxoIndex.Name())
	_ = utxoIndexDB.Update(func(dbTx database.Tx) error {
		err := testUtxoIndex.Create(dbTx)
		assert.NoError(t, err)

		// initialize test utxo
		utxos1 := make([]*common2.UTXO, 0)
		utxos2 := make([]*common2.UTXO, 0)
		for i, output := range testUtxoIndexReferTx.Outputs {
			if output.Value == 0 {
				continue
			}
			if output.ProgramHash.IsEqual(*referRecipient1) {
				utxos1 = append(utxos1, &common2.UTXO{
					TxID:  testUtxoIndexReferTx.Hash(),
					Index: uint16(i),
					Value: output.Value,
				})
			}
			if output.ProgramHash.IsEqual(*referRecipient2) {
				utxos2 = append(utxos2, &common2.UTXO{
					TxID:  testUtxoIndexReferTx.Hash(),
					Index: uint16(i),
					Value: output.Value,
				})
			}
		}
		err = dbPutUtxoIndexEntry(dbTx, referRecipient1, referHeight, utxos1)
		assert.NoError(t, err)
		err = dbPutUtxoIndexEntry(dbTx, referRecipient2, referHeight, utxos2)
		assert.NoError(t, err)

		// check the initialization
		utxos1, err = dbFetchUtxoIndexEntryByHeight(dbTx, referRecipient1, 200)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(utxos1))

		utxos1, err = dbFetchUtxoIndexEntryByHeight(dbTx, referRecipient1, referHeight)
		assert.NoError(t, err)
		assert.Equal(t, []*common2.UTXO{
			{
				TxID:  testUtxoIndexReferTx.Hash(),
				Index: 1,
				Value: 100,
			},
		}, utxos1)
		utxos2, err = dbFetchUtxoIndexEntryByHeight(dbTx, referRecipient2, referHeight)
		assert.NoError(t, err)
		assert.Equal(t, []*common2.UTXO{
			{
				TxID:  testUtxoIndexReferTx.Hash(),
				Index: 2,
				Value: 200,
			},
			{
				TxID:  testUtxoIndexReferTx.Hash(),
				Index: 3,
				Value: 300,
			},
		}, utxos2)

		return nil
	})
}

func TestUtxoIndex_ConnectBlock(t *testing.T) {
	_ = utxoIndexDB.Update(func(dbTx database.Tx) error {
		err := testUtxoIndex.ConnectBlock(dbTx, testUtxoIndexBlock)
		assert.NoError(t, err)
		return nil
	})

	_ = utxoIndexDB.View(func(dbTx database.Tx) error {
		// input items should be removed from db
		inputUtxo1, err := dbFetchUtxoIndexEntry(dbTx, referRecipient1)
		assert.NoError(t, err)
		assert.Equal(t, []*common2.UTXO{}, inputUtxo1)
		inputUtxo2, err := dbFetchUtxoIndexEntry(dbTx, referRecipient2)
		assert.NoError(t, err)
		assert.Equal(t, []*common2.UTXO{
			{
				TxID:  testUtxoIndexReferTx.Hash(),
				Index: 3,
				Value: 300,
			},
		}, inputUtxo2)

		// output items should be added in db
		outputUtxo1, err := dbFetchUtxoIndexEntry(dbTx, recipient1)
		assert.NoError(t, err)
		assert.Equal(t, []*common2.UTXO{
			{
				TxID:  testUtxoIndexTx1.Hash(),
				Index: 0,
				Value: 10,
			},
			{
				TxID:  testUtxoIndexTx1.Hash(),
				Index: 1,
				Value: 20,
			},
			{
				TxID:  testUtxoIndexTx2.Hash(),
				Index: 0,
				Value: 30,
			},
		}, outputUtxo1)
		outputUtxo2, err := dbFetchUtxoIndexEntry(dbTx, recipient2)
		assert.NoError(t, err)
		assert.Equal(t, []*common2.UTXO{
			{
				TxID:  testUtxoIndexTx2.Hash(),
				Index: 1,
				Value: 40,
			},
		}, outputUtxo2)

		return nil
	})
}

func TestUtxoIndex_DisconnectBlock(t *testing.T) {
	_ = utxoIndexDB.Update(func(dbTx database.Tx) error {
		err := testUtxoIndex.DisconnectBlock(dbTx, testUtxoIndexBlock)
		assert.NoError(t, err)

		// input items should be added to db
		utxos1, err := dbFetchUtxoIndexEntry(dbTx, referRecipient1)
		assert.NoError(t, err)
		assert.Equal(t, []*common2.UTXO{
			{
				TxID:  testUtxoIndexReferTx.Hash(),
				Index: 1,
				Value: 100,
			},
		}, utxos1)
		utxos2, err := dbFetchUtxoIndexEntry(dbTx, referRecipient2)
		assert.NoError(t, err)
		assert.Equal(t, []*common2.UTXO{
			{
				TxID:  testUtxoIndexReferTx.Hash(),
				Index: 3,
				Value: 300,
			},
			{
				TxID:  testUtxoIndexReferTx.Hash(),
				Index: 2,
				Value: 200,
			},
		}, utxos2)

		// output items should be removed from db
		outputUtxo1, err := dbFetchUtxoIndexEntry(dbTx, recipient1)
		assert.NoError(t, err)
		assert.Equal(t, []*common2.UTXO{}, outputUtxo1)
		outputUtxo2, err := dbFetchUtxoIndexEntry(dbTx, recipient2)
		assert.NoError(t, err)
		assert.Equal(t, []*common2.UTXO{}, outputUtxo2)

		return nil
	})
}

func TestUtxoIndexEnd(t *testing.T) {
	_ = utxoIndexDB.Update(func(dbTx database.Tx) error {
		meta := dbTx.Metadata()
		err := meta.DeleteBucket(utxoIndexKey)
		assert.NoError(t, err)
		return nil
	})
	utxoIndexDB.Close()
}
