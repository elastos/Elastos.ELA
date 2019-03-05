package pow

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/elastos/Elastos.ELA/auxpow"
	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/common/log"
	"github.com/elastos/Elastos.ELA/core/types"
	"github.com/elastos/Elastos.ELA/core/types/outputpayload"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/crypto"
	"github.com/elastos/Elastos.ELA/dpos/state"
	"github.com/elastos/Elastos.ELA/elanet/pact"
	"github.com/elastos/Elastos.ELA/errors"
	"github.com/elastos/Elastos.ELA/mempool"
	"github.com/elastos/Elastos.ELA/version"
)

const (
	maxNonce       = ^uint32(0) // 2^32 - 1
	updateInterval = 5 * time.Second
)

type Config struct {
	PayToAddr      string
	MinerInfo      string
	Chain          *blockchain.BlockChain
	ChainParams    *config.Params
	TxMemPool      *mempool.TxPool
	Versions       version.HeightVersions
	BroadcastBlock func(block *types.Block)
}

type blockPool struct {
	mutex       sync.RWMutex
	mapNewBlock map[common.Uint256]*types.Block
}

func (p *blockPool) AppendBlock(block *types.Block) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.mapNewBlock[block.Hash()] = block
}

func (p *blockPool) ClearBlock() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	for key := range p.mapNewBlock {
		delete(p.mapNewBlock, key)
	}
}

func (p *blockPool) GetBlock(hash common.Uint256) (*types.Block, bool) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	block, ok := p.mapNewBlock[hash]
	return block, ok
}

type Service struct {
	PayToAddr   string
	MinerInfo   string
	chain       *blockchain.BlockChain
	chainParams *config.Params
	dutyState   *state.DutyState
	versions    version.HeightVersions
	txMemPool   *mempool.TxPool
	broadcast   func(block *types.Block)

	mutex           sync.Mutex
	started         bool
	discreteMining  bool
	blockPool       blockPool
	preChainHeight  uint32
	preTime         time.Time
	currentAuxBlock *types.Block

	wg   sync.WaitGroup
	quit chan struct{}

	lock      sync.Mutex
	lastBlock *types.Block
}

func (s *Service) CreateCoinbaseTx(minerAddr string) (*types.Transaction, error) {
	minerProgramHash, err := common.Uint168FromAddress(minerAddr)
	if err != nil {
		return nil, err
	}

	currentHeight := s.chain.GetHeight() + 1
	version := types.TransactionVersion(s.versions.GetDefaultTxVersion(currentHeight))
	tx := &types.Transaction{
		Version:        version,
		TxType:         types.CoinBase,
		PayloadVersion: payload.CoinBaseVersion,
		Payload: &payload.CoinBase{
			Content: []byte(s.MinerInfo),
		},
		Inputs: []*types.Input{
			{
				Previous: types.OutPoint{
					TxID:  common.EmptyHash,
					Index: math.MaxUint16,
				},
				Sequence: math.MaxUint32,
			},
		},
		Outputs: []*types.Output{
			{
				AssetID:     config.ELAAssetID,
				Value:       0,
				ProgramHash: s.chainParams.Foundation,
				Type:        types.OTNone,
				Payload:     &outputpayload.DefaultOutput{},
			},
			{
				AssetID:     config.ELAAssetID,
				Value:       0,
				ProgramHash: *minerProgramHash,
				Type:        types.OTNone,
				Payload:     &outputpayload.DefaultOutput{},
			},
		},
		Attributes: []*types.Attribute{},
		LockTime:   currentHeight,
	}

	nonce := make([]byte, 8)
	binary.BigEndian.PutUint64(nonce, rand.Uint64())
	txAttr := types.NewAttribute(types.Nonce, nonce)
	tx.Attributes = append(tx.Attributes, &txAttr)

	return tx, nil
}

func (s *Service) assignCoinbaseRewards(block *types.Block, totalReward common.Fixed64) error {
	// The point open arbiters join into DPOS consensus.
	if block.Height >= s.chainParams.OpenArbitersHeight {
		rewardCyberRepublic := common.Fixed64(math.Ceil(float64(totalReward) * 0.3))
		rewardDposArbiter := common.Fixed64(float64(totalReward) * 0.35)

		var dposChange common.Fixed64
		var err error
		if dposChange, err = s.distributeDposReward(block.Transactions[0], rewardDposArbiter); err != nil {
			return err
		}
		rewardMergeMiner := common.Fixed64(totalReward) - rewardCyberRepublic - rewardDposArbiter + dposChange
		block.Transactions[0].Outputs[0].Value = rewardCyberRepublic
		block.Transactions[0].Outputs[1].Value = rewardMergeMiner
		return nil
	}

	if block.Height >= s.chainParams.DPOSStartHeight {
		return s.assignCoinbaseRewardsV1(block, totalReward)
	}

	return s.assignCoinbaseRewardsV0(block, totalReward)
}

