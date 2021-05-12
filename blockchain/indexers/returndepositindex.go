package indexers

import (
	"bytes"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/types"
	"github.com/elastos/Elastos.ELA/core/types/outputpayload"
	"github.com/elastos/Elastos.ELA/database"
)

const (
	// returnDepositIndexName is the human-readable name for the index.
	returnDepositIndexName = "return deposit index"
)

var (
	// returnDepositIndexKey is the key of the returnDeposit index and the db bucket used
	// to house it.
	returnDepositIndexKey = []byte("returnDeposithash")

	// returnDepositIndexValue is placeholder for returnDeposit index
	returnDepositIndexValue = []byte{1}
)

func dbFetchReturnDepositIndexEntry(dbTx database.Tx, txHash *common.Uint256) bool {
	hashIndex := dbTx.Metadata().Bucket(returnDepositIndexKey)
	value := hashIndex.Get(txHash[:])
	if bytes.Equal(value, returnDepositIndexValue) {
		return true
	}
	return false
}

func dbPutReturnDepositIndexEntry(dbTx database.Tx, txHash *common.Uint256) error {
	returnDepositIndex := dbTx.Metadata().Bucket(returnDepositIndexKey)
	return returnDepositIndex.Put(txHash[:], returnDepositIndexValue)
}

// dbRemoveTxIndexEntry uses an existing database transaction to remove the most
// recent returnDeposit entry for the given hash.
func dbRemoveReturnDepositIndexEntry(dbTx database.Tx, txHash *common.Uint256) error {
	returnDepositIndex := dbTx.Metadata().Bucket(returnDepositIndexKey)

	return returnDepositIndex.Delete(txHash[:])
}

// returnDepositIndex implements returnDeposit hash set which come from side chain.
type ReturnDepositIndex struct {
	db database.DB
}

// Init initializes the hash-based returnDeposit index. This is part of the Indexer
// interface.
func (idx *ReturnDepositIndex) Init() error {
	return nil // Nothing to do.
}

// Key returns the database key to use for the index as a byte slice.
//
// This is part of the Indexer interface.
func (idx *ReturnDepositIndex) Key() []byte {
	return returnDepositIndexKey
}

// Name returns the human-readable name of the index.
//
// This is part of the Indexer interface.
func (idx *ReturnDepositIndex) Name() string {
	return returnDepositIndexName
}

// Create is invoked when the indexer manager determines the index needs
// to be created for the first time.  It creates the buckets for the returnDeposit
// index.
//
// This is part of the Indexer interface.
func (idx *ReturnDepositIndex) Create(dbTx database.Tx) error {
	meta := dbTx.Metadata()
	_, err := meta.CreateBucket(returnDepositIndexKey)
	return err
}

// ConnectBlock is invoked by the index manager when a new block has been
// connected to the main chain.  This indexer maintains a returnDeposit hash
// mapping for every transaction in the passed block.
//
// This is part of the Indexer interface.
func (idx *ReturnDepositIndex) ConnectBlock(dbTx database.Tx, block *types.Block) error {
	for _, txn := range block.Transactions {
		if txn.TxType != types.ReturnSideChainDepositCoin {
			continue
		}
		for _, output := range txn.Outputs {
			if output.Type == types.OTReturnSideChainDepositCoin {
				payload, ok := output.Payload.(*outputpayload.ReturnSideChainDeposit)
				if ok {
					err := dbPutReturnDepositIndexEntry(dbTx, &payload.DepositTransactionHash)
					if err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

// DisconnectBlock is invoked by the index manager when a block has been
// disconnected from the main chain.  This indexer removes the
// returnDeposit hash mapping for every transaction in the block.
//
// This is part of the Indexer interface.
func (idx *ReturnDepositIndex) DisconnectBlock(dbTx database.Tx, block *types.Block) error {
	for _, txn := range block.Transactions {
		if txn.TxType != types.ReturnSideChainDepositCoin {
			continue
		}
		for _, output := range txn.Outputs {
			if output.Type == types.OTReturnSideChainDepositCoin {
				payload, ok := output.Payload.(*outputpayload.ReturnSideChainDeposit)
				if ok {
					err := dbRemoveReturnDepositIndexEntry(dbTx, &payload.DepositTransactionHash)
					if err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

// NewreturnDepositIndex returns a new instance of an indexer that is used to create a
// mapping of the program hashes of all addresses be used in the blockchain to
// the their utxo.
//
// It implements the Indexer interface which plugs into the IndexManager that in
// turn is used by the blockchain package.  This allows the index to be
// seamlessly maintained along with the chain.
func NewReturnDepositIndex(db database.DB) *ReturnDepositIndex {
	return &ReturnDepositIndex{db}
}
