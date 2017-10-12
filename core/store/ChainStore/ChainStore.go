package ChainStore

import (
	. "DNA_POW/common"
	"DNA_POW/common/config"
	"DNA_POW/common/log"
	"DNA_POW/common/serialization"
	. "DNA_POW/core/asset"
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
	"sort"
	"sync"
	"time"
)

const (
	HeaderHashListCount = 2000
	CleanCacheThreshold = 2
	TaskChanCap         = 4
)

var (
	ErrDBNotFound = errors.New("leveldb: not found")
)

type persistTask interface{}
type persistHeaderTask struct {
	header *Header
}

type rollbackBlockTask struct {
	blockHash Uint256
}
type persistBlockTask struct {
	block  *Block
	ledger *Ledger
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
}

func NewStore(file string) (IStore, error) {
	ldbs, err := NewLevelDBStore(file)

	return ldbs, err
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
				tcall := float64(time.Now().Sub(now)) / float64(time.Second)
				log.Debugf("handle header exetime: %g \n", tcall)

			case *persistBlockTask:
				self.handlePersistBlockTask(task.block, task.ledger)
				tcall := float64(time.Now().Sub(now)) / float64(time.Second)
				log.Debugf("handle block exetime: %g num transactions:%d \n", tcall, len(task.block.Transactions))
			case *rollbackBlockTask:
				self.handleRollbackBlockTask(task.blockHash)
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
func (self *ChainStore) clearCache() {
	self.mu.Lock()
	defer self.mu.Unlock()

	currBlockHeight := self.currentBlockHeight
	for hash, header := range self.headerCache {
		if header.Blockdata.Height+CleanCacheThreshold < currBlockHeight {
			delete(self.headerCache, hash)
		}
	}

	for hash, block := range self.blockCache {
		if block.Blockdata.Height+CleanCacheThreshold < currBlockHeight {
			delete(self.blockCache, hash)
		}
	}

}

func (bd *ChainStore) InitLedgerStoreWithGenesisBlock(genesisBlock *Block, defaultBookKeeper []*crypto.PubKey) (uint32, error) {
	hash := genesisBlock.Hash()
	bd.headerIndex[0] = hash
	log.Debugf("listhash genesis: %x\n", hash)

	prefix := []byte{byte(CFG_Version)}
	version, err := bd.Get(prefix)
	if err != nil {
		version = []byte{0x00}
	}

	if version[0] == 0x01 {
		// GenesisBlock should exist in chain
		// Or the bookkeepers are not consistent with the chain
		if !bd.IsBlockInStore(hash) {
			return 0, errors.New("bookkeepers are not consistent with the chain")
		}
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
		current_Header_Height := bd.currentBlockHeight

		log.Debugf("blockHash: %x\n", blockHash.ToArray())

		var listHash Uint256
		iter := bd.NewIterator([]byte{byte(IX_HeaderHashList)})
		for iter.Next() {
			rk := bytes.NewReader(iter.Key())
			// read prefix
			_, _ = serialization.ReadBytes(rk, 1)
			startNum, err := serialization.ReadUint32(rk)
			if err != nil {
				return 0, err
			}
			log.Debugf("start index: %d\n", startNum)

			r = bytes.NewReader(iter.Value())
			listNum, err := serialization.ReadVarUint(r, 0)
			if err != nil {
				return 0, err
			}

			for i := 0; i < int(listNum); i++ {
				listHash.Deserialize(r)
				bd.headerIndex[startNum+uint32(i)] = listHash
				bd.storedHeaderCount++
				//log.Debug( fmt.Sprintf( "listHash %d: %x\n", startNum+uint32(i), listHash ) )
			}
		}

		if bd.storedHeaderCount == 0 {
			iter = bd.NewIterator([]byte{byte(DATA_BlockHash)})
			for iter.Next() {
				rk := bytes.NewReader(iter.Key())
				// read prefix
				_, _ = serialization.ReadBytes(rk, 1)
				listheight, err := serialization.ReadUint32(rk)
				if err != nil {
					return 0, err
				}
				//log.Debug(fmt.Sprintf( "DATA_BlockHash block height: %d\n", listheight ))

				r := bytes.NewReader(iter.Value())
				listHash.Deserialize(r)
				//log.Debug(fmt.Sprintf( "DATA_BlockHash block hash: %x\n", listHash ))

				bd.headerIndex[listheight] = listHash
			}
		} else if current_Header_Height >= bd.storedHeaderCount {
			hash = blockHash
			for {
				if hash == bd.headerIndex[bd.storedHeaderCount-1] {
					break
				}

				header, err := bd.GetHeader(hash)
				if err != nil {
					return 0, err
				}

				//log.Debug(fmt.Sprintf( "header height: %d\n", header.Blockdata.Height ))
				//log.Debug(fmt.Sprintf( "header hash: %x\n", hash ))

				bd.headerIndex[header.Blockdata.Height] = hash
				hash = header.Blockdata.PrevBlockHash
			}
		}

		return bd.currentBlockHeight, nil

	} else {

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

		///////////////////////////////////////////////////
		// process defaultBookKeeper
		///////////////////////////////////////////////////
		// sort defaultBookKeeper
		sort.Sort(crypto.PubKeySlice(defaultBookKeeper))

		// currBookKeeper key
		bkListKey := bytes.NewBuffer(nil)
		bkListKey.WriteByte(byte(SYS_CurrentBookKeeper))

		// currBookKeeper value
		bkListValue := bytes.NewBuffer(nil)
		serialization.WriteUint8(bkListValue, uint8(len(defaultBookKeeper)))
		for k := 0; k < len(defaultBookKeeper); k++ {
			defaultBookKeeper[k].Serialize(bkListValue)
		}

		// nextBookKeeper value
		serialization.WriteUint8(bkListValue, uint8(len(defaultBookKeeper)))
		for k := 0; k < len(defaultBookKeeper); k++ {
			defaultBookKeeper[k].Serialize(bkListValue)
		}

		// defaultBookKeeper put value
		bd.Put(bkListKey.Bytes(), bkListValue.Bytes())
		///////////////////////////////////////////////////
		// persist genesis block
		bd.persist(genesisBlock)

		// put version to db
		err = bd.Put(prefix, []byte{0x01})
		if err != nil {
			return 0, err
		}

		return 0, nil
	}
}

func (bd *ChainStore) InitLedgerStore(l *Ledger) error {
	// TODO: InitLedgerStore
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
	bd.mu.RLock()
	defer bd.mu.RUnlock()

	return bd.headerIndex[bd.currentBlockHeight]
}

func (bd *ChainStore) dumpCache() {
	for key, data := range bd.headerCache {
		log.Trace("dumpCache ", key, data.Blockdata)
	}
}
func (bd *ChainStore) DumpCache() {
	bd.dumpCache()
}
func (bd *ChainStore) getHeaderWithCache(hash Uint256) *Header {
	if _, ok := bd.headerCache[hash]; ok {
		return bd.headerCache[hash]
	}
	header, _ := bd.GetHeader(hash)

	return header
}

func (bd *ChainStore) verifyHeader(header *Header) bool {
	prevHeader := bd.getHeaderWithCache(header.Blockdata.PrevBlockHash)

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
	prevHeader := bd.getHeaderWithCache(header.Blockdata.PrevBlockHash)

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
		self.taskCh <- &persistHeaderTask{header: &headers[i]}
	}

	return nil

}

