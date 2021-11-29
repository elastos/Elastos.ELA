// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package blockchain

import (
	"container/list"
	"errors"
	"sync"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
)

const (
	memoryFirstReferenceSize = 5000
)

var (
	MaxReferenceSize = 100000
)

type IUTXOCacheStore interface {
	GetTransaction(txID common.Uint256) (interfaces.Transaction, uint32, error)
}

type UTXOCache struct {
	sync.Mutex

	DB        IUTXOCacheStore
	Inputs    *list.List
	Reference map[common2.Input]common2.Output
	TxCache   map[common.Uint256]interfaces.Transaction
}

func (up *UTXOCache) InsertReference(input *common2.Input, output *common2.Output) {
	if up.Inputs.Len() >= MaxReferenceSize {
		for e := up.Inputs.Front(); e != nil; e = e.Next() {
			up.Inputs.Remove(e)
			delete(up.Reference, e.Value.(common2.Input))
			if up.Inputs.Len() < MaxReferenceSize {
				break
			}
		}
	}

	up.Inputs.PushBack(*input)
	up.Reference[*input] = *output
}

func (up *UTXOCache) GetTxReference(tx interfaces.Transaction) (map[*common2.Input]common2.Output, error) {
	up.Lock()
	defer up.Unlock()

	result := make(map[*common2.Input]common2.Output)
	for _, input := range tx.Inputs() {
		if output, exist := up.Reference[*input]; exist {
			result[input] = output
		} else {
			prevTx, err := up.getTransaction(input.Previous.TxID)
			if err != nil {
				return nil, errors.New("GetTxReference failed, " + err.Error())
			}
			if int(input.Previous.Index) >= len(prevTx.Outputs()) {
				return nil, errors.New("GetTxReference failed, refIdx out of range")
			}

			result[input] = *prevTx.Outputs()[input.Previous.Index]
			up.InsertReference(input, prevTx.Outputs()[input.Previous.Index])
		}
	}

	return result, nil
}

func (up *UTXOCache) GetTransaction(txID common.Uint256) (interfaces.Transaction, error) {
	up.Lock()
	defer up.Unlock()

	return up.getTransaction(txID)
}

func (up *UTXOCache) insertTransaction(txID common.Uint256, tx interfaces.Transaction) {
	if len(up.TxCache) > MaxReferenceSize {
		for k := range up.TxCache {
			delete(up.TxCache, k)

			if len(up.TxCache) <= MaxReferenceSize {
				break
			}
		}
	}

	up.TxCache[txID] = tx
}

func (up *UTXOCache) getTransaction(txID common.Uint256) (interfaces.Transaction, error) {
	prevTx, exist := up.TxCache[txID]
	if !exist {
		var err error
		prevTx, _, err = up.DB.GetTransaction(txID)
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

	up.TxCache = make(map[common.Uint256]interfaces.Transaction)
}

func (up *UTXOCache) CleanCache() {
	up.Lock()
	defer up.Unlock()

	up.Inputs.Init()
	up.Reference = make(map[common2.Input]common2.Output)
	up.TxCache = make(map[common.Uint256]interfaces.Transaction)
}

func NewUTXOCache(db IUTXOCacheStore, params *config.Params) *UTXOCache {
	if params.NodeProfileStrategy == config.MemoryFirst.String() {
		MaxReferenceSize = memoryFirstReferenceSize
	}

	return &UTXOCache{
		DB:        db,
		Inputs:    list.New(),
		Reference: make(map[common2.Input]common2.Output),
		TxCache:   make(map[common.Uint256]interfaces.Transaction),
	}
}
