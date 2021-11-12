// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package blockchain

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	. "github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/common/log"
	. "github.com/elastos/Elastos.ELA/core/types"
	"github.com/elastos/Elastos.ELA/core/types/outputpayload"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	_ "github.com/elastos/Elastos.ELA/database/ffldb"
	"github.com/elastos/Elastos.ELA/events"
	"github.com/elastos/Elastos.ELA/utils"
)

const (
	RECEIVED                    string = "received"
	SENT                        string = "sent"
	MOVED                       string = "moved"
	ELA                         uint64 = 100000000
	DPOS_CHECK_POINT                   = 100
	CHECK_POINT_ROLLBACK_HEIGHT        = 100
)

type ProducerState byte

var SMALL_CROSS_TRANSFER_RPEFIX = []byte("SMALL_CROSS_TRANSFER")

type ProducerInfo struct {
	Payload   *payload.ProducerInfo
	RegHeight uint32
	Vote      Fixed64
}

type ChainStore struct {
	levelDB            IStore
	fflDB              IFFLDBChainStore
	currentBlockHeight uint32
	persistMutex       sync.Mutex
}

func NewChainStore(dataDir string, params *config.Params) (IChainStore, error) {
	db, err := NewLevelDB(filepath.Join(dataDir, "chain"))
	if err != nil {
		return nil, err
	}
	fflDB, err := NewChainStoreFFLDB(dataDir, params)
	if err != nil {
		return nil, err
	}
	s := &ChainStore{
		levelDB: db,
		fflDB:   fflDB,
	}

	return s, nil
}

func (c *ChainStore) CleanSmallCrossTransferTx(txHash Uint256) error {
	var key bytes.Buffer
	key.Write(SMALL_CROSS_TRANSFER_RPEFIX)
	key.Write(txHash.Bytes())
	return c.levelDB.Delete(key.Bytes())
}

func (c *ChainStore) SaveSmallCrossTransferTx(tx *Transaction) error {
	buf := new(bytes.Buffer)
	tx.Serialize(buf)
	var key bytes.Buffer
	key.Write(SMALL_CROSS_TRANSFER_RPEFIX)
	key.Write(tx.Hash().Bytes())
	c.levelDB.Put(key.Bytes(), buf.Bytes())
	return nil
}

func (c *ChainStore) GetSmallCrossTransferTxs() ([]*Transaction, error) {
	Iter := c.levelDB.NewIterator(SMALL_CROSS_TRANSFER_RPEFIX)
	txs := make([]*Transaction, 0)
	for Iter.Next() {
		val := Iter.Value()
		r := bytes.NewReader(val)
		tx := new(Transaction)
		if err := tx.Deserialize(r); err != nil {
			return nil, err
		}
		txs = append(txs, tx)
	}
	return txs, nil
}

func (c *ChainStore) GetSmallCrossTransferTx() ([]string, error) {
	Iter := c.levelDB.NewIterator(SMALL_CROSS_TRANSFER_RPEFIX)
	txns := make([]string, 0)
	for Iter.Next() {
		val := Iter.Value()
		txns = append(txns, hex.EncodeToString(val))
	}
	return txns, nil
}

func (c *ChainStore) CloseLeveldb() {
	c.levelDB.Close()
}

func (c *ChainStore) Close() {
	c.persistMutex.Lock()
	defer c.persistMutex.Unlock()
	if err := c.fflDB.Close(); err != nil {
		log.Error("fflDB close failed:", err)
	}
}

func (c *ChainStore) IsTxHashDuplicate(txID Uint256) bool {
	txn, _, err := c.fflDB.GetTransaction(txID)
	if err != nil || txn == nil {
		return false
	}

	return true
}

func (c *ChainStore) IsSidechainTxHashDuplicate(sidechainTxHash Uint256) bool {
	return c.GetFFLDB().IsTx3Exist(&sidechainTxHash)
}

func (c *ChainStore) IsSidechainReturnDepositTxHashDuplicate(sidechainReturnDepositTxHash Uint256) bool {
	return c.GetFFLDB().IsSideChainReturnDepositExist(&sidechainReturnDepositTxHash)
}

