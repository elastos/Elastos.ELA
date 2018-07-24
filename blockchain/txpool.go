// In fact, a transaction pool, which hold unconfirmed transactions,
// is merely a database in memory, rather than a disk.
// so it should satisfy some mathematical laws.
// and if we treated it as a relationship database,
// it should satisfy relationship algebra.
// that is:
// it have two databases by now. transactions list and sidechain transaction list.
// transaction database has two tables: txnList and inputUTXOList.
// transaction list: (transaction hash, transaction object) and the transaction hash is the primary key.
// input UTXO list: (referKey, transaction object) and the referKey is the composite-id, which is a sha256 hash digest
// of reference tx id and index.
// sidechain transaction list has one table.
// so, the memory pool as a database shoud have CURD these four actions.
package blockchain

import (
	"bytes"
	"errors"
	"fmt"
	"sync"

	"github.com/elastos/Elastos.ELA/config"
	. "github.com/elastos/Elastos.ELA/core"
	. "github.com/elastos/Elastos.ELA/errors"
	"github.com/elastos/Elastos.ELA/events"
	"github.com/elastos/Elastos.ELA/log"

	. "github.com/elastos/Elastos.ELA.Utility/common"
)

type TxPool struct {
	sync.RWMutex
	txnList         map[Uint256]*Transaction // transaction which have been verifyed will put into this map
	inputUTXOList   map[string]*Transaction  // transaction which pass the verify will add the UTXO to this map
	sidechainTxList map[Uint256]*Transaction // sidechain tx pool
}

func (pool *TxPool) Init() {
	pool.inputUTXOList = make(map[string]*Transaction)
	pool.txnList = make(map[Uint256]*Transaction)
	pool.sidechainTxList = make(map[Uint256]*Transaction)
}

//append transaction to transaction pool when check is ok.
//1.refuse coinbase transaction  2.check with ledger(db) 3.check with pool 4. add transaction
func (pool *TxPool) AppendToTxnPool(txn *Transaction) ErrCode {
	log.Info("transaction ready to validate:", txn.Hash().String())
	if txn.IsCoinBaseTx() {
		log.Warn("coinbase cannot be added into transaction pool", txn.Hash().String())
		return ErrIneffectiveCoinbase
	}

	//verify transaction with ledger
	if errCode := CheckTransactionSanity(CheckTxOut, txn); errCode != Success {
		log.Warn("[TxPool CheckTransactionSanity] failed", txn.Hash().String())
		return errCode
	}
	if errCode := CheckTransactionContext(txn); errCode != Success {
		log.Warn("[TxPool CheckTransactionContext] failed", txn.Hash().String())
		return errCode
	}

	pool.Lock()
	defer pool.Unlock()
	//verify transaction with tx pool
	if errCode := pool.verifyTransactionWithTxnPool(txn); errCode != Success {
		log.Warn("[TxPool verifyTransactionWithTxnPool] failed", txn.Hash())
		return errCode
	}
	txn.Fee = GetTxFee(txn, DefaultLedger.Blockchain.AssetID)
	buf := new(bytes.Buffer)
	txn.Serialize(buf)
	txn.FeePerKB = txn.Fee * 1000 / Fixed64(len(buf.Bytes()))

	// add transaction
	log.Info("add transaction:", txn.Hash().String())
	pool.addTransaction(txn)

	return Success
}

// clean the trasaction Pool with committed block.
// first read all transactions from the new-coming block.
// then we process all transactions.
func (pool *TxPool) CleanTxPool(block *Block) error {
	pool.Lock()
	defer pool.Unlock()
	blockTxs := block.Transactions
	txCountInPool := len(pool.txnList)
	deleteCount := 0

	for _, blockTx := range blockTxs {
		// 1. we don't check coinbase txs.
		if blockTx.TxType == CoinBase {
			continue
		}
		// 2. delete txs in tx pool which are double spent with txs in block
		if doubleSpendTxs := pool.verifyDoubleSpend(blockTx); doubleSpendTxs != nil {
			for _, tx := range doubleSpendTxs {
				pool.removeTransaction(tx)
				deleteCount++
			}
			continue
		}
		// 3. check duplicate sidechain hashes in withdrawfromsidechain txs
		if blockTx.TxType == WithdrawFromSideChain {
			if txs := pool.verifyDuplicateSidechainTx(blockTx); txs != nil {
				for _, tx := range txs {
					pool.removeTransaction(tx)
					deleteCount++
				}
				continue
			}
		}
	}

	log.Info(fmt.Sprintf("[cleanTransactionList],transaction %d in block, %d in transaction pool before, %d deleted,"+
		" Remains %d in TxPool",
		len(blockTxs), txCountInPool, deleteCount, len(pool.txnList)))

	return nil
}