func (db *ChainStore) RollbackBlock(blockHash Uint256) error {

	db.taskCh <- &rollbackBlockTask{blockHash: blockHash}

	return nil
}

func (bd *ChainStore) GetHeader(hash Uint256) (*Header, error) {
	bd.mu.RLock()
	if header, ok := bd.headerCache[hash]; ok {
		bd.mu.RUnlock()
		return header, nil
	}
	bd.mu.RUnlock()

	var h *Header = new(Header)

	h.Blockdata = new(Blockdata)
	h.Blockdata.Program = new(program.Program)

	prefix := []byte{byte(DATA_Header)}
	log.Debug("GetHeader Data:", hash.ToArray())
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
	log.Debug(fmt.Sprintf("sysfee: %d\n", sysfee))

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
	bd.mu.RLock()
	if block, ok := bd.blockCache[hash]; ok {
		bd.mu.RUnlock()
		return block, nil
	}
	bd.mu.RUnlock()

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

	if config.Parameters.ConsensusType == "pow" {
		if !self.powVerifyHeader(header) {
			return
		}

		self.addHeader(header)
	} else {
		if !self.verifyHeader(header) {
			return
		}
		self.addHeader(header)
	}
}

func (self *ChainStore) SaveBlock(b *Block, ledger *Ledger) error {
	log.Debug("SaveBlock()")
	self.mu.RLock()
	headerHeight := uint32(len(self.headerIndex))
	currBlockHeight := self.currentBlockHeight
	self.mu.RUnlock()

	if b.Blockdata.Height <= currBlockHeight {
		return nil
	}

	if b.Blockdata.Height > headerHeight {
		log.Infof("Info: [SaveBlock] block height - headerIndex.count >= 1, block height:%d, headerIndex.count:%d",
			b.Blockdata.Height, headerHeight)
		return nil
	}

	if b.Blockdata.Height == headerHeight {
		if config.Parameters.ConsensusType == "pow" {
			err := validation.PowVerifyBlock(b, ledger, false)
			if err != nil {
				log.Error("PowVerifyBlock error!")
				return err
			}
			log.Trace("------------>PowVerifyBlock passs!")
		} else {
			err := validation.VerifyBlock(b, ledger, false)
			if err != nil {
				log.Error("VerifyBlock error!")
				return err
			}
		}
		self.taskCh <- &persistHeaderTask{header: &Header{Blockdata: b.Blockdata}}
	} else {
		if config.Parameters.ConsensusType == "pow" {
			// Fixme Consider the Pow case
		} else {
			flag, err := validation.VerifySignableData(b)
			if flag == false || err != nil {
				log.Error("VerifyBlock error!")
				return err
			}
		}
	}

	self.taskCh <- &persistBlockTask{block: b, ledger: ledger}

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

	if b.Blockdata.Height <= self.currentBlockHeight {
		return
	}

	self.mu.Lock()
	self.blockCache[b.Hash()] = b
	self.mu.Unlock()

	//log.Trace(b.Blockdata.Height)
	//log.Trace(b.Blockdata)
	//log.Trace(b.Transactions[0])
	if b.Blockdata.Height < uint32(len(self.headerIndex)) {
		self.persistBlocks(ledger)

		self.NewBatch()
		storedHeaderCount := self.storedHeaderCount
		for self.currentBlockHeight-storedHeaderCount >= HeaderHashListCount {
			hashBuffer := new(bytes.Buffer)
			serialization.WriteVarUint(hashBuffer, uint64(HeaderHashListCount))
			var hashArray []byte
			for i := 0; i < HeaderHashListCount; i++ {
				index := storedHeaderCount + uint32(i)
				thash := self.headerIndex[index]
				thehash := thash.ToArray()
				hashArray = append(hashArray, thehash...)
			}
			hashBuffer.Write(hashArray)

			hhlPrefix := bytes.NewBuffer(nil)
			hhlPrefix.WriteByte(byte(IX_HeaderHashList))
			serialization.WriteUint32(hhlPrefix, storedHeaderCount)

			self.BatchPut(hhlPrefix.Bytes(), hashBuffer.Bytes())
			storedHeaderCount += HeaderHashListCount
		}

		err := self.BatchCommit()
		if err != nil {
			log.Error("failed to persist header hash list:", err)
			return
		}
		self.mu.Lock()
		self.storedHeaderCount = storedHeaderCount
		self.mu.Unlock()

		self.clearCache()
	}
}

