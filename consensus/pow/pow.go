package pow

import (
	"DNA/common/config"
	cl "DNA_POW/account"
	. "DNA_POW/common"
	"DNA_POW/common/log"
	"DNA_POW/core/contract/program"
	"DNA_POW/core/ledger"
	tx "DNA_POW/core/transaction"
	"DNA_POW/core/transaction/payload"
	"DNA_POW/crypto"
	"DNA_POW/events"
	"DNA_POW/net"
	msg "DNA_POW/net/message"
	//	"bytes"
	"math/big"
	"sync"
	"time"
)

var TaskCh chan bool

const (
	MINGENBLOCKTIME = 6
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
}

func (pow *PowService) CollectTransactions(MsgBlock *ledger.Block) int {
	txs := 0
	transactionsPool := pow.localNet.GetTxnPool(true)

	for _, tx := range transactionsPool {
		log.Trace(tx)
		MsgBlock.Transactions = append(MsgBlock.Transactions, tx)
		txs++
	}

	return txs
}

func (pow *PowService) CreateCoinbaseTrx(MsgBlock *ledger.Block) bool {
	//setp1 combine the coinbase transaction
	return true
}

func (pow *PowService) CreateFeesTrx(MsgBlock *ledger.Block) bool {
	if 0 == len(MsgBlock.Transactions) {
		return false
	}

	//setp1 create the fees transaction
	return true
}

func (pow *PowService) GenerateBlock(MsgBlock *ledger.Block) bool {
	if 0 == len(MsgBlock.Transactions) {
		return false
	}

	txHash := []Uint256{}
	//	txHash = append(txHash, pow.coinbaseTx.Hash())

	//for _, t := range pow.feesTx {
	//txHash = append(txHash, t.Hash())
	//}

	for _, t := range MsgBlock.Transactions {
		txHash = append(txHash, t.Hash())
	}

	txRoot, _ := crypto.ComputeRoot(txHash)
	MsgBlock.Blockdata.TransactionsRoot = txRoot

	return true
}

func CompactToBig(compact uint32) *big.Int {
	// Extract the mantissa, sign bit, and exponent.
	mantissa := compact & 0x007fffff
	isNegative := compact&0x00800000 != 0
	exponent := uint(compact >> 24)

	// Since the base for the exponent is 256, the exponent can be treated
	// as the number of bytes to represent the full 256-bit number.  So,
	// treat the exponent as the number of bytes and shift the mantissa
	// right or left accordingly.  This is equivalent to:
	// N = mantissa * 256^(exponent-3)
	var bn *big.Int
	if exponent <= 3 {
		mantissa >>= 8 * (3 - exponent)
		bn = big.NewInt(int64(mantissa))
	} else {
		bn = big.NewInt(int64(mantissa))
		bn.Lsh(bn, 8*(exponent-3))
	}

	// Make it negative if the sign bit is set.
	if isNegative {
		bn = bn.Neg(bn)
	}

	return bn
}

// HashToBig converts a chainhash.Hash into a big.Int that can be used to
// perform math comparisons.
func HashToBig(hash *Uint256) *big.Int {
	// A Hash is in little-endian, but the big package wants the bytes in
	// big-endian, so reverse them.
	buf := *hash
	blen := len(buf)
	for i := 0; i < blen/2; i++ {
		buf[i], buf[blen-1-i] = buf[blen-1-i], buf[i]
	}

	return new(big.Int).SetBytes(buf[:])
}

