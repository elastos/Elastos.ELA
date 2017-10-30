package ChainStore

import (
	. "DNA_POW/common"
	"DNA_POW/common/config"
	"DNA_POW/common/log"
	"DNA_POW/common/serialization"
	. "DNA_POW/core/asset"
	"DNA_POW/core/auxpow"
	"DNA_POW/core/contract/program"
	. "DNA_POW/core/ledger"
	. "DNA_POW/core/store"
	. "DNA_POW/core/store/LevelDBStore"
	tx "DNA_POW/core/transaction"
	"DNA_POW/core/validation"
	"DNA_POW/crypto"
	"DNA_POW/events"
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"sync"
	"time"
)

const (
	HeaderHashListCount  = 2000
	CleanCacheThreshold  = 2
	TaskChanCap          = 4
	MaxTimeOffsetSeconds = 2 * 60 * 60
)

var (
	ErrDBNotFound = errors.New("leveldb: not found")
)

type persistTask interface{}
type persistHeaderTask struct {
	header *Header
	reply  chan bool
}

type rollbackBlockTask struct {
	blockHash Uint256
	reply     chan bool
}
type persistBlockTask struct {
	block  *Block
	ledger *Ledger
	reply  chan bool
}

type ChainStore struct {
	IStore

	taskCh chan persistTask
	quit   chan chan bool

	mu          sync.RWMutex // guard the following var
	headerIndex map[uint32]Uint256
	blockCache  map[Uint256]*Block
	headerCache map[Uint256]*Header

	currentBlockHeight uint32
	storedHeaderCount  uint32
	ledger             *Ledger
}

func NewLedgerStore() (ILedgerStore, error) {
	// TODO: read config file decide which db to use.
	cs, err := NewChainStore("Chain")
	if err != nil {
		return nil, err
	}

	return cs, nil
}

func NewChainStore(file string) (*ChainStore, error) {

	st, err := NewStore(file)
	if err != nil {
		return nil, err
	}

	chain := &ChainStore{
		IStore:             st,
		headerIndex:        map[uint32]Uint256{},
		blockCache:         map[Uint256]*Block{},
		headerCache:        map[Uint256]*Header{},
		currentBlockHeight: 0,
		storedHeaderCount:  0,
		taskCh:             make(chan persistTask, TaskChanCap),
		quit:               make(chan chan bool, 1),
	}

	go chain.loop()

	return chain, nil
}

func NewStore(file string) (IStore, error) {
	ldbs, err := NewLevelDBStore(file)

	return ldbs, err
}

func (self *ChainStore) Close() {
	closed := make(chan bool)
	self.quit <- closed
	<-closed

	self.Close()
}

func (self *ChainStore) loop() {
	for {
		select {
		case t := <-self.taskCh:
			now := time.Now()
			switch task := t.(type) {
			case *persistHeaderTask:
				self.handlePersistHeaderTask(task.header)
				task.reply <- true
				tcall := float64(time.Now().Sub(now)) / float64(time.Second)
				log.Debugf("handle header exetime: %g \n", tcall)

			case *persistBlockTask:
				self.handlePersistBlockTask(task.block, task.ledger)
				task.reply <- true
				tcall := float64(time.Now().Sub(now)) / float64(time.Second)
				log.Debugf("handle block exetime: %g num transactions:%d \n", tcall, len(task.block.Transactions))
			case *rollbackBlockTask:
				self.handleRollbackBlockTask(task.blockHash)
				task.reply <- true
				tcall := float64(time.Now().Sub(now)) / float64(time.Second)
				log.Debugf("handle block rollback exetime: %g \n", tcall)
			}

		case closed := <-self.quit:
			closed <- true
			return
		}
	}
}

// can only be invoked by backend write goroutine
//func (self *ChainStore) clearCache() {
//	self.mu.Lock()
//	defer self.mu.Unlock()
//
//	currBlockHeight := self.currentBlockHeight
//	for hash, header := range self.headerCache {
//		if header.Blockdata.Height+CleanCacheThreshold < currBlockHeight {
//			delete(self.headerCache, hash)
//		}
//	}
//}

