// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package blockchain

import (
	"time"

	"github.com/elastos/Elastos.ELA/blockchain/indexers"
	. "github.com/elastos/Elastos.ELA/common"
	. "github.com/elastos/Elastos.ELA/core/types"
	"github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/database"
)

// IChainStore provides func with store package.
type IChainStore interface {
	GetFFLDB() IFFLDBChainStore

	SaveBlock(b *Block, node *BlockNode, confirm *payload.Confirm,
		medianTimePast time.Time) error
	IsDoubleSpend(tx interfaces.Transaction) bool

	GetConfirm(hash Uint256) (*payload.Confirm, error)

	RollbackBlock(b *Block, node *BlockNode,
		confirm *payload.Confirm, medianTimePast time.Time) error

	GetTransaction(txID Uint256) (interfaces.Transaction, uint32, error)
	GetTxReference(tx interfaces.Transaction) (map[*common.Input]*common.Output, error)

	SetHeight(height uint32)
	GetHeight() uint32

	IsTxHashDuplicate(txhash Uint256) bool
	IsSidechainTxHashDuplicate(sidechainTxHash Uint256) bool
	IsSidechainReturnDepositTxHashDuplicate(sidechainReturnDepositTxHash Uint256) bool

	GetProposalDraftDataByDraftHash(draftHash *Uint256) ([]byte, error)

	SaveSmallCrossTransferTx(tx interfaces.Transaction) error
	GetSmallCrossTransferTxs() ([]interfaces.Transaction, error)
	GetSmallCrossTransferTx() ([]string, error)
	CleanSmallCrossTransferTx(txHash Uint256) error

	Close()
	CloseLeveldb()
}

// IChainStore provides func with store package.
type IFFLDBChainStore interface {
	database.DB

	// SaveBlock will write block into file DB.
	SaveBlock(b *Block, node *BlockNode, confirm *payload.Confirm,
		medianTimePast time.Time) error

	// RollbackBlock only remove block state and block index.
	RollbackBlock(b *Block, node *BlockNode,
		confirm *payload.Confirm, medianTimePast time.Time) error

	// Get block from file DB.
	GetBlock(hash Uint256) (*DposBlock, error)

	// Get block from file DB.
	GetOldBlock(hash Uint256) (*Block, error)

	// Get block header from file DB.
	GetHeader(hash Uint256) (*common.Header, error)

	// If already exist in main chain(exist in file DB and exist block index),
	// will return true.
	BlockExists(hash *Uint256) (bool, uint32, error)

	// If already exist in file DB (rollback will not remove from file DB), will
	// return true.
	IsBlockInStore(hash *Uint256) bool

	// Get a transaction by transaction hash.
	GetTransaction(txID Uint256) (interfaces.Transaction, uint32, error)

	// InitIndex use to initialize the index manager.
	InitIndex(chain indexers.IChain, interrupt <-chan struct{}) error

	// Get unspent by transaction hash.
	GetUnspent(txID Uint256) ([]uint16, error)

	// Get utxo by program hash.
	GetUTXO(programHash *Uint168) ([]*common.UTXO, error)

	// IsTx3Exist use to find if tx3 exist in DB.
	IsTx3Exist(txHash *Uint256) bool

	// IsSideChainReturnDepositExist use to find if return deposit exist in DB.
	IsSideChainReturnDepositExist(txHash *Uint256) bool

	// Get proposal draft data by draft hash.
	GetProposalDraftDataByDraftHash(draftHash *Uint256) ([]byte, error)
}