func (c *ChainStore) IsDoubleSpend(txn *Transaction) bool {
	if len(txn.Inputs) == 0 {
		return false
	}
	for i := 0; i < len(txn.Inputs); i++ {
		txID := txn.Inputs[i].Previous.TxID
		unspents, err := c.GetFFLDB().GetUnspent(txID)
		if err != nil {
			return true
		}
		findFlag := false
		for k := 0; k < len(unspents); k++ {
			if unspents[k] == txn.Inputs[i].Previous.Index {
				findFlag = true
				break
			}
		}
		if !findFlag {
			return true
		}
	}

	return false
}

func (c *ChainStore) RollbackBlock(b *Block, node *BlockNode,
	confirm *payload.Confirm, medianTimePast time.Time) error {
	now := time.Now()
	err := c.handleRollbackBlockTask(b, node, confirm, medianTimePast)
	tcall := float64(time.Now().Sub(now)) / float64(time.Second)
	log.Debugf("handle block rollback exetime: %g", tcall)
	return err
}

func (c *ChainStore) GetTransaction(txID Uint256) (*Transaction, uint32, error) {
	return c.fflDB.GetTransaction(txID)
}

func (c *ChainStore) GetProposalDraftDataByDraftHash(draftHash *Uint256) ([]byte, error) {
	return c.fflDB.GetProposalDraftDataByDraftHash(draftHash)
}

func (c *ChainStore) GetTxReference(tx *Transaction) (map[*Input]*Output, error) {
	if tx.TxType == RegisterAsset {
		return nil, nil
	}
	txOutputsCache := make(map[Uint256][]*Output)
	//UTXO input /  Outputs
	reference := make(map[*Input]*Output)
	// Key index，v UTXOInput
	for _, input := range tx.Inputs {
		txID := input.Previous.TxID
		index := input.Previous.Index
		if outputs, ok := txOutputsCache[txID]; ok {
			reference[input] = outputs[index]
		} else {
			transaction, _, err := c.GetTransaction(txID)

			if err != nil {
				return nil, errors.New("GetTxReference failed, previous transaction not found")
			}
			if int(index) >= len(transaction.Outputs) {
				return nil, errors.New("GetTxReference failed, refIdx out of range")
			}
			reference[input] = transaction.Outputs[index]
			txOutputsCache[txID] = transaction.Outputs
		}
	}
	return reference, nil
}

func (c *ChainStore) rollback(b *Block, node *BlockNode,
	confirm *payload.Confirm, medianTimePast time.Time) error {
	if err := c.fflDB.RollbackBlock(b, node, confirm, medianTimePast); err != nil {
		return err
	}
	atomic.StoreUint32(&c.currentBlockHeight, b.Height-1)

	return nil
}

func (c *ChainStore) persist(b *Block, node *BlockNode,
	confirm *payload.Confirm, medianTimePast time.Time) error {
	c.persistMutex.Lock()
	defer c.persistMutex.Unlock()

	if err := c.fflDB.SaveBlock(b, node, confirm, medianTimePast); err != nil {
		return err
	}
	return nil
}

func (c *ChainStore) GetFFLDB() IFFLDBChainStore {
	return c.fflDB
}

func (c *ChainStore) SaveBlock(b *Block, node *BlockNode,
	confirm *payload.Confirm, medianTimePast time.Time) error {
	log.Info("SaveBlock ", b.Height)

	now := time.Now()
	err := c.handlePersistBlockTask(b, node, confirm, medianTimePast)

	tcall := float64(time.Now().Sub(now)) / float64(time.Second)
	log.Debugf("handle block exetime: %g num transactions:%d",
		tcall, len(b.Transactions))
	return err
}

func (c *ChainStore) handleRollbackBlockTask(b *Block, node *BlockNode,
	confirm *payload.Confirm, medianTimePast time.Time) error {
	_, err := c.fflDB.GetBlock(b.Hash())
	if err != nil {
		log.Errorf("block %x can't be found", BytesToHexString(b.Hash().Bytes()))
		return err
	}
	return c.rollback(b, node, confirm, medianTimePast)
}

func (c *ChainStore) handlePersistBlockTask(b *Block, node *BlockNode,
	confirm *payload.Confirm, medianTimePast time.Time) error {
	if b.Header.Height <= c.currentBlockHeight {
		return errors.New("block height less than current block height")
	}

	return c.persistBlock(b, node, confirm, medianTimePast)
}