func (s *Service) distributeDposReward(coinBaseTx *types.Transaction, reward common.Fixed64) (common.Fixed64, error) {
	arbitratorsHashes := s.dutyState.GetArbiterProgramHashes()
	if len(arbitratorsHashes) == 0 {
		return 0, fmt.Errorf("not found arbiters when distributeDposReward")
	}
	candidatesHashes := s.dutyState.GetCandidateProgramHashes()

	totalBlockConfirmReward := float64(reward) * 0.25
	totalTopProducersReward := float64(reward) * 0.75
	individualBlockConfirmReward := common.Fixed64(math.Floor(totalBlockConfirmReward / float64(len(arbitratorsHashes))))
	individualProducerReward := common.Fixed64(math.Floor(totalTopProducersReward / float64(int(s.chainParams.ArbitersCount)+len(candidatesHashes))))

	realDposReward := common.Fixed64(0)
	for _, v := range arbitratorsHashes {
		reward := individualBlockConfirmReward + individualProducerReward
		if s.dutyState.IsCRCArbiterProgramHash(v) {
			reward = individualBlockConfirmReward
		}

		coinBaseTx.Outputs = append(coinBaseTx.Outputs, &types.Output{
			AssetID:     config.ELAAssetID,
			Value:       reward,
			ProgramHash: *v,
			Type:        types.OTNone,
			Payload:     &outputpayload.DefaultOutput{},
		})

		realDposReward += reward
	}

	for _, v := range candidatesHashes {

		coinBaseTx.Outputs = append(coinBaseTx.Outputs, &types.Output{
			AssetID:     config.ELAAssetID,
			Value:       individualProducerReward,
			ProgramHash: *v,
			Type:        types.OTNone,
			Payload:     &outputpayload.DefaultOutput{},
		})

		realDposReward += individualProducerReward
	}

	change := reward - realDposReward
	if change < 0 {
		return 0, fmt.Errorf("real dpos reward more than reward limit")
	}
	return change, nil
}

func (s *Service) GenerateBlock(minerAddr string) (*types.Block, error) {
	bestChain := s.chain.BestChain
	nextBlockHeight := bestChain.Height + 1
	coinBaseTx, err := s.CreateCoinbaseTx(minerAddr)
	if err != nil {
		return nil, err
	}

	header := types.Header{
		Version:    s.versions.GetDefaultBlockVersion(nextBlockHeight),
		Previous:   *s.chain.BestChain.Hash,
		MerkleRoot: common.EmptyHash,
		Timestamp:  uint32(s.chain.MedianAdjustedTime().Unix()),
		Bits:       config.Parameters.ChainParam.PowLimitBits,
		Height:     nextBlockHeight,
		Nonce:      0,
	}

	msgBlock := &types.Block{
		Header:       header,
		Transactions: []*types.Transaction{},
	}

	msgBlock.Transactions = append(msgBlock.Transactions, coinBaseTx)
	totalTxsSize := coinBaseTx.GetSize()
	txCount := 1
	totalTxFee := common.Fixed64(0)
	txs := s.txMemPool.GetTxsInPool()
	sort.Slice(txs, func(i, j int) bool {
		if txs[i].IsIllegalTypeTx() || txs[i].IsInactiveArbiters() {
			return true
		}
		if txs[j].IsIllegalTypeTx() || txs[j].IsInactiveArbiters() {
			return false
		}
		return txs[i].FeePerKB > txs[j].FeePerKB
	})

	for _, tx := range txs {
		totalTxsSize = totalTxsSize + tx.GetSize()
		if totalTxsSize > pact.MaxBlockSize {
			break
		}
		if txCount >= config.Parameters.MaxTxsInBlock {
			break
		}

		if !blockchain.IsFinalizedTransaction(tx, nextBlockHeight) {
			continue
		}
		if errCode := s.chain.CheckTransactionContext(nextBlockHeight, tx); errCode != errors.Success {
			log.Warn("check transaction context failed, wrong transaction:", tx.Hash().String())
			continue
		}
		fee := blockchain.GetTxFee(tx, config.ELAAssetID)
		if fee != tx.Fee {
			continue
		}
		msgBlock.Transactions = append(msgBlock.Transactions, tx)
		totalTxFee += fee
		txCount++
	}

	totalReward := totalTxFee + s.chainParams.RewardPerBlock
	if err := s.assignCoinbaseRewards(msgBlock, totalReward); err != nil {
		return nil, err
	}

	txHash := make([]common.Uint256, 0, len(msgBlock.Transactions))
	for _, tx := range msgBlock.Transactions {
		txHash = append(txHash, tx.Hash())
	}
	txRoot, _ := crypto.ComputeRoot(txHash)
	msgBlock.Header.MerkleRoot = txRoot

	msgBlock.Header.Bits, err = s.chain.CalcNextRequiredDifficulty(bestChain, time.Now())
	log.Info("difficulty: ", msgBlock.Header.Bits)

	return msgBlock, err
}

