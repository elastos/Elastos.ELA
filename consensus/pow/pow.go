package pow

import (
	cl "DNA_POW/account"
	. "DNA_POW/common"
	"DNA_POW/common/config"
	"DNA_POW/common/log"
	"DNA_POW/core/auxpow"
	"DNA_POW/core/contract/program"
	"DNA_POW/core/ledger"
	tx "DNA_POW/core/transaction"
	"DNA_POW/core/transaction/payload"
	"DNA_POW/core/validation"
	"DNA_POW/crypto"
	"DNA_POW/events"
	"DNA_POW/net"
	//	msg "DNA_POW/net/message"
	//	"bytes"
	//	"math/big"
	"sync"
	"time"
)

var TaskCh chan bool

const (
	MINGENBLOCKTIME = 10
	// maxNonce is the maximum value a nonce can be in a block header.
	maxNonce = ^uint32(0) // 2^32 - 1

	// maxExtraNonce is the maximum value an extra nonce used in a coinbase
	// transaction can be.
	maxExtraNonce = ^uint64(0) // 2^64 - 1

	// hpsUpdateSecs is the number of seconds to wait in between each
	// update to the hashes per second monitor.
	hpsUpdateSecs = 10

	// hashUpdateSec is the number of seconds each worker waits in between
	// notifying the speed monitor with how many hashes have been completed
	// while they are actively searching for a solution.  This is done to
	// reduce the amount of syncs between the workers that must be done to
	// keep track of the hashes per second.
	hashUpdateSecs = 15
)

var GenBlockTime = (MINGENBLOCKTIME * time.Second)

type PowService struct {
	// Miner's receiving address for earning coin
	PayToAddr         string
	coinbaseTx        tx.Transaction
	feesTx            []tx.Transaction
	MsgBlock          *ledger.Block
	ZMQPublish        chan bool
	Mutex             sync.Mutex
	Client            cl.Client
	timer             *time.Timer
	timerHeight       uint32
	timeView          byte
	blockReceivedTime time.Time
	logDictionary     string
	started           bool
	localNet          net.Neter

	newInventorySubscriber          events.Subscriber
	blockPersistCompletedSubscriber events.Subscriber

	submitBlockLock sync.Mutex
	wg              sync.WaitGroup
	quit            chan struct{}
}

func (pow *PowService) CreateCoinbaseTrx(nextBlockHeight uint32, addr string) (*tx.Transaction, error) {
	//1. create script hash
	pkScript, err := ToScriptHash(addr)
	if err != nil {
		return nil, err
	}

	//2. create coinbase tx
	//TODO coinbase payload
	txn, err := tx.NewCoinBaseTransaction(&payload.CoinBase{})
	if err != nil {
		return nil, err
	}

	txn.Outputs = []*tx.TxOutput{
		&tx.TxOutput{
			//TODO asset id
			AssetID: Uint256{},
			//TODO  calc block subsidy
			Value:       0,
			ProgramHash: pkScript,
		},
	}

	return txn, nil
}

func (pow *PowService) GenerateBlock(addr string) (*ledger.Block, error) {
	//nextBlockHeight := ledger.DefaultLedger.Blockchain.BlockHeight + 1
	nextBlockHeight := ledger.DefaultLedger.Blockchain.GetBestHeight() + 1
	coinBaseTx, err := pow.CreateCoinbaseTrx(nextBlockHeight, addr)
	if err != nil {
		return &ledger.Block{}, err
	}

	//TODO TimeSource
	blockData := &ledger.Blockdata{
		Version:          0,
		PrevBlockHash:    ledger.DefaultLedger.Blockchain.CurrentBlockHash(),
		TransactionsRoot: Uint256{},
		Timestamp:        uint32(time.Now().Unix()),
		//Bits:             0x2007ffff,
		Bits:           0x1e03ffff,
		Height:         nextBlockHeight,
		Nonce:          0,
		ConsensusData:  0,
		NextBookKeeper: Uint160{},
		AuxPow:         auxpow.AuxPow{},
		Program:        &program.Program{},
	}

	msgBlock := &ledger.Block{
		Blockdata:    blockData,
		Transactions: []*tx.Transaction{},
	}

	msgBlock.Transactions = append(msgBlock.Transactions, coinBaseTx)
	txHash := []Uint256{}
	txHash = append(txHash, coinBaseTx.Hash())

	//TODO tx ordering
	transactionsPool := pow.localNet.GetTxnPool(true)
	for _, tx := range transactionsPool {
		msgBlock.Transactions = append(msgBlock.Transactions, tx)
		txHash = append(txHash, tx.Hash())
	}

	txRoot, _ := crypto.ComputeRoot(txHash)
	msgBlock.Blockdata.TransactionsRoot = txRoot

	//TODO fee & subsidy

	prevBlock, _ := ledger.DefaultLedger.GetBlockWithHeight(nextBlockHeight - 1)
	msgBlock.Blockdata.Bits, err = validation.CalcNextRequiredDifficulty(prevBlock, time.Now())
	log.Trace("difficulty: ", msgBlock.Blockdata.Bits)

	return msgBlock, err
}

func (pow *PowService) SolveBlock(MsgBlock *ledger.Block) bool {
	header := MsgBlock.Blockdata
	targetDifficulty := validation.CompactToBig(header.Bits)

	for extraNonce := uint64(0); extraNonce < maxExtraNonce; extraNonce++ {
		for i := uint32(0); i <= maxNonce; i++ {
			header.Nonce = i
			hash := header.Hash()
			if validation.HashToBig(&hash).Cmp(targetDifficulty) <= 0 {
				log.Trace(header)
				log.Trace(hash)
				return true
			}
		}
	}
	return false
}