func (c *ChainStore) persistBlock(b *Block, node *BlockNode,
	confirm *payload.Confirm, medianTimePast time.Time) error {
	err := c.persist(b, node, confirm, medianTimePast)
	if err != nil {
		log.Fatal("[persistBlocks]: error to persist block:", err.Error())
		return err
	}

	atomic.StoreUint32(&c.currentBlockHeight, b.Height)
	return nil
}

func (c *ChainStore) GetConfirm(hash Uint256) (*payload.Confirm, error) {
	var confirm = new(payload.Confirm)
	prefix := []byte{byte(DATAConfirm)}
	confirmBytes, err := c.levelDB.Get(append(prefix, hash.Bytes()...))
	if err != nil {
		return nil, err
	}

	if err = confirm.Deserialize(bytes.NewReader(confirmBytes)); err != nil {
		return nil, err
	}

	return confirm, nil
}

func (c *ChainStore) GetHeight() uint32 {
	return atomic.LoadUint32(&c.currentBlockHeight)
}

func (c *ChainStore) SetHeight(height uint32) {
	atomic.StoreUint32(&c.currentBlockHeight, height)
}

var (
	MINING_ADDR  = Uint168{}
	ELA_ASSET, _ = Uint256FromHexString("b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a3")
)

type ChainStoreExtend struct {
	IChainStore
	IStore
	chain    *BlockChain
	taskChEx chan interface{}
	quitEx   chan chan bool
	mu       sync.RWMutex
	rp         chan bool
	checkPoint bool
}

func (c *ChainStoreExtend) AddTask(task interface{}) {
	c.taskChEx <- task
}

func NewChainStoreEx(chain *BlockChain, chainstore IChainStore, filePath string) (*ChainStoreExtend, error) {
	if !utils.FileExisted(filePath) {
		os.MkdirAll(filePath, 0700)
	}
	st, err := NewLevelDB(filePath)
	if err != nil {
		return nil, err
	}

	c := &ChainStoreExtend{
		IChainStore: chainstore,
		IStore:      st,
		chain:       chain,
		taskChEx:    make(chan interface{}, 100),
		quitEx:      make(chan chan bool, 1),
		mu:          sync.RWMutex{},
		rp:          make(chan bool, 1),
		checkPoint:  true,
	}
	StoreEx = c
	MemPoolEx = MemPool{
		c:    StoreEx,
		is_p: make(map[Uint256]bool),
		p:    make(map[string][]byte),
	}
	go c.loop()

	events.Subscribe(func(e *events.Event) {
		switch e.Type {
		case events.ETBlockConnected:
			b, ok := e.Data.(*Block)
			if ok {
				go StoreEx.AddTask(b)
			}
		case events.ETTransactionAccepted:
			tx, ok := e.Data.(*Transaction)
			if ok {
				go MemPoolEx.AppendToMemPool(tx)
			}
		}
	})
	return c, nil
}

func (c *ChainStoreExtend) Close() {

}

func (c *ChainStoreExtend) processVote(block *Block, voteTxHolder *map[string]VoteCategory) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	bestHeight, _ := c.GetBestHeightExt()
	if block.Height >= DPOS_CHECK_POINT {
		if block.Height > bestHeight {
			err := doProcessVote(block, voteTxHolder)
			if err != nil {
				return err
			}
		}
	}
	c.persistBestHeight(block.Height)
	return nil
}

