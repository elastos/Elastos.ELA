package pow

import (
	"encoding/binary"
	"errors"
	"math"
	"math/rand"
	"sort"
	"sync"
	"time"

	chain "github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/config"
	"github.com/elastos/Elastos.ELA/events"
	"github.com/elastos/Elastos.ELA/log"
	"github.com/elastos/Elastos.ELA/protocol"

	. "github.com/elastos/Elastos.ELA.Utility/core"
	. "github.com/elastos/Elastos.ELA.Utility/common"
	"github.com/elastos/Elastos.ELA.Utility/crypto"
)

var TaskCh chan bool

const (
	maxNonce       = ^uint32(0) // 2^32 - 1
	maxExtraNonce  = ^uint64(0) // 2^64 - 1
	hpsUpdateSecs  = 10
	hashUpdateSecs = 15
)

var (
	TargetTimePerBlock = int64(config.Parameters.ChainParam.TargetTimePerBlock / time.Second)

	OrginAmountOfEla = 3300 * 10000 * 100000000
	SubsidyInterval  = 365 * 24 * 60 * 60 / TargetTimePerBlock
	RetargetPersent  = 25
)

type msgBlock struct {
	BlockData map[string]*Block
	Mutex     sync.Mutex
}

type PowService struct {
	PayToAddr     string
	MsgBlock      msgBlock
	Mutex         sync.Mutex
	logDictionary string
	started       bool
	manualMining  bool
	localNode     protocol.Noder

	blockPersistCompletedSubscriber events.Subscriber
	RollbackTransactionSubscriber   events.Subscriber

	wg   sync.WaitGroup
	quit chan struct{}
}

func (pow *PowService) GetTransactionCount() int {
	transactionsPool := pow.localNode.GetTxnPool(true)
	return len(transactionsPool)
}

func (pow *PowService) CollectTransactions(MsgBlock *Block) int {
	txs := 0
	transactionsPool := pow.localNode.GetTxnPool(true)

	for _, tx := range transactionsPool {
		log.Trace(tx)
		MsgBlock.Transactions = append(MsgBlock.Transactions, tx)
		txs++
	}
	return txs
}

func (pow *PowService) CreateCoinbaseTrx(nextBlockHeight uint32, addr string) (*Transaction, error) {
	minerProgramHash, err := Uint168FromAddress(addr)
	if err != nil {
		return nil, err
	}
	foundationProgramHash, err := Uint168FromAddress(FoundationAddress)
	if err != nil {
		return nil, err
	}

	pd := &PayloadCoinBase{
		CoinbaseData: []byte(config.Parameters.PowConfiguration.MinerInfo),
	}

	txn := NewCoinBaseTransaction(pd, chain.DefaultLedger.Blockchain.GetBestHeight()+1)
	txn.Inputs = []*Input{
		{
			Previous: OutPoint{
				TxID:  EmptyHash,
				Index: math.MaxUint16,
			},
			Sequence: math.MaxUint32,
		},
	}
	txn.Outputs = []*Output{
		{
			AssetID:     chain.DefaultLedger.Blockchain.AssetID,
			Value:       0,
			ProgramHash: *foundationProgramHash,
		},
		{
			AssetID:     chain.DefaultLedger.Blockchain.AssetID,
			Value:       0,
			ProgramHash: *minerProgramHash,
		},
	}

	nonce := make([]byte, 8)
	binary.BigEndian.PutUint64(nonce, rand.Uint64())
	txAttr := NewAttribute(Nonce, nonce)
	txn.Attributes = append(txn.Attributes, &txAttr)
	// log.Trace("txAttr", txAttr)

	return txn, nil
}

func calcBlockSubsidy(currentHeight uint32) Fixed64 {
	ToTalAmountOfEla := int64(OrginAmountOfEla)
	for i := uint32(0); i < (currentHeight / uint32(SubsidyInterval)); i++ {
		incr := float64(ToTalAmountOfEla) / float64(RetargetPersent)
		subsidyPerBlock := int64(float64(incr) / float64(SubsidyInterval))
		ToTalAmountOfEla += subsidyPerBlock * int64(SubsidyInterval)
	}
	incr := float64(ToTalAmountOfEla) / float64(RetargetPersent)
	subsidyPerBlock := Fixed64(float64(incr) / float64(SubsidyInterval))
	log.Trace("subsidyPerBlock: ", subsidyPerBlock)

	return subsidyPerBlock
}

type txSorter []*Transaction

func (s txSorter) Len() int {
	return len(s)
}