func (s *Service) CreateAuxBlock(payToAddr string) (*types.Block, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.chain.GetHeight() == 0 || s.preChainHeight != s.chain.GetHeight() ||
		time.Now().After(s.preTime.Add(updateInterval)) {

		if s.preChainHeight != s.chain.GetHeight() {
			// Clear old blocks since they're obsolete now.
			s.currentAuxBlock = nil
			s.blockPool.ClearBlock()
		}

		// Create new block with nonce = 0
		auxBlock, err := s.GenerateBlock(payToAddr)
		if err != nil {
			return nil, err
		}

		// Update state only when CreateNewBlock succeeded
		s.preChainHeight = s.chain.GetHeight()
		s.preTime = time.Now()

		// Save
		s.currentAuxBlock = auxBlock
		s.blockPool.AppendBlock(auxBlock)
	}

	// At this point, currentAuxBlock is always initialised: If we make it here without creating
	// a new block above, it means that, in particular, preChainHeight == ServerNode.Height().
	// But for that to happen, we must already have created a currentAuxBlock in a previous call,
	// as preChainHeight is initialised only when currentAuxBlock is.
	if s.currentAuxBlock == nil {
		return nil, fmt.Errorf("no block cached")
	}

	return s.currentAuxBlock, nil
}

func (s *Service) SubmitAuxBlock(hash *common.Uint256, auxPow *auxpow.AuxPow) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	msgAuxBlock, ok := s.blockPool.GetBlock(*hash)
	if !ok {
		log.Debug("[json-rpc:SubmitAuxBlock] block hash unknown", hash)
		return fmt.Errorf("block hash unknown")
	}

	msgAuxBlock.Header.AuxPow = *auxPow
	_, _, err := s.versions.AddDposBlock(&types.DposBlock{
		BlockFlag: true,
		Block:     msgAuxBlock,
	})
	return err
}

func (s *Service) DiscreteMining(n uint32) ([]*common.Uint256, error) {
	s.mutex.Lock()

	if s.started || s.discreteMining {
		s.mutex.Unlock()
		return nil, fmt.Errorf("Server is already CPU mining.")
	}

	s.started = true
	s.discreteMining = true
	s.mutex.Unlock()

	log.Debugf("Pow generating %d blocks", n)
	i := uint32(0)
	blockHashes := make([]*common.Uint256, 0)

	for {
		log.Debug("<================Discrete Mining==============>\n")

		msgBlock, err := s.GenerateBlock(s.PayToAddr)
		if err != nil {
			log.Debug("generage block err", err)
			continue
		}

		if s.SolveBlock(msgBlock, nil) {
			if msgBlock.Header.Height == s.chain.GetHeight()+1 {

				_, _, err := s.versions.AddDposBlock(&types.DposBlock{
					BlockFlag: true,
					Block:     msgBlock,
				})
				if err != nil {
					continue
				}

				h := msgBlock.Hash()
				blockHashes = append(blockHashes, &h)
				i++
				if i == n {
					s.mutex.Lock()
					s.started = false
					s.discreteMining = false
					s.mutex.Unlock()
					return blockHashes, nil
				}
			}
		}
	}
}

