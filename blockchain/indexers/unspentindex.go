// Copyright (c) 2017-2022 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package indexers

import (
	"errors"
	"fmt"

	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/core/types"
	"github.com/elastos/Elastos.ELA/database"
)

const (
	// UnspentIndexName is the human-readable name for the index.
	UnspentIndexName = "unspent index"
)

var (
	// UnspentIndexKey is the key of the unspent index and the DB bucket used
	// to house it.
	UnspentIndexKey = []byte("unspentbyhashidx")

	// hashIndexBucketName is the name of the DB bucket used to house to the
	// block hash -> block height index.
	hashIndexBucketName = []byte("hashidx")
)

func toByteArray(source []uint16) []byte {
	dst := make([]byte, len(source)*2)
	for i := 0; i < len(source); i++ {
		dst[i*2] = byte(source[i] % 256)
		dst[i*2+1] = byte(source[i] / 256)
	}
	return dst
}

// DBPutUnspentIndexEntry uses an existing database transaction to update the
// index of unspent output.
func DBPutUnspentIndexEntry(dbTx database.Tx, txHash *common.Uint256, outputIndexes []uint16) error {
	unspentIndex := dbTx.Metadata().Bucket(UnspentIndexKey)
	return unspentIndex.Put(txHash[:], toByteArray(outputIndexes))
}

func getUint16Array(source []byte) ([]uint16, error) {
	if source == nil {
		return nil, errors.New("getUint16Array err, source = nil")
	}
	if len(source)%2 != 0 {
		return nil, errors.New("getUint16Array err, length of source is odd")
	}
	dst := make([]uint16, len(source)/2)
	for i := 0; i < len(source)/2; i++ {
		dst[i] = uint16(source[i*2]) + uint16(source[i*2+1])*256
	}
	return dst, nil
}

// DBFetchUnspentIndexEntry uses an existing database transaction to fetch its
// index of unspent output. When there is no entry for the provided hash, nil
// will be returned for the both the index and the error.
func DBFetchUnspentIndexEntry(dbTx database.Tx, txHash *common.Uint256) ([]uint16, error) {
	// Load the record from the database and return now if it doesn't exist.
	unspentIndex := dbTx.Metadata().Bucket(UnspentIndexKey)
	serializedData := unspentIndex.Get(txHash[:])
	if len(serializedData) == 0 {
		return nil, nil
	}

	return getUint16Array(serializedData)
}

// dbRemoveUnspentIndexEntry removes an unspent item by the given hash.
func dbRemoveUnspentIndexEntry(dbTx database.Tx, txHash *common.Uint256) error {
	unspentIndex := dbTx.Metadata().Bucket(UnspentIndexKey)
	serializedData := unspentIndex.Get(txHash[:])
	if len(serializedData) == 0 {
		return fmt.Errorf("can't remove non-existent unspent item by %s "+
			"from the unspent index", txHash)
	}

	return unspentIndex.Delete(txHash[:])
}

// UnspentIndex implements a unspent set by tx hash index. That is to say,
// it supports querying all unspent index by their tx hash.
type UnspentIndex struct {
	DB      database.DB
	TxCache *TxCache
}

// Init initializes the hash-based unspent index. This is part of the Indexer
// interface.
func (idx *UnspentIndex) Init() error {
	return nil // Nothing to do.
}

// Key returns the database key to use for the index as a byte slice.
//
// This is part of the Indexer interface.
func (idx *UnspentIndex) Key() []byte {
	return UnspentIndexKey
}

// Name returns the human-readable name of the index.
//
// This is part of the Indexer interface.
func (idx *UnspentIndex) Name() string {
	return UnspentIndexName
}

// Create is invoked when the indexer manager determines the index needs
// to be created for the first time.  It creates the buckets for the unspent
// index.
//
// This is part of the Indexer interface.
func (idx *UnspentIndex) Create(dbTx database.Tx) error {
	meta := dbTx.Metadata()
	_, err := meta.CreateBucket(UnspentIndexKey)
	return err
}

