package mempool

import (
	"bytes"
	"errors"
	"fmt"
	"sync"

	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/config"
	. "github.com/elastos/Elastos.ELA/core"
	. "github.com/elastos/Elastos.ELA/errors"

	. "github.com/elastos/Elastos.ELA.Utility/common"
)

// ruleError creates an RuleError given a set of arguments.
func ruleError(c ErrCode, desc string) RuleError {
	return RuleError{ErrorCode: c, Description: desc}
}

type TxPool struct {
	sync.RWMutex
	txnCnt  uint64                   // count
	txnList map[Uint256]*Transaction // transaction which have been verifyed will put into this map
	//issueSummary  map[Uint256]Fixed64           // transaction which pass the verify will summary the amout to this map
	inputUTXOList   map[string]*Transaction  // transaction which pass the verify will add the UTXO to this map
	sidechainTxList map[Uint256]*Transaction // sidechain tx pool
}

func (p *TxPool) Init() {
	p.Lock()
	defer p.Unlock()
	p.txnCnt = 0
	p.inputUTXOList = make(map[string]*Transaction)
	//pool.issueSummary = make(map[Uint256]Fixed64)
	p.txnList = make(map[Uint256]*Transaction)
	p.sidechainTxList = make(map[Uint256]*Transaction)
}

//append transaction to txnpool when check ok.
//1.check  2.check with ledger(db) 3.check with pool
func (p *TxPool) AppendToTxPool(txn *Transaction) error {
	if txn.IsCoinBaseTx() {
		errMsg := fmt.Sprintf("coinbase cannot be added into transaction"+
			" pool", txn.Hash().String())
		return ruleError(ErrIneffectiveCoinbase, errMsg)
	}

	//verify transaction with Concurrency
	if err := blockchain.CheckTransactionSanity(CheckTxOut, txn); err != nil {
		return err
	}
	if err := blockchain.CheckTransactionContext(txn); err != nil {
		return err
	}
	//verify transaction by pool with lock
	if err := p.verifyTransactionWithTxnPool(txn); err != nil {
		return err
	}

	txn.Fee = blockchain.GetTxFee(txn, config.ELAAssetID)
	buf := new(bytes.Buffer)
	txn.Serialize(buf)
	txn.FeePerKB = txn.Fee * 1000 / Fixed64(len(buf.Bytes()))
	//add the transaction to process scope
	if ok := p.addToTxList(txn); !ok {
		// reject duplicated transaction
		errMsg := fmt.Sprintf("Transaction duplicate %s", txn.Hash().String())
		return ruleError(ErrTransactionDuplicate, errMsg)
	}
	return nil
}

// HaveTransaction returns if a transaction is in transaction pool by the given
// transaction id. If no transaction match the transaction id, return false
func (p *TxPool) HaveTransaction(txId Uint256) bool {
	p.RLock()
	defer p.RUnlock()
	_, ok := p.txnList[txId]
	return ok
}

// GetTxsInPool returns a copy of the transactions in transaction pool,
// It is safe to modify the returned map.
func (p *TxPool) GetTxsInPool() map[Uint256]*Transaction {
	p.RLock()
	defer p.RUnlock()
	copy := make(map[Uint256]*Transaction)
	for txnId, tx := range p.txnList {
		copy[txnId] = tx
	}
	return copy
}

//clean the trasaction Pool with committed block.
func (p *TxPool) CleanSubmittedTransactions(block *Block) error {
	p.cleanTransactions(block.Transactions)
	p.cleanSidechainTx(block.Transactions)
	p.cleanSideChainPowTx()

	return nil
}

