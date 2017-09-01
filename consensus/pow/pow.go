package pow

import (
	cl "DNA_POW/account"
	. "DNA_POW/common"
	"DNA_POW/common/config"
	"DNA_POW/common/log"
	"DNA_POW/core/contract/program"
	"DNA_POW/core/ledger"
	sig "DNA_POW/core/signature"
	tx "DNA_POW/core/transaction"
	"DNA_POW/core/transaction/payload"
	msg "DNA_POW/net/message"
	"DNA_POW/events"
	"DNA_POW/net"
	"time"
	"sync"
)

const (
	MINGENBLOCKTIME = 6
)

var GenBlockTime = (MINGENBLOCKTIME * time.Second)

type PowService struct {
	sync.Mutex
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

func (pow *PowService) Start() error {
	log.Debug()
	pow.started = true

	if config.Parameters.GenBlockTime > MINGENBLOCKTIME {
		GenBlockTime = time.Duration(config.Parameters.GenBlockTime) * time.Second
	} else {
		log.Warn("The Generate block time should be longer than 6 seconds, so set it to be 6.")
	}

	pow.blockPersistCompletedSubscriber = ledger.DefaultLedger.Blockchain.BCEvents.Subscribe(events.EventBlockPersistCompleted, pow.BlockPersistCompleted)
	pow.newInventorySubscriber = pow.localNet.GetEvent("consensus").Subscribe(events.EventNewInventory, pow.LocalNodeNewInventory)

	go pow.InitializeConsensus(0)
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
		pow.localNet.GetEvent("consensus").UnSubscribe(events.EventNewInventory, pow.newInventorySubscriber)
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

	pow.blockReceivedTime = time.Now()

	go pow.InitializeConsensus(0)
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
	log.Debug()
	pow.Mutex.Lock()
	defer pow.MutexUnlock()

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
}

func NewPowService(client cl.Client, logDictionary string, localNet net.Neter) *PowService {
	log.Debug()

	pow := &PowService{
		Client:        client,
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
	log.Debug()
	pow.context.contextMu.Lock()
	defer pow.context.contextMu.Unlock()
	if pow.timerHeight != pow.context.Height || pow.timeView != pow.context.ViewNumber {
		return
	}

	log.Info("Timeout: height: ", pow.timerHeight, " View: ", pow.timeView, " State: ", pow.context.GetStateDetail())

	if pow.context.State.HasFlag(Primary) && !pow.context.State.HasFlag(RequestSent) {
		//primary node send the prepare request
		log.Info("Send prepare request: height: ", pow.timerHeight, " View: ", pow.timeView, " State: ", pow.context.GetStateDetail())
		pow.context.State |= RequestSent
		if !pow.context.State.HasFlag(SignatureSent) {
			now := uint32(time.Now().Unix())
			header, _ := ledger.DefaultLedger.Blockchain.GetHeader(pow.context.PrevHash)

			//set context Timestamp
			blockTime := header.Blockdata.Timestamp + 1
			if blockTime > now {
				pow.context.Timestamp = blockTime
			} else {
				pow.context.Timestamp = now
			}

			pow.context.Nonce = GetNonce()
			transactionsPool := pow.localNet.GetTxnPool(true)
			//TODO: add policy
			//TODO: add max TX limitation

			txBookkeeping := pow.CreateBookkeepingTransaction(pow.context.Nonce)
			//add book keeping transaction first
			pow.context.Transactions = append(pow.context.Transactions, txBookkeeping)
			//add transactions from transaction pool
			for _, tx := range transactionsPool {
				pow.context.Transactions = append(pow.context.Transactions, tx)
			}
			pow.context.header = nil
			//build block and sign
			block := pow.context.MakeHeader()
			account, _ := pow.Client.GetAccount(pow.context.BookKeepers[pow.context.BookKeeperIndex]) //TODO: handle error
			pow.context.Signatures[pow.context.BookKeeperIndex], _ = sig.SignBySigner(block, account)
		}
		payload := pow.context.MakePrepareRequest()
		pow.SignAndRelay(payload)
		pow.timer.Stop()
		pow.timer.Reset(GenBlockTime << (pow.timeView + 1))
	} else if (pow.context.State.HasFlag(Primary) && pow.context.State.HasFlag(RequestSent)) || pow.context.State.HasFlag(Backup) {
		pow.RequestChangeView()
	}
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

}