// ConnectBlock is invoked by the index manager when a new block has been
// connected to the main chain.  This indexer maintains a hash-to-unspent
// mapping for every transaction in the passed block.
//
// This is part of the Indexer interface.
func (idx *UnspentIndex) ConnectBlock(dbTx database.Tx, block *types.Block) error {
	unspents := make(map[common.Uint256][]uint16)
	// Trim the cache before connect block so the extra txns can be stored at least
	// one block.
	idx.TxCache.trim()

	for _, txn := range block.Transactions {
		if txn.TxType() == common2.RegisterAsset {
			continue
		}
		txnHash := txn.Hash()
		idx.TxCache.setTxn(block.Height, txn)
		for index := range txn.Outputs() {
			unspents[txnHash] = append(unspents[txnHash], uint16(index))
		}
		if !txn.IsCoinBaseTx() {
			for index, input := range txn.Inputs() {
				referTxnHash := input.Previous.TxID
				if _, ok := unspents[referTxnHash]; !ok {
					unspentValue, err := DBFetchUnspentIndexEntry(dbTx, &referTxnHash)
					if err != nil {
						return err
					}
					unspents[referTxnHash] = unspentValue
				}

				unspentLen := len(unspents[referTxnHash])
				for k, outputIndex := range unspents[referTxnHash] {
					if outputIndex == uint16(txn.Inputs()[index].Previous.Index) {
						unspents[referTxnHash][k] = unspents[referTxnHash][unspentLen-1]
						unspents[referTxnHash] = unspents[referTxnHash][:unspentLen-1]
						break
					}
				}
			}
		}
	}

	for txHash, value := range unspents {
		if len(value) == 0 {
			idx.TxCache.deleteTxn(txHash)
			err := dbRemoveUnspentIndexEntry(dbTx, &txHash)
			if err != nil {
				return err
			}
		} else {
			err := DBPutUnspentIndexEntry(dbTx, &txHash, value)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// DisconnectBlock is invoked by the index manager when a block has been
// disconnected from the main chain.  This indexer removes the
// hash-to-unspent mapping for every transaction in the block.
//
// This is part of the Indexer interface.
func (idx *UnspentIndex) DisconnectBlock(dbTx database.Tx, block *types.Block) error {
	unspents := make(map[common.Uint256][]uint16)
	for _, txn := range block.Transactions {
		if txn.TxType() == common2.RegisterAsset {
			continue
		}
		// remove all utxos created by this transaction
		txnHash := txn.Hash()
		idx.TxCache.deleteTxn(txnHash)
		if len(txn.Outputs()) != 0 {
			err := dbRemoveUnspentIndexEntry(dbTx, &txnHash)
			if err != nil {
				return err
			}
		}
		if !txn.IsCoinBaseTx() {
			for _, input := range txn.Inputs() {
				referTxnHash := input.Previous.TxID
				referTxnOutIndex := input.Previous.Index
				if _, ok := unspents[referTxnHash]; !ok {
					unspentValue, err := DBFetchUnspentIndexEntry(dbTx, &referTxnHash)
					if err != nil {
						return err
					}
					if len(unspentValue) != 0 {
						unspents[referTxnHash] = unspentValue
					}
				}
				unspents[referTxnHash] = append(unspents[referTxnHash], referTxnOutIndex)
			}
		}
	}

	for txHash, value := range unspents {
		if len(value) == 0 {
			idx.TxCache.deleteTxn(txHash)
			err := dbRemoveUnspentIndexEntry(dbTx, &txHash)
			if err != nil {
				return err
			}
		} else {
			err := DBPutUnspentIndexEntry(dbTx, &txHash, value)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (idx *UnspentIndex) FetchTx(txID common.Uint256) (interfaces.Transaction, uint32, error) {
	if txnInfo := idx.TxCache.GetTxn(txID); txnInfo != nil {
		return txnInfo.Txn, txnInfo.BlockHeight, nil
	}

	var txn interfaces.Transaction
	var height uint32
	err := idx.DB.View(func(dbTx database.Tx) error {
		var err error
		var blockHash *common.Uint256
		txn, blockHash, err = dbFetchTx(dbTx, &txID)
		if err != nil {
			return err
		}
		height, err = dbFetchHeightByHash(dbTx, blockHash)
		return err
	})
	if err != nil {
		return nil, 0, err
	}

	return txn, height, nil
}

// dbFetchHeightByHash uses an existing database transaction to retrieve the
// height for the provided hash from the index.
func dbFetchHeightByHash(dbTx database.Tx, hash *common.Uint256) (uint32, error) {
	meta := dbTx.Metadata()
	hashIndex := meta.Bucket(hashIndexBucketName)
	serializedHeight := hashIndex.Get(hash[:])
	if serializedHeight == nil {
		return 0, fmt.Errorf("block %s is not in the main chain", hash)
	}

	return byteOrder.Uint32(serializedHeight), nil
}

// NewUnspentIndex returns a new instance of an indexer that is used to create a
// mapping of the hashes of all transactions in the blockchain to the index of
// output which unspent in the transaction.
//
// It implements the Indexer interface which plugs into the IndexManager that in
// turn is used by the blockchain package.  This allows the index to be
// seamlessly maintained along with the chain.
func NewUnspentIndex(db database.DB, params *config.Params) *UnspentIndex {
	unspentIndex := &UnspentIndex{
		DB:      db,
		TxCache: NewTxCache(params),
	}
	//if params.NodeProfileStrategy !=
	//	config.MemoryFirst.String() {
	//	params.CkpManager.Register(NewCheckpoint(unspentIndex))
	//}
	return unspentIndex
}
