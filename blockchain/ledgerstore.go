package blockchain

import (
	"github.com/elastos/Elastos.ELA/common"
	. "github.com/elastos/Elastos.ELA/core/types"
	. "github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/protocol"
)

// IChainStoreDpos provides func for dpos
type IChainStoreDpos interface {
	GetRegisteredProducers() []*PayloadRegisterProducer
	GetActiveRegisteredProducers() []*PayloadRegisterProducer
	GetRegisteredProducersSorted() ([]*PayloadRegisterProducer, error)
	GetProducerVote(publicKey []byte) common.Fixed64
	GetProducerStatus(publicKey string) ProducerState

	GetIllegalProducers() map[string]struct{}
	GetCancelProducerHeight(publicKey []byte) (uint32, error)
}

// IChainStore provides func with store package.
type IChainStore interface {
	IChainStoreDpos
	protocol.TxnPoolListener

	InitWithGenesisBlock(genesisblock *Block) (uint32, error)
	InitProducerVotes() error

	SaveBlock(b *Block) error
	GetBlock(hash common.Uint256) (*Block, error)
	GetBlockHash(height uint32) (common.Uint256, error)
	IsDoubleSpend(tx *Transaction) bool

	SaveConfirm(confirm *DPosProposalVoteSlot) error
	GetConfirm(hash common.Uint256) (*DPosProposalVoteSlot, error)

	GetHeader(hash common.Uint256) (*Header, error)

	RollbackBlock(hash common.Uint256) error

	GetTransaction(txID common.Uint256) (*Transaction, uint32, error)
	GetTxReference(tx *Transaction) (map[*Input]*Output, error)

	PersistAsset(assetid common.Uint256, asset Asset) error
	GetAsset(hash common.Uint256) (*Asset, error)

	PersistSidechainTx(sidechainTxHash common.Uint256)
	GetSidechainTx(sidechainTxHash common.Uint256) (byte, error)

	GetCurrentBlockHash() common.Uint256
	GetHeight() uint32

	GetUnspent(txID common.Uint256, index uint16) (*Output, error)
	ContainsUnspent(txID common.Uint256, index uint16) (bool, error)
	GetUnspentFromProgramHash(programHash common.Uint168, assetid common.Uint256) ([]*UTXO, error)
	GetUnspentsFromProgramHash(programHash common.Uint168) (map[common.Uint256][]*UTXO, error)
	GetAssets() map[common.Uint256]*Asset

	IsTxHashDuplicate(txhash common.Uint256) bool
	IsSidechainTxHashDuplicate(sidechainTxHash common.Uint256) bool
	IsBlockInStore(hash common.Uint256) bool

	Close()
}