func (p *TxPool) cleanTransactions(blockTxs []*Transaction) error {
	txCountInPool := p.GetTransactionCount()
	deleteCount := 0
	for _, blockTx := range blockTxs {
		if blockTx.TxType == CoinBase {
			continue
		}
		inputUtxos, err := blockchain.DefaultLedger.Store.GetTxReference(blockTx)
		if err != nil {
			log.Info(fmt.Sprintf("Transaction =%x not Exist in Pool when delete.", blockTx.Hash()), err)
			continue
		}
		for input := range inputUtxos {
			// we search transactions in transaction pool which have the same utxos with those transactions
			// in block. That is, if a transaction in the new-coming block uses the same utxo which a transaction
			// in transaction pool uses, then the latter one should be deleted, because one of its utxos has been used
			// by a confirmed transaction packed in the new-coming block.
			if tx := p.getInputUTXOList(input); tx != nil {
				if tx.Hash() == blockTx.Hash() {
					// it is evidently that two transactions with the same transaction id has exactly the same utxos with each
					// other. This is a special case of what we've said above.
					log.Debugf("duplicated transactions detected when adding a new block. "+
						" Delete transaction in the transaction pool. Transaction id: %x", tx.Hash())
				} else {
					log.Debugf("double spent UTXO inputs detected in transaction pool when adding a new block. "+
						"Delete transaction in the transaction pool. "+
						"block transaction hash: %x, transaction hash: %x, the same input: %s, index: %d",
						blockTx.Hash(), tx.Hash(), input.Previous.TxID, input.Previous.Index)
				}
				//1.remove from txnList
				p.delFromTxList(tx.Hash())
				//2.remove from UTXO list map
				for _, input := range tx.Inputs {
					p.delInputUTXOList(input)
				}

				//delete sidechain tx list
				if tx.TxType == WithdrawFromSideChain {
					payload, ok := tx.Payload.(*PayloadWithdrawFromSideChain)
					if !ok {
						log.Error("type cast failed when clean sidechain tx:", tx.Hash())
					}
					for _, hash := range payload.SideChainTransactionHashes {
						p.delSidechainTx(hash)
					}
				}
				deleteCount++
			}
		}
	}
	log.Debug(fmt.Sprintf("[cleanTransactionList],transaction %d in block, %d in transaction pool before, %d deleted,"+
		" Remains %d in TxPool",
		len(blockTxs), txCountInPool, deleteCount, p.GetTransactionCount()))
	return nil
}

//get the transaction by hash
func (p *TxPool) GetTransaction(hash Uint256) *Transaction {
	p.RLock()
	defer p.RUnlock()
	return p.txnList[hash]
}

//verify transaction with txnpool
func (p *TxPool) verifyTransactionWithTxnPool(txn *Transaction) error {
	if txn.IsSideChainPowTx() {
		// check and replace the duplicate sidechainpow tx
		p.replaceDuplicateSideChainPowTx(txn)
	} else if txn.IsWithdrawFromSideChainTx() {
		// check if the withdraw transaction includes duplicate sidechain tx in pool
		if err := p.verifyDuplicateSidechainTx(txn); err != nil {
			return ruleError(ErrSidechainTxDuplicate, err.Error())
		}
	}

	// check if the transaction includes double spent UTXO inputs
	if err := p.verifyDoubleSpend(txn); err != nil {
		return ruleError(ErrDoubleSpend, err.Error())
	}

	return nil
}

//remove from associated map
func (p *TxPool) removeTransaction(txn *Transaction) {
	//1.remove from txnList
	p.delFromTxList(txn.Hash())
	//2.remove from UTXO list map
	result, err := blockchain.DefaultLedger.Store.GetTxReference(txn)
	if err != nil {
		log.Info(fmt.Sprintf("Transaction =%x not Exist in Pool when delete.", txn.Hash()))
		return
	}
	for UTXOTxInput := range result {
		p.delInputUTXOList(UTXOTxInput)
	}
}

//check and add to utxo list pool
func (p *TxPool) verifyDoubleSpend(txn *Transaction) error {
	reference, err := blockchain.DefaultLedger.Store.GetTxReference(txn)
	if err != nil {
		return err
	}
	inputs := []*Input{}
	for k := range reference {
		if txn := p.getInputUTXOList(k); txn != nil {
			return errors.New(fmt.Sprintf("double spent UTXO inputs detected, "+
				"transaction hash: %x, input: %s, index: %d",
				txn.Hash(), k.Previous.TxID, k.Previous.Index))
		}
		inputs = append(inputs, k)
	}
	for _, v := range inputs {
		p.addInputUTXOList(txn, v)
	}

	return nil
}