func (bd *ChainStore) persistBlocks(ledger *Ledger) {
	stopHeight := uint32(len(bd.headerIndex))
	for h := bd.currentBlockHeight + 1; h <= stopHeight; h++ {
		hash := bd.headerIndex[h]
		block, ok := bd.blockCache[hash]
		if !ok {
			break
		}
		log.Trace(block.Blockdata)
		log.Trace(block.Transactions[0])
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

		ledger.Blockchain.BCEvents.Notify(events.EventBlockPersistCompleted, block)
		log.Tracef("The latest block height:%d, block hash: %x", block.Blockdata.Height, hash)
	}

}

func (bd *ChainStore) BlockInCache(hash Uint256) bool {
	bd.mu.RLock()
	defer bd.mu.RUnlock()

	_, ok := bd.blockCache[hash]
	return ok
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
	return bd.headerIndex[uint32(len(bd.headerIndex)-1)]
}

func (bd *ChainStore) GetHeaderHashByHeight(height uint32) Uint256 {
	bd.mu.RLock()
	defer bd.mu.RUnlock()

	return bd.headerIndex[height]
}

func (bd *ChainStore) GetHeaderHeight() uint32 {
	bd.mu.RLock()
	defer bd.mu.RUnlock()

	return uint32(len(bd.headerIndex) - 1)
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
