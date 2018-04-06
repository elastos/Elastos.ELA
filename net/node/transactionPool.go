package node

import (
	"bytes"
	"errors"
	"fmt"
	"sync"

	"github.com/elastos/Elastos.ELA.Utility/common"
	"github.com/elastos/Elastos.ELA.Utility/core/transaction"
	. "github.com/elastos/Elastos.ELA.Utility/errors"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/common/log"
	"github.com/elastos/Elastos.ELA/core/ledger"
	tx "github.com/elastos/Elastos.ELA/core/transaction"
	"github.com/elastos/Elastos.ELA/events"
)

var (
	zeroHash = common.Uint256{}
)

type TXNPool struct {
	sync.RWMutex
	txnCnt  uint64                                 // count
	txnList map[common.Uint256]*tx.NodeTransaction // transaction which have been verifyed will put into this map
	//issueSummary  map[common.Uint256]common.Fixed64           // transaction which pass the verify will summary the amout to this map
	inputUTXOList map[string]*tx.NodeTransaction // transaction which pass the verify will add the UTXO to this map
}

func (this *TXNPool) init() {
	this.Lock()
	defer this.Unlock()
	this.txnCnt = 0
	this.inputUTXOList = make(map[string]*tx.NodeTransaction)
	//this.issueSummary = make(map[common.Uint256]common.Fixed64)
	this.txnList = make(map[common.Uint256]*tx.NodeTransaction)
}

//append transaction to txnpool when check ok.
//1.check transaction. 2.check with ledger(db) 3.check with pool
func (this *TXNPool) AppendToTxnPool(txn *tx.NodeTransaction) ErrCode {
	//verify transaction with Concurrency
	if errCode := ledger.CheckTransactionSanity(txn); errCode != Success {
		log.Info("NodeTransaction verification failed", txn.Hash())
		return errCode
	}
	if errCode := ledger.CheckTransactionContext(txn, ledger.DefaultLedger); errCode != Success {
		log.Info("NodeTransaction verification with ledger failed", txn.Hash())
		return errCode
	}
	//verify transaction by pool with lock
	if ok := this.verifyTransactionWithTxnPool(txn); !ok {
		return ErrDoubleSpend
	}

	txn.Fee = common.Fixed64(txn.GetFee(ledger.DefaultLedger.Blockchain.AssetID))
	b_buf := new(bytes.Buffer)
	txn.Serialize(b_buf)
	txn.FeePerKB = txn.Fee * 1000 / common.Fixed64(len(b_buf.Bytes()))
	//add the transaction to process scope
	this.addtxnList(txn)
	return Success
}

//get the transaction in txnpool
func (this *TXNPool) GetTxnPool(byCount bool) map[common.Uint256]*tx.NodeTransaction {
	this.RLock()
	count := config.Parameters.MaxTxInBlock
	if count <= 0 {
		byCount = false
	}
	if len(this.txnList) < count || !byCount {
		count = len(this.txnList)
	}
	var num int
	txnMap := make(map[common.Uint256]*tx.NodeTransaction, count)
	for txnId, tx := range this.txnList {
		txnMap[txnId] = tx
		num++
		if num >= count {
			break
		}
	}
	this.RUnlock()
	return txnMap
}

//clean the trasaction Pool with committed block.
func (this *TXNPool) CleanSubmittedTransactions(block *ledger.Block) error {
	this.cleanTransactionList(block.Transactions)
	this.cleanUTXOList(block.Transactions)
	//this.cleanIssueSummary(block.Transactions)
	return nil
}

//get the transaction by hash
func (this *TXNPool) GetTransaction(hash common.Uint256) *tx.NodeTransaction {
	this.RLock()
	defer this.RUnlock()
	return this.txnList[hash]
}

//verify transaction with txnpool
func (this *TXNPool) verifyTransactionWithTxnPool(txn *tx.NodeTransaction) bool {
	// check if the transaction includes double spent UTXO inputs
	if err := this.verifyDoubleSpend(txn); err != nil {
		log.Info(err)
		return false
	}

	return true
}

//remove from associated map
func (this *TXNPool) removeTransaction(txn *tx.NodeTransaction) {
	//1.remove from txnList
	this.deltxnList(txn)
	//2.remove from UTXO list map
	result, err := txn.GetReference()
	if err != nil {
		log.Info(fmt.Sprintf("NodeTransaction =%x not Exist in Pool when delete.", txn.Hash()))
		return
	}
	for UTXOTxInput := range result {
		this.delInputUTXOList(UTXOTxInput)
	}
}