//get the transaction in txnpool
//get the transaction in txnpool
func (pool *TxPool) GetTransactionPool(hasMaxCount bool) map[Uint256]*Transaction {
	pool.RLock()
	count := config.Parameters.MaxTxsInBlock
	if count <= 0 {
		hasMaxCount = false
	}
	if len(pool.txnList) < count || !hasMaxCount {
		count = len(pool.txnList)
	}
	var num int
	txnMap := make(map[Uint256]*Transaction, count)
	for txnId, tx := range pool.txnList {
		txnMap[txnId] = tx
		num++
		if num >= count {
			break
		}
	}
	pool.RUnlock()
	return txnMap
}

//verify transaction with txnpool
func (pool *TxPool) verifyTransactionWithTxnPool(tx *Transaction) ErrCode {
	// check if the transaction includes double spent UTXO inputs
	if doubleSpendTxs := pool.verifyDoubleSpend(tx); doubleSpendTxs != nil {
		var hashes string
		for _, doubleSpendTx := range doubleSpendTxs {
			hashes = hashes + "," + doubleSpendTx.Hash().String()
		}
		log.Warn("double spend transactions detected", hashes)
		return ErrDoubleSpend
	}

	if tx.IsSideChainPowTx() {
		// check and replace the duplicate sidechainpow tx
		pool.removeDuplicateSideChainPowTx(tx)
	} else if tx.IsWithdrawFromSideChainTx() {
		// check if the withdraw transaction includes duplicate sidechain tx in pool
		if txList := pool.verifyDuplicateSidechainTx(tx); txList != nil {
			log.Warn("duplicate sidechain tx detected")
			return ErrSidechainTxDuplicate
		}
	}
	return Success
}

//return txs from tx pool which are double spent with given tx.
func (pool *TxPool) verifyDoubleSpend(tx *Transaction) []*Transaction {
	var doubleSpendTxs []*Transaction
	for _, input := range tx.Inputs {
		if poolTx := pool.inputUTXOList[input.ReferKey()]; poolTx != nil {
			if poolTx.Hash() == tx.Hash() {
				// it is evidently that two transactions with the same transaction id has exactly the same utxos with each
				// other. This is a special case of double-spent transactions.
				log.Infof("duplicated transactions detected when adding a new block. "+
					" Delete transaction in the transaction pool. Transaction id: %x", tx.Hash())
			} else {
				log.Infof("double spent UTXO inputs detected in transaction pool when adding a new block. "+
					"Delete transaction in the transaction pool. "+
					"block transaction hash: %x, transaction hash: %x, the same input: %s, index: %d",
					tx.Hash(), tx.Hash(), input.Previous.TxID, input.Previous.Index)
			}
			doubleSpendTxs = append(doubleSpendTxs, poolTx)
		}
	}
	return doubleSpendTxs
}

func (pool *TxPool) IsDuplicateSidechainTx(sidechainTxHash Uint256) bool {
	_, ok := pool.sidechainTxList[sidechainTxHash]
	return ok
}

//check and add to sidechain tx pool
func (pool *TxPool) verifyDuplicateSidechainTx(txn *Transaction) (tx []*Transaction) {
	withPayload, ok := txn.Payload.(*PayloadWithdrawFromSideChain)
	if !ok {
		return nil
	}

	var txList []*Transaction
	for _, hash := range withPayload.SideChainTransactionHashes {
		tx := pool.sidechainTxList[hash]
		if tx != nil {
			txList = append(txList, tx)
		}
	}

	return txList
}

// check and replace the duplicate sidechainpow tx
func (pool *TxPool) removeDuplicateSideChainPowTx(txn *Transaction) {
	for _, v := range pool.txnList {
		oldPayload := v.Payload.Data(SideChainPowPayloadVersion)
		oldGenesisHashData := oldPayload[32:64]

		newPayload := txn.Payload.Data(SideChainPowPayloadVersion)
		newGenesisHashData := newPayload[32:64]

		if bytes.Equal(oldGenesisHashData, newGenesisHashData) {
			txid := txn.Hash()
			log.Warn("replace sidechainpow transaction, txid=", txid.String())
			pool.removeTransaction(v)
		}
	}
}

// INSERT, UPDATE and DELETE operation must be atomic.
// all the tables must be consistent
func (pool *TxPool) removeTransaction(txn *Transaction) {
	//1.remove from tx list
	delete(pool.txnList, txn.Hash())
	//2.remove from UTXO list map
	for _, input := range txn.Inputs {
		delete(pool.inputUTXOList, input.ReferKey())
	}
	//3. remove from sidechain tx list
	if txn.TxType == WithdrawFromSideChain {
		payload := txn.Payload.(*PayloadWithdrawFromSideChain)
		for _, hash := range payload.SideChainTransactionHashes {
			delete(pool.sidechainTxList, hash)
		}
	}
}