func doProcessVote(block *Block, voteTxHolder *map[string]VoteCategory) error {
	for i := 0; i < len(block.Transactions); i++ {
		tx := block.Transactions[i]
		version := tx.Version
		txid, err := ReverseHexString(tx.Hash().String())
		vt := VoteCategory(0x00)
		if err != nil {
			return err
		}
		if version == 0x09 {
			vout := tx.Outputs
			for _, v := range vout {
				if v.Type == 0x01 && v.AssetID == *ELA_ASSET {
					outputPayload, ok := v.Payload.(*outputpayload.VoteOutput)
					if !ok || outputPayload == nil {
						continue
					}
					contents := outputPayload.Contents
					for _, cv := range contents {
						votetype := cv.VoteType
						votetypeStr := ""
						if votetype == 0x00 {
							votetypeStr = "Delegate"
						} else if votetype == 0x01 {
							votetypeStr = "CRC"
						} else if votetype == 0x02 {
							votetypeStr = "CRCProposal"
						} else if votetype == 0x03 {
							votetypeStr = "CRCImpeachment"
						} else {
							continue
						}

						if len(cv.CandidateVotes) > 0 {
							switch votetypeStr {
							case "Delegate":
								vt = vt | DPoS
							case "CRC":
								vt = vt | CRC
							case "CRCProposal":
								vt = vt | Proposal
							case "CRCImpeachment":
								vt = vt | Impeachment
							}
						}
					}
				}
			}
		}

		(*voteTxHolder)[txid] = vt
	}
	return nil
}

func (c *ChainStoreExtend) assembleRollbackBlock(rollbackStart uint32, blk *Block, blocks *[]*Block) error {
	for i := rollbackStart; i < blk.Height; i++ {
		blockHash, err := c.chain.GetBlockHash(i)
		if err != nil {
			return err
		}
		b, err := c.chain.GetBlockByHash(blockHash)
		if err != nil {
			return err
		}
		*blocks = append(*blocks, b)
	}
	return nil
}

