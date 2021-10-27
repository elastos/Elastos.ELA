// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package blockchain

import (
	"container/list"
	"errors"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/transactions"
	"sync"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
)

const (
	memoryFirstReferenceSize = 5000
)

var (
	maxReferenceSize = 100000
)

type IUTXOCacheStore interface {
	GetTransaction(txID common.Uint256) (*transactions.BaseTransaction, uint32, error)
}

type UTXOCache struct {
	sync.Mutex

	db        IUTXOCacheStore
	inputs    *list.List
	reference map[common2.Input]common2.Output
	txCache   map[common.Uint256]*transactions.BaseTransaction
}

func (up *UTXOCache) insertReference(input *common2.Input, output *common2.Output) {
	if up.inputs.Len() >= maxReferenceSize {
		for e := up.inputs.Front(); e != nil; e = e.Next() {
			up.inputs.Remove(e)
			delete(up.reference, e.Value.(common2.Input))
			if up.inputs.Len() < maxReferenceSize {
				break
			}
		}
	}

	up.inputs.PushBack(*input)
	up.reference[*input] = *output
}

func (up *UTXOCache) GetTxReference(tx *transactions.BaseTransaction) (map[*common2.Input]common2.Output, error) {
	up.Lock()
	defer up.Unlock()

	result := make(map[*common2.Input]common2.Output)
	for _, input := range tx.Inputs {
		if output, exist := up.reference[*input]; exist {
			result[input] = output
		} else {
			prevTx, err := up.getTransaction(input.Previous.TxID)
			if err != nil {
				return nil, errors.New("GetTxReference failed, " + err.Error())
			}
			if int(input.Previous.Index) >= len(prevTx.Outputs) {
				return nil, errors.New("GetTxReference failed, refIdx out of range")
			}

			result[input] = *prevTx.Outputs[input.Previous.Index]
			up.insertReference(input, prevTx.Outputs[input.Previous.Index])
		}
	}

	return result, nil
}

func (up *UTXOCache) GetTransaction(txID common.Uint256) (*transactions.BaseTransaction, error) {
	up.Lock()
	defer up.Unlock()

	return up.getTransaction(txID)
}

func (up *UTXOCache) insertTransaction(txID common.Uint256, tx *transactions.BaseTransaction) {
	if len(up.txCache) > maxReferenceSize {
		for k := range up.txCache {
			delete(up.txCache, k)

			if len(up.txCache) <= maxReferenceSize {
				break
			}
		}
	}

	up.txCache[txID] = tx
}

func (up *UTXOCache) getTransaction(txID common.Uint256) (*transactions.BaseTransaction, error) {
	prevTx, exist := up.txCache[txID]
	if !exist {
		var err error
		prevTx, _, err = up.db.GetTransaction(txID)
		if err != nil {
			return nil, errors.New("transaction not found, " + err.Error())
		}
		up.insertTransaction(txID, prevTx)
	}

	return prevTx, nil
}

func (up *UTXOCache) CleanTxCache() {
	up.Lock()
	defer up.Unlock()

	up.txCache = make(map[common.Uint256]*transactions.BaseTransaction)
}

func (up *UTXOCache) CleanCache() {
	up.Lock()
	defer up.Unlock()

	up.inputs.Init()
	up.reference = make(map[common2.Input]common2.Output)
	up.txCache = make(map[common.Uint256]*transactions.BaseTransaction)
}

func NewUTXOCache(db IUTXOCacheStore, params *config.Params) *UTXOCache {
	if params.NodeProfileStrategy == config.MemoryFirst.String() {
		maxReferenceSize = memoryFirstReferenceSize
	}

	return &UTXOCache{
		db:        db,
		inputs:    list.New(),
		reference: make(map[common2.Input]common2.Output),
		txCache:   make(map[common.Uint256]*transactions.BaseTransaction),
	}
}
