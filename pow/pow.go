package pow

import (
	"encoding/binary"
	"errors"
	"math"
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/elastos/Elastos.ELA/auxpow"
	. "github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/config"
	. "github.com/elastos/Elastos.ELA/core"
	"github.com/elastos/Elastos.ELA/core/outputpayload"
	. "github.com/elastos/Elastos.ELA/errors"
	"github.com/elastos/Elastos.ELA/events"
	"github.com/elastos/Elastos.ELA/log"
	"github.com/elastos/Elastos.ELA/node"

	"github.com/elastos/Elastos.ELA.Utility/common"
	"github.com/elastos/Elastos.ELA.Utility/crypto"
)

var TaskCh chan bool

const (
	maxNonce       = ^uint32(0) // 2^32 - 1
	hashUpdateSecs = 15
)

type auxBlockPool struct {
	mapNewBlock map[common.Uint256]*Block
	mutex       sync.RWMutex
}

func (auxpool *auxBlockPool) AppendBlock(block *Block) {
	auxpool.mutex.Lock()
	defer auxpool.mutex.Unlock()

	auxpool.mapNewBlock[block.Hash()] = block
}

func (auxpool *auxBlockPool) ClearBlock() {
	auxpool.mutex.Lock()
	defer auxpool.mutex.Unlock()

	for key := range auxpool.mapNewBlock {
		delete(auxpool.mapNewBlock, key)
	}
}

func (auxpool *auxBlockPool) GetBlock(hash common.Uint256) (*Block, bool) {
	auxpool.mutex.RLock()
	defer auxpool.mutex.RUnlock()

	block, ok := auxpool.mapNewBlock[hash]
	return block, ok
}

type PowService struct {
	NewBlocksListener

	PayToAddr      string
	Started        bool
	discreteMining bool
	AuxBlockPool   auxBlockPool
	Mutex          sync.Mutex

	blockPersistCompletedSubscriber events.Subscriber
	RollbackTransactionSubscriber   events.Subscriber

	wg   sync.WaitGroup
	quit chan struct{}
	wait chan bool

	lastBlock *Block
	lastNode  *BlockNode
}

func (pow *PowService) CreateCoinbaseTx(nextBlockHeight uint32, minerAddr string) (*Transaction, error) {
	minerProgramHash, err := common.Uint168FromAddress(minerAddr)
	if err != nil {
		return nil, err
	}

	pd := &PayloadCoinBase{
		CoinbaseData: []byte(config.Parameters.PowConfiguration.MinerInfo),
	}

	txn := NewCoinBaseTransaction(pd, DefaultLedger.Blockchain.GetBestHeight()+1)
	txn.Inputs = []*Input{
		{
			Previous: OutPoint{
				TxID:  common.EmptyHash,
				Index: math.MaxUint16,
			},
			Sequence: math.MaxUint32,
		},
	}
	txn.Outputs = []*Output{
		{
			AssetID:       DefaultLedger.Blockchain.AssetID,
			Value:         0,
			ProgramHash:   FoundationAddress,
			OutputType:    DefaultOutput,
			OutputPayload: &outputpayload.DefaultOutput{},
		},
		{
			AssetID:       DefaultLedger.Blockchain.AssetID,
			Value:         0,
			ProgramHash:   *minerProgramHash,
			OutputType:    DefaultOutput,
			OutputPayload: &outputpayload.DefaultOutput{},
		},
	}

	nonce := make([]byte, 8)
	binary.BigEndian.PutUint64(nonce, rand.Uint64())
	txAttr := NewAttribute(Nonce, nonce)
	txn.Attributes = append(txn.Attributes, &txAttr)

	return txn, nil
}

type byFeeDesc []*Transaction

func (s byFeeDesc) Len() int           { return len(s) }
func (s byFeeDesc) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s byFeeDesc) Less(i, j int) bool { return s[i].FeePerKB > s[j].FeePerKB }