func (c *ChainStoreExtend) persistTxHistory(blk *Block) error {
	var blocks []*Block
	var rollbackStart uint32 = 0
	if c.checkPoint {
		bestHeight, err := c.GetBestHeightExt()
		if err == nil && bestHeight > CHECK_POINT_ROLLBACK_HEIGHT {
			rollbackStart = bestHeight - CHECK_POINT_ROLLBACK_HEIGHT
		}
		c.assembleRollbackBlock(rollbackStart, blk, &blocks)
		c.checkPoint = false
	} else if blk.Height > DPOS_CHECK_POINT {
		rollbackStart = blk.Height - 5
		c.assembleRollbackBlock(rollbackStart, blk, &blocks)
	}

	blocks = append(blocks, blk)

	for _, block := range blocks {
		_, err := c.GetStoredHeightExt(block.Height)
		if err == nil {
			continue
		}

		voteTxHolder := make(map[string]VoteCategory)
		err = c.processVote(block, &voteTxHolder)
		if err != nil {
			return err
		}

		txs := block.Transactions
		txhs := make([]TransactionHistory, 0)
		for i := 0; i < len(txs); i++ {
			tx := txs[i]
			txid, err := ReverseHexString(tx.Hash().String())
			if err != nil {
				return err
			}
			var memo []byte
			var txType = tx.TxType
			for _, attr := range tx.Attributes {
				if attr.Usage == Memo {
					memo = attr.Data
				}
			}

			if txType == CoinBase {
				var to []Uint168
				hold := make(map[Uint168]Fixed64)
				txhsCoinBase := make([]TransactionHistory, 0)
				for _, vout := range tx.Outputs {
					if !ContainsU168(vout.ProgramHash, to) {
						to = append(to, vout.ProgramHash)
						txh := TransactionHistory{}
						txh.Address = vout.ProgramHash
						txh.Txid = tx.Hash()
						txh.Type = []byte(RECEIVED)
						txh.Time = uint64(block.Header.Timestamp)
						txh.Height = uint64(block.Height)
						txh.Fee = 0
						txh.Inputs = []Uint168{MINING_ADDR}
						txh.TxType = txType
						txh.Memo = memo

						hold[vout.ProgramHash] = vout.Value
						txhsCoinBase = append(txhsCoinBase, txh)
					} else {
						hold[vout.ProgramHash] += vout.Value
					}
				}
				for i := 0; i < len(txhsCoinBase); i++ {
					txhsCoinBase[i].Outputs = []Uint168{txhsCoinBase[i].Address}
					txhsCoinBase[i].Value = hold[txhsCoinBase[i].Address]
				}
				txhs = append(txhs, txhsCoinBase...)
			} else {

				isCrossTx := false
				if txType == TransferCrossChainAsset {
					isCrossTx = true
				}
				//if voteTxHolder[txid] == DPoS || voteTxHolder[txid] == CRC || voteTxHolder[txid] == DPoSAndCRC {
				//	txType = voteTxHolder[txid]
				//}
				voteType := voteTxHolder[txid]
				spend := make(map[Uint168]Fixed64)
				var totalInput int64 = 0
				var fromAddress []Uint168
				var toAddress []Uint168
				for _, input := range tx.Inputs {
					txid := input.Previous.TxID
					index := input.Previous.Index
					referTx, _, err := c.GetTransaction(txid)
					if err != nil {
						return err
					}
					address := referTx.Outputs[index].ProgramHash
					totalInput += int64(referTx.Outputs[index].Value)
					v, ok := spend[address]
					if ok {
						spend[address] = v + referTx.Outputs[index].Value
					} else {
						spend[address] = referTx.Outputs[index].Value
					}
					if !ContainsU168(address, fromAddress) {
						fromAddress = append(fromAddress, address)
					}
				}
				receive := make(map[Uint168]Fixed64)
				var totalOutput int64 = 0
				for _, output := range tx.Outputs {
					address, _ := output.ProgramHash.ToAddress()
					var valueCross int64
					if isCrossTx == true && (output.ProgramHash == MINING_ADDR || strings.Index(address, "X") == 0 || address == "4oLvT2") {
						payloadVersion := tx.PayloadVersion
						if payloadVersion > 0 {
							switch outputPayload:= output.Payload.(type){
								case *outputpayload.CrossChainOutput:
									valueCross = int64(outputPayload.TargetAmount)
								}
						} else {
							switch pl := tx.Payload.(type) {
							case *payload.TransferCrossChainAsset:
								valueCross = int64(pl.CrossChainAmounts[0])
							}
						}
					}
					if valueCross != 0 {
						totalOutput += valueCross
					} else {
						totalOutput += int64(output.Value)
					}
					v, ok := receive[output.ProgramHash]
					if ok {
						receive[output.ProgramHash] = v + output.Value
					} else {
						receive[output.ProgramHash] = output.Value
					}
					if !ContainsU168(output.ProgramHash, toAddress) {
						toAddress = append(toAddress, output.ProgramHash)
					}
				}
				fee := totalInput - totalOutput
				for addressReceiver, valueReceived := range receive {
					transferType := RECEIVED
					valueSpent, ok := spend[addressReceiver]
					var txValue Fixed64
					if ok {
						if valueSpent > valueReceived {
							txValue = valueSpent - valueReceived
							transferType = SENT
						} else {
							txValue = valueReceived - valueSpent
						}
						delete(spend, addressReceiver)
					} else {
						txValue = valueReceived
					}
					var realFee = uint64(fee)
					var txOutput = toAddress
					if transferType == RECEIVED {
						realFee = 0
						txOutput = []Uint168{addressReceiver}
					}

					if transferType == SENT {
						fromAddress = []Uint168{addressReceiver}
					}

					txh := TransactionHistory{}
					txh.Value = txValue
					txh.Address = addressReceiver
					txh.Inputs = fromAddress
					txh.TxType = txType
					txh.Txid = tx.Hash()
					txh.Height = uint64(block.Height)
					txh.Time = uint64(block.Header.Timestamp)
					txh.Type = []byte(transferType)
					txh.VoteType = voteType
					txh.Fee = Fixed64(realFee)
					if len(txOutput) > 10 {
						txh.Outputs = txOutput[0:10]
					} else {
						txh.Outputs = txOutput
					}
					txh.Memo = memo
					txhs = append(txhs, txh)
				}

				for addr, value := range spend {
					txh := TransactionHistory{}
					txh.Value = value
					txh.Address = addr
					txh.Inputs = []Uint168{addr}
					txh.TxType = txType
					txh.VoteType = voteType
					txh.Txid = tx.Hash()
					txh.Height = uint64(block.Height)
					txh.Time = uint64(block.Header.Timestamp)
					txh.Type = []byte(SENT)
					txh.Fee = Fixed64(fee)
					if len(toAddress) > 10 {
						txh.Outputs = toAddress[0:10]
					} else {
						txh.Outputs = toAddress
					}
					txh.Memo = memo
					txhs = append(txhs, txh)
				}
			}
		}
		c.persistTransactionHistory(txhs)
		c.persistStoredHeight(block.Height)
	}
	return nil
}