//check and add to utxo list pool
func (this *TXNPool) verifyDoubleSpend(txn *tx.NodeTransaction) error {
	reference, err := txn.GetReference()
	if err != nil {
		return err
	}
	inputs := []*transaction.UTXOTxInput{}
	for k := range reference {
		if txn := this.getInputUTXOList(k); txn != nil {
			return errors.New(fmt.Sprintf("double spent UTXO inputs detected, "+
				"transaction hash: %x, input: %s, index: %s",
				txn.Hash(), k.ToString()[:64], k.ToString()[64:]))
		}
		inputs = append(inputs, k)
	}
	for _, v := range inputs {
		this.addInputUTXOList(txn, v)
	}

	return nil
}

//clean txnpool utxo map
func (this *TXNPool) cleanUTXOList(txs []*tx.NodeTransaction) {
	for _, txn := range txs {
		inputUtxos, _ := txn.GetReference()
		for Utxoinput, _ := range inputUtxos {
			this.delInputUTXOList(Utxoinput)
		}
	}
}

// clean the trasaction Pool with committed transactions.
func (this *TXNPool) cleanTransactionList(txns []*tx.NodeTransaction) error {
	cleaned := 0
	txnsNum := len(txns)
	for _, txn := range txns {
		if txn.TxType == transaction.CoinBase {
			txnsNum = txnsNum - 1
			continue
		}
		if this.deltxnList(txn) {
			cleaned++
		}
	}
	if txnsNum != cleaned {
		log.Info(fmt.Sprintf("The Transactions num Unmatched. Expect %d, got %d .\n", txnsNum, cleaned))
	}
	log.Debug(fmt.Sprintf("[cleanTransactionList],transaction %d Requested, %d cleaned, Remains %d in TxPool", txnsNum, cleaned, this.GetTransactionCount()))
	return nil
}

func (this *TXNPool) addtxnList(txn *tx.NodeTransaction) bool {
	this.Lock()
	defer this.Unlock()
	txnHash := txn.Hash()
	if _, ok := this.txnList[txnHash]; ok {
		return false
	}
	this.txnList[txnHash] = txn
	ledger.DefaultLedger.Blockchain.BCEvents.Notify(events.EventNewTransactionPutInPool, txn)
	return true
}

func (this *TXNPool) deltxnList(tx *tx.NodeTransaction) bool {
	this.Lock()
	defer this.Unlock()
	txHash := tx.Hash()
	if _, ok := this.txnList[txHash]; !ok {
		return false
	}
	delete(this.txnList, tx.Hash())
	return true
}

func (this *TXNPool) copytxnList() map[common.Uint256]*tx.NodeTransaction {
	this.RLock()
	defer this.RUnlock()
	txnMap := make(map[common.Uint256]*tx.NodeTransaction, len(this.txnList))
	for txnId, txn := range this.txnList {
		txnMap[txnId] = txn
	}
	return txnMap
}

func (this *TXNPool) GetTransactionCount() int {
	this.RLock()
	defer this.RUnlock()
	return len(this.txnList)
}

func (this *TXNPool) getInputUTXOList(input *transaction.UTXOTxInput) *tx.NodeTransaction {
	this.RLock()
	defer this.RUnlock()
	return this.inputUTXOList[input.ToString()]
}

func (this *TXNPool) addInputUTXOList(tx *tx.NodeTransaction, input *transaction.UTXOTxInput) bool {
	this.Lock()
	defer this.Unlock()
	id := input.ToString()
	_, ok := this.inputUTXOList[id]
	if ok {
		return false
	}
	this.inputUTXOList[id] = tx

	return true
}

func (this *TXNPool) delInputUTXOList(input *transaction.UTXOTxInput) bool {
	this.Lock()
	defer this.Unlock()
	id := input.ToString()
	_, ok := this.inputUTXOList[id]
	if !ok {
		return false
	}
	delete(this.inputUTXOList, id)
	return true
}

func (this *TXNPool) MaybeAcceptTransaction(txn *tx.NodeTransaction) error {
	txHash := txn.Hash()

	// Don't accept the transaction if it already exists in the pool.  This
	// applies to orphan transactions as well.  This check is intended to
	// be a quick check to weed out duplicates.
	if txn := this.GetTransaction(txHash); txn != nil {
		return fmt.Errorf("already have transaction")
	}

	// A standalone transaction must not be a coinbase transaction.
	if txn.IsCoinBaseTx() {
		return fmt.Errorf("transaction is an individual coinbase")
	}

	if errCode := this.AppendToTxnPool(txn); errCode != Success {
		return fmt.Errorf("VerifyTxs failed when AppendToTxnPool")
	}

	return nil
}

func (this *TXNPool) RemoveTransaction(txn *tx.NodeTransaction) {
	txHash := txn.Hash()
	for i := range txn.Outputs {
		input := transaction.UTXOTxInput{
			ReferTxID:          txHash,
			ReferTxOutputIndex: uint16(i),
		}

		txn := this.getInputUTXOList(&input)
		if txn != nil {
			this.removeTransaction(txn)
		}
	}
}