func (pool *TxPool) addTransaction(tx *Transaction) {
	// 1. add to txList
	pool.txnList[tx.Hash()] = tx
	// 2. add to UTXO list map
	for _, input := range tx.Inputs {
		pool.inputUTXOList[input.ReferKey()] = tx
	}
	// 3. add to sidechain tx list
	if tx.TxType == WithdrawFromSideChain {
		payload := tx.Payload.(*PayloadWithdrawFromSideChain)
		for _, hash := range payload.SideChainTransactionHashes {
			pool.sidechainTxList[hash] = tx
		}
	}

	DefaultLedger.Blockchain.BCEvents.Notify(events.EventNewTransactionPutInPool, tx)
}

func (pool *TxPool) RemoveSubsequentTransactions(txn *Transaction) {
	//if a transaction (let's call it as tx1) is rolled back, then those transactions which used tx1's outputs as inputs,
	// should also be deleted from transaction pool, because those inputs are no longer valid.
	txHash := txn.Hash()

	pool.Lock()
	defer pool.Unlock()
	for i := range txn.Outputs {
		input := Input{
			Previous: OutPoint{
				TxID:  txHash,
				Index: uint16(i),
			},
		}

		txn := pool.inputUTXOList[input.ReferKey()]
		if txn != nil {
			pool.removeTransaction(txn)
		}
	}
}

func (pool *TxPool) isTransactionCleaned(tx *Transaction) error {
	if tx := pool.txnList[tx.Hash()]; tx != nil {
		return errors.New("has transaction in transaction pool" + tx.Hash().String())
	}
	for _, input := range tx.Inputs {
		if poolInput := pool.inputUTXOList[input.ReferKey()]; poolInput != nil {
			return errors.New("has utxo inputs in input list pool" + input.String())
		}
	}
	if tx.TxType == WithdrawFromSideChain {
		payload := tx.Payload.(*PayloadWithdrawFromSideChain)
		for _, hash := range payload.SideChainTransactionHashes {
			if sidechainPoolTx := pool.sidechainTxList[hash]; sidechainPoolTx != nil {
				return errors.New("has sidechain hash in sidechain list pool" + hash.String())
			}
		}
	}
	return nil
}

func (pool *TxPool) isTransactionExisted(tx *Transaction) error {
	if tx := pool.txnList[tx.Hash()]; tx == nil {
		return errors.New("does not have transaction in transaction pool" + tx.Hash().String())
	}
	for _, input := range tx.Inputs {
		if poolInput := pool.inputUTXOList[input.ReferKey()]; poolInput == nil {
			return errors.New("does not have utxo inputs in input list pool" + input.String())
		}
	}
	if tx.TxType == WithdrawFromSideChain {
		payload := tx.Payload.(*PayloadWithdrawFromSideChain)
		for _, hash := range payload.SideChainTransactionHashes {
			if sidechainPoolTx := pool.sidechainTxList[hash]; sidechainPoolTx == nil {
				return errors.New("does not have sidechain hash in sidechain list pool" + hash.String())
			}
		}
	}
	return nil
}

func GetTxFee(tx *Transaction, assetId Uint256) Fixed64 {
	feeMap, err := GetTxFeeMap(tx)
	if err != nil {
		return 0
	}

	return feeMap[assetId]
}

func GetTxFeeMap(tx *Transaction) (map[Uint256]Fixed64, error) {
	feeMap := make(map[Uint256]Fixed64)
	reference, err := DefaultLedger.Store.GetTxReference(tx)
	if err != nil {
		return nil, err
	}

	var inputs = make(map[Uint256]Fixed64)
	var outputs = make(map[Uint256]Fixed64)
	for _, v := range reference {
		amout, ok := inputs[v.AssetID]
		if ok {
			inputs[v.AssetID] = amout + v.Value
		} else {
			inputs[v.AssetID] = v.Value
		}
	}

	for _, v := range tx.Outputs {
		amout, ok := outputs[v.AssetID]
		if ok {
			outputs[v.AssetID] = amout + v.Value
		} else {
			outputs[v.AssetID] = v.Value
		}
	}

	//calc the balance of input vs output
	for outputAssetid, outputValue := range outputs {
		if inputValue, ok := inputs[outputAssetid]; ok {
			feeMap[outputAssetid] = inputValue - outputValue
		} else {
			feeMap[outputAssetid] -= outputValue
		}
	}
	for inputAssetid, inputValue := range inputs {
		if _, exist := feeMap[inputAssetid]; !exist {
			feeMap[inputAssetid] += inputValue
		}
	}
	return feeMap, nil
}