func (c *ChainStoreExtend) CloseEx() {
	closed := make(chan bool)
	c.quitEx <- closed
	<-closed
	log.Info("Extend chainStore shutting down")
}

func (c *ChainStoreExtend) loop() {
	for {
		select {
		case t := <-c.taskChEx:
			now := time.Now()
			switch kind := t.(type) {
			case *Block:
				err := c.persistTxHistory(kind)
				if err != nil {
					log.Errorf("Error persist transaction history %s", err.Error())
					os.Exit(-1)
					return
				}
				tcall := float64(time.Now().Sub(now)) / float64(time.Second)
				log.Debugf("handle SaveHistory time cost: %g num transactions:%d", tcall, len(kind.Transactions))
			}
		case closed := <-c.quitEx:
			closed <- true
			return
		}
	}
}

func (c *ChainStoreExtend) GetTxHistory(addr string, order string, timestamp uint64) interface{} {
	key := new(bytes.Buffer)
	key.WriteByte(byte(DataTxHistoryPrefix))
	var txhs interface{}
	if order == "desc" {
		txhs = make(TransactionHistorySorterDesc, 0)
	} else {
		txhs = make(TransactionHistorySorter, 0)
	}
	programHash, err := Uint168FromAddress(addr)
	if err != nil {
		return txhs
	}
	WriteVarBytes(key, programHash[:])
	iter := c.NewIterator(key.Bytes())
	defer iter.Release()

	for iter.Next() {
		val := new(bytes.Buffer)
		val.Write(iter.Value())
		txh := TransactionHistory{}
		txhd, _ := txh.Deserialize(val)
		if txhd.Type == "received" {
			if len(txhd.Inputs) > 10 {
				txhd.Inputs = txhd.Inputs[0:10]
			}
			txhd.Outputs = []string{txhd.Address}
		} else {
			txhd.Inputs = []string{txhd.Address}
			if len(txhd.Outputs) > 10 {
				txhd.Outputs = txhd.Outputs[0:10]
			}
		}

		if (timestamp > 0 && txhd.Time > timestamp) || timestamp == 0 {
			if order == "desc" {
				txhs = append(txhs.(TransactionHistorySorterDesc), *txhd)
			} else {
				txhs = append(txhs.(TransactionHistorySorter), *txhd)
			}
		}
	}

	txInMempool := MemPoolEx.GetMemPoolTx(programHash)
	for _, txh := range txInMempool {
		if order == "desc" {
			txhs = append(txhs.(TransactionHistorySorterDesc), txh)
		} else {
			txhs = append(txhs.(TransactionHistorySorter), txh)
		}
	}

	if order == "desc" {
		sort.Sort(txhs.(TransactionHistorySorterDesc))
	} else {
		sort.Sort(txhs.(TransactionHistorySorter))
	}
	return txhs
}

func (c *ChainStoreExtend) GetTxHistoryByLimit(addr, order string, skip, limit, timestamp uint32) (interface{}, int) {
	txhs := c.GetTxHistory(addr, order, uint64(timestamp))
	if order == "desc" {
		return txhs.(TransactionHistorySorterDesc).Filter(skip, limit), len(txhs.(TransactionHistorySorterDesc))
	} else {
		return txhs.(TransactionHistorySorter).Filter(skip, limit), len(txhs.(TransactionHistorySorter))
	}
}

func (c *ChainStoreExtend) GetBestHeightExt() (uint32, error) {
	key := new(bytes.Buffer)
	key.WriteByte(byte(DataBestHeightPrefix))
	data, err := c.Get(key.Bytes())
	if err != nil {
		return 0, err
	}
	buf := bytes.NewBuffer(data)
	return binary.LittleEndian.Uint32(buf.Bytes()), nil
}

func (c *ChainStoreExtend) GetStoredHeightExt(height uint32) (bool, error) {
	key := new(bytes.Buffer)
	key.WriteByte(byte(DataStoredHeightPrefix))
	WriteUint32(key, height)
	_, err := c.Get(key.Bytes())
	if err != nil {
		return false, err
	}
	return true, nil
}

func (c *ChainStoreExtend) LockDposData() {
	c.mu.RLock()
}

func (c *ChainStoreExtend) UnlockDposData() {
	c.mu.RUnlock()
}