func (s txSorter) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s txSorter) Less(i, j int) bool {
	return s[i].FeePerKB < s[j].FeePerKB
}

func (pow *PowService) GenerateBlock(addr string) (*Block, error) {
	nextBlockHeight := chain.DefaultLedger.Blockchain.GetBestHeight() + 1
	coinBaseTx, err := pow.CreateCoinbaseTrx(nextBlockHeight, addr)
	if err != nil {
		return nil, err
	}

	blockData := &Header{
		Version:    0,
		Previous:   *chain.DefaultLedger.Blockchain.BestChain.Hash,
		MerkleRoot: EmptyHash,
		Timestamp:  uint32(chain.DefaultLedger.Blockchain.MedianAdjustedTime().Unix()),
		Bits:       config.Parameters.ChainParam.PowLimitBits,
		Height:     nextBlockHeight,
		Nonce:      0,
		AuxPow:     AuxPow{},
	}

	msgBlock := &Block{
		Header:       blockData,
		Transactions: []*Transaction{},
	}

	msgBlock.Transactions = append(msgBlock.Transactions, coinBaseTx)
	calcTxsSize := coinBaseTx.GetSize()
	calcTxsAmount := 1
	totalFee := Fixed64(0)
	var txPool txSorter
	txPool = make([]*Transaction, 0)
	transactionsPool := pow.localNode.GetTxnPool(false)
	for _, v := range transactionsPool {
		txPool = append(txPool, v)
	}
	sort.Sort(sort.Reverse(txPool))

	for _, tx := range txPool {
		if (tx.GetSize() + calcTxsSize) > config.Parameters.MaxBlockSize {
			break
		}
		if calcTxsAmount >= config.Parameters.MaxTxInBlock {
			break
		}

		if !chain.IsFinalizedTransaction(tx, nextBlockHeight) {
			continue
		}

		fee := chain.GetTxFee(tx, chain.DefaultLedger.Blockchain.AssetID)
		if fee != tx.Fee {
			continue
		}
		msgBlock.Transactions = append(msgBlock.Transactions, tx)
		calcTxsSize = calcTxsSize + tx.GetSize()
		calcTxsAmount++
		totalFee += fee
	}

	subsidy := calcBlockSubsidy(nextBlockHeight)
	reward := totalFee + subsidy
	rewardFoundation := Fixed64(float64(reward) * 0.3)
	msgBlock.Transactions[0].Outputs[0].Value = rewardFoundation
	msgBlock.Transactions[0].Outputs[1].Value = Fixed64(reward) - rewardFoundation

	txHash := make([]Uint256, 0, len(msgBlock.Transactions))
	for _, tx := range msgBlock.Transactions {
		txHash = append(txHash, tx.Hash())
	}
	txRoot, _ := crypto.ComputeRoot(txHash)
	msgBlock.Header.MerkleRoot = txRoot

	msgBlock.Header.Bits, err = chain.CalcNextRequiredDifficulty(chain.DefaultLedger.Blockchain.BestChain, time.Now())
	log.Info("difficulty: ", msgBlock.Header.Bits)

	return msgBlock, err
}

func (pow *PowService) ManualMining(n uint32) ([]*Uint256, error) {
	pow.Mutex.Lock()

	if pow.started || pow.manualMining {
		pow.Mutex.Unlock()
		return nil, errors.New("Server is already CPU mining.")
	}

	pow.started = true
	pow.manualMining = true
	pow.Mutex.Unlock()

	log.Tracef("Pow generating %d blocks", n)
	i := uint32(0)
	blockHashes := make([]*Uint256, n)
	ticker := time.NewTicker(time.Second * hashUpdateSecs)
	defer ticker.Stop()

	for {
		log.Trace("<================Manual Mining==============>\n")

		msgBlock, err := pow.GenerateBlock(pow.PayToAddr)
		if err != nil {
			log.Trace("generage block err", err)
			continue
		}

		if pow.SolveBlock(msgBlock, ticker) {
			if msgBlock.Header.Height == chain.DefaultLedger.Blockchain.GetBestHeight()+1 {
				inMainChain, isOrphan, err := chain.DefaultLedger.Blockchain.AddBlock(msgBlock)
				if err != nil {
					log.Trace(err)
					continue
				}
				//TODO if co-mining condition
				if isOrphan || !inMainChain {
					continue
				}
				pow.BroadcastBlock(msgBlock)
				h := msgBlock.Hash()
				blockHashes[i] = &h
				i++
				if i == n {
					pow.Mutex.Lock()
					pow.started = false
					pow.manualMining = false
					pow.Mutex.Unlock()
					return blockHashes, nil
				}
			}
		}
	}
}

