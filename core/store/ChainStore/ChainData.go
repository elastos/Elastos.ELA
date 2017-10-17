package ChainStore

import (
	"bytes"
	"errors"
	"fmt"

	. "DNA_POW/common"
	"DNA_POW/common/serialization"
	. "DNA_POW/core/ledger"
	tx "DNA_POW/core/transaction"
	"DNA_POW/core/transaction/payload"
)

func (db *ChainStore) BatchInit() error {
	return db.NewBatch()
}

func (db *ChainStore) BatchFinish() error {
	return db.BatchCommit()
}

// key: DATA_Header || block hash
// value: sysfee(8bytes) || trimmed block
func (db *ChainStore) PersistTrimmedBlock(b *Block) error {
	key := bytes.NewBuffer(nil)
	key.WriteByte(byte(DATA_Header))
	blockHash := b.Hash()
	blockHash.Serialize(key)

	value := bytes.NewBuffer(nil)
	var sysfee uint64 = 0x0000000000000000
	serialization.WriteUint64(value, sysfee)
	b.Trim(value)

	if err := db.BatchPut(key.Bytes(), value.Bytes()); err != nil {
		return err
	}

	return nil
}

func (db *ChainStore) RollbackTrimemedBlock(b *Block) error {
	key := bytes.NewBuffer(nil)
	key.WriteByte(byte(DATA_Header))
	blockHash := b.Hash()
	blockHash.Serialize(key)

	if err := db.BatchDelete(key.Bytes()); err != nil {
		return err
	}

	return nil
}

// key: DATA_BlockHash || height
// value: block hash
func (db *ChainStore) PersistBlockHash(b *Block) error {
	key := bytes.NewBuffer(nil)
	key.WriteByte(byte(DATA_BlockHash))
	if err := serialization.WriteUint32(key, b.Blockdata.Height); err != nil {
		return err
	}

	value := bytes.NewBuffer(nil)
	hashValue := b.Blockdata.Hash()
	hashValue.Serialize(value)

	if err := db.BatchPut(key.Bytes(), value.Bytes()); err != nil {
		return err
	}

	return nil
}

func (db *ChainStore) RollbackBlockHash(b *Block) error {
	key := bytes.NewBuffer(nil)
	key.WriteByte(byte(DATA_BlockHash))
	if err := serialization.WriteUint32(key, b.Blockdata.Height); err != nil {
		return err
	}

	if err := db.BatchDelete(key.Bytes()); err != nil {
		return err
	}

	return nil
}

// key: SYS_CurrentBlock
// value: current block hash || height
func (db *ChainStore) PersistCurrentBlock(b *Block) error {

	currentBlockKey := bytes.NewBuffer(nil)
	currentBlockKey.WriteByte(byte(SYS_CurrentBlock))

	currentBlock := bytes.NewBuffer(nil)
	blockHash := b.Hash()
	blockHash.Serialize(currentBlock)
	serialization.WriteUint32(currentBlock, b.Blockdata.Height)

	if err := db.BatchPut(currentBlockKey.Bytes(), currentBlock.Bytes()); err != nil {
		return err
	}

	return nil
}

func (db *ChainStore) RollbackCurrentBlock(b *Block) error {
	key := bytes.NewBuffer(nil)
	key.WriteByte(byte(SYS_CurrentBlock))

	value := bytes.NewBuffer(nil)
	blockHash := b.Blockdata.PrevBlockHash
	blockHash.Serialize(value)
	serialization.WriteUint32(value, b.Blockdata.Height-1)

	if err := db.BatchPut(key.Bytes(), value.Bytes()); err != nil {
		return err
	}

	return nil
}