func (pow *PowService) BroadcastBlock(MsgBlock *ledger.Block) error {
	pow.localNet.Xmit(MsgBlock)
	return nil
}

func (pow *PowService) Start() error {
	pow.Mutex.Lock()
	defer pow.Mutex.Unlock()
	pow.wg.Add(1)
	pow.started = true

	pow.blockPersistCompletedSubscriber = ledger.DefaultLedger.Blockchain.BCEvents.Subscribe(events.EventBlockPersistCompleted, pow.BlockPersistCompleted)

	fstBookking, _ := HexToBytes(config.Parameters.BookKeepers[0])
	acct, _ := pow.Client.GetDefaultAccount()
	dftPubkey, _ := acct.PubKey().EncodePoint(true)
	if IsEqualBytes(fstBookking, dftPubkey) {
		log.Trace(fstBookking)
		log.Trace(acct.PubKey().EncodePoint(true))
		fstBookking, _ := HexToBytes(config.Parameters.BookKeepers[0])
		acct, _ := pow.Client.GetDefaultAccount()
		dftPubkey, _ := acct.PubKey().EncodePoint(true)
		if IsEqualBytes(fstBookking, dftPubkey) {
			log.Trace(fstBookking)
			log.Trace(acct.PubKey().EncodePoint(true))
			go pow.Timeout()
		}
	}
	return nil
}

func (pow *PowService) Halt() error {
	log.Debug()
	log.Info("DBFT Stop")
	if pow.timer != nil {
		pow.timer.Stop()
	}

	if pow.started {
		ledger.DefaultLedger.Blockchain.BCEvents.UnSubscribe(events.EventBlockPersistCompleted, pow.blockPersistCompletedSubscriber)
		//pow.localNet.GetEvent("consensus").UnSubscribe(events.EventNewInventory, pow.newInventorySubscriber)

		close(pow.quit)
		pow.wg.Wait()
		pow.started = false
	}
	return nil
}

func (pow *PowService) BlockPersistCompleted(v interface{}) {
	log.Debug()
	if block, ok := v.(*ledger.Block); ok {
		log.Infof("persist block: %x", block.Hash())
		err := pow.localNet.CleanSubmittedTransactions(block)
		if err != nil {
			log.Warn(err)
		}

		pow.localNet.Xmit(block.Hash())
	}

	//pow.blockReceivedTime = time.Now()

	//go pow.InitializeConsensus(0)
}

func (pow *PowService) InitializeConsensus(viewNum byte) error {
	log.Debug("[InitializeConsensus] Start InitializeConsensus.")
	pow.Mutex.Lock()
	defer pow.Mutex.Unlock()

	return nil
}

func NewPowService(client cl.Client, logDictionary string, localNet net.Neter) *PowService {
	log.Debug()

	pow := &PowService{
		Client: client,
		//	MsgBlock:      msgBlock,
		timer:         time.NewTimer(time.Second * 15),
		started:       false,
		ZMQPublish:    make(chan bool, 1),
		localNet:      localNet,
		logDictionary: logDictionary,
	}

	if !pow.timer.Stop() {
		<-pow.timer.C
	}
	log.Debug()
	//go pow.timerRoutine()
	go pow.ZMQServer()
	return pow
}

func (pow *PowService) Timeout() {

out:
	for {
		// Quit when the miner is stopped.
		select {
		case <-pow.quit:
			break out
		default:
			// Non-blocking select to fall through
		}

		//time.Sleep(15 * time.Second)

		addr := "Abn4A5BMXNBrzEfuRbxWpgtYtGbQ6xqK2m"
		msgBlock, err := pow.GenerateBlock(addr)
		if err != nil {
			return
		}
		//if 0 == len(msgBlock.Transactions) {
		//	return
		//}

		pow.MsgBlock = msgBlock

		// push notifyed message into ZMQ
		generateStatus := true
		if true == generateStatus {
			pow.ZMQPublish <- true
		}

		isAuxPow := config.Parameters.PowConfiguration.CoMining
		//begin to mine the block with POW
		if generateStatus && !isAuxPow && pow.SolveBlock(msgBlock) {
			//send the valid block to p2p networkd
			if msgBlock.Blockdata.Height == ledger.DefaultLedger.Blockchain.BlockHeight+1 {

				if err := ledger.DefaultLedger.Blockchain.AddBlock(msgBlock); err != nil {
					log.Trace(err)
					return
				}
				//TODO if co-mining condition
				pow.ZMQClientSend(*msgBlock)
				pow.BroadcastBlock(msgBlock)
			}
			//when the block send succeed, the transaction need to be removed from transaction pool
			//pow.localNet.CleanSubmittedTransactions(pow.MsgBlock)
			//pow.CleanSubmittedTransactions()
		} else {
			//if mining failed, have to give transaction back to transaction pool
			//pow.ReleaseTransactions()
		}

	}

	pow.wg.Done()
	//	pow.timer.Stop()
	//	pow.timer.Reset(GenBlockTime)
}

func (pow *PowService) timerRoutine() {
	log.Debug()
	for {
		select {
		case <-pow.timer.C:
			log.Debug("******Get a timeout notice")
			go pow.Timeout()
		}
	}
}
