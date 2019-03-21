package mempool

import (
	"errors"
	"sync"

	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/common/log"
	"github.com/elastos/Elastos.ELA/core/types"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/events"
)

type BlockPool struct {
	Chain     *blockchain.BlockChain
	Store     blockchain.IChainStore
	IsCurrent func() bool

	sync.RWMutex
	blocks      map[common.Uint256]*types.Block
	confirms    map[common.Uint256]*payload.Confirm
	chainParams *config.Params
}

func (bm *BlockPool) AppendConfirm(confirm *payload.Confirm) (bool,
	bool, error) {
	bm.Lock()
	defer bm.Unlock()

	return bm.appendConfirm(confirm)
}

func (bm *BlockPool) AddDposBlock(block *types.DPOSBlock) (bool, bool, error) {
	// main version >=H1
	if block.Height >= bm.chainParams.CRCOnlyDPOSHeight {
		return bm.AppendDposBlock(block)
	}

	// old version [0, H1)
	return bm.Chain.ProcessBlock(block.ToBlock(), &block.Confirm)
}

func (bm *BlockPool) AppendDposBlock(dposBlock *types.DPOSBlock) (bool, bool, error) {
	bm.Lock()
	defer bm.Unlock()
	if !dposBlock.HaveConfirm {
		return bm.appendBlock(dposBlock)
	}
	return bm.appendBlockAndConfirm(dposBlock)
}

func (bm *BlockPool) appendBlock(dposBlock *types.DPOSBlock) (bool, bool, error) {
	// add block
	block := dposBlock.ToBlock()
	hash := block.Hash()
	if _, ok := bm.blocks[hash]; ok {
		return false, false, errors.New("duplicate block in pool")
	}
	// verify block
	if err := bm.Chain.CheckBlockSanity(block); err != nil {
		return false, false, err
	}
	bm.blocks[hash] = block

	// confirm block
	inMainChain, isOrphan, err := bm.confirmBlock(hash)
	if err != nil {
		log.Debugf("[AppendDposBlock] ConfirmBlock %s failed, %s", hash, err)

		// Notify the caller that the new block without confirm was accepted.
		// The caller would typically want to react by relaying the inventory
		// to other peers.
		events.Notify(events.ETBlockAccepted, block)
		if block.Height == blockchain.DefaultLedger.Blockchain.GetHeight()+1 {
			events.Notify(events.ETNewBlockReceived, dposBlock)
		}
		return inMainChain, isOrphan, nil
	}

	copyBlock := types.NewDPOSBlock(block, bm.confirms[hash])

	// notify new block received
	events.Notify(events.ETNewBlockReceived, &copyBlock)

	return inMainChain, isOrphan, nil
}

func (bm *BlockPool) appendBlockAndConfirm(dposBlock *types.DPOSBlock) (bool, bool, error) {
	block := dposBlock.ToBlock()
	hash := block.Hash()
	// verify block
	if err := bm.Chain.CheckBlockSanity(block); err != nil {
		return false, false, err
	}
	// add block
	bm.blocks[block.Hash()] = block
	// confirm block
	inMainChain, isOrphan, err := bm.appendConfirm(&dposBlock.Confirm)
	if err != nil {
		log.Debug("[appendBlockAndConfirm] ConfirmBlock failed, hash:", hash.String(), "err: ", err)
		return inMainChain, isOrphan, nil
	}

	// notify new block received
	events.Notify(events.ETNewBlockReceived, dposBlock)

	return inMainChain, isOrphan, nil
}

func (bm *BlockPool) appendConfirm(confirm *payload.Confirm) (
	bool, bool, error) {

	// verify confirmation
	if err := blockchain.ConfirmSanityCheck(confirm); err != nil {
		return false, false, err
	}
	bm.confirms[confirm.Proposal.BlockHash] = confirm

	inMainChain, isOrphan, err := bm.confirmBlock(confirm.Proposal.BlockHash)
	if err != nil {
		return inMainChain, isOrphan, err
	}

	// notify new confirm accepted.
	events.Notify(events.ETConfirmAccepted, confirm)

	return inMainChain, isOrphan, nil
}

func (bm *BlockPool) ConfirmBlock(hash common.Uint256) (bool, bool, error) {
	bm.Lock()
	inMainChain, isOrphan, err := bm.confirmBlock(hash)
	bm.Unlock()
	return inMainChain, isOrphan, err
}

func (bm *BlockPool) confirmBlock(hash common.Uint256) (bool, bool, error) {
	log.Info("[ConfirmBlock] block hash:", hash)

	block, ok := bm.blocks[hash]
	if !ok {
		return false, false, errors.New("there is no block in pool when confirming block")
	}

	confirm, ok := bm.confirms[hash]
	if !ok {
		return false, false, errors.New("there is no block confirmation in pool when confirming block")
	}

	log.Info("[ConfirmBlock] block height:", block.Height)
	if !bm.Chain.BlockExists(&hash) {
		inMainChain, isOrphan, err := bm.Chain.ProcessBlock(block, confirm)
		if err != nil {
			return inMainChain, isOrphan, errors.New("add block failed," + err.Error())
		}

		if !inMainChain && !isOrphan {
			if err := bm.CheckConfirmedBlockOnFork(bm.Store.GetHeight(), block); err != nil {
				return inMainChain, isOrphan, err
			}
		}

		if isOrphan && !inMainChain {
			bm.Chain.AddOrphanConfirm(confirm)
		}

		if isOrphan || !inMainChain {
			return inMainChain, isOrphan, errors.New("add orphan block")
		}
	} else {
		return false, false, errors.New("already processed block")
	}

	return true, false, nil
}

func (bm *BlockPool) AddToBlockMap(block *types.Block) {
	bm.Lock()
	defer bm.Unlock()

	bm.blocks[block.Hash()] = block
}

func (bm *BlockPool) GetBlock(hash common.Uint256) (*types.Block, bool) {
	bm.RLock()
	defer bm.RUnlock()

	block, ok := bm.blocks[hash]
	return block, ok
}

func (bm *BlockPool) GetDposBlockByHash(hash common.Uint256) (*types.DPOSBlock, error) {
	bm.RLock()
	defer bm.RUnlock()

	if block := bm.blocks[hash]; block != nil {
		confirm := bm.confirms[hash]
		return types.NewDPOSBlock(block, confirm), nil
	}
	return nil, errors.New("not found dpos block")
}

func (bm *BlockPool) AddToConfirmMap(confirm *payload.Confirm) {
	bm.Lock()
	defer bm.Unlock()

	bm.confirms[confirm.Proposal.BlockHash] = confirm
}

func (bm *BlockPool) GetConfirm(hash common.Uint256) (
	*payload.Confirm, bool) {
	bm.Lock()
	defer bm.Unlock()

	confirm, ok := bm.confirms[hash]
	return confirm, ok
}

func NewBlockPool(params *config.Params) *BlockPool {
	return &BlockPool{
		blocks:      make(map[common.Uint256]*types.Block),
		confirms:    make(map[common.Uint256]*payload.Confirm),
		chainParams: params,
	}
}