func (p *TxPool) IsDuplicateSidechainTx(sidechainTxHash Uint256) bool {
	_, ok := p.sidechainTxList[sidechainTxHash]
	if ok {
		return true
	}

	return false
}

//check and add to sidechain tx pool
func (p *TxPool) verifyDuplicateSidechainTx(txn *Transaction) error {
	withPayload, ok := txn.Payload.(*PayloadWithdrawFromSideChain)
	if !ok {
		return errors.New("convert the payload of withdraw tx failed")
	}

	for _, hash := range withPayload.SideChainTransactionHashes {
		_, ok := p.sidechainTxList[hash]
		if ok {
			return errors.New("duplicate sidechain tx detected")
		}
	}
	p.addSidechainTx(txn)

	return nil
}

// check and replace the duplicate sidechainpow tx
func (p *TxPool) replaceDuplicateSideChainPowTx(txn *Transaction) {
	for _, v := range p.txnList {
		if v.TxType == SideChainPow {
			oldPayload := v.Payload.Data(SideChainPowPayloadVersion)
			oldGenesisHashData := oldPayload[32:64]

			newPayload := txn.Payload.Data(SideChainPowPayloadVersion)
			newGenesisHashData := newPayload[32:64]

			if bytes.Equal(oldGenesisHashData, newGenesisHashData) {
				txid := txn.Hash()
				log.Warn("replace sidechainpow transaction, txid=", txid.String())
				p.removeTransaction(v)
			}
		}
	}
}

// clean the sidechain tx pool
func (p *TxPool) cleanSidechainTx(txs []*Transaction) {
	for _, txn := range txs {
		if txn.IsWithdrawFromSideChainTx() {
			withPayload := txn.Payload.(*PayloadWithdrawFromSideChain)
			for _, hash := range withPayload.SideChainTransactionHashes {
				poolTx := p.sidechainTxList[hash]
				if poolTx != nil {
					// delete tx
					p.delFromTxList(poolTx.Hash())
					//delete utxo map
					for _, input := range poolTx.Inputs {
						p.delInputUTXOList(input)
					}
					//delete sidechain tx map
					payload, ok := poolTx.Payload.(*PayloadWithdrawFromSideChain)
					if !ok {
						log.Error("type cast failed when clean sidechain tx:", poolTx.Hash())
					}
					for _, hash := range payload.SideChainTransactionHashes {
						p.delSidechainTx(hash)
					}
				}
			}
		}
	}
}

// clean the sidechainpow tx pool
func (p *TxPool) cleanSideChainPowTx() {
	arbitrtor, err := blockchain.GetOnDutyArbiter()
	if err != nil {
		log.Error("get current arbiter failed")
		return
	}
	p.Lock()
	defer p.Unlock()
	for hash, txn := range p.txnList {
		if txn.IsSideChainPowTx() {
			if err = blockchain.CheckSideChainPowConsensus(txn, arbitrtor); err != nil {
				// delete tx
				delete(p.txnList, hash)
				//delete utxo map
				for _, input := range txn.Inputs {
					delete(p.inputUTXOList, input.ReferKey())
				}
			}
		}
	}
}

func (p *TxPool) addToTxList(txn *Transaction) bool {
	p.Lock()
	defer p.Unlock()
	txnHash := txn.Hash()
	if _, ok := p.txnList[txnHash]; ok {
		return false
	}
	p.txnList[txnHash] = txn
	// TODO move notification out of this method.
	// blockchain.DefaultLedger.Blockchain.BCEvents.Notify(events.EventNewTransactionPutInPool, txn)
	return true
}

func (p *TxPool) delFromTxList(txId Uint256) bool {
	p.Lock()
	defer p.Unlock()
	if _, ok := p.txnList[txId]; !ok {
		return false
	}
	delete(p.txnList, txId)
	return true
}

func (p *TxPool) copyTxList() map[Uint256]*Transaction {
	p.RLock()
	defer p.RUnlock()
	txnMap := make(map[Uint256]*Transaction, len(p.txnList))
	for txnId, txn := range p.txnList {
		txnMap[txnId] = txn
	}
	return txnMap
}

func (p *TxPool) GetTransactionCount() int {
	p.RLock()
	defer p.RUnlock()
	return len(p.txnList)
}

