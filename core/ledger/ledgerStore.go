package ledger

import (
	. "github.com/elastos/Elastos.ELA.Utility/common"
	. "github.com/elastos/Elastos.ELA.Utility/core/asset"
	uti_tx "github.com/elastos/Elastos.ELA.Utility/core/transaction"
	tx "github.com/elastos/Elastos.ELA/core/transaction"
)

// ILedgerStore provides func with store package.
type ILedgerStore interface {
	//TODO: define the state store func

	SaveBlock(b *Block, ledger *Ledger) error
	GetBlock(hash Uint256) (*Block, error)
	BlockInCache(hash Uint256) bool

	GetBlockHash(height uint32) (Uint256, error)
	InitLedgerStore(ledger *Ledger) error
	IsDoubleSpend(tx *tx.NodeTransaction) bool

	//SaveHeader(header *Header,ledger *Ledger) error
	GetHeader(hash Uint256) (*Header, error)

	RollbackBlock(blockHash Uint256) error

	GetTransaction(hash Uint256) (*tx.NodeTransaction, uint32, error)

	PersistAsset(assetid Uint256, asset *Asset) error
	GetAsset(hash Uint256) (*Asset, error)

	GetCurrentBlockHash() Uint256
	GetHeight() uint32

	RemoveHeaderListElement(hash Uint256)

	InitLedgerStoreWithGenesisBlock(genesisblock *Block) (uint32, error)

	GetUnspent(txid Uint256, index uint16) (*uti_tx.TxOutput, error)
	ContainsUnspent(txid Uint256, index uint16) (bool, error)
	GetUnspentFromProgramHash(programHash Uint168, assetid Uint256) ([]*tx.UTXOUnspent, error)
	GetUnspentsFromProgramHash(programHash Uint168) (map[Uint256][]*tx.UTXOUnspent, error)
	GetAssets() map[Uint256]*Asset

	IsTxHashDuplicate(txhash Uint256) bool
	IsBlockInStore(hash Uint256) bool
	Close()
}