func (db *ChainStore) PersistUnspendUTXOs(b *Block) error {
	unspendUTXOs := make(map[Uint160]map[Uint256][]*tx.UTXOUnspent)
	for _, txn := range b.Transactions {
		if txn.TxType == tx.RegisterAsset {
			continue
		}
		for index, output := range txn.Outputs {
			programHash := output.ProgramHash
			assetID := output.AssetID
			value := output.Value
			if _, ok := unspendUTXOs[programHash]; !ok {
				unspendUTXOs[programHash] = make(map[Uint256][]*tx.UTXOUnspent)
			}
			if _, ok := unspendUTXOs[programHash][assetID]; !ok {
				var err error
				unspendUTXOs[programHash][assetID], err = db.GetUnspentFromProgramHash(programHash, assetID)
				if err != nil {
					unspendUTXOs[programHash][assetID] = make([]*tx.UTXOUnspent, 0)
				}
			}
			u := tx.UTXOUnspent{
				Txid:  txn.Hash(),
				Index: uint32(index),
				Value: value,
			}
			unspendUTXOs[programHash][assetID] = append(unspendUTXOs[programHash][assetID], &u)
		}

		for _, input := range txn.UTXOInputs {
			referTxn, err := db.GetTransaction(input.ReferTxID)
			if err != nil {
				return err
			}
			index := input.ReferTxOutputIndex
			referTxnOutput := referTxn.Outputs[index]
			programHash := referTxnOutput.ProgramHash
			assetID := referTxnOutput.AssetID
			if _, ok := unspendUTXOs[programHash]; !ok {
				unspendUTXOs[programHash] = make(map[Uint256][]*tx.UTXOUnspent)
			}
			if _, ok := unspendUTXOs[programHash][assetID]; !ok {
				unspendUTXOs[programHash][assetID], err = db.GetUnspentFromProgramHash(programHash, assetID)
				if err != nil {
					return errors.New(fmt.Sprintf("[persist] utxoUnspents programHash:%v, assetId:%v has no unspent UTXO.", programHash, assetID))
				}
			}
			flag := false
			listnum := len(unspendUTXOs[programHash][assetID])
			for i := 0; i < listnum; i++ {
				if unspendUTXOs[programHash][assetID][i].Txid.CompareTo(referTxn.Hash()) == 0 && unspendUTXOs[programHash][assetID][i].Index == uint32(index) {
					unspendUTXOs[programHash][assetID][i] = unspendUTXOs[programHash][assetID][listnum-1]
					unspendUTXOs[programHash][assetID] = unspendUTXOs[programHash][assetID][:listnum-1]
					flag = true
					break
				}
			}
			if !flag {
				return errors.New(fmt.Sprintf("[persist] utxoUnspents NOT find UTXO by txid: %x, index: %d.", referTxn.Hash(), index))
			}
		}
	}
	// batch put the utxoUnspents
	for programHash, programHash_value := range unspendUTXOs {
		for assetId, unspents := range programHash_value {
			err := db.PersistUnspentWithProgramHash(programHash, assetId, unspents)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (db *ChainStore) RollbackUnspendUTXOs(b *Block) error {
	unspendUTXOs := make(map[Uint160]map[Uint256][]*tx.UTXOUnspent)
	for _, txn := range b.Transactions {
		if txn.TxType == tx.RegisterAsset {
			continue
		}
		for index, output := range txn.Outputs {
			programHash := output.ProgramHash
			assetID := output.AssetID
			value := output.Value
			if _, ok := unspendUTXOs[programHash]; !ok {
				unspendUTXOs[programHash] = make(map[Uint256][]*tx.UTXOUnspent)
			}
			if _, ok := unspendUTXOs[programHash][assetID]; !ok {
				var err error
				unspendUTXOs[programHash][assetID], err = db.GetUnspentFromProgramHash(programHash, assetID)
				if err != nil {
					unspendUTXOs[programHash][assetID] = make([]*tx.UTXOUnspent, 0)
				}
			}
			u := tx.UTXOUnspent{
				Txid:  txn.Hash(),
				Index: uint32(index),
				Value: value,
			}
			var position int
			for i, unspend := range unspendUTXOs[programHash][assetID] {
				if unspend.Txid == u.Txid && unspend.Index == u.Index {
					position = i
					break
				}
			}
			unspendUTXOs[programHash][assetID] = append(unspendUTXOs[programHash][assetID][:position], unspendUTXOs[programHash][assetID][position+1:]...)
		}

		for _, input := range txn.UTXOInputs {
			referTxn, err := db.GetTransaction(input.ReferTxID)
			if err != nil {
				return err
			}
			index := input.ReferTxOutputIndex
			referTxnOutput := referTxn.Outputs[index]
			programHash := referTxnOutput.ProgramHash
			assetID := referTxnOutput.AssetID
			if _, ok := unspendUTXOs[programHash]; !ok {
				unspendUTXOs[programHash] = make(map[Uint256][]*tx.UTXOUnspent)
			}
			if _, ok := unspendUTXOs[programHash][assetID]; !ok {
				unspendUTXOs[programHash][assetID], err = db.GetUnspentFromProgramHash(programHash, assetID)
				if err != nil {
					return errors.New(fmt.Sprintf("[persist] utxoUnspents programHash:%v, assetId:%v has no unspent UTXO.", programHash, assetID))
				}
			}
			u := tx.UTXOUnspent{
				Txid:  referTxn.Hash(),
				Index: uint32(index),
				Value: referTxnOutput.Value,
			}
			unspendUTXOs[programHash][assetID] = append(unspendUTXOs[programHash][assetID], &u)
		}
	}
	// batch put the utxoUnspents
	for programHash, programHash_value := range unspendUTXOs {
		for assetId, unspents := range programHash_value {
			err := db.PersistUnspentWithProgramHash(programHash, assetId, unspents)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (db *ChainStore) PersistTransactions(b *Block) error {

	for _, txn := range b.Transactions {
		if err := db.PersistTransaction(txn, b.Blockdata.Height); err != nil {
			return err
		}
		if txn.TxType == tx.RegisterAsset {
			regPayload := txn.Payload.(*payload.RegisterAsset)
			if err := db.PersistAsset(txn.Hash(), regPayload.Asset); err != nil {
				return err
			}
		}
	}
	return nil
}

func (db *ChainStore) RollbackTransactions(b *Block) error {
	for _, txn := range b.Transactions {
		if err := db.RollbackTransaction(txn); err != nil {
			return err
		}
		if txn.TxType == tx.RegisterAsset {
			if err := db.RollbackAsset(txn.Hash()); err != nil {
				return err
			}
		}
	}

	return nil
}

func (db *ChainStore) RollbackTransaction(txn *tx.Transaction) error {

	key := bytes.NewBuffer(nil)
	key.WriteByte(byte(DATA_Transaction))
	txnHash := txn.Hash()
	txnHash.Serialize(key)

	if err := db.BatchDelete(key.Bytes()); err != nil {
		return err
	}

	return nil
}

func (bd *ChainStore) RollbackAsset(assetId Uint256) error {
	key := bytes.NewBuffer(nil)
	key.WriteByte(byte(ST_Info))
	assetId.Serialize(key)

	if err := bd.BatchDelete(key.Bytes()); err != nil {
		return err
	}

	return nil
}

func (db *ChainStore) PersistUnspend(b *Block) error {
	unspentPrefix := []byte{byte(IX_Unspent)}
	unspents := make(map[Uint256][]uint16)
	for _, txn := range b.Transactions {
		if txn.TxType == tx.RegisterAsset {
			continue
		}
		txnHash := txn.Hash()
		for index := range txn.Outputs {
			unspents[txnHash] = append(unspents[txnHash], uint16(index))
		}
		for index, input := range txn.UTXOInputs {
			referTxnHash := input.ReferTxID
			if _, ok := unspents[referTxnHash]; !ok {
				unspentValue, err := db.Get(append(unspentPrefix, referTxnHash.ToArray()...))
				if err != nil {
					return err
				}
				unspents[referTxnHash], err = GetUint16Array(unspentValue)
				if err != nil {
					return err
				}
			}

			unspentLen := len(unspents[referTxnHash])
			for k, outputIndex := range unspents[referTxnHash] {
				if outputIndex == uint16(txn.UTXOInputs[index].ReferTxOutputIndex) {
					unspents[referTxnHash][k] = unspents[referTxnHash][unspentLen-1]
					unspents[referTxnHash] = unspents[referTxnHash][:unspentLen-1]
					break
				}
			}
		}
	}

	for txhash, value := range unspents {
		key := bytes.NewBuffer(nil)
		key.WriteByte(byte(IX_Unspent))
		txhash.Serialize(key)

		if len(value) == 0 {
			db.BatchDelete(key.Bytes())
		} else {
			unspentArray := ToByteArray(value)
			db.BatchPut(key.Bytes(), unspentArray)
		}
	}

	return nil
}

func (db *ChainStore) RollbackUnspend(b *Block) error {
	unspentPrefix := []byte{byte(IX_Unspent)}
	unspents := make(map[Uint256][]uint16)
	for _, txn := range b.Transactions {
		if txn.TxType == tx.RegisterAsset {
			continue
		}
		// remove all utxos created by this transaction
		txnHash := txn.Hash()
		if err := db.BatchDelete(append(unspentPrefix, txnHash.ToArray()...)); err != nil {
			return err
		}

		for _, input := range txn.UTXOInputs {
			referTxnHash := input.ReferTxID
			referTxnOutIndex := input.ReferTxOutputIndex
			if _, ok := unspents[referTxnHash]; !ok {
				unspentValue, err := db.Get(append(unspentPrefix, referTxnHash.ToArray()...))
				if err != nil {
					return err
				}
				unspents[referTxnHash], err = GetUint16Array(unspentValue)
				if err != nil {
					return err
				}
			}
			unspents[referTxnHash] = append(unspents[referTxnHash], referTxnOutIndex)
		}
	}

	for txhash, value := range unspents {
		key := bytes.NewBuffer(nil)
		key.WriteByte(byte(IX_Unspent))
		txhash.Serialize(key)

		if len(value) == 0 {
			db.BatchDelete(key.Bytes())
		} else {
			unspentArray := ToByteArray(value)
			db.BatchPut(key.Bytes(), unspentArray)
		}
	}

	return nil
}
