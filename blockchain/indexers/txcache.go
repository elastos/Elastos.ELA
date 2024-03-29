// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package indexers

import (
	"io"
	"sync"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
)

const (
	// TrimmingInterval is the interval number for each cache trimming.
	TrimmingInterval = 10000

	// MaxCacheInputsCountPerTransaction is the max inputs count of transaction for cache.
	MaxCacheInputsCountPerTransaction = 100
)

type TxInfo struct {
	BlockHeight uint32
	Txn         interfaces.Transaction
}

func (t *TxInfo) Serialize(w io.Writer) (err error) {
	err = common.WriteUint32(w, t.BlockHeight)
	if err != nil {
		return
	}
	return t.Txn.Serialize(w)
}

func (t *TxInfo) Deserialize(r io.Reader) (err error) {

	t.BlockHeight, err = common.ReadUint32(r)
	if err != nil {
		return
	}
	txn, err := functions.GetTransactionByBytes(r)
	if err != nil {
		return err
	}
	err = txn.Deserialize(r)
	if err != nil {
		return
	}
	t.Txn = txn
	return nil
}

type TxCache struct {
	txns map[common.Uint256]*TxInfo
	sync.RWMutex

	params *config.Configuration
}

func (t *TxCache) Serialize(w io.Writer) (err error) {
	t.RLock()
	defer t.RUnlock()

	count := uint64(len(t.txns))
	err = common.WriteVarUint(w, count)
	if err != nil {
		return err
	}
	for _, txnInfo := range t.txns {
		err = txnInfo.Serialize(w)
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *TxCache) Deserialize(r io.Reader) (err error) {
	count, err := common.ReadVarUint(r, 0)
	if err != nil {
		return err
	}

	t.txns = make(map[common.Uint256]*TxInfo)
	for i := uint64(0); i < count; i++ {
		var txInfo TxInfo
		err = txInfo.Deserialize(r)
		if err != nil {
			return err
		}
		t.setTxn(txInfo.BlockHeight, txInfo.Txn)
	}

	return nil
}

func (t *TxCache) setTxn(height uint32, txn interfaces.Transaction) {
	if t.params.MemoryFirst {
		return
	}

	if len(txn.Inputs()) > MaxCacheInputsCountPerTransaction {
		return
	}

	t.Lock()
	defer t.Unlock()
	t.txns[txn.Hash()] = &TxInfo{
		BlockHeight: height,
		Txn:         txn,
	}
}

func (t *TxCache) deleteTxn(hash common.Uint256) {
	if t.params.MemoryFirst {
		return
	}

	t.Lock()
	defer t.Unlock()
	delete(t.txns, hash)
}

func (t *TxCache) GetTxn(hash common.Uint256) *TxInfo {
	t.RLock()
	defer t.RUnlock()

	return t.txns[hash]
}

func (t *TxCache) trim() {
	if t.params.MemoryFirst {
		return
	}

	t.Lock()
	defer t.Unlock()

	trigger := t.params.TxCacheVolume + TrimmingInterval
	if len(t.txns) > int(trigger) {
		extra := len(t.txns) - int(t.params.TxCacheVolume)
		for k := range t.txns {
			delete(t.txns, k)
			extra--
			if extra < 0 {
				break
			}
		}
	}
}

func NewTxCache(params *config.Configuration) *TxCache {
	return &TxCache{
		txns:   make(map[common.Uint256]*TxInfo),
		params: params,
	}
}