func (bd *ChainStore) InitLedgerStoreWithGenesisBlock(genesisBlock *Block, defaultBookKeeper []*crypto.PubKey) (uint32, error) {
	prefix := []byte{byte(CFG_Version)}
	version, err := bd.Get(prefix)
	if err != nil {
		version = []byte{0x00}
	}

	log.Trace("version: ", version)
	if version[0] == 0x00 {
		// batch delete old data
		bd.NewBatch()
		iter := bd.NewIterator(nil)
		for iter.Next() {
			bd.BatchDelete(iter.Key())
		}
		iter.Release()

		err := bd.BatchCommit()
		if err != nil {
			return 0, err
		}

		// persist genesis block
		bd.persist(genesisBlock)

		// put version to db
		err = bd.Put(prefix, []byte{0x01})
		if err != nil {
			return 0, err
		}

	}

	log.Trace("version2: ", version)
	// GenesisBlock should exist in chain
	// Or the bookkeepers are not consistent with the chain
	hash := genesisBlock.Hash()
	log.Trace("hash: ", hash)
	if !bd.IsBlockInStore(hash) {
		log.Trace("bookkeepers are not consistent with the chain")
		return 0, errors.New("bookkeepers are not consistent with the chain")
	}
	bd.ledger.Blockchain.GenesisHash = hash
	log.Trace("hash2: ", hash)

	// Get Current Block
	currentBlockPrefix := []byte{byte(SYS_CurrentBlock)}
	data, err := bd.Get(currentBlockPrefix)
	if err != nil {
		return 0, err
	}

	r := bytes.NewReader(data)
	var blockHash Uint256
	blockHash.Deserialize(r)
	bd.currentBlockHeight, err = serialization.ReadUint32(r)
	log.Tracef("blockHash: %x\n", blockHash.ToArray())
	endHeight := bd.currentBlockHeight

	startHeight := uint32(0)
	if endHeight > MinMemoryNodes {
		startHeight = endHeight - MinMemoryNodes
	}
	log.Tracef("startHeight: %x, endHeight: %x", startHeight, endHeight)

	for start := startHeight; start <= endHeight; start++ {
		hash, err := bd.GetBlockHash(start)
		if err != nil {
			return 0, err
		}
		header, err := bd.GetHeader(hash)
		if err != nil {
			return 0, err
		}
		node, err := bd.ledger.Blockchain.LoadBlockNode(header.Blockdata, &hash)
		if err != nil {
			return 0, err
		}

		// This node is now the end of the best chain.
		bd.ledger.Blockchain.BestChain = node

	}
	bd.ledger.Blockchain.DumpState()

	return bd.currentBlockHeight, nil

}

func (bd *ChainStore) InitLedgerStore(l *Ledger) error {
	// TODO: InitLedgerStore
	bd.ledger = l
	return nil
}

func (bd *ChainStore) IsTxHashDuplicate(txhash Uint256) bool {
	prefix := []byte{byte(DATA_Transaction)}
	_, err_get := bd.Get(append(prefix, txhash.ToArray()...))
	if err_get != nil {
		return false
	} else {
		return true
	}
}