func (p *TxPool) getInputUTXOList(input *Input) *Transaction {
	p.RLock()
	defer p.RUnlock()
	return p.inputUTXOList[input.ReferKey()]
}

func (p *TxPool) addInputUTXOList(tx *Transaction, input *Input) bool {
	p.Lock()
	defer p.Unlock()
	id := input.ReferKey()
	_, ok := p.inputUTXOList[id]
	if ok {
		return false
	}
	p.inputUTXOList[id] = tx

	return true
}

func (p *TxPool) delInputUTXOList(input *Input) bool {
	p.Lock()
	defer p.Unlock()
	id := input.ReferKey()
	_, ok := p.inputUTXOList[id]
	if !ok {
		return false
	}
	delete(p.inputUTXOList, id)
	return true
}

func (p *TxPool) addSidechainTx(txn *Transaction) {
	p.Lock()
	defer p.Unlock()
	witPayload := txn.Payload.(*PayloadWithdrawFromSideChain)
	for _, hash := range witPayload.SideChainTransactionHashes {
		p.sidechainTxList[hash] = txn
	}
}

func (p *TxPool) delSidechainTx(hash Uint256) bool {
	p.Lock()
	defer p.Unlock()
	_, ok := p.sidechainTxList[hash]
	if !ok {
		return false
	}
	delete(p.sidechainTxList, hash)
	return true
}

func (p *TxPool) MaybeAcceptTransaction(txn *Transaction) error {
	txHash := txn.Hash()

	// Don't accept the transaction if it already exists in the pool.  This
	// applies to orphan transactions as well.  This check is intended to
	// be a quick check to weed out duplicates.
	if txn := p.GetTransaction(txHash); txn != nil {
		return fmt.Errorf("already have transaction")
	}

	// A standalone transaction must not be a coinbase
	if txn.IsCoinBaseTx() {
		return fmt.Errorf("transaction is an individual coinbase")
	}

	if err := p.AppendToTxPool(txn); err != nil {
		return err
	}

	return nil
}

func (p *TxPool) RemoveTransaction(txn *Transaction) {
	txHash := txn.Hash()
	for i := range txn.Outputs {
		input := Input{
			Previous: OutPoint{
				TxID:  txHash,
				Index: uint16(i),
			},
		}

		txn := p.getInputUTXOList(&input)
		if txn != nil {
			p.removeTransaction(txn)
		}
	}
}

func (p *TxPool) isTransactionCleaned(tx *Transaction) error {
	if tx := p.txnList[tx.Hash()]; tx != nil {
		return errors.New("has transaction in transaction pool" + tx.Hash().String())
	}
	for _, input := range tx.Inputs {
		if poolInput := p.inputUTXOList[input.ReferKey()]; poolInput != nil {
			return errors.New("has utxo inputs in input list pool" + input.String())
		}
	}
	if tx.TxType == WithdrawFromSideChain {
		payload := tx.Payload.(*PayloadWithdrawFromSideChain)
		for _, hash := range payload.SideChainTransactionHashes {
			if sidechainPoolTx := p.sidechainTxList[hash]; sidechainPoolTx != nil {
				return errors.New("has sidechain hash in sidechain list pool" + hash.String())
			}
		}
	}
	return nil
}

func (p *TxPool) isTransactionExisted(tx *Transaction) error {
	if tx := p.txnList[tx.Hash()]; tx == nil {
		return errors.New("does not have transaction in transaction pool" + tx.Hash().String())
	}
	for _, input := range tx.Inputs {
		if poolInput := p.inputUTXOList[input.ReferKey()]; poolInput == nil {
			return errors.New("does not have utxo inputs in input list pool" + input.String())
		}
	}
	if tx.TxType == WithdrawFromSideChain {
		payload := tx.Payload.(*PayloadWithdrawFromSideChain)
		for _, hash := range payload.SideChainTransactionHashes {
			if sidechainPoolTx := p.sidechainTxList[hash]; sidechainPoolTx == nil {
				return errors.New("does not have sidechain hash in sidechain list pool" + hash.String())
			}
		}
	}
	return nil
}