func (pow *PowService) SolveBlock(MsgBlock *ledger.Block) bool {
	header := MsgBlock.Blockdata
	header.Timestamp = uint32(time.Now().Unix())
	header.PrevBlockHash = ledger.DefaultLedger.Blockchain.CurrentBlockHash()
	header.Height = ledger.DefaultLedger.Blockchain.BlockHeight + 1
	targetDifficulty := CompactToBig(header.Bits)

	for extraNonce := uint64(0); extraNonce < maxExtraNonce; extraNonce++ {
		for i := uint32(0); i <= maxNonce; i++ {
			header.Nonce = i
			hash := header.Hash()
			if HashToBig(&hash).Cmp(targetDifficulty) <= 0 {
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

/*
func (pow *PowService) CleanSubmittedTransactions() error {
	return nil
}
*/

func (pow *PowService) ReleaseTransactions() error {
	return nil
}

func (pow *PowService) Start() error {
	pow.Mutex.Lock()
	defer pow.Mutex.Unlock()
	pow.started = true

	pow.blockPersistCompletedSubscriber = ledger.DefaultLedger.Blockchain.BCEvents.Subscribe(events.EventBlockPersistCompleted, pow.BlockPersistCompleted)
	//pow.newInventorySubscriber = ds.localNet.GetEvent("consensus").Subscribe(events.EventNewInventory,pow.LocalNodeNewInventory)

	fstBookking, _ := HexToBytes(config.Parameters.BookKeepers[0])
	acct, _ := pow.Client.GetDefaultAccount()
	dftPubkey, _ := acct.PubKey().EncodePoint(true)
	if IsEqualBytes(fstBookking, dftPubkey) {
		log.Trace(fstBookking)
		log.Trace(acct.PubKey().EncodePoint(true))
		pow.timer.Stop()
		pow.timer.Reset(GenBlockTime)
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

func (pow *PowService) LocalNodeNewInventory(v interface{}) {
	log.Debug()
	if inventory, ok := v.(Inventory); ok {
		if inventory.Type() == CONSENSUS {
			payload, ret := inventory.(*msg.ConsensusPayload)
			if ret == true {
				pow.NewConsensusPayload(payload)
			}
		}
	}
}

func (pow *PowService) InitializeConsensus(viewNum byte) error {
	log.Debug("[InitializeConsensus] Start InitializeConsensus.")
	pow.Mutex.Lock()
	defer pow.Mutex.Unlock()

	log.Debug("[InitializeConsensus] viewNum: ", viewNum)

	pow.timer.Stop()
	pow.timer.Reset(GenBlockTime << (viewNum + 1))

	return nil
}

//TODO: add invenory receiving
func (pow *PowService) NewConsensusPayload(payload *msg.ConsensusPayload) {
	/*
		log.Debug()
		pow.Mutex.Lock()
		defer pow.Mutex.Unlock()

		//if payload is not same height with current contex, ignore it
		if payload.Version != ContextVersion || payload.PrevHash != pow.context.PrevHash || payload.Height != pow.context.Height {
			return
		}

		message, err := DeserializeMessage(payload.Data)
		if err != nil {
			log.Error(fmt.Sprintf("DeserializeMessage failed: %s\n", err))
			return
		}

		if message.ViewNumber() != ds.context.ViewNumber && message.Type() != ChangeViewMsg {
			return
		}

		switch message.Type() {
		case ChangeViewMsg:
			if cv, ok := message.(*ChangeView); ok {
				ds.ChangeViewReceived(payload, cv)
			}
			break
		}
	*/
}

func NewPowService(client cl.Client, logDictionary string, localNet net.Neter) *PowService {
	log.Debug()

	pow := &PowService{
		Client: client,
		//	MsgBlock:      msgBlock,
		timer:         time.NewTimer(time.Second * 15),
		started:       false,
		localNet:      localNet,
		logDictionary: logDictionary,
	}

	if !pow.timer.Stop() {
		<-pow.timer.C
	}
	log.Debug()
	go pow.timerRoutine()
	//TODO add condition: if co-mining start
	go pow.ZMQServer()
	return pow
}

func (pow *PowService) CreateBookkeepingTransaction(nonce uint64) *tx.Transaction {
	log.Debug()
	//TODO: sysfee
	bookKeepingPayload := &payload.BookKeeping{
		Nonce: uint64(time.Now().UnixNano()),
	}
	return &tx.Transaction{
		TxType:         tx.BookKeeping,
		PayloadVersion: payload.BookKeepingPayloadVersion,
		Payload:        bookKeepingPayload,
		Attributes:     []*tx.TxAttribute{},
		UTXOInputs:     []*tx.UTXOTxInput{},
		BalanceInputs:  []*tx.BalanceTxInput{},
		Outputs:        []*tx.TxOutput{},
		Programs:       []*program.Program{},
	}
}

func (pow *PowService) Timeout() {

	blockData := &ledger.Blockdata{
		//Version: ContextVersion,
		Version: 0,
		//PrevBlockHash:    cxt.PrevHash,
		//TransactionsRoot: txRoot,
		//Timestamp:        cxt.Timestamp,
		//Height:           cxt.Height,
		//Bits:             0x1d00ffff,
		//Bits:           0x2007ffff,
		Bits: 0x1f07ffff,
		//ConsensusData:  uint64(Nonce),
		//NextBookKeeper: cxt.NextBookKeeper,
		Program: &program.Program{},
	}

	msgBlock := &ledger.Block{
		Blockdata:    blockData,
		Transactions: []*tx.Transaction{},
	}

	if 0 == pow.CollectTransactions(msgBlock) {
		pow.timer.Stop()
		pow.timer.Reset(GenBlockTime)
		return
	}
	generateStatus := pow.GenerateBlock(msgBlock)

	// push notifyed message into ZMQ
	if true == generateStatus {
		pow.ZMQPublish <- true
	}

	//begin to mine the block with POW
	if generateStatus && false && pow.SolveBlock(msgBlock) {
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

	pow.timer.Stop()
	pow.timer.Reset(GenBlockTime)
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

func miner(blkHdrHash []byte) []byte {
	return nil
}
