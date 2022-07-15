package blockchain

import (
	"bytes"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/log"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
)

func (c ChainStoreExtend) begin() {
	c.NewBatch()
}

func (c ChainStoreExtend) commit() {
	c.BatchCommit()
}

func (c ChainStoreExtend) rollback() {

}

// key: DataEntryPrefix + height + address
// value: serialized history
func (c ChainStoreExtend) persistTransactionHistory(txhs []common2.TransactionHistory) error {
	c.begin()
	for i, txh := range txhs {
		err := c.doPersistTransactionHistory(uint64(i), txh)
		if err != nil {
			c.rollback()
			log.Fatal("Error persist transaction history")
			return err
		}
		c.deleteMemPoolTx(txh.Txid)
	}
	c.commit()
	return nil
}

func (c ChainStoreExtend) deleteMemPoolTx(txid common.Uint256) {
	MemPoolEx.DeleteMemPoolTx(txid)
}

func (c ChainStoreExtend) persistBestHeight(height uint32) error {
	bestHeight, err := c.GetBestHeightExt()
	if (err == nil && bestHeight < height) || err != nil {
		c.begin()
		err = c.doPersistBestHeight(height)
		if err != nil {
			c.rollback()
			log.Fatal("Error persist best height")
			return err
		}
		c.commit()
	}
	return nil
}

func (c ChainStoreExtend) doPersistBestHeight(height uint32) error {
	key := new(bytes.Buffer)
	key.WriteByte(byte(DataBestHeightPrefix))
	value := new(bytes.Buffer)
	common.WriteUint32(value, height)
	c.BatchPut(key.Bytes(), value.Bytes())
	return nil
}

func (c ChainStoreExtend) persistStoredHeight(height uint32) error {
	c.begin()
	err := c.doPersistStoredHeight(height)
	if err != nil {
		c.rollback()
		log.Fatal("Error persist best height")
		return err
	}
	c.commit()
	return nil
}

func (c ChainStoreExtend) doPersistStoredHeight(height uint32) error {
	key := new(bytes.Buffer)
	key.WriteByte(byte(DataStoredHeightPrefix))
	common.WriteUint32(key, height)
	value := new(bytes.Buffer)
	common.WriteVarBytes(value, []byte{1})
	c.BatchPut(key.Bytes(), value.Bytes())
	return nil
}

func (c ChainStoreExtend) doPersistTransactionHistory(i uint64, history common2.TransactionHistory) error {
	key := new(bytes.Buffer)
	key.WriteByte(byte(DataTxHistoryPrefix))
	err := common.WriteVarBytes(key, history.Address[:])
	if err != nil {
		return err
	}
	err = common.WriteUint64(key, history.Height)
	if err != nil {
		return err
	}
	err = common.WriteUint64(key, i)
	if err != nil {
		return err
	}

	value := new(bytes.Buffer)
	history.Serialize(value)
	c.BatchPut(key.Bytes(), value.Bytes())
	return nil
}