func (s *Service) SolveBlock(msgBlock *types.Block, lastBlockHash *common.Uint256) bool {
	ticker := time.NewTicker(updateInterval)
	defer ticker.Stop()
	// fake a btc blockheader and coinbase
	auxPow := auxpow.GenerateAuxPow(msgBlock.Hash())
	header := msgBlock.Header
	targetDifficulty := blockchain.CompactToBig(header.Bits)
	for i := uint32(0); i <= maxNonce; i++ {
		select {
		case <-ticker.C:
			log.Info("five second countdown ends. Re-generate block.")
			return false
		default:
			// Non-blocking select to fall through
		}

		auxPow.ParBlockHeader.Nonce = i
		hash := auxPow.ParBlockHeader.Hash() // solve parBlockHeader hash
		if blockchain.HashToBig(&hash).Cmp(targetDifficulty) <= 0 {
			msgBlock.Header.AuxPow = *auxPow
			return true
		}
	}

	return false
}

func (s *Service) Start() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.started || s.discreteMining {
		log.Debug("cpuMining is already started")
	}

	s.quit = make(chan struct{})
	s.wg.Add(1)
	s.started = true

	go s.cpuMining()
}

func (s *Service) Halt() {
	log.Info("POW Stop")
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.started || s.discreteMining {
		return
	}

	close(s.quit)
	s.wg.Wait()
	s.started = false
}

func (s *Service) cpuMining() {
	blockVersion := s.versions.GetDefaultBlockVersion(s.chain.BestChain.Height + 1)
	switch blockVersion {
	case 0:
		s.cpuMiningV0()
	default:
		s.cpuMiningMain()
	}
}

func (s *Service) cpuMiningMain() {
out:
	for {
		select {
		case <-s.quit:
			break out
		default:
			// Non-blocking select to fall through
		}
		log.Info("<================Packing Block==============>")

		s.lock.Lock()
		msgBlock, err := s.GenerateBlock(s.PayToAddr)
		if err != nil {
			log.Error("generage block err", err)
			s.lock.Unlock()
			continue
		}
		s.lock.Unlock()

		//begin to mine the block with POW
		hash := s.lastBlock.Hash()
		if s.SolveBlock(msgBlock, &hash) {
			log.Info("<================Solved Block==============>")
			//send the valid block to p2p networkd

			_, _, err := s.versions.AddDposBlock(&types.DposBlock{
				BlockFlag: true,
				Block:     msgBlock,
			})
			if err != nil {
				log.Debug(err)
				continue
			}

			s.broadcast(msgBlock)
			hash := msgBlock.Hash()
			node := blockchain.NewBlockNode(&msgBlock.Header, &hash)
			node.InMainChain = true
			prevHash := &msgBlock.Previous
			if parentNode, ok := s.chain.LookupNodeInIndex(prevHash); ok {
				node.WorkSum = node.WorkSum.Add(parentNode.WorkSum, node.WorkSum)
				node.Parent = parentNode
			}
			s.lock.Lock()
			s.lastBlock = msgBlock
			s.lock.Unlock()
		}
	}

	s.wg.Done()
}

func (s *Service) cpuMiningV0() {
out:
	for {
		select {
		case <-s.quit:
			break out
		default:
			// Non-blocking select to fall through
		}
		log.Debug("<================Packing Block==============>")
		//time.Sleep(15 * time.Second)

		msgBlock, err := s.GenerateBlock(s.PayToAddr)
		if err != nil {
			log.Debug("generage block err", err)
			continue
		}

		//begin to mine the block with POW
		if s.SolveBlock(msgBlock, nil) {
			log.Info("<================Solved Block==============>")
			//send the valid block to p2p networkd
			if msgBlock.Header.Height == s.chain.GetHeight()+1 {

				inMainChain, isOrphan, err := s.versions.AddDposBlock(&types.DposBlock{
					BlockFlag: true,
					Block:     msgBlock,
				})
				if err != nil {
					log.Debug(err)
					continue
				}

				if isOrphan || !inMainChain {
					continue
				}
			}
		}

	}

	s.wg.Done()
}

func NewService(cfg *Config) *Service {
	block, _ := cfg.Chain.GetBlockByHash(*cfg.Chain.BestChain.Hash)
	pow := &Service{
		PayToAddr:   cfg.PayToAddr,
		MinerInfo:   cfg.MinerInfo,
		chain:       cfg.Chain,
		chainParams: cfg.ChainParams,
		versions:    cfg.Versions,
		txMemPool:   cfg.TxMemPool,
		broadcast:   cfg.BroadcastBlock,

		started:        false,
		discreteMining: false,
		blockPool:      blockPool{mapNewBlock: make(map[common.Uint256]*types.Block)},
		lastBlock:      block,
	}

	return pow
}
