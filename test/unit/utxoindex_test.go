// Copyright (c) 2017-2022 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package unit

import (
	"errors"
	"fmt"
	"testing"

	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/core/contract/program"
	"github.com/elastos/Elastos.ELA/core/transaction"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"

	"github.com/elastos/Elastos.ELA/blockchain/indexers"
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
	recipient1, _      = common.Uint168FromAddress("EQr9qjiXGF2y7YMtDCHtHNewZynakbDzF7")
	recipient2, _      = common.Uint168FromAddress("EWQfnxDhXQ4vHjncuAG5si2zpbKR79CjLp")

	// refer tx hash: 160da301e49617c037ae9b630919af52b8ac458202cd64558af7e0dcc753e307
	testUtxoIndexReferTx interfaces.Transaction
	testUtxoIndexTx1     interfaces.Transaction
	testUtxoIndexTx2     interfaces.Transaction
	testUtxoIndexBlock   *types.Block

	testUtxoIndex *indexers.UtxoIndex
	utxoIndexDB   database.DB
	testTxStore   *TestTxStore
)

func init() {
	functions.GetTransactionByTxType = transaction.GetTransaction
	functions.GetTransactionByBytes = transaction.GetTransactionByBytes
	functions.CreateTransaction = transaction.CreateTransaction
	functions.GetTransactionParameters = transaction.GetTransactionparameters
	config.DefaultParams = config.GetDefaultParams()

	testUtxoIndexReferTx = functions.CreateTransaction(
		common2.TxVersion09,
		common2.TransferAsset,
		0,
		&payload.TransferAsset{},
		[]*common2.Attribute{},
		[]*common2.Input{
			{
				Previous: common2.OutPoint{
					Index: 0,
					TxID:  common.EmptyHash,
				},
				Sequence: 0,
			},
		},
		[]*common2.Output{
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
		5,
		[]*program.Program{},
	)

	testUtxoIndexTx1 = functions.CreateTransaction(
		0,
		common2.CoinBase,
		0,
		new(payload.CoinBase),
		[]*common2.Attribute{},
		nil,
		[]*common2.Output{
			{
				Value:       10,
				ProgramHash: *recipient1,
			},
			{
				Value:       20,
				ProgramHash: *recipient1,
			},
		},
		0,
		[]*program.Program{},
	)

	testUtxoIndexTx2 = functions.CreateTransaction(
		0,
		common2.TransferAsset,
		0,
		&payload.TransferAsset{},
		[]*common2.Attribute{},
		[]*common2.Input{
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
		[]*common2.Output{
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
		0,
		[]*program.Program{},
	)

	testUtxoIndexBlock = &types.Block{
		Header: common2.Header{
			Height: 200,
		},
		Transactions: []interfaces.Transaction{
			testUtxoIndexTx1,
			testUtxoIndexTx2,
		},
	}
}

type TestTxStore struct {
	transactions map[common.Uint256]interfaces.Transaction
	heights      map[common.Uint256]uint32
}

func (s *TestTxStore) FetchTx(txID common.Uint256) (interfaces.Transaction,
	uint32, error) {
	Txn, exist := s.transactions[txID]
	if exist {
		return Txn, s.heights[txID], nil
	}
	return nil, 0, errors.New("leveldb: not found")
}

func (s *TestTxStore) SetTx(Txn interfaces.Transaction, height uint32) {
	s.transactions[Txn.Hash()] = Txn
	s.heights[Txn.Hash()] = height
}

func (s *TestTxStore) RemoveTx(txID common.Uint256) {
	delete(s.transactions, txID)
}

func NewTestTxStore() *TestTxStore {
	var DB TestTxStore
	DB.transactions = make(map[common.Uint256]interfaces.Transaction)
	DB.heights = make(map[common.Uint256]uint32)
	return &DB
}

func TestUTXOIndexInit(t *testing.T) {
	log.NewDefault(test.NodeLogPath, 0, 0, 0)

	var err error
	utxoIndexDB, err = LoadBlockDB(test.DataPath)
	assert.NoError(t, err)
	testTxStore = NewTestTxStore()
	testTxStore.SetTx(testUtxoIndexReferTx, referHeight)
	fmt.Println("refer tx hash:", testUtxoIndexReferTx.Hash().String())

	testUtxoIndex = indexers.NewUtxoIndex(utxoIndexDB, testTxStore)
	assert.NotEqual(t, nil, testUtxoIndex)
	assert.Equal(t, []byte("utxobyhashidx"), testUtxoIndex.Key())
	assert.Equal(t, "utxo index", testUtxoIndex.Name())
	_ = utxoIndexDB.Update(func(dbTx database.Tx) error {
		err := testUtxoIndex.Create(dbTx)
		assert.NoError(t, err)

		// initialize test utxo
		utxos1 := make([]*common2.UTXO, 0)
		utxos2 := make([]*common2.UTXO, 0)
		for i, output := range testUtxoIndexReferTx.Outputs() {
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
		err = indexers.DBPutUtxoIndexEntry(dbTx, referRecipient1, referHeight, utxos1)
		assert.NoError(t, err)
		err = indexers.DBPutUtxoIndexEntry(dbTx, referRecipient2, referHeight, utxos2)
		assert.NoError(t, err)

		// check the initialization
		utxos1, err = indexers.DBFetchUtxoIndexEntryByHeight(dbTx, referRecipient1, 200)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(utxos1))

		utxos1, err = indexers.DBFetchUtxoIndexEntryByHeight(dbTx, referRecipient1, referHeight)
		assert.NoError(t, err)
		assert.Equal(t, []*common2.UTXO{
			{
				TxID:  testUtxoIndexReferTx.Hash(),
				Index: 1,
				Value: 100,
			},
		}, utxos1)
		utxos2, err = indexers.DBFetchUtxoIndexEntryByHeight(dbTx, referRecipient2, referHeight)
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
		// input items should be removed from DB
		inputUtxo1, err := indexers.DBFetchUtxoIndexEntry(dbTx, referRecipient1)
		assert.NoError(t, err)
		assert.Equal(t, []*common2.UTXO{}, inputUtxo1)
		inputUtxo2, err := indexers.DBFetchUtxoIndexEntry(dbTx, referRecipient2)
		assert.NoError(t, err)
		assert.Equal(t, []*common2.UTXO{
			{
				TxID:  testUtxoIndexReferTx.Hash(),
				Index: 3,
				Value: 300,
			},
		}, inputUtxo2)

		// output items should be added in DB
		outputUtxo1, err := indexers.DBFetchUtxoIndexEntry(dbTx, recipient1)
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
		outputUtxo2, err := indexers.DBFetchUtxoIndexEntry(dbTx, recipient2)
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

		// input items should be added to DB
		utxos1, err := indexers.DBFetchUtxoIndexEntry(dbTx, referRecipient1)
		assert.NoError(t, err)
		assert.Equal(t, []*common2.UTXO{
			{
				TxID:  testUtxoIndexReferTx.Hash(),
				Index: 1,
				Value: 100,
			},
		}, utxos1)
		utxos2, err := indexers.DBFetchUtxoIndexEntry(dbTx, referRecipient2)
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

		// output items should be removed from DB
		outputUtxo1, err := indexers.DBFetchUtxoIndexEntry(dbTx, recipient1)
		assert.NoError(t, err)
		assert.Equal(t, []*common2.UTXO{}, outputUtxo1)
		outputUtxo2, err := indexers.DBFetchUtxoIndexEntry(dbTx, recipient2)
		assert.NoError(t, err)
		assert.Equal(t, []*common2.UTXO{}, outputUtxo2)

		return nil
	})
}

func TestUtxoIndexEnd(t *testing.T) {
	_ = utxoIndexDB.Update(func(dbTx database.Tx) error {
		meta := dbTx.Metadata()
		err := meta.DeleteBucket(indexers.UTXOIndexKey)
		assert.NoError(t, err)
		return nil
	})
	utxoIndexDB.Close()
}
