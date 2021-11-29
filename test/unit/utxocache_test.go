// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package unit

import (
	"errors"
	"fmt"
	"testing"

	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/core/contract/program"
	"github.com/elastos/Elastos.ELA/core/transaction"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/outputpayload"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/utils/test"

	"github.com/stretchr/testify/assert"
)

var (
	utxoCacheDB *UtxoCacheDB
	utxoCache   *blockchain.UTXOCache

	// refer tx hash: 160da301e49617c037ae9b630919af52b8ac458202cd64558af7e0dcc753e307
	referTx interfaces.Transaction
	spendTx interfaces.Transaction
)

func init() {
	functions.GetTransactionByTxType = transaction.GetTransaction
	functions.GetTransactionByBytes = transaction.GetTransactionByBytes
	functions.CreateTransaction = transaction.CreateTransaction
	functions.GetTransactionParameters = transaction.GetTransactionparameters
	config.DefaultParams = config.GetDefaultParams()

	referTx = functions.CreateTransaction(
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
				Value:   100,
				Type:    common2.OTVote,
				Payload: &outputpayload.VoteOutput{},
			},
		},
		5,
		[]*program.Program{
			{
				Code:      randomPublicKey(),
				Parameter: randomSignature(),
			},
		},
	)

	spendTx = functions.CreateTransaction(
		0,
		0,
		0,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{
			{
				Previous: common2.OutPoint{
					Index: 0,
					TxID:  referTx.Hash(),
				},
				Sequence: 0,
			},
		},
		[]*common2.Output{},
		0,
		[]*program.Program{
			{
				Code:      randomPublicKey(),
				Parameter: randomSignature(),
			},
		},
	)
}

type UtxoCacheDB struct {
	transactions map[common.Uint256]interfaces.Transaction
}

func init() {
	testing.Init()
}

func (s *UtxoCacheDB) GetTransaction(txID common.Uint256) (
	interfaces.Transaction, uint32, error) {
	txn, exist := s.transactions[txID]
	if exist {
		return txn, 0, nil
	}
	return nil, 0, errors.New("leveldb: not found")
}

func (s *UtxoCacheDB) SetTransaction(txn interfaces.Transaction) {
	s.transactions[txn.Hash()] = txn
}

func (s *UtxoCacheDB) RemoveTransaction(txID common.Uint256) {
	delete(s.transactions, txID)
}

func NewUtxoCacheDB() *UtxoCacheDB {
	var db UtxoCacheDB
	db.transactions = make(map[common.Uint256]interfaces.Transaction)
	return &db
}

func TestUTXOCache_Init(t *testing.T) {
	utxoCacheDB = NewUtxoCacheDB()
	fmt.Println("refer tx hash:", referTx.Hash().String())
	utxoCacheDB.SetTransaction(referTx)
}

func TestUTXOCache_GetTxReferenceInfo(t *testing.T) {
	utxoCache = blockchain.NewUTXOCache(utxoCacheDB, &config.DefaultParams)

	// get tx reference form db and cache it first time.
	reference, err := utxoCache.GetTxReference(spendTx)
	assert.NoError(t, err)
	for input, output := range reference {
		assert.Equal(t, referTx.Hash(), input.Previous.TxID)
		assert.Equal(t, uint16(0), input.Previous.Index)
		assert.Equal(t, uint32(0), input.Sequence)

		assert.Equal(t, common.Fixed64(100), output.Value)
		assert.Equal(t, common2.OTVote, output.Type)
	}

	// ensure above reference have been cached.
	utxoCacheDB.RemoveTransaction(referTx.Hash())
	_, _, err = utxoCacheDB.GetTransaction(referTx.Hash())
	assert.Equal(t, "leveldb: not found", err.Error())

	reference, err = utxoCache.GetTxReference(spendTx)
	assert.NoError(t, err)
	for input, output := range reference {
		assert.Equal(t, referTx.Hash(), input.Previous.TxID)
		assert.Equal(t, uint16(0), input.Previous.Index)
		assert.Equal(t, uint32(0), input.Sequence)

		assert.Equal(t, common.Fixed64(100), output.Value)
		assert.Equal(t, common2.OTVote, output.Type)
	}
}

func TestUTXOCache_CleanSpent(t *testing.T) {
	utxoCache.CleanTxCache()
	_, err := utxoCache.GetTransaction(spendTx.Hash())
	assert.Equal(t, "transaction not found, leveldb: not found", err.Error())
}