func (pow *PowService) GenerateBlock(minerAddr string, bestChain *BlockNode) (*Block, error) {
	nextBlockHeight := bestChain.Height + 1
	coinBaseTx, err := pow.CreateCoinbaseTx(nextBlockHeight, minerAddr)
	if err != nil {
		return nil, err
	}

	header := Header{
		Version:    DefaultLedger.HeightVersions.GetDefaultBlockVersion(nextBlockHeight),
		Previous:   *bestChain.Hash,
		MerkleRoot: common.EmptyHash,
		Timestamp:  uint32(DefaultLedger.Blockchain.MedianAdjustedTime().Unix()),
		Bits:       config.Parameters.ChainParam.PowLimitBits,
		Height:     nextBlockHeight,
		Nonce:      0,
	}

	msgBlock := &Block{
		Header:       header,
		Transactions: []*Transaction{},
	}

	msgBlock.Transactions = append(msgBlock.Transactions, coinBaseTx)
	totalTxsSize := coinBaseTx.GetSize()
	txCount := 1
	totalTxFee := common.Fixed64(0)
	var txsByFeeDesc byFeeDesc
	txsInPool := node.LocalNode.GetTransactionPool(false)
	txsByFeeDesc = make([]*Transaction, 0, len(txsInPool))
	for _, v := range txsInPool {
		txsByFeeDesc = append(txsByFeeDesc, v)
	}
	sort.Sort(txsByFeeDesc)

	for _, tx := range txsByFeeDesc {
		totalTxsSize = totalTxsSize + tx.GetSize()
		if totalTxsSize > config.Parameters.MaxBlockSize {
			break
		}
		if txCount >= config.Parameters.MaxTxsInBlock {
			break
		}

		if !IsFinalizedTransaction(tx, nextBlockHeight) {
			continue
		}
		if errCode := CheckTransactionContext(nextBlockHeight, tx); errCode != Success {
			log.Warn("check transaction context failed, wrong transaction:", tx.Hash().String())
			continue
		}
		fee := GetTxFee(tx, DefaultLedger.Blockchain.AssetID)
		if fee != tx.Fee {
			continue
		}
		msgBlock.Transactions = append(msgBlock.Transactions, tx)
		totalTxFee += fee
		txCount++
	}

	blockReward := RewardAmountPerBlock
	totalReward := totalTxFee + blockReward
	if err := DefaultLedger.HeightVersions.AssignCoinbaseTxRewards(msgBlock, totalReward); err != nil {
		return nil, err
	}

	txHash := make([]common.Uint256, 0, len(msgBlock.Transactions))
	for _, tx := range msgBlock.Transactions {
		txHash = append(txHash, tx.Hash())
	}
	txRoot, _ := crypto.ComputeRoot(txHash)
	msgBlock.Header.MerkleRoot = txRoot

	msgBlock.Header.Bits, err = CalcNextRequiredDifficulty(bestChain, time.Now())
	log.Info("difficulty: ", msgBlock.Header.Bits)

	return msgBlock, err
}

func (pow *PowService) DiscreteMining(n uint32) ([]*common.Uint256, error) {
	pow.Mutex.Lock()

	if pow.Started || pow.discreteMining {
		pow.Mutex.Unlock()
		return nil, errors.New("Server is already CPU mining.")
	}

	pow.Started = true
	pow.discreteMining = true
	pow.Mutex.Unlock()

	log.Debugf("Pow generating %d blocks", n)
	i := uint32(0)
	blockHashes := make([]*common.Uint256, 0)

	for {
		log.Debug("<================Discrete Mining==============>\n")

		msgBlock, err := pow.GenerateBlock(pow.PayToAddr, DefaultLedger.Blockchain.BestChain)
		if err != nil {
			log.Debug("generage block err", err)
			continue
		}

		if pow.SolveBlock(msgBlock) {
			if msgBlock.Header.Height == DefaultLedger.Blockchain.GetBestHeight()+1 {

				if err := DefaultLedger.HeightVersions.AddBlock(msgBlock); err != nil {
					continue
				}

				pow.BroadcastBlock(msgBlock)
				h := msgBlock.Hash()
				blockHashes = append(blockHashes, &h)
				i++
				if i == n {
					pow.Mutex.Lock()
					pow.Started = false
					pow.discreteMining = false
					pow.Mutex.Unlock()
					return blockHashes, nil
				}
			}
		}
	}
}

func (pow *PowService) SolveBlock(MsgBlock *Block) bool {
	ticker := time.NewTicker(time.Second * hashUpdateSecs)
	defer ticker.Stop()

	// fake a btc blockheader and coinbase
	auxPow := auxpow.GenerateAuxPow(MsgBlock.Hash())
	header := MsgBlock.Header
	targetDifficulty := CompactToBig(header.Bits)

	for i := uint32(0); i <= maxNonce; i++ {
		select {
		case <-ticker.C:
			// if !MsgBlock.Header.Previous.IsEqual(*DefaultLedger.Blockchain.BestChain.Hash) {
			// 	return false
			// }
			return false

		default:
			// Non-blocking select to fall through
		}

		auxPow.ParBlockHeader.Nonce = i
		hash := auxPow.ParBlockHeader.Hash() // solve parBlockHeader hash
		if HashToBig(&hash).Cmp(targetDifficulty) <= 0 {
			MsgBlock.Header.AuxPow = *auxPow
			return true
		}
	}

	return false
}

func (pow *PowService) BroadcastBlock(block *Block) error {
	return node.LocalNode.Relay(nil, &BlockConfirm{
		BlockFlag: true,
		Block:     block,
	})
}

