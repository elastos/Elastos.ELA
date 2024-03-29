// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

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
	"github.com/elastos/Elastos.ELA/dpos/state"
	"github.com/elastos/Elastos.ELA/events"
)

const cachedCount = 6

type ConfirmInfo struct {
	Confirm *payload.Confirm
	Height  uint32
}

type BlockPool struct {
	Chain     *blockchain.BlockChain
	Store     blockchain.IChainStore
	IsCurrent func() bool

	sync.RWMutex
	blocks      map[common.Uint256]*types.Block
	confirms    map[common.Uint256]*payload.Confirm
	chainParams *config.Configuration
}

func (bm *BlockPool) AppendConfirm(confirm *payload.Confirm) (bool,
	bool, error) {
	bm.Lock()
	defer bm.Unlock()

	return bm.appendConfirm(confirm)
}

func (bm *BlockPool) AddDposBlock(dposBlock *types.DposBlock) (bool, bool, error) {
	if bm.Chain.GetState().ConsensusAlgorithm == state.POW {
		return bm.Chain.ProcessBlock(dposBlock.Block, dposBlock.Confirm)
	}

	if dposBlock.Block.Height > bm.chainParams.DPoSConfiguration.RevertToPOWStartHeight && !dposBlock.HaveConfirm {
		for _, tx := range dposBlock.Transactions {
			if tx.IsRevertToPOW() {
				return bm.Chain.ProcessBlock(dposBlock.Block, dposBlock.Confirm)
			}
		}
	}

	// main version >=H1
	if dposBlock.Block.Height >= bm.chainParams.CRCOnlyDPOSHeight {
		if dposBlock.Block.Height >= bm.chainParams.CRConfiguration.CRCommitteeStartHeight {
			if len(dposBlock.Block.Transactions) > 0 &&
				len(dposBlock.Block.Transactions[0].Outputs()) >= 1 &&
				dposBlock.Block.Transactions[0].Outputs()[0].ProgramHash.
					IsEqual(*bm.chainParams.DestroyELAProgramHash) {
				return bm.Chain.ProcessBlock(dposBlock.Block, dposBlock.Confirm)
			}
		}

		return bm.AppendDposBlock(dposBlock)
	}

	// old version [0, H1)
	return bm.Chain.ProcessBlock(dposBlock.Block, dposBlock.Confirm)
}

func (bm *BlockPool) AppendDposBlock(dposBlock *types.DposBlock) (bool, bool, error) {
	bm.Lock()
	defer bm.Unlock()
	if !dposBlock.HaveConfirm {
		return bm.appendBlock(dposBlock)
	}
	return bm.appendBlockAndConfirm(dposBlock)
}

func (bm *BlockPool) appendBlock(dposBlock *types.DposBlock) (bool, bool, error) {
	// add block
	block := dposBlock.Block
	hash := block.Hash()
	if _, ok := bm.blocks[hash]; ok {
		return false, false, errors.New("duplicate block in pool")
	}
	// verify block
	if err := bm.Chain.CheckBlockSanity(block); err != nil {
		log.Info("[AppendBlock] check block sanity failed, ", err)
		return false, false, err
	}
	if block.Height == bm.Chain.GetHeight()+1 {
		prevNode, exist := bm.Chain.LookupNodeInIndex(&block.Header.Previous)
		if !exist {
			log.Info("[AppendBlock] check block context failed, there is no previous block on the chain")
			return false, false, errors.New("there is no previous block on the chain")
		}
		if err := bm.Chain.CheckBlockContext(block, prevNode); err != nil {
			log.Info("[AppendBlock] check block context failed, ", err)
			return false, false, err
		}
	}

	bm.blocks[block.Hash()] = block

	// confirm block
	inMainChain, isOrphan, err := bm.confirmBlock(hash)
	if err != nil {
		log.Debug("[AppendDPOSBlock] ConfirmBlock failed, height", block.Height, "len(txs):",
			len(block.Transactions), "hash:", hash.String(), "err: ", err)

		// Notify the caller that the new block without confirm was accepted.
		// The caller would typically want to react by relaying the inventory
		// to other peers.
		events.Notify(events.ETBlockAccepted, block)
		if block.Height == blockchain.DefaultLedger.Blockchain.GetHeight()+1 {
			events.Notify(events.ETNewBlockReceived, dposBlock)
		}
		return inMainChain, isOrphan, nil
	}

	copyBlock := *dposBlock
	confirm := bm.confirms[hash]
	copyBlock.HaveConfirm = confirm != nil
	copyBlock.Confirm = confirm

	// notify new block received
	events.Notify(events.ETNewBlockReceived, &copyBlock)

	return inMainChain, isOrphan, nil
}

func (bm *BlockPool) appendBlockAndConfirm(dposBlock *types.DposBlock) (bool, bool, error) {
	block := dposBlock.Block
	hash := block.Hash()
	// verify block
	if err := bm.Chain.CheckBlockSanity(block); err != nil {
		return false, false, err
	}
	// add block
	bm.blocks[block.Hash()] = block
	// confirm block
	inMainChain, isOrphan, err := bm.appendConfirm(dposBlock.Confirm)
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
	block := bm.blocks[confirm.Proposal.BlockHash]

	// notify new confirm accepted.
	events.Notify(events.ETConfirmAccepted, &ConfirmInfo{
		Confirm: confirm,
		Height:  block.Height,
	})

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
	log.Info("[ConfirmBlock] block height:", block.Height)

	confirm, ok := bm.confirms[hash]
	if !ok {
		return false, false, errors.New("there is no block confirmation in pool when confirming block")
	}

	if !bm.Chain.BlockExists(&hash) {
		inMainChain, isOrphan, err := bm.Chain.ProcessBlock(block, confirm)
		if err != nil {
			return inMainChain, isOrphan, errors.New("add block failed," + err.Error())
		}

		if !inMainChain && !isOrphan {
			if err := bm.CheckConfirmedBlockOnFork(bm.Chain.GetHeight(), block); err != nil {
				log.Error("[CheckConfirmedBlockOnFork] error:", err)
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

func (bm *BlockPool) GetDposBlockByHash(hash common.Uint256) (*types.DposBlock, error) {
	bm.RLock()
	defer bm.RUnlock()

	if block := bm.blocks[hash]; block != nil {
		confirm := bm.confirms[hash]
		return &types.DposBlock{
			Block:       block,
			HaveConfirm: confirm != nil,
			Confirm:     confirm,
		}, nil
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

func (bm *BlockPool) CleanFinalConfirmedBlock(height uint32) {
	bm.Lock()
	defer bm.Unlock()

	for _, block := range bm.blocks {
		if block.Height < height-cachedCount {
			delete(bm.blocks, block.Hash())
			delete(bm.confirms, block.Hash())
		}
	}
}

func NewBlockPool(params *config.Configuration) *BlockPool {
	return &BlockPool{
		blocks:      make(map[common.Uint256]*types.Block),
		confirms:    make(map[common.Uint256]*payload.Confirm),
		chainParams: params,
	}
}