func TestUTXOCache_CleanCache(t *testing.T) {
	utxoCacheDB.SetTransaction(referTx)

	reference, err := utxoCache.GetTxReference(spendTx)
	assert.NoError(t, err)
	for input, output := range reference {
		assert.Equal(t, referTx.Hash(), input.Previous.TxID)
		assert.Equal(t, uint16(0), input.Previous.Index)
		assert.Equal(t, uint32(0), input.Sequence)

		assert.Equal(t, common.Fixed64(100), output.Value)
		assert.Equal(t, common2.OTVote, output.Type)
	}

	utxoCacheDB.RemoveTransaction(referTx.Hash())
	_, _, err = utxoCacheDB.GetTransaction(referTx.Hash())
	assert.Equal(t, "leveldb: not found", err.Error())

	utxoCache.CleanCache()
	_, err = utxoCache.GetTxReference(spendTx)
	assert.Equal(t,
		"GetTxReference failed, transaction not found, leveldb: not found",
		err.Error())
}

// Test for case that a map use pointer as a key
func Test_PointerKeyForMap(t *testing.T) {
	test.SkipShort(t)
	i1 := common2.Input{
		Previous: common2.OutPoint{
			TxID:  common.EmptyHash,
			Index: 15,
		},
		Sequence: 10,
	}

	i2 := common2.Input{
		Previous: common2.OutPoint{
			TxID:  common.EmptyHash,
			Index: 15,
		},
		Sequence: 10,
	}
	// ensure i1 and i2 have the same data
	assert.Equal(t, i1, i2)

	// pointer as a key
	m1 := make(map[*common2.Input]int)
	m1[&i1] = 1
	m1[&i2] = 2
	assert.Equal(t, 2, len(m1))
	//fmt.Println(m1)
	// NOTE: &i1 and &i2 are different keys in m1
	// map[{TxID: 0000000000000000000000000000000000000000000000000000000000000000 Index: 15 Sequence: 10}:1 {TxID: 0000000000000000000000000000000000000000000000000000000000000000 Index: 15 Sequence: 10}:2]

	// object as a key
	m2 := make(map[common2.Input]int)
	m2[i1] = 1
	m2[i2] = 2
	assert.Equal(t, 1, len(m2))
	//fmt.Println(m2)
	// map[{TxID: 0000000000000000000000000000000000000000000000000000000000000000 Index: 15 Sequence: 10}:2]

	// pointer as a key
	m4 := make(map[*int]int)
	i3 := 0
	i4 := 0
	m4[&i3] = 3
	m4[&i4] = 4
	assert.Equal(t, 2, len(m4))
	//fmt.Println(m4)
	// map[0xc0000b43d8:3 0xc0000b4400:4]
}

func TestUTXOCache_InsertReference(t *testing.T) {
	// init reference
	for i := uint32(0); i < uint32(blockchain.MaxReferenceSize); i++ {
		input := &common2.Input{
			Sequence: i,
		}
		output := &common2.Output{
			OutputLock: i,
		}
		utxoCache.InsertReference(input, output)
	}
	assert.Equal(t, blockchain.MaxReferenceSize, len(utxoCache.Reference))
	assert.Equal(t, blockchain.MaxReferenceSize, utxoCache.Inputs.Len())
	assert.Equal(t, uint32(0), utxoCache.Inputs.Front().Value.(common2.Input).Sequence)
	assert.Equal(t, uint32(blockchain.MaxReferenceSize-1), utxoCache.Inputs.Back().Value.(common2.Input).Sequence)
	assert.Equal(t, uint32(0), utxoCache.Reference[utxoCache.Inputs.Front().Value.(common2.Input)].OutputLock)
	assert.Equal(t, uint32(blockchain.MaxReferenceSize-1), utxoCache.Reference[utxoCache.Inputs.Back().Value.(common2.Input)].OutputLock)

	for i := uint32(blockchain.MaxReferenceSize); i < uint32(blockchain.MaxReferenceSize+500); i++ {
		input := &common2.Input{
			Sequence: i,
		}
		output := &common2.Output{
			OutputLock: i,
		}
		utxoCache.InsertReference(input, output)
	}
	assert.Equal(t, blockchain.MaxReferenceSize, len(utxoCache.Reference))
	assert.Equal(t, blockchain.MaxReferenceSize, utxoCache.Inputs.Len())
	assert.Equal(t, uint32(500), utxoCache.Inputs.Front().Value.(common2.Input).Sequence)
	assert.Equal(t, uint32(blockchain.MaxReferenceSize+499), utxoCache.Inputs.Back().Value.(common2.Input).Sequence)
	assert.Equal(t, uint32(500), utxoCache.Reference[utxoCache.Inputs.Front().Value.(common2.Input)].OutputLock)
	assert.Equal(t, uint32(blockchain.MaxReferenceSize+499), utxoCache.Reference[utxoCache.Inputs.Back().Value.(common2.Input)].
		OutputLock)
}