func (pow *PowService) Start() {
	pow.Mutex.Lock()
	defer pow.Mutex.Unlock()
	if pow.Started || pow.discreteMining {
		log.Debug("cpuMining is already Started")
	}

	pow.quit = make(chan struct{})
	pow.wait = make(chan bool, 1)
	pow.wg.Add(1)
	pow.Started = true
	pow.wait <- true

	go pow.cpuMining()
}

func (pow *PowService) Halt() {
	log.Info("POW Stop")
	pow.Mutex.Lock()
	defer pow.Mutex.Unlock()

	if !pow.Started || pow.discreteMining {
		return
	}

	close(pow.quit)
	close(pow.wait)
	pow.wg.Wait()
	pow.Started = false
}

func (pow *PowService) RollbackTransaction(v interface{}) {
	if block, ok := v.(*Block); ok {
		for _, tx := range block.Transactions[1:] {
			err := node.LocalNode.MaybeAcceptTransaction(tx)
			if err == nil {
				node.LocalNode.RemoveTransaction(tx)
			} else {
				log.Error(err)
			}
		}
	}
}

func (pow *PowService) BlockPersistCompleted(v interface{}) {
	log.Debug()
	if block, ok := v.(*Block); ok {
		log.Infof("persist block: %x", block.Hash())
		err := node.LocalNode.CleanSubmittedTransactions(block)
		if err != nil {
			log.Warn(err)
		}
		node.LocalNode.SetHeight(uint64(DefaultLedger.Blockchain.GetBestHeight()))
	}
}

func NewPowService() *PowService {
	block, _ := DefaultLedger.Store.GetBlock(*DefaultLedger.Blockchain.BestChain.Hash)
	pow := &PowService{
		PayToAddr:      config.Parameters.PowConfiguration.PayToAddr,
		Started:        false,
		discreteMining: false,
		AuxBlockPool:   auxBlockPool{mapNewBlock: make(map[common.Uint256]*Block)},
		lastBlock:      block,
		lastNode:       DefaultLedger.Blockchain.BestChain,
	}

	pow.blockPersistCompletedSubscriber = DefaultLedger.Blockchain.BCEvents.Subscribe(events.EventBlockPersistCompleted, pow.BlockPersistCompleted)
	pow.RollbackTransactionSubscriber = DefaultLedger.Blockchain.BCEvents.Subscribe(events.EventRollbackTransaction, pow.RollbackTransaction)

	DefaultLedger.Blockchain.NewBlocksListeners = append(DefaultLedger.Blockchain.NewBlocksListeners, pow)

	log.Debug("pow Service Init succeed")
	return pow
}

func (pow *PowService) OnBlockReceived(b *Block, confirmed bool) {}

func (pow *PowService) OnConfirmReceived(p *DPosProposalVoteSlot) {
	block, err := DefaultLedger.Store.GetBlock(p.Hash)
	if err != nil {
		return
	}

	if block.Height == pow.lastBlock.Height {
		if p.Hash.IsEqual(block.Hash()) {
			pow.wait <- true
		} else {
			pow.wait <- false
			pow.lastBlock = block
			pow.lastNode = DefaultLedger.Blockchain.BestChain
		}
	}
}

func (pow *PowService) cpuMining() {

out:
	for {
		select {
		case <-pow.quit:
			break out
		default:
			// Non-blocking select to fall through
		}
		log.Debug("<================Packing Block==============>")

		msgBlock, err := pow.GenerateBlock(pow.PayToAddr, pow.lastNode)
		if err != nil {
			log.Debug("generage block err", err)
			continue
		}

		//begin to mine the block with POW
		if pow.SolveBlock(msgBlock) {
			log.Info("<================Solved Block==============>")
			//send the valid block to p2p networkd
			if msgBlock.Header.Height == pow.lastNode.Height+1 {

				if !<-pow.wait {
					continue
				}

				time.Sleep(time.Second * 1)

				if err := DefaultLedger.HeightVersions.AddBlock(msgBlock); err != nil {
					log.Debug(err)
					continue
				}

				pow.BroadcastBlock(msgBlock)
				hash := msgBlock.Hash()
				node := NewBlockNode(&msgBlock.Header, &hash)
				node.InMainChain = true
				prevHash := &msgBlock.Previous
				if parentNode, ok := DefaultLedger.Blockchain.LookupNodeInIndex(prevHash); ok {
					node.WorkSum = node.WorkSum.Add(parentNode.WorkSum, node.WorkSum)
					node.Parent = parentNode
				}
				pow.lastBlock = msgBlock
				pow.lastNode = node
			}
		}

	}

	pow.wg.Done()
}