func (pow *PowService) SolveBlock(MsgBlock *Block, ticker *time.Ticker) bool {
	// fake a btc blockheader and coinbase
	auxPow := generateAuxPow(MsgBlock.Hash())
	header := MsgBlock.Header
	targetDifficulty := chain.CompactToBig(header.Bits)

	for i := uint32(0); i <= maxNonce; i++ {
		select {
		case <-ticker.C:
			if !MsgBlock.Header.Previous.IsEqual(*chain.DefaultLedger.Blockchain.BestChain.Hash) {
				return false
			}
			//UpdateBlockTime(msgBlock, m.server.blockManager)

		default:
			// Non-blocking select to fall through
		}

		auxPow.ParBlockHeader.Nonce = i
		hash := auxPow.ParBlockHeader.Hash() // solve parBlockHeader hash
		if chain.HashToBig(&hash).Cmp(targetDifficulty) <= 0 {
			MsgBlock.Header.AuxPow = *auxPow
			return true
		}
	}

	return false
}

func (pow *PowService) BroadcastBlock(MsgBlock *Block) error {
	return pow.localNode.Relay(nil, MsgBlock)
}

func (pow *PowService) Start() {
	pow.Mutex.Lock()
	defer pow.Mutex.Unlock()
	if pow.started || pow.manualMining {
		log.Trace("cpuMining is already started")
	}

	pow.quit = make(chan struct{})
	pow.wg.Add(1)
	pow.started = true

	go pow.cpuMining()
}

func (pow *PowService) Halt() {
	log.Info("POW Stop")
	pow.Mutex.Lock()
	defer pow.Mutex.Unlock()

	if !pow.started || pow.manualMining {
		return
	}

	close(pow.quit)
	pow.wg.Wait()
	pow.started = false
}

func (pow *PowService) RollbackTransaction(v interface{}) {
	if block, ok := v.(*Block); ok {
		for _, tx := range block.Transactions[1:] {
			err := pow.localNode.MaybeAcceptTransaction(tx)
			if err == nil {
				pow.localNode.RemoveTransaction(tx)
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
		err := pow.localNode.CleanSubmittedTransactions(block)
		if err != nil {
			log.Warn(err)
		}
		pow.localNode.SetHeight(uint64(chain.DefaultLedger.Blockchain.GetBestHeight()))
	}
}

func NewPowService(logDictionary string, localNode protocol.Noder) *PowService {
	pow := &PowService{
		PayToAddr:     config.Parameters.PowConfiguration.PayToAddr,
		started:       false,
		manualMining:  false,
		MsgBlock:      msgBlock{BlockData: make(map[string]*Block)},
		localNode:     localNode,
		logDictionary: logDictionary,
	}

	pow.blockPersistCompletedSubscriber = chain.DefaultLedger.Blockchain.BCEvents.Subscribe(events.EventBlockPersistCompleted, pow.BlockPersistCompleted)
	pow.RollbackTransactionSubscriber = chain.DefaultLedger.Blockchain.BCEvents.Subscribe(events.EventRollbackTransaction, pow.RollbackTransaction)

	log.Trace("pow Service Init succeed")
	return pow
}

func (pow *PowService) cpuMining() {
	ticker := time.NewTicker(time.Second * hashUpdateSecs)
	defer ticker.Stop()

out:
	for {
		select {
		case <-pow.quit:
			break out
		default:
			// Non-blocking select to fall through
		}
		log.Trace("<================POW Mining==============>\n")
		//time.Sleep(15 * time.Second)

		msgBlock, err := pow.GenerateBlock(pow.PayToAddr)
		if err != nil {
			log.Trace("generage block err", err)
			continue
		}

		//begin to mine the block with POW
		if pow.SolveBlock(msgBlock, ticker) {
			//send the valid block to p2p networkd
			if msgBlock.Header.Height == chain.DefaultLedger.Blockchain.GetBestHeight()+1 {
				inMainChain, isOrphan, err := chain.DefaultLedger.Blockchain.AddBlock(msgBlock)
				if err != nil {
					log.Trace(err)
					continue
				}
				//TODO if co-mining condition
				if isOrphan || !inMainChain {
					continue
				}
				pow.BroadcastBlock(msgBlock)
			}
		}

	}

	pow.wg.Done()
}