func (bd *ChainStore) IsDoubleSpend(tx *tx.Transaction) bool {
	if len(tx.UTXOInputs) == 0 {
		return false
	}

	unspentPrefix := []byte{byte(IX_Unspent)}
	for i := 0; i < len(tx.UTXOInputs); i++ {
		txhash := tx.UTXOInputs[i].ReferTxID
		unspentValue, err_get := bd.Get(append(unspentPrefix, txhash.ToArray()...))
		if err_get != nil {
			return true
		}

		unspents, _ := GetUint16Array(unspentValue)
		findFlag := false
		for k := 0; k < len(unspents); k++ {
			if unspents[k] == tx.UTXOInputs[i].ReferTxOutputIndex {
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

func (bd *ChainStore) GetBlockHash(height uint32) (Uint256, error) {
	queryKey := bytes.NewBuffer(nil)
	queryKey.WriteByte(byte(DATA_BlockHash))
	err := serialization.WriteUint32(queryKey, height)

	if err != nil {
		return Uint256{}, err
	}
	blockHash, err_get := bd.Get(queryKey.Bytes())
	if err_get != nil {
		//TODO: implement error process
		return Uint256{}, err_get
	}
	blockHash256, err_parse := Uint256ParseFromBytes(blockHash)
	if err_parse != nil {
		return Uint256{}, err_parse
	}

	return blockHash256, nil
}

func (bd *ChainStore) GetCurrentBlockHash() Uint256 {
	hash, err := bd.GetBlockHash(bd.currentBlockHeight)
	if err != nil {
		return Uint256{}
	}

	return hash
}

func (bd *ChainStore) verifyHeader(header *Header) bool {
	prevHeader, _ := bd.GetHeader(header.Blockdata.PrevBlockHash)

	if prevHeader == nil {
		log.Error("[verifyHeader] failed, not found prevHeader.")
		return false
	}

	if prevHeader.Blockdata.Height+1 != header.Blockdata.Height {
		log.Error("[verifyHeader] failed, prevHeader.Height + 1 != header.Height")
		return false
	}

	if prevHeader.Blockdata.Timestamp >= header.Blockdata.Timestamp {
		log.Error("[verifyHeader] failed, prevHeader.Timestamp >= header.Timestamp")
		return false
	}

	flag, err := validation.VerifySignableData(header.Blockdata)
	if flag == false || err != nil {
		log.Error("[verifyHeader] failed, VerifySignableData failed.")
		log.Error(err)
		return false
	}

	return true
}

func (bd *ChainStore) powVerifyHeader(header *Header) bool {
	prevHeader, _ := bd.GetHeader(header.Blockdata.PrevBlockHash)

	if prevHeader == nil {
		log.Error("[verifyHeader] failed, not found prevHeader.")
		return false
	}

	// Fixme Consider the forking case
	if prevHeader.Blockdata.Height+1 != header.Blockdata.Height {
		log.Error("[verifyHeader] failed, prevHeader.Height + 1 != header.Height")
		return false
	}

	if prevHeader.Blockdata.Timestamp >= header.Blockdata.Timestamp {
		log.Error("[verifyHeader] failed, prevHeader.Timestamp >= header.Timestamp")
		return false
	}

	//flag, err := validation.VerifySignableData(header.Blockdata)
	//if flag == false || err != nil {
	//	log.Error("[verifyHeader] failed, VerifySignableData failed.")
	//	log.Error(err)
	//	return false
	//}

	//pow verify

	return true
}

func (self *ChainStore) AddHeaders(headers []Header, ledger *Ledger) error {

	sort.Slice(headers, func(i, j int) bool {
		return headers[i].Blockdata.Height < headers[j].Blockdata.Height
	})

	for i := 0; i < len(headers); i++ {
		reply := make(chan bool)
		self.taskCh <- &persistHeaderTask{header: &headers[i], reply: reply}
		<-reply
	}

	return nil

}

func (db *ChainStore) RollbackBlock(blockHash Uint256) error {

	reply := make(chan bool)
	db.taskCh <- &rollbackBlockTask{blockHash: blockHash, reply: reply}
	<-reply

	return nil
}

func (bd *ChainStore) GetHeader(hash Uint256) (*Header, error) {
	var h *Header = new(Header)

	h.Blockdata = new(Blockdata)
	h.Blockdata.Program = new(program.Program)

	prefix := []byte{byte(DATA_Header)}
	log.Trace("GetHeader Data:", hash.ToArray())
	data, err_get := bd.Get(append(prefix, hash.ToArray()...))
	//log.Debug( "Get Header Data: %x\n",  data )
	if err_get != nil {
		//TODO: implement error process
		return nil, err_get
	}

	r := bytes.NewReader(data)

	// first 8 bytes is sys_fee
	sysfee, err := serialization.ReadUint64(r)
	if err != nil {
		return nil, err
	}
	log.Trace(fmt.Sprintf("sysfee: %d\n", sysfee))

	// Deserialize block data
	err = h.Deserialize(r)
	if err != nil {
		return nil, err
	}

	return h, err
}

func (bd *ChainStore) PersistAsset(assetId Uint256, asset *Asset) error {
	w := bytes.NewBuffer(nil)

	asset.Serialize(w)

	// generate key
	assetKey := bytes.NewBuffer(nil)
	// add asset prefix.
	assetKey.WriteByte(byte(ST_Info))
	// contact asset id
	assetId.Serialize(assetKey)

	log.Debug(fmt.Sprintf("asset key: %x\n", assetKey))

	// PUT VALUE
	err := bd.BatchPut(assetKey.Bytes(), w.Bytes())
	if err != nil {
		return err
	}

	return nil
}

func (bd *ChainStore) GetAsset(hash Uint256) (*Asset, error) {
	log.Debug(fmt.Sprintf("GetAsset Hash: %x\n", hash))

	asset := new(Asset)

	prefix := []byte{byte(ST_Info)}
	data, err_get := bd.Get(append(prefix, hash.ToArray()...))

	log.Debug(fmt.Sprintf("GetAsset Data: %x\n", data))
	if err_get != nil {
		//TODO: implement error process
		return nil, err_get
	}

	r := bytes.NewReader(data)
	asset.Deserialize(r)

	return asset, nil
}

func (bd *ChainStore) GetTransaction(hash Uint256) (*tx.Transaction, error) {
	log.Debugf("GetTransaction Hash: %x\n", hash)

	t := new(tx.Transaction)
	err := bd.getTx(t, hash)

	if err != nil {
		return nil, err
	}

	return t, nil
}

func (bd *ChainStore) getTx(tx *tx.Transaction, hash Uint256) error {
	prefix := []byte{byte(DATA_Transaction)}
	tHash, err_get := bd.Get(append(prefix, hash.ToArray()...))
	if err_get != nil {
		//TODO: implement error process
		return err_get
	}

	r := bytes.NewReader(tHash)

	// get height
	_, err := serialization.ReadUint32(r)
	if err != nil {
		return err
	}

	// Deserialize Transaction
	err = tx.Deserialize(r)

	return err
}

func (bd *ChainStore) PersistTransaction(tx *tx.Transaction, height uint32) error {
	//////////////////////////////////////////////////////////////
	// generate key with DATA_Transaction prefix
	txhash := bytes.NewBuffer(nil)
	// add transaction header prefix.
	txhash.WriteByte(byte(DATA_Transaction))
	// get transaction hash
	txHashValue := tx.Hash()
	txHashValue.Serialize(txhash)
	log.Debug(fmt.Sprintf("transaction header + hash: %x\n", txhash))

	// generate value
	w := bytes.NewBuffer(nil)
	serialization.WriteUint32(w, height)
	tx.Serialize(w)
	log.Debug(fmt.Sprintf("transaction tx data: %x\n", w))

	// put value
	err := bd.BatchPut(txhash.Bytes(), w.Bytes())
	if err != nil {
		return err
	}

	return nil
}

func (bd *ChainStore) GetBlock(hash Uint256) (*Block, error) {
	var b *Block = new(Block)

	b.Blockdata = new(Blockdata)
	b.Blockdata.Program = new(program.Program)

	prefix := []byte{byte(DATA_Header)}
	bHash, err_get := bd.Get(append(prefix, hash.ToArray()...))
	if err_get != nil {
		//TODO: implement error process
		return nil, err_get
	}

	r := bytes.NewReader(bHash)

	// first 8 bytes is sys_fee
	_, err := serialization.ReadUint64(r)
	if err != nil {
		return nil, err
	}

	// Deserialize block data
	err = b.FromTrimmedData(r)
	if err != nil {
		return nil, err
	}

	// Deserialize transaction
	for i := 0; i < len(b.Transactions); i++ {
		err = bd.getTx(b.Transactions[i], b.Transactions[i].Hash())
		if err != nil {
			return nil, err
		}
	}

	return b, nil
}

func (self *ChainStore) GetBookKeeperList() ([]*crypto.PubKey, []*crypto.PubKey, error) {
	prefix := []byte{byte(SYS_CurrentBookKeeper)}
	bkListValue, err_get := self.Get(prefix)
	if err_get != nil {
		return nil, nil, err_get
	}

	r := bytes.NewReader(bkListValue)

	// first 1 bytes is length of list
	currCount, err := serialization.ReadUint8(r)
	if err != nil {
		return nil, nil, err
	}

	var currBookKeeper = make([]*crypto.PubKey, currCount)
	for i := uint8(0); i < currCount; i++ {
		bk := new(crypto.PubKey)
		err := bk.DeSerialize(r)
		if err != nil {
			return nil, nil, err
		}

		currBookKeeper[i] = bk
	}

	nextCount, err := serialization.ReadUint8(r)
	if err != nil {
		return nil, nil, err
	}

	var nextBookKeeper = make([]*crypto.PubKey, nextCount)
	for i := uint8(0); i < nextCount; i++ {
		bk := new(crypto.PubKey)
		err := bk.DeSerialize(r)
		if err != nil {
			return nil, nil, err
		}

		nextBookKeeper[i] = bk
	}

	return currBookKeeper, nextBookKeeper, nil
}

func (db *ChainStore) rollback(b *Block) error {
	db.BatchInit()
	db.RollbackTrimemedBlock(b)
	db.RollbackBlockHash(b)
	db.RollbackTransactions(b)
	db.RollbackUnspendUTXOs(b)
	db.RollbackUnspend(b)
	db.RollbackCurrentBlock(b)
	db.BatchFinish()

	db.ledger.Blockchain.UpdateBestHeight(b.Blockdata.Height - 1)
	db.mu.Lock()
	db.currentBlockHeight = b.Blockdata.Height - 1
	db.mu.Unlock()

	db.ledger.Blockchain.BCEvents.Notify(events.EventRollbackTransaction, b)

	return nil
}

func (db *ChainStore) persist(b *Block) error {
	//unspents := make(map[Uint256][]uint16)

	db.BatchInit()
	db.PersistTrimmedBlock(b)
	db.PersistBlockHash(b)
	db.PersistTransactions(b)
	db.PersistUnspendUTXOs(b)
	db.PersistUnspend(b)
	db.PersistCurrentBlock(b)
	db.BatchFinish()

	return nil
}

// can only be invoked by backend write goroutine
func (bd *ChainStore) addHeader(header *Header) {

	log.Debugf("addHeader(), Height=%d\n", header.Blockdata.Height)

	hash := header.Blockdata.Hash()

	bd.mu.Lock()
	bd.headerCache[header.Blockdata.Hash()] = header
	bd.headerIndex[header.Blockdata.Height] = hash
	bd.mu.Unlock()

	log.Debug("[addHeader]: finish, header height:", header.Blockdata.Height)
}

func (self *ChainStore) handlePersistHeaderTask(header *Header) {
	if header.Blockdata.Height != uint32(len(self.headerIndex)) {
		return
	}

	//if config.Parameters.ConsensusType == "pow" {
	if !self.powVerifyHeader(header) {
		return
	}

	self.addHeader(header)
	//} else {
	//	if !self.verifyHeader(header) {
	//		return
	//	}
	//	self.addHeader(header)
	//}
}

func (self *ChainStore) SaveBlock(b *Block, ledger *Ledger) error {
	log.Debug("SaveBlock()")
	//log.Trace("validation.PowVerifyBlock(b, ledger, false)")
	//err := validation.PowVerifyBlock(b, ledger, false)
	//if err != nil {
	//	log.Error("PowVerifyBlock error!")
	//	return err
	//}
	//log.Trace("validation.PowVerifyBlock(b, ledger, false)222222")

	reply := make(chan bool)
	self.taskCh <- &persistBlockTask{block: b, ledger: ledger, reply: reply}
	<-reply

	return nil
}

func (db *ChainStore) PowCheckBlockSanity(block *Block, powLimit *big.Int, timeSource MedianTimeSource) error {
	header := block.Blockdata
	isAuxPow := config.Parameters.PowConfiguration.CoMining
	if isAuxPow && !header.AuxPow.Check(header.Hash(), auxpow.AuxPowChainID) {
		return errors.New("[PowCheckBlockSanity] block check proof is failed")
	}
	if validation.CheckProofOfWork(header, powLimit, isAuxPow) != nil {
		return errors.New("[PowCheckBlockSanity] block check proof is failed.")
	}

	//TODO A block timestamp must not have a greater precision than one second.

	// Ensure the block time is not too far in the future.
	maxTimestamp := timeSource.AdjustedTime().Add(time.Second * MaxTimeOffsetSeconds)
	tempTime := time.Unix(int64(header.Timestamp), 0)
	if tempTime.After(maxTimestamp) {
		return errors.New("[PowCheckBlockSanity] block timestamp of is too far in the future")
	}

	// A block must have at least one transaction.
	numTx := len(block.Transactions)
	if numTx == 0 {
		return errors.New("[PowCheckBlockSanity]  block does not contain any transactions")
	}

	// A block must not have more transactions than the max block payload.
	//TODO
	//config.Parameters.MaxTxInBlock = 100
	if numTx > config.Parameters.MaxTxInBlock {
		return errors.New("[PowCheckBlockSanity]  block contains too many transactions")
	}

	// TODO The first transaction in a block must be a coinbase.
	//transactions := block.Transactions
	//if !IsCoinBase(transactions[0]) {
	//	return ruleError(ErrFirstTxNotCoinbase, "first transaction in "+
	//		"block is not a coinbase")
	//}

	//// A block must not have more than one coinbase.
	//for i, tx := range transactions[1:] {
	//	if IsCoinBase(tx) {
	//		str := fmt.Sprintf("block contains second coinbase at "+
	//			"index %d", i)
	//		return ruleError(ErrMultipleCoinbases, str)
	//	}
	//}

	// Do some preliminary checks on each transaction to ensure they are
	// sane before continuing.
	//for _, tx := range transactions {
	//	err := CheckTransactionSanity(tx)
	//	if err != nil {
	//		return err
	//	}
	//}

	// Build merkle tree and ensure the calculated merkle root matches the
	// entry in the block header.  This also has the effect of caching all
	// of the transaction hashes in the block to speed up future hash
	// checks.  Bitcoind builds the tree here and checks the merkle root
	// after the following checks, but there is no reason not to check the
	// merkle root matches here.
	//merkles := BuildMerkleTreeStore(block.Transactions())
	//calculatedMerkleRoot := merkles[len(merkles)-1]
	//if !header.MerkleRoot.IsEqual(calculatedMerkleRoot) {
	//	str := fmt.Sprintf("block merkle root is invalid - block "+
	//		"header indicates %v, but calculated value is %v",
	//		header.MerkleRoot, calculatedMerkleRoot)
	//	return ruleError(ErrBadMerkleRoot, str)
	//}

	// Check for duplicate transactions.  This check will be fairly quick
	// since the transaction hashes are already cached due to building the
	// merkle tree above.
	//existingTxHashes := make(map[wire.ShaHash]struct{})
	//for _, tx := range transactions {
	//	hash := tx.Sha()
	//	if _, exists := existingTxHashes[*hash]; exists {
	//		str := fmt.Sprintf("block contains duplicate "+
	//			"transaction %v", hash)
	//		return ruleError(ErrDuplicateTx, str)
	//	}
	//	existingTxHashes[*hash] = struct{}{}
	//}

	//// The number of signature operations must be less than the maximum
	//// allowed per block.
	//totalSigOps := 0
	//for _, tx := range transactions {
	//	// We could potentially overflow the accumulator so check for
	//	// overflow.
	//	lastSigOps := totalSigOps
	//	totalSigOps += CountSigOps(tx)
	//	if totalSigOps < lastSigOps || totalSigOps > MaxSigOpsPerBlock {
	//		str := fmt.Sprintf("block contains too many signature "+
	//			"operations - got %v, max %v", totalSigOps,
	//			MaxSigOpsPerBlock)
	//		return ruleError(ErrTooManySigOps, str)
	//	}
	//}

	return nil
}

func (db *ChainStore) PowCheckBlockContext(block *Block, prevNode *BlockNode, ledger *Ledger) error {
	// The genesis block is valid by definition.
	if prevNode == nil {
		return nil
	}

	// Perform all block header related validation checks.
	//header := block.Blockdata
	// Ensure the difficulty specified in the block header matches
	// the calculated difficulty based on the previous block and
	// difficulty retarget rules.

	//if prevHeader.Blockdata.Height+1 != bd.Height {
	//return NewDetailErr(errors.New("[BlockValidator] error"), ErrNoCode, "[BlockValidator], block height is incorrect.")
	//}
	//expectedDifficulty, err := b.CalcNextRequiredDifficulty(prevNode,
	//	header.Timestamp)
	//if err != nil {
	//	return err
	//}
	//blockDifficulty := header.Bits
	//if blockDifficulty != expectedDifficulty {
	//	str := "block difficulty of %d is not the expected value of %d"
	//	str = fmt.Sprintf(str, blockDifficulty, expectedDifficulty)
	//	return ruleError(ErrUnexpectedDifficulty, str)
	//}

	// Ensure the timestamp for the block header is after the
	// median time of the last several blocks (medianTimeBlocks).
	//medianTime, err := b.calcPastMedianTime(prevNode)
	//if err != nil {
	//	log.Errorf("calcPastMedianTime: %v", err)
	//	return err
	//}
	//if !header.Timestamp.After(medianTime) {
	//	str := "block timestamp of %v is not after expected %v"
	//	str = fmt.Sprintf(str, header.Timestamp, medianTime)
	//	return ruleError(ErrTimeTooOld, str)
	//}

	// The height of this block is one more than the referenced
	// previous block.
	//blockHeight := prevNode.Height + 1

	// Ensure all transactions in the block are finalized.
	//for _, tx := range block.Transactions() {
	//	if !IsFinalizedTransaction(tx, blockHeight,
	//		header.Timestamp) {

	//		str := fmt.Sprintf("block contains unfinalized "+
	//			"transaction %v", tx.Sha())
	//		return ruleError(ErrUnfinalizedTx, str)
	//	}
	//}

	// Ensure coinbase starts with serialized block heights for
	// blocks whose version is the serializedHeightVersion or newer
	// once a majority of the network has upgraded.  This is part of
	// BIP0034.
	//if ShouldHaveSerializedBlockHeight(header) &&
	//	b.isMajorityVersion(serializedHeightVersion, prevNode,
	//		b.chainParams.BlockEnforceNumRequired) {

	//	coinbaseTx := block.Transactions()[0]
	//	err := checkSerializedHeight(coinbaseTx, blockHeight)
	//	if err != nil {
	//		return err
	//	}
	//}

	return nil
}

func (db *ChainStore) handleRollbackBlockTask(blockHash Uint256) {
	block, err := db.GetBlock(blockHash)
	if err != nil {
		log.Errorf("block %x can't be found", BytesToHexString(blockHash.ToArray()))
		return
	}
	db.rollback(block)
}

func (self *ChainStore) handlePersistBlockTask(b *Block, ledger *Ledger) {

	log.Trace(b.Blockdata.Height, " ", self.currentBlockHeight)
	if b.Blockdata.Height <= self.currentBlockHeight {
		return
	}

	//	self.mu.Lock()
	//self.blockCache[b.Hash()] = b
	//self.mu.Unlock()

	//log.Trace(b.Blockdata.Height)
	//log.Trace(b.Blockdata)
	//log.Trace(b.Transactions[0])
	//if b.Blockdata.Height < uint32(len(self.headerIndex)) {
	self.persistBlocks(b, ledger)

	//self.NewBatch()
	//storedHeaderCount := self.storedHeaderCount
	//for self.currentBlockHeight-storedHeaderCount >= HeaderHashListCount {
	//	hashBuffer := new(bytes.Buffer)
	//	serialization.WriteVarUint(hashBuffer, uint64(HeaderHashListCount))
	//	var hashArray []byte
	//	for i := 0; i < HeaderHashListCount; i++ {
	//		index := storedHeaderCount + uint32(i)
	//		thash := self.headerIndex[index]
	//		thehash := thash.ToArray()
	//		hashArray = append(hashArray, thehash...)
	//	}
	//	hashBuffer.Write(hashArray)

	//	hhlPrefix := bytes.NewBuffer(nil)
	//	hhlPrefix.WriteByte(byte(IX_HeaderHashList))
	//	serialization.WriteUint32(hhlPrefix, storedHeaderCount)

	//	self.BatchPut(hhlPrefix.Bytes(), hashBuffer.Bytes())
	//	storedHeaderCount += HeaderHashListCount
	//}

	//err := self.BatchCommit()
	//if err != nil {
	//	log.Error("failed to persist header hash list:", err)
	//	return
	//}
	//self.mu.Lock()
	//self.storedHeaderCount = storedHeaderCount
	//self.mu.Unlock()
	//self.clearCache()
	//}
}

func (bd *ChainStore) persistBlocks(block *Block, ledger *Ledger) {
	//stopHeight := uint32(len(bd.headerIndex))
	//for h := bd.currentBlockHeight + 1; h <= stopHeight; h++ {
	//hash := bd.headerIndex[h]
	//block, ok := bd.blockCache[hash]
	//if !ok {
	//	break
	//}
	//log.Trace(block.Blockdata)
	//log.Trace(block.Transactions[0])
	err := bd.persist(block)
	if err != nil {
		log.Fatal("[persistBlocks]: error to persist block:", err.Error())
		return
	}

	// PersistCompleted event
	//ledger.Blockchain.BlockHeight = block.Blockdata.Height
	ledger.Blockchain.UpdateBestHeight(block.Blockdata.Height)
	bd.mu.Lock()
	bd.currentBlockHeight = block.Blockdata.Height
	bd.mu.Unlock()
	log.Trace("persist height", bd.currentBlockHeight)
	log.Trace("persist height2", bd.ledger.Blockchain.BlockHeight)

	ledger.Blockchain.BCEvents.Notify(events.EventBlockPersistCompleted, block)
	//log.Tracef("The latest block height:%d, block hash: %x", block.Blockdata.Height, hash)
	//}

}

func (bd *ChainStore) BlockInCache(hash Uint256) bool {
	//TODO mutex
	//_, ok := bd.ledger.Blockchain.Index[hash]
	//return ok
	return false
}

func (bd *ChainStore) GetUnspent(txid Uint256, index uint16) (*tx.TxOutput, error) {
	if ok, _ := bd.ContainsUnspent(txid, index); ok {
		Tx, err := bd.GetTransaction(txid)
		if err != nil {
			return nil, err
		}

		return Tx.Outputs[index], nil
	}

	return nil, errors.New("[GetUnspent] NOT ContainsUnspent.")
}

func (bd *ChainStore) ContainsUnspent(txid Uint256, index uint16) (bool, error) {
	unspentPrefix := []byte{byte(IX_Unspent)}
	unspentValue, err_get := bd.Get(append(unspentPrefix, txid.ToArray()...))

	if err_get != nil {
		return false, err_get
	}

	unspentArray, err_get := GetUint16Array(unspentValue)
	if err_get != nil {
		return false, err_get
	}

	for i := 0; i < len(unspentArray); i++ {
		if unspentArray[i] == index {
			return true, nil
		}
	}

	return false, nil
}

func (bd *ChainStore) GetCurrentHeaderHash() Uint256 {
	bd.mu.RLock()
	defer bd.mu.RUnlock()
	return bd.GetCurrentBlockHash()
	//return bd.headerIndex[uint32(len(bd.headerIndex)-1)]
}

func (bd *ChainStore) GetHeaderHashByHeight(height uint32) Uint256 {
	bd.mu.RLock()
	defer bd.mu.RUnlock()

	//return bd.headerIndex[height]
	hash, err := bd.GetBlockHash(height)
	if err != nil {
		return Uint256{}
	}
	return hash
}

func (bd *ChainStore) GetHeaderHeight() uint32 {
	bd.mu.RLock()
	defer bd.mu.RUnlock()

	return bd.currentBlockHeight

	//return uint32(len(bd.headerIndex) - 1)

}

func (bd *ChainStore) GetHeight() uint32 {
	bd.mu.RLock()
	defer bd.mu.RUnlock()

	return bd.currentBlockHeight
}

func (bd *ChainStore) IsBlockInStore(hash Uint256) bool {
	var b *Block = new(Block)

	b.Blockdata = new(Blockdata)
	b.Blockdata.Program = new(program.Program)

	prefix := []byte{byte(DATA_Header)}
	blockData, err_get := bd.Get(append(prefix, hash.ToArray()...))
	if err_get != nil {
		return false
	}

	r := bytes.NewReader(blockData)

	// first 8 bytes is sys_fee
	_, err := serialization.ReadUint64(r)
	if err != nil {
		return false
	}

	// Deserialize block data
	err = b.FromTrimmedData(r)
	if err != nil {
		return false
	}

	if b.Blockdata.Height > bd.currentBlockHeight {
		return false
	}

	return true
}

func (bd *ChainStore) GetUnspentFromProgramHash(programHash Uint160, assetid Uint256) ([]*tx.UTXOUnspent, error) {

	prefix := []byte{byte(IX_Unspent_UTXO)}

	key := append(prefix, programHash.ToArray()...)
	key = append(key, assetid.ToArray()...)
	unspentsData, err := bd.Get(key)
	if err != nil {
		return nil, err
	}

	r := bytes.NewReader(unspentsData)
	listNum, err := serialization.ReadVarUint(r, 0)
	if err != nil {
		return nil, err
	}

	//log.Trace(fmt.Printf("[getUnspentFromProgramHash] listNum: %d, unspentsData: %x\n", listNum, unspentsData ))

	// read unspent list in store
	unspents := make([]*tx.UTXOUnspent, listNum)
	for i := 0; i < int(listNum); i++ {
		uu := new(tx.UTXOUnspent)
		err := uu.Deserialize(r)
		if err != nil {
			return nil, err
		}

		unspents[i] = uu
	}

	return unspents, nil
}

func (bd *ChainStore) PersistUnspentWithProgramHash(programHash Uint160, assetid Uint256, unspents []*tx.UTXOUnspent) error {
	prefix := []byte{byte(IX_Unspent_UTXO)}

	key := append(prefix, programHash.ToArray()...)
	key = append(key, assetid.ToArray()...)

	listnum := len(unspents)
	w := bytes.NewBuffer(nil)
	serialization.WriteVarUint(w, uint64(listnum))
	for i := 0; i < listnum; i++ {
		unspents[i].Serialize(w)
	}

	// BATCH PUT VALUE
	if err := bd.BatchPut(key, w.Bytes()); err != nil {
		return err
	}

	return nil
}

func (bd *ChainStore) GetUnspentsFromProgramHash(programHash Uint160) (map[Uint256][]*tx.UTXOUnspent, error) {
	uxtoUnspents := make(map[Uint256][]*tx.UTXOUnspent)

	prefix := []byte{byte(IX_Unspent_UTXO)}
	key := append(prefix, programHash.ToArray()...)
	iter := bd.NewIterator(key)
	for iter.Next() {
		rk := bytes.NewReader(iter.Key())

		// read prefix
		_, _ = serialization.ReadBytes(rk, 1)
		var ph Uint160
		ph.Deserialize(rk)
		var assetid Uint256
		assetid.Deserialize(rk)

		r := bytes.NewReader(iter.Value())
		listNum, err := serialization.ReadVarUint(r, 0)
		if err != nil {
			return nil, err
		}

		// read unspent list in store
		unspents := make([]*tx.UTXOUnspent, listNum)
		for i := 0; i < int(listNum); i++ {
			uu := new(tx.UTXOUnspent)
			err := uu.Deserialize(r)
			if err != nil {
				return nil, err
			}

			unspents[i] = uu
		}
		uxtoUnspents[assetid] = unspents
	}

	return uxtoUnspents, nil
}

func (bd *ChainStore) GetAssets() map[Uint256]*Asset {
	assets := make(map[Uint256]*Asset)

	iter := bd.NewIterator([]byte{byte(ST_Info)})
	for iter.Next() {
		rk := bytes.NewReader(iter.Key())

		// read prefix
		_, _ = serialization.ReadBytes(rk, 1)
		var assetid Uint256
		assetid.Deserialize(rk)
		log.Tracef("[GetAssets] assetid: %x\n", assetid.ToArray())

		asset := new(Asset)
		r := bytes.NewReader(iter.Value())
		asset.Deserialize(r)

		assets[assetid] = asset
	}

	return assets
}
